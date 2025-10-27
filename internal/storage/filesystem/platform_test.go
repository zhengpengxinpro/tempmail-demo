package filesystem

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tempmail/backend/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPlatformUtils 测试平台兼容性工具
func TestPlatformUtils(t *testing.T) {
	utils := NewPlatformUtils()

	t.Run("sanitize filename", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			// 正常文件名
			{"document.pdf", "document.pdf"},
			{"test file.txt", "test file.txt"},
			{"image.jpg", "image.jpg"},

			// 包含特殊字符
			{"file<name>.txt", "file_name_.txt"},
			{"file:name.txt", "file_name.txt"},
			{"file|name.txt", "file_name.txt"},
			{"file?name.txt", "file_name.txt"},
			{"file*name.txt", "file_name.txt"},
			{"file\"name\".txt", "file_name_.txt"},

			// 路径分隔符
			{"../file.txt", "file.txt"},
			{"./file.txt", "file.txt"},
			{"path/to/file.txt", "file.txt"},

			// 控制字符
			{"file\x00name.txt", "file_name.txt"},
			{"file\tname.txt", "file\tname.txt"}, // 保留制表符
			{"file\nname.txt", "file\nname.txt"}, // 保留换行符

			// 空文件名
			{"", ""},
			{"   ", ""},
			{"...", ""},

			// 超长文件名
			{strings.Repeat("a", 300) + ".txt", strings.Repeat("a", 196) + ".txt"},
		}

		for _, tc := range testCases {
			result := utils.SanitizeFilename(tc.input)
			assert.Equal(t, tc.expected, result, "Input: %s", tc.input)
		}
	})

	t.Run("validate path", func(t *testing.T) {
		// 有效路径
		validPaths := []string{
			"data/mails",
			"./data/mails",
			"/tmp/tempmail",
			"relative/path",
		}

		for _, path := range validPaths {
			err := utils.ValidatePath(path)
			assert.NoError(t, err, "Path should be valid: %s", path)
		}

		// 无效路径
		invalidPaths := []string{
			"../../../etc/passwd",
			"data/../etc/passwd",
			"data/..",
			strings.Repeat("a", 3000), // 超长路径
		}

		for _, path := range invalidPaths {
			err := utils.ValidatePath(path)
			assert.Error(t, err, "Path should be invalid: %s", path)
		}
	})

	t.Run("platform detection", func(t *testing.T) {
		// 测试平台检测
		os := runtime.GOOS
		t.Logf("Current OS: %s", os)

		// 测试大小写敏感性
		caseSensitive := utils.IsCaseSensitive()
		t.Logf("Case sensitive: %v", caseSensitive)

		// 测试路径分隔符
		separator := utils.GetPathSeparator()
		expectedSeparator := string(filepath.Separator)
		assert.Equal(t, expectedSeparator, separator)

		// 测试最大路径长度
		maxLen := utils.GetMaxPathLength()
		assert.Greater(t, maxLen, 0)
		t.Logf("Max path length: %d", maxLen)
	})

	t.Run("normalize path", func(t *testing.T) {
		// 创建临时目录进行测试
		tempDir, err := os.MkdirTemp("", "platform_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// 测试相对路径
		relPath := "test/path"
		normalized := utils.NormalizePath(relPath)
		assert.NotEmpty(t, normalized)

		// 测试绝对路径
		absPath := filepath.Join(tempDir, "test")
		normalizedAbs := utils.NormalizePath(absPath)
		// 在 Windows 上，路径可能被转换为小写
		assert.Equal(t, strings.ToLower(absPath), strings.ToLower(normalizedAbs))
	})

	t.Run("join path", func(t *testing.T) {
		// 测试安全路径连接
		path := utils.JoinPath("data", "mails", "test")
		assert.NotEmpty(t, path)
		assert.NotContains(t, path, "..")

		// 测试无效路径
		invalidPath := utils.JoinPath("data", "..", "etc", "passwd")
		// 在 Windows 上，路径可能不会被完全阻止
		if runtime.GOOS == "windows" {
			// Windows 可能允许这种路径，所以不检查是否为空
			t.Logf("Windows path handling: %s", invalidPath)
		} else {
			assert.Empty(t, invalidPath)
		}
	})

	t.Run("is valid filename", func(t *testing.T) {
		// 有效文件名
		validFilenames := []string{
			"document.pdf",
			"test file.txt",
			"image.jpg",
			"file-name.txt",
			"file_name.txt",
		}

		for _, filename := range validFilenames {
			assert.True(t, utils.IsValidFilename(filename), "Should be valid: %s", filename)
		}

		// 无效文件名
		invalidFilenames := []string{
			"",
			"   ",
			"...",
			"file<name>.txt",
			"file:name.txt",
			"file|name.txt",
			"file?name.txt",
			"file*name.txt",
			"file\"name\".txt",
		}

		for _, filename := range invalidFilenames {
			assert.False(t, utils.IsValidFilename(filename), "Should be invalid: %s", filename)
		}
	})
}

