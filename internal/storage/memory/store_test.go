package memory

import (
	"testing"
	"time"

	"tempmail/backend/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_MailboxOperations(t *testing.T) {
	store := NewStore(24 * time.Hour)
	userID := "test-user-1"

	// Test SaveMailbox
	mailbox := &domain.Mailbox{
		ID:        "test-mailbox-1",
		UserID:    &userID,
		Address:   "test@temp.mail",
		LocalPart: "test",
		Domain:    "temp.mail",
		Token:     "test-token",
		CreatedAt: time.Now(),
	}

	err := store.SaveMailbox(mailbox)
	require.NoError(t, err)

	// Test GetMailbox
	retrievedMailbox, err := store.GetMailbox("test-mailbox-1")
	require.NoError(t, err)
	assert.Equal(t, mailbox.Address, retrievedMailbox.Address)
	assert.Equal(t, mailbox.Domain, retrievedMailbox.Domain)

	// Test GetMailboxByAddress
	retrievedMailbox, err = store.GetMailboxByAddress("test@temp.mail")
	require.NoError(t, err)
	assert.Equal(t, mailbox.ID, retrievedMailbox.ID)

	// Test ListMailboxesByUserID
	mailboxes := store.ListMailboxesByUserID("test-user-1")
	assert.Len(t, mailboxes, 1)
	assert.Equal(t, mailbox.ID, mailboxes[0].ID)

	// Test DeleteMailbox
	err = store.DeleteMailbox("test-mailbox-1")
	require.NoError(t, err)

	_, err = store.GetMailbox("test-mailbox-1")
	assert.Error(t, err)
}

func TestMemoryStore_MessageOperations(t *testing.T) {
	store := NewStore(24 * time.Hour)
	userID := "test-user-1"

	// Create test mailbox first
	mailbox := &domain.Mailbox{
		ID:        "test-mailbox-1",
		UserID:    &userID,
		Address:   "test@temp.mail",
		LocalPart: "test",
		Domain:    "temp.mail",
		Token:     "test-token",
		CreatedAt: time.Now(),
	}
	err := store.SaveMailbox(mailbox)
	require.NoError(t, err)

	// Test SaveMessage
	message := &domain.Message{
		ID:        "test-message-1",
		MailboxID: "test-mailbox-1",
		From:      "sender@example.com",
		To:        "test@temp.mail",
		Subject:   "Test Message",
		Text:      "This is a test message",
		CreatedAt: time.Now(),
	}

	err = store.SaveMessage(message)
	require.NoError(t, err)

	// Test GetMessage
	retrievedMessage, err := store.GetMessage("test-mailbox-1", "test-message-1")
	require.NoError(t, err)
	assert.Equal(t, message.Subject, retrievedMessage.Subject)
	assert.Equal(t, message.From, retrievedMessage.From)

	// Test ListMessages
	messages, err := store.ListMessages("test-mailbox-1")
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, message.ID, messages[0].ID)

	// Test MarkMessageRead
	err = store.MarkMessageRead("test-mailbox-1", "test-message-1")
	require.NoError(t, err)

	retrievedMessage, err = store.GetMessage("test-mailbox-1", "test-message-1")
	require.NoError(t, err)
	assert.True(t, retrievedMessage.IsRead)
}

func TestMemoryStore_Cleanup(t *testing.T) {
	store := NewStore(1 * time.Millisecond) // Very short TTL
	userID := "test-user-1"

	// Create test mailbox with expiration
	expiresAt := time.Now().Add(-1 * time.Hour) // Already expired
	mailbox := &domain.Mailbox{
		ID:        "test-mailbox-1",
		UserID:    &userID,
		Address:   "test@temp.mail",
		LocalPart: "test",
		Domain:    "temp.mail",
		Token:     "test-token",
		CreatedAt: time.Now(),
		ExpiresAt: &expiresAt,
	}

	err := store.SaveMailbox(mailbox)
	require.NoError(t, err)

	// Run cleanup
	count, err := store.DeleteExpiredMailboxes()
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify mailbox is deleted
	_, err = store.GetMailbox("test-mailbox-1")
	assert.Error(t, err)
}