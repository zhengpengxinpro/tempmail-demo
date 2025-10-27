package memory

import (
	"fmt"
	"testing"
	"time"

	"tempmail/backend/internal/domain"
)

func BenchmarkMemoryStore_SaveMailbox(b *testing.B) {
	store := NewStore(24 * time.Hour)
	userID := "test-user"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mailbox := &domain.Mailbox{
			ID:        fmt.Sprintf("mailbox-%d", i),
			UserID:    &userID,
			Address:   fmt.Sprintf("test%d@temp.mail", i),
			LocalPart: fmt.Sprintf("test%d", i),
			Domain:    "temp.mail",
			Token:     fmt.Sprintf("token-%d", i),
			CreatedAt: time.Now(),
		}
		store.SaveMailbox(mailbox)
	}
}

func BenchmarkMemoryStore_GetMailbox(b *testing.B) {
	store := NewStore(24 * time.Hour)
	userID := "test-user"
	
	// Pre-populate with test data
	for i := 0; i < 1000; i++ {
		mailbox := &domain.Mailbox{
			ID:        fmt.Sprintf("mailbox-%d", i),
			UserID:    &userID,
			Address:   fmt.Sprintf("test%d@temp.mail", i),
			LocalPart: fmt.Sprintf("test%d", i),
			Domain:    "temp.mail",
			Token:     fmt.Sprintf("token-%d", i),
			CreatedAt: time.Now(),
		}
		store.SaveMailbox(mailbox)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := fmt.Sprintf("mailbox-%d", i%1000)
		store.GetMailbox(id)
	}
}

func BenchmarkMemoryStore_SaveMessage(b *testing.B) {
	store := NewStore(24 * time.Hour)
	userID := "test-user"
	
	// Create test mailbox
	mailbox := &domain.Mailbox{
		ID:        "test-mailbox",
		UserID:    &userID,
		Address:   "test@temp.mail",
		LocalPart: "test",
		Domain:    "temp.mail",
		Token:     "test-token",
		CreatedAt: time.Now(),
	}
	store.SaveMailbox(mailbox)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		message := &domain.Message{
			ID:        fmt.Sprintf("message-%d", i),
			MailboxID: "test-mailbox",
			From:      "sender@example.com",
			To:        "test@temp.mail",
			Subject:   fmt.Sprintf("Test Message %d", i),
			Text:      "This is a test message body",
			CreatedAt: time.Now(),
		}
		store.SaveMessage(message)
	}
}

func BenchmarkMemoryStore_ConcurrentAccess(b *testing.B) {
	store := NewStore(24 * time.Hour)
	userID := "test-user"
	
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			mailbox := &domain.Mailbox{
				ID:        fmt.Sprintf("mailbox-%d", i),
				UserID:    &userID,
				Address:   fmt.Sprintf("test%d@temp.mail", i),
				LocalPart: fmt.Sprintf("test%d", i),
				Domain:    "temp.mail",
				Token:     fmt.Sprintf("token-%d", i),
				CreatedAt: time.Now(),
			}
			store.SaveMailbox(mailbox)
			store.GetMailbox(mailbox.ID)
			i++
		}
	})
}