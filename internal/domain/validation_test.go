package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{"Valid email", "test@example.com", true},
		{"Valid email with subdomain", "user@mail.example.com", true},
		{"Valid email with numbers", "user123@example.com", true},
		{"Valid email with dots", "user.name@example.com", true},
		{"Valid email with plus", "user+tag@example.com", true},
		{"Invalid email - no @", "testexample.com", false},
		{"Invalid email - no domain", "test@", false},
		{"Invalid email - no local part", "@example.com", false},
		{"Invalid email - multiple @", "test@@example.com", false},
		{"Invalid email - empty", "", false},
		{"Invalid email - spaces", "test @example.com", false},
		{"Invalid email - invalid characters", "test$@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateEmail(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		expected bool
	}{
		{"Valid username", "testuser", true},
		{"Valid username with numbers", "user123", true},
		{"Valid username with underscore", "test_user", true},
		{"Valid username with dash", "test-user", true},
		{"Valid minimum length", "abc", true},
		{"Valid maximum length", "abcdefghijklmnopqrstuvwxyz123456", true},
		{"Invalid - too short", "ab", false},
		{"Invalid - too long", "abcdefghijklmnopqrstuvwxyz1234567", false},
		{"Invalid - empty", "", false},
		{"Invalid - spaces", "test user", false},
		{"Invalid - special characters", "test@user", false},
		{"Invalid - starts with number", "123user", false},
		{"Invalid - starts with dash", "-testuser", false},
		{"Invalid - starts with underscore", "_testuser", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateUsernameBool(tt.username)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"Valid password", "Password123!", true},
		{"Valid password - minimum length", "Pass123!", true},
		{"Valid password - maximum length", "ThisIsAVeryLongPasswordThatIsStillValidBecauseItMeetsAllRequirements123!", true},
		{"Invalid - too short", "Pass1!", false},
		{"Invalid - no uppercase", "password123!", false},
		{"Invalid - no lowercase", "PASSWORD123!", false},
		{"Invalid - no numbers", "Password!", false},
		{"Invalid - no special characters", "Password123", false},
		{"Invalid - empty", "", false},
		{"Invalid - only spaces", "        ", false},
		{"Invalid - common password", "password", false},
		{"Invalid - common password", "123456", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePassword(tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected bool
	}{
		{"Valid domain", "example.com", true},
		{"Valid subdomain", "mail.example.com", true},
		{"Valid domain with numbers", "example123.com", true},
		{"Valid domain with dash", "my-domain.com", true},
		{"Valid long domain", "very-long-domain-name.example.com", true},
		{"Invalid - empty", "", false},
		{"Invalid - no TLD", "example", false},
		{"Invalid - starts with dot", ".example.com", false},
		{"Invalid - ends with dot", "example.com.", false},
		{"Invalid - double dots", "example..com", false},
		{"Invalid - spaces", "example .com", false},
		{"Invalid - special characters", "example@.com", false},
		{"Invalid - starts with dash", "-example.com", false},
		{"Invalid - ends with dash", "example-.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDomain(tt.domain)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateTagName(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		expected bool
	}{
		{"Valid tag name", "Important", true},
		{"Valid tag name with spaces", "Very Important", true},
		{"Valid tag name with numbers", "Priority 1", true},
		{"Valid tag name with dash", "Work-Related", true},
		{"Valid tag name with underscore", "Personal_Items", true},
		{"Valid minimum length", "A", true},
		{"Valid maximum length", "This is a very long tag name that should still be valid", true},
		{"Invalid - empty", "", false},
		{"Invalid - only spaces", "   ", false},
		{"Invalid - too long", "This tag name is way too long and exceeds the maximum allowed length for tag names in the system which should not be allowed", false},
		{"Invalid - special characters", "Tag@Name", false},
		{"Invalid - newlines", "Tag\nName", false},
		{"Invalid - tabs", "Tag\tName", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateTagName(tt.tagName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateColorCode(t *testing.T) {
	tests := []struct {
		name      string
		colorCode string
		expected  bool
	}{
		{"Valid hex color - lowercase", "#ff0000", true},
		{"Valid hex color - uppercase", "#FF0000", true},
		{"Valid hex color - mixed case", "#Ff0000", true},
		{"Valid hex color - short form", "#f00", true},
		{"Valid hex color - short form uppercase", "#F00", true},
		{"Invalid - no hash", "ff0000", false},
		{"Invalid - wrong length", "#ff00", false},
		{"Invalid - wrong length", "#ff00000", false},
		{"Invalid - invalid characters", "#gg0000", false},
		{"Invalid - empty", "", false},
		{"Invalid - only hash", "#", false},
		{"Invalid - spaces", "#ff 0000", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateColorCode(tt.colorCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSubject(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		expected bool
	}{
		{"Valid subject", "Test Subject", true},
		{"Valid empty subject", "", true}, // Empty subjects are allowed
		{"Valid subject with numbers", "Meeting 123", true},
		{"Valid subject with special chars", "Re: Important Meeting!", true},
		{"Valid long subject", "This is a very long subject line that contains multiple words and should still be valid", true},
		{"Invalid - too long", "This subject line is extremely long and exceeds the maximum allowed length for email subjects which is typically around 255 characters but this one goes way beyond that limit and should be rejected by the validation function because it's just too long to be practical", false},
		{"Invalid - control characters", "Subject\nwith\nnewlines", false},
		{"Invalid - tabs", "Subject\twith\ttabs", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSubject(tt.subject)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateMessageBody(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected bool
	}{
		{"Valid body", "This is a test message body.", true},
		{"Valid empty body", "", true}, // Empty bodies are allowed
		{"Valid body with newlines", "Line 1\nLine 2\nLine 3", true},
		{"Valid body with special chars", "Hello! How are you? I'm fine. 50% off!", true},
		{"Valid long body", "This is a very long message body that contains multiple paragraphs and should still be valid even though it's quite lengthy.", true},
		{"Invalid - too long", generateLongString(100001), false}, // Assuming max length is 100000
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateMessageBody(tt.body)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name     string
		user     *User
		expected bool
	}{
		{
			"Valid user",
			&User{
				Username: "testuser",
				Email:    "test@example.com",
				Role:     RoleUser,
			},
			true,
		},
		{
			"Invalid username",
			&User{
				Username: "ab", // too short
				Email:    "test@example.com",
				Role:     RoleUser,
			},
			false,
		},
		{
			"Invalid email",
			&User{
				Username: "testuser",
				Email:    "invalid-email",
				Role:     RoleUser,
			},
			false,
		},
		{
			"Invalid role",
			&User{
				Username: "testuser",
				Email:    "test@example.com",
				Role:     "invalid-role",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCreateMailboxRequest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		req      *CreateMailboxRequest
		expected bool
	}{
		{
			"Valid request",
			&CreateMailboxRequest{
				UserID: "user-123",
				Domain: "temp.mail",
			},
			true,
		},
		{
			"Invalid - empty user ID",
			&CreateMailboxRequest{
				UserID: "",
				Domain: "temp.mail",
			},
			false,
		},
		{
			"Invalid - invalid domain",
			&CreateMailboxRequest{
				UserID: "user-123",
				Domain: "invalid-domain",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCreateMessageRequest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		req      *CreateMessageRequest
		expected bool
	}{
		{
			"Valid request",
			&CreateMessageRequest{
				MailboxID: "mailbox-123",
				From:      "sender@example.com",
				To:        []string{"recipient@example.com"},
				Subject:   "Test Subject",
				Body:      "Test body",
			},
			true,
		},
		{
			"Invalid - empty mailbox ID",
			&CreateMessageRequest{
				MailboxID: "",
				From:      "sender@example.com",
				To:        []string{"recipient@example.com"},
				Subject:   "Test Subject",
				Body:      "Test body",
			},
			false,
		},
		{
			"Invalid - invalid from email",
			&CreateMessageRequest{
				MailboxID: "mailbox-123",
				From:      "invalid-email",
				To:        []string{"recipient@example.com"},
				Subject:   "Test Subject",
				Body:      "Test body",
			},
			false,
		},
		{
			"Invalid - empty recipients",
			&CreateMessageRequest{
				MailboxID: "mailbox-123",
				From:      "sender@example.com",
				To:        []string{},
				Subject:   "Test Subject",
				Body:      "Test body",
			},
			false,
		},
		{
			"Invalid - invalid recipient email",
			&CreateMessageRequest{
				MailboxID: "mailbox-123",
				From:      "sender@example.com",
				To:        []string{"invalid-email"},
				Subject:   "Test Subject",
				Body:      "Test body",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Helper function to generate long strings for testing
func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}