package security

import (
	"regexp"
	"strings"
)

// ContentFilter 内容过滤器
type ContentFilter struct {
	// 恶意内容模式
	maliciousPatterns []*regexp.Regexp

	// 垃圾邮件关键词
	spamKeywords []string

	// 危险文件扩展名
	dangerousExtensions []string
}

// NewContentFilter 创建内容过滤器
func NewContentFilter() *ContentFilter {
	return &ContentFilter{
		maliciousPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
			regexp.MustCompile(`(?i)javascript:`),
			regexp.MustCompile(`(?i)onload\s*=`),
			regexp.MustCompile(`(?i)onerror\s*=`),
			regexp.MustCompile(`(?i)eval\s*\(`),
			regexp.MustCompile(`(?i)document\.cookie`),
			regexp.MustCompile(`(?i)<iframe[^>]*>`),
			regexp.MustCompile(`(?i)<object[^>]*>`),
			regexp.MustCompile(`(?i)<embed[^>]*>`),
		},
		spamKeywords: []string{
			"viagra", "casino", "lottery", "winner", "congratulations",
			"free money", "click here", "limited time", "act now",
			"guaranteed", "no risk", "earn money", "work from home",
		},
		dangerousExtensions: []string{
			".exe", ".bat", ".cmd", ".scr", ".pif", ".com",
			".vbs", ".js", ".jar", ".zip", ".rar", ".7z",
		},
	}
}

// FilterEmail 过滤邮件内容
func (cf *ContentFilter) FilterEmail(content string) (bool, string) {
	// 检查恶意内容
	if malicious, reason := cf.checkMaliciousContent(content); malicious {
		return false, reason
	}

	// 检查垃圾邮件
	if spam, reason := cf.checkSpamContent(content); spam {
		return false, reason
	}

	return true, ""
}

// checkMaliciousContent 检查恶意内容
func (cf *ContentFilter) checkMaliciousContent(content string) (bool, string) {
	for _, pattern := range cf.maliciousPatterns {
		if pattern.MatchString(content) {
			return true, "Malicious content detected: " + pattern.String()
		}
	}
	return false, ""
}

// checkSpamContent 检查垃圾邮件内容
func (cf *ContentFilter) checkSpamContent(content string) (bool, string) {
	contentLower := strings.ToLower(content)

	spamCount := 0
	for _, keyword := range cf.spamKeywords {
		if strings.Contains(contentLower, keyword) {
			spamCount++
		}
	}

	if spamCount >= 3 {
		return true, "Spam content detected: multiple spam keywords found"
	}

	return false, ""
}