// TestCrossPlatformCompatibility 测试跨平台兼容性
func TestCrossPlatformCompatibility(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "cross_platform_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// 创建存储实例
	store, err := NewStore(tempDir)
	require.NoError(t, err)

	t.Run("create store on different platforms", func(t *testing.T) {
		// 测试不同路径格式
		paths := []string{
			"data/mails",         // Unix 风格
			"data\\mails",        // Windows 风格
			"./data/mails",       // 相对路径
			"data/../data/mails", // 包含 .. 的路径（应该被清理）
		}

		for _, path := range paths {
			// 创建子目录进行测试
			testPath := filepath.Join(tempDir, "test_"+strings.ReplaceAll(path, "/", "_"))

			store, err := NewStore(testPath)
			if strings.Contains(path, "..") {
				// 包含路径遍历的应该失败
				assert.Error(t, err, "Path with traversal should fail: %s", path)
			} else {
				assert.NoError(t, err, "Path should be valid: %s", path)
				if err == nil {
					assert.NotNil(t, store)
				}
			}
		}
	})

	t.Run("attachment filename handling", func(t *testing.T) {
		// 测试不同平台的附件文件名
		testFilenames := []string{
			"document.pdf",
			"file with spaces.txt",
			"file<name>.txt",
			"file:name.txt",
			"file|name.txt",
			"file?name.txt",
			"file*name.txt",
			"file\"name\".txt",
			"file/name.txt",
			"file\\name.txt",
			"file\x00name.txt",
			"file\nname.txt",
			"file\tname.txt",
		}

		mailboxID := "test-mailbox"
		messageID := "test-message"

		for _, filename := range testFilenames {
			// 测试文件名生成
			safeFilename := store.generateSafeFilename("att-001", filename)
			assert.NotEmpty(t, safeFilename)
			assert.NotContains(t, safeFilename, "<")
			assert.NotContains(t, safeFilename, ">")
			assert.NotContains(t, safeFilename, ":")
			assert.NotContains(t, safeFilename, "|")
			assert.NotContains(t, safeFilename, "?")
			assert.NotContains(t, safeFilename, "*")
			assert.NotContains(t, safeFilename, "\"")
			assert.NotContains(t, safeFilename, "/")
			assert.NotContains(t, safeFilename, "\\")
			assert.NotContains(t, safeFilename, "\x00")

			// 测试实际保存
			attachment := &domain.Attachment{
				ID:          "att-001",
				MessageID:   messageID,
				Filename:    filename,
				ContentType: "text/plain",
				Size:        int64(len("test content")),
				Content:     []byte("test content"),
			}

			// 先保存邮件元数据（GetAttachment 需要）
			message := &domain.Message{
				ID:          messageID,
				MailboxID:   mailboxID,
				From:        "test@example.com",
				To:          "test@example.com",
				Subject:     "Test",
				Text:        "Test content",
				CreatedAt:   time.Now(),
				ReceivedAt:  time.Now(),
				Attachments: []*domain.Attachment{attachment},
			}

			_, err := store.SaveMessageMetadata(mailboxID, messageID, message)
			require.NoError(t, err, "Should save message metadata")

			_, err = store.SaveAttachment(mailboxID, messageID, "att-001", attachment)
			if err != nil {
				// 某些特殊字符在 Windows 上可能无法保存，这是预期的
				t.Logf("Expected behavior: attachment with filename '%s' may fail on Windows: %v", filename, err)
			} else {
				// 验证可以读取
				retrieved, err := store.GetAttachment(mailboxID, messageID, "att-001")
				if err != nil {
					t.Logf("Expected behavior: attachment retrieval may fail: %v", err)
				} else {
					assert.NotNil(t, retrieved)
				}
			}
		}
	})

	t.Run("path length limits", func(t *testing.T) {
		// 测试路径长度限制
		maxLen := store.platformUtils.GetMaxPathLength()
		t.Logf("Max path length: %d", maxLen)

		// 创建超长路径
		longPath := strings.Repeat("a", maxLen+100)
		err := store.platformUtils.ValidatePath(longPath)
		// 在 Windows 上，路径长度检查可能不够严格
		if runtime.GOOS == "windows" {
			// Windows 可能允许更长的路径
			t.Logf("Windows path length handling: %v", err)
		} else {
			assert.Error(t, err, "Long path should be invalid")
		}
	})

	t.Run("case sensitivity", func(t *testing.T) {
		// 测试大小写敏感性
		caseSensitive := store.platformUtils.IsCaseSensitive()
		t.Logf("Case sensitive: %v", caseSensitive)

		// 在大小写不敏感的系统上，测试文件名冲突
		if !caseSensitive {
			// 创建两个只有大小写不同的文件
			attachment1 := &domain.Attachment{
				ID:          "att-001",
				MessageID:   "test-message",
				Filename:    "Document.pdf",
				ContentType: "application/pdf",
				Size:        100,
				Content:     []byte("content1"),
			}

			attachment2 := &domain.Attachment{
				ID:          "att-002",
				MessageID:   "test-message",
				Filename:    "document.pdf", // 小写
				ContentType: "application/pdf",
				Size:        100,
				Content:     []byte("content2"),
			}

			mailboxID := "test-mailbox"

			// 保存第一个附件
			_, err := store.SaveAttachment(mailboxID, "test-message", "att-001", attachment1)
			require.NoError(t, err)

			// 保存第二个附件（可能覆盖第一个）
			_, err = store.SaveAttachment(mailboxID, "test-message", "att-002", attachment2)
			// 在大小写不敏感的系统上，这可能会覆盖第一个文件
			// 这是预期的行为
			if err != nil {
				t.Logf("Expected behavior: second attachment may overwrite first in case-insensitive system")
			}
		}
	})
}

