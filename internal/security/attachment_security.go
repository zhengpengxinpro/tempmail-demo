package security

import (
	"bytes"
	"io"
	"mime"
	"path/filepath"
	"strings"
)

// AttachmentSecurity 附件安全检查器
type AttachmentSecurity struct {
	// 允许的文件类型
	allowedMimeTypes map[string]bool

	// 最大文件大小（字节）
	maxFileSize int64

	// 危险文件扩展名
	dangerousExtensions map[string]bool
}

// NewAttachmentSecurity 创建附件安全检查器
func NewAttachmentSecurity() *AttachmentSecurity {
	return &AttachmentSecurity{
		allowedMimeTypes: map[string]bool{
			"text/plain":                   true,
			"text/html":                    true,
			"text/css":                     true,
			"application/json":             true,
			"application/pdf":              true,
			"image/jpeg":                   true,
			"image/png":                    true,
			"image/gif":                    true,
			"image/webp":                   true,
			"application/zip":              true,
			"application/x-zip-compressed": true,
		},
		maxFileSize: 10 * 1024 * 1024, // 10MB
		dangerousExtensions: map[string]bool{
			".exe": true,
			".bat": true,
			".cmd": true,
			".scr": true,
			".pif": true,
			".com": true,
			".vbs": true,
			".js":  true,
			".jar": true,
			".php": true,
			".asp": true,
			".jsp": true,
		},
	}
}

// CheckAttachment 检查附件安全性
func (as *AttachmentSecurity) CheckAttachment(filename string, content io.Reader, mimeType string) (bool, string) {
	// 检查文件扩展名
	if dangerous, reason := as.checkFileExtension(filename); dangerous {
		return false, reason
	}

	// 检查 MIME 类型
	if allowed, reason := as.checkMimeType(mimeType); !allowed {
		return false, reason
	}

	// 检查文件大小
	if tooLarge, reason := as.checkFileSize(content); tooLarge {
		return false, reason
	}

	// 检查文件内容
	if malicious, reason := as.checkFileContent(content, mimeType); malicious {
		return false, reason
	}

	return true, ""
}

// checkFileExtension 检查文件扩展名
func (as *AttachmentSecurity) checkFileExtension(filename string) (bool, string) {
	ext := strings.ToLower(filepath.Ext(filename))

	if as.dangerousExtensions[ext] {
		return true, "Dangerous file extension: " + ext
	}

	return false, ""
}

// checkMimeType 检查 MIME 类型
func (as *AttachmentSecurity) checkMimeType(mimeType string) (bool, string) {
	// 解析 MIME 类型
	mediaType, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		return false, "Invalid MIME type: " + mimeType
	}

	if !as.allowedMimeTypes[mediaType] {
		return false, "Disallowed MIME type: " + mediaType
	}

	return true, ""
}

// checkFileSize 检查文件大小
func (as *AttachmentSecurity) checkFileSize(content io.Reader) (bool, string) {
	// 读取文件内容到缓冲区
	buf := make([]byte, as.maxFileSize+1)
	n, err := content.Read(buf)

	if err != nil && err != io.EOF {
		return true, "Error reading file: " + err.Error()
	}

	if int64(n) > as.maxFileSize {
		return true, "File too large: exceeds " + string(rune(as.maxFileSize)) + " bytes"
	}

	return false, ""
}

// checkFileContent 检查文件内容
func (as *AttachmentSecurity) checkFileContent(content io.Reader, mimeType string) (bool, string) {
	// 读取文件头部
	header := make([]byte, 512)
	n, err := content.Read(header)
	if err != nil && err != io.EOF {
		return true, "Error reading file header: " + err.Error()
	}

	// 检查文件魔数
	if malicious, reason := as.checkFileMagic(header[:n]); malicious {
		return true, reason
	}

	// 检查文本文件中的恶意内容
	if strings.HasPrefix(mimeType, "text/") {
		if malicious, reason := as.checkTextContent(string(header[:n])); malicious {
			return true, reason
		}
	}

	return false, ""
}

// checkFileMagic 检查文件魔数
func (as *AttachmentSecurity) checkFileMagic(header []byte) (bool, string) {
	// 检查可执行文件魔数
	executableSignatures := [][]byte{
		{0x4D, 0x5A},             // PE executable
		{0x7F, 0x45, 0x4C, 0x46}, // ELF executable
		{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O executable
		{0xCE, 0xFA, 0xED, 0xFE}, // Mach-O executable (reverse)
	}

	for _, sig := range executableSignatures {
		if bytes.HasPrefix(header, sig) {
			return true, "Executable file detected"
		}
	}

	return false, ""
}

// checkTextContent 检查文本内容
func (as *AttachmentSecurity) checkTextContent(content string) (bool, string) {
	// 检查脚本标签
	if strings.Contains(strings.ToLower(content), "<script") {
		return true, "Script tag detected in text file"
	}

	// 检查 JavaScript 代码
	if strings.Contains(strings.ToLower(content), "javascript:") {
		return true, "JavaScript code detected in text file"
	}

	return false, ""
}
