package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

// PlatformUtils 平台兼容性工具
type PlatformUtils struct{}

// NewPlatformUtils 创建平台工具实例
func NewPlatformUtils() *PlatformUtils {
	return &PlatformUtils{}
}

// SanitizeFilename 清理文件名，确保跨平台兼容
func (p *PlatformUtils) SanitizeFilename(filename string) string {
	// 1. 移除路径分隔符
	filename = filepath.Base(filename)

	// 2. 移除或替换不允许的字符
	invalidChars := p.getInvalidChars()
	for _, char := range invalidChars {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	// 3. 移除控制字符
	filename = p.removeControlChars(filename)

	// 4. 限制长度
	filename = p.limitLength(filename, 200)

	// 5. 确保不为空
	if filename == "" {
		filename = "unnamed"
	}

	// 6. 移除前后空格和点
	filename = strings.Trim(filename, " .")

	return filename
}

// getInvalidChars 获取当前平台不允许的字符
func (p *PlatformUtils) getInvalidChars() []string {
	switch runtime.GOOS {
	case "windows":
		// Windows 不允许的字符
		return []string{"<", ">", ":", "\"", "|", "?", "*", "\\", "/", "\x00"}
	case "darwin", "linux":
		// Unix-like 系统不允许的字符
		return []string{"/", "\x00"}
	default:
		// 保守处理，移除所有可能的问题字符
		return []string{"<", ">", ":", "\"", "|", "?", "*", "\\", "/", "\x00"}
	}
}

// removeControlChars 移除控制字符
func (p *PlatformUtils) removeControlChars(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return -1 // 移除字符
		}
		return r
	}, s)
}

// limitLength 限制字符串长度
func (p *PlatformUtils) limitLength(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// 保留扩展名
	ext := filepath.Ext(s)
	nameWithoutExt := strings.TrimSuffix(s, ext)

	// 计算可用长度
	availableLen := maxLen - len(ext)
	if availableLen <= 0 {
		return ext
	}

	// 截断并添加扩展名
	truncated := nameWithoutExt[:availableLen]
	return truncated + ext
}

// ValidatePath 验证路径是否安全
func (p *PlatformUtils) ValidatePath(path string) error {
	// 1. 检查路径长度
	if len(path) > 2000 { // 保守的长度限制
		return fmt.Errorf("path too long: %d characters", len(path))
	}

	// 2. 检查是否包含路径遍历
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// 3. 检查绝对路径（如果配置不允许）
	// 这里可以根据配置决定是否允许绝对路径

	return nil
}

// GetMaxPathLength 获取当前平台的最大路径长度
func (p *PlatformUtils) GetMaxPathLength() int {
	switch runtime.GOOS {
	case "windows":
		// Windows 10 支持长路径，但为了兼容性使用保守值
		return 200
	case "darwin", "linux":
		// Unix-like 系统通常支持更长的路径
		return 400
	default:
		// 保守值
		return 200
	}
}

// IsCaseSensitive 检查当前文件系统是否大小写敏感
func (p *PlatformUtils) IsCaseSensitive() bool {
	switch runtime.GOOS {
	case "windows":
		return false
	case "darwin", "linux":
		return true
	default:
		// 保守假设为大小写敏感
		return true
	}
}

// NormalizePath 标准化路径
func (p *PlatformUtils) NormalizePath(path string) string {
	// 1. 转换为绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		// 如果转换失败，返回原路径
		return path
	}

	// 2. 清理路径
	cleanPath := filepath.Clean(absPath)

	// 3. 如果文件系统不区分大小写，转换为小写
	if !p.IsCaseSensitive() {
		cleanPath = strings.ToLower(cleanPath)
	}

	return cleanPath
}

// GetPathSeparator 获取当前平台的路径分隔符
func (p *PlatformUtils) GetPathSeparator() string {
	return string(filepath.Separator)
}

// JoinPath 安全地连接路径
func (p *PlatformUtils) JoinPath(elem ...string) string {
	// 1. 使用 filepath.Join
	path := filepath.Join(elem...)

	// 2. 验证路径
	if err := p.ValidatePath(path); err != nil {
		// 如果路径不安全，返回错误或使用安全路径
		return ""
	}

	return path
}

// GetTempDir 获取临时目录
func (p *PlatformUtils) GetTempDir() string {
	return filepath.Join(os.TempDir(), "tempmail-filesystem")
}

// IsValidFilename 检查文件名是否有效
func (p *PlatformUtils) IsValidFilename(filename string) bool {
	// 1. 不能为空
	if filename == "" {
		return false
	}

	// 2. 不能只包含空格和点
	trimmed := strings.Trim(filename, " .")
	if trimmed == "" {
		return false
	}

	// 3. 不能包含不允许的字符
	invalidChars := p.getInvalidChars()
	for _, char := range invalidChars {
		if strings.Contains(filename, char) {
			return false
		}
	}

	// 4. 长度检查
	if len(filename) > p.GetMaxPathLength() {
		return false
	}

	return true
}



