package storage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-memdb"
)

func (d *Storage) SaveChatMessage(msg *ChatMessage) error {
	if msg == nil {
		return fmt.Errorf("chat message is nil")
	}

	txn := d.db.Txn(true)
	defer txn.Abort()

	stored := cloneChatMessage(msg)
	stored.SessionID = strings.TrimSpace(stored.SessionID)
	if stored.SessionID == "" {
		return fmt.Errorf("session id required")
	}
	if strings.TrimSpace(stored.Role) == "" {
		return fmt.Errorf("role required")
	}
	if stored.ID == "" {
		stored.ID = generateID("chat")
	}
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = time.Now()
	}

	if _, err := d.getSessionRecordTxn(txn, stored.SessionID); err != nil {
		return err
	}
	if err := txn.Insert(tableChat, chatMessageToRecord(stored)); err != nil {
		return fmt.Errorf("failed to save chat message: %w", err)
	}

	msg.ID = stored.ID
	msg.SessionID = stored.SessionID
	msg.Role = stored.Role
	msg.Content = stored.Content
	msg.CreatedAt = stored.CreatedAt

	txn.Commit()
	return nil
}

func (d *Storage) ListChatMessages(sessionID string) ([]*ChatMessage, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session id required")
	}

	txn := d.db.Txn(false)
	defer txn.Abort()

	if _, err := d.getSessionRecordTxn(txn, sessionID); err != nil {
		if err.Error() == "session not found" {
			return []*ChatMessage{}, nil
		}
		return nil, err
	}

	return listChatMessagesBySessionTxn(txn, sessionID)
}

func (d *Storage) DeleteChatMessages(sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id required")
	}

	txn := d.db.Txn(true)
	defer txn.Abort()

	if _, err := txn.DeleteAll(tableChat, "session_id", sessionID); err != nil {
		return fmt.Errorf("failed to delete session chat messages: %w", err)
	}

	txn.Commit()
	return nil
}

func listChatMessagesBySessionTxn(txn *memdb.Txn, sessionID string) ([]*ChatMessage, error) {
	it, err := txn.Get(tableChat, "session_id", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session chat messages: %w", err)
	}

	messages := make([]*ChatMessage, 0)
	for obj := it.Next(); obj != nil; obj = it.Next() {
		rec := obj.(*chatMessageRecord)
		messages = append(messages, cloneChatMessage(recordToChatMessage(rec)))
	}

	sort.Slice(messages, func(i, j int) bool {
		if messages[i].CreatedAt.Equal(messages[j].CreatedAt) {
			return messages[i].ID < messages[j].ID
		}
		return messages[i].CreatedAt.Before(messages[j].CreatedAt)
	})
	return messages, nil
}