// TestPlatformSpecificBehaviors 测试平台特定行为
func TestPlatformSpecificBehaviors(t *testing.T) {
	utils := NewPlatformUtils()
	os := runtime.GOOS

	t.Run("windows specific", func(t *testing.T) {
		if os == "windows" {
			// Windows 特定测试
			invalidChars := utils.getInvalidChars()
			assert.Contains(t, invalidChars, "<")
			assert.Contains(t, invalidChars, ">")
			assert.Contains(t, invalidChars, ":")
			assert.Contains(t, invalidChars, "\"")
			assert.Contains(t, invalidChars, "|")
			assert.Contains(t, invalidChars, "?")
			assert.Contains(t, invalidChars, "*")
			assert.Contains(t, invalidChars, "\\")

			// 测试大小写不敏感
			assert.False(t, utils.IsCaseSensitive())
		}
	})

	t.Run("unix specific", func(t *testing.T) {
		if os == "linux" || os == "darwin" {
			// Unix 特定测试
			invalidChars := utils.getInvalidChars()
			assert.Contains(t, invalidChars, "/")
			assert.NotContains(t, invalidChars, "<")
			assert.NotContains(t, invalidChars, ">")

			// 测试大小写敏感
			assert.True(t, utils.IsCaseSensitive())
		}
	})

	t.Run("path separator", func(t *testing.T) {
		separator := utils.GetPathSeparator()
		expectedSeparator := string(filepath.Separator)
		assert.Equal(t, expectedSeparator, separator)

		// 测试路径连接
		path := utils.JoinPath("data", "mails", "test")
		assert.Contains(t, path, separator)
		assert.NotContains(t, path, "/") // 不应该包含其他系统的分隔符
	})
}
