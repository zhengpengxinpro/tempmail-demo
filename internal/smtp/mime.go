package smtp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"

	"tempmail/backend/internal/domain"
)

// ParsedEmail 表示解析后的邮件内容。
type ParsedEmail struct {
	Subject     string
	From        string
	To          string
	Text        string
	HTML        string
	Attachments []*domain.Attachment
}

// ParseEmail 解析邮件，提取文本、HTML 和附件。
func ParseEmail(rawEmail []byte) (*ParsedEmail, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(rawEmail))
	if err != nil {
		return nil, fmt.Errorf("parse mail: %w", err)
	}

	parsed := &ParsedEmail{
		Subject:     decodeHeader(msg.Header.Get("Subject")),
		From:        msg.Header.Get("From"),
		To:          msg.Header.Get("To"),
		Attachments: make([]*domain.Attachment, 0),
	}

	contentType := msg.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		// 如果没有 Content-Type 或解析失败，当作纯文本处理
		body, _ := io.ReadAll(msg.Body)
		parsed.Text = string(body)
		return parsed, nil
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		// 多部分邮件，需要解析各个部分
		boundary := params["boundary"]
		if boundary == "" {
			return nil, fmt.Errorf("multipart message without boundary")
		}

		mr := multipart.NewReader(msg.Body, boundary)
		if err := parseMultipart(mr, parsed); err != nil {
			return nil, fmt.Errorf("parse multipart: %w", err)
		}
	} else {
		// 单部分邮件
		body, err := decodeBody(msg.Body, msg.Header.Get("Content-Transfer-Encoding"), params["charset"])
		if err != nil {
			return nil, fmt.Errorf("decode body: %w", err)
		}

		if strings.HasPrefix(mediaType, "text/html") {
			parsed.HTML = body
		} else {
			parsed.Text = body
		}
	}

	return parsed, nil
}

// parseMultipart 递归解析多部分邮件。
func parseMultipart(mr *multipart.Reader, parsed *ParsedEmail) error {
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		contentType := part.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			mediaType = "text/plain"
		}

		// 检查是否是附件
		disposition := part.Header.Get("Content-Disposition")
		if disposition != "" {
			dispType, dispParams, _ := mime.ParseMediaType(disposition)
			if dispType == "attachment" || dispType == "inline" {
				// 这是一个附件
				filename := dispParams["filename"]
				if filename == "" {
					filename = params["name"]
				}
				if filename == "" {
					filename = "unnamed"
				}

				// 解码文件名
				filename = decodeHeader(filename)

				// 读取附件内容
				content, err := io.ReadAll(part)
				if err != nil {
					continue
				}

				// 解码附件内容（如果是 base64 编码）
				encoding := part.Header.Get("Content-Transfer-Encoding")
				if strings.ToLower(encoding) == "base64" {
					decoded, err := base64.StdEncoding.DecodeString(string(content))
					if err == nil {
						content = decoded
					}
				}

				attachment := &domain.Attachment{
					ID:          uuid.NewString(),
					Filename:    filename,
					ContentType: mediaType,
					Size:        int64(len(content)),
					Content:     content,
				}
				parsed.Attachments = append(parsed.Attachments, attachment)
				continue
			}
		}

		// 处理嵌套的 multipart
		if strings.HasPrefix(mediaType, "multipart/") {
			boundary := params["boundary"]
			if boundary != "" {
				nestedReader := multipart.NewReader(part, boundary)
				if err := parseMultipart(nestedReader, parsed); err != nil {
					return err
				}
			}
			continue
		}

		// 处理文本内容
		body, err := decodeBody(part, part.Header.Get("Content-Transfer-Encoding"), params["charset"])
		if err != nil {
			continue
		}

		if strings.HasPrefix(mediaType, "text/html") {
			if parsed.HTML == "" {
				parsed.HTML = body
			}
		} else if strings.HasPrefix(mediaType, "text/plain") {
			if parsed.Text == "" {
				parsed.Text = body
			}
		}
	}

	return nil
}

// decodeBody 根据编码方式解码邮件体。
func decodeBody(reader io.Reader, transferEncoding string, charset string) (string, error) {
	transferEncoding = strings.ToLower(strings.TrimSpace(transferEncoding))

	var decoded io.Reader = reader

	switch transferEncoding {
	case "base64":
		decoded = base64.NewDecoder(base64.StdEncoding, reader)
	case "quoted-printable":
		decoded = quotedprintable.NewReader(reader)
	case "7bit", "8bit", "binary", "":
		// 不需要解码
		decoded = reader
	default:
		// 未知编码，尝试直接读取
		decoded = reader
	}

	body, err := io.ReadAll(decoded)
	if err != nil {
		return "", err
	}

	// 字符集转换
	charset = strings.ToLower(strings.TrimSpace(charset))
	if charset != "" && charset != "utf-8" && charset != "us-ascii" {
		if enc := getCharsetEncoding(charset); enc != nil {
			decoder := enc.NewDecoder()
			converted, _, err := transform.Bytes(decoder, body)
			if err == nil {
				body = converted
			}
		}
	}

	return string(body), nil
}

// getCharsetEncoding 根据字符集名称返回编码器
func getCharsetEncoding(charset string) encoding.Encoding {
	switch charset {
	case "gb2312", "gbk", "gb18030":
		return simplifiedchinese.GBK
	case "big5":
		return traditionalchinese.Big5
	case "iso-2022-jp", "shift_jis", "euc-jp":
		return japanese.ShiftJIS
	case "euc-kr", "ks_c_5601-1987":
		return korean.EUCKR
	default:
		return nil
	}
}
