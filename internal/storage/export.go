package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/go-memdb"
)

type exportSession struct {
	ID           string              `json:"id"`
	Name         string              `json:"name"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
	IsActive     bool                `json:"is_active"`
	Description  string              `json:"description"`
	Requests     []*Request          `json:"requests,omitempty"`
	ChatMessages []*ChatMessage      `json:"chat_messages,omitempty"`
	HostGroups   map[string][]string `json:"host_groups,omitempty"`
}

type exportData struct {
	Sessions      []*exportSession `json:"sessions"`
	ActiveSession string           `json:"active_session"`
}

func (d *Storage) ExportAll() ([]byte, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	records, err := d.listSessionRecordsTxn(txn)
	if err != nil {
		return nil, err
	}

	activeSessionID, err := d.activeSessionIDTxn(txn)
	if err != nil {
		return nil, err
	}

	out := exportData{
		Sessions:      make([]*exportSession, 0, len(records)),
		ActiveSession: activeSessionID,
	}

	for _, rec := range records {
		session, err := d.buildSessionViewTxn(txn, rec)
		if err != nil {
			return nil, err
		}
		out.Sessions = append(out.Sessions, exportSessionFromView(txn, session))
	}

	sort.Slice(out.Sessions, func(i, j int) bool { return out.Sessions[i].ID < out.Sessions[j].ID })

	data, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("failed to export data: %w", err)
	}
	return data, nil
}

func (d *Storage) ImportAll(data []byte) error {
	var in exportData
	if err := json.Unmarshal(data, &in); err != nil {
		return fmt.Errorf("failed to parse import data: %w", err)
	}

	newDB, err := memdb.NewMemDB(newSchema())
	if err != nil {
		return fmt.Errorf("failed to create memdb: %w", err)
	}

	txn := newDB.Txn(true)
	defer txn.Abort()

	validSessionIDs := make(map[string]struct{}, len(in.Sessions))
	for _, session := range in.Sessions {
		if session == nil || session.ID == "" {
			continue
		}
		validSessionIDs[session.ID] = struct{}{}
	}

	selectedActive := in.ActiveSession
	if selectedActive != "" {
		if _, ok := validSessionIDs[selectedActive]; !ok {
			selectedActive = ""
		}
	}
	if selectedActive == "" {
		for _, session := range in.Sessions {
			if session != nil && session.ID != "" && session.IsActive {
				selectedActive = session.ID
				break
			}
		}
	}
	if selectedActive == "" {
		var newest *exportSession
		for _, session := range in.Sessions {
			if session == nil || session.ID == "" {
				continue
			}
			if newest == nil || session.CreatedAt.After(newest.CreatedAt) || (session.CreatedAt.Equal(newest.CreatedAt) && session.ID > newest.ID) {
				newest = session
			}
		}
		if newest != nil {
			if _, ok := validSessionIDs[newest.ID]; ok {
				selectedActive = newest.ID
			}
		}
	}

	for _, session := range in.Sessions {
		if session == nil || session.ID == "" {
			continue
		}

		storedSession := sessionFromExport(session)
		storedSession.IsActive = storedSession.ID == selectedActive
		if storedSession.CreatedAt.IsZero() {
			storedSession.CreatedAt = time.Now()
		}
		if storedSession.UpdatedAt.IsZero() {
			storedSession.UpdatedAt = storedSession.CreatedAt
		}

		if err := txn.Insert(tableSession, sessionToRecord(storedSession)); err != nil {
			return fmt.Errorf("failed to import session: %w", err)
		}

		for _, req := range session.Requests {
			if req == nil || req.ID == "" {
				continue
			}
			storedReq := cloneRequest(req)
			storedReq.SessionID = storedSession.ID
			if storedReq.CreatedAt.IsZero() {
				storedReq.CreatedAt = storedSession.CreatedAt
			}
			if storedReq.UpdatedAt.IsZero() {
				storedReq.UpdatedAt = storedReq.CreatedAt
			}
			if err := txn.Insert(tableRequest, requestToRecord(storedReq)); err != nil {
				return fmt.Errorf("failed to import request: %w", err)
			}
		}

		for _, msg := range session.ChatMessages {
			if msg == nil || msg.ID == "" {
				continue
			}
			storedMsg := cloneChatMessage(msg)
			storedMsg.SessionID = storedSession.ID
			if storedMsg.CreatedAt.IsZero() {
				storedMsg.CreatedAt = storedSession.CreatedAt
			}
			if err := txn.Insert(tableChat, chatMessageToRecord(storedMsg)); err != nil {
				return fmt.Errorf("failed to import chat message: %w", err)
			}
		}
	}

	txn.Commit()
	d.db = newDB

	return nil
}

func exportSessionFromView(txn *memdb.Txn, session *SessionView) *exportSession {
	if session == nil {
		return nil
	}
	view := &exportSession{
		ID:          session.ID,
		Name:        session.Name,
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   session.UpdatedAt,
		IsActive:    session.IsActive,
		Description: session.Description,
		HostGroups:  cloneHostGroups(session.HostGroups),
	}
	if len(session.Requests) > 0 {
		view.Requests = make([]*Request, 0, len(session.Requests))
		for _, req := range session.Requests {
			view.Requests = append(view.Requests, cloneRequest(req))
		}
	}
	if txn != nil {
		if storageMsgs, err := listChatMessagesBySessionTxn(txn, session.ID); err == nil && len(storageMsgs) > 0 {
			view.ChatMessages = make([]*ChatMessage, 0, len(storageMsgs))
			for _, msg := range storageMsgs {
				view.ChatMessages = append(view.ChatMessages, cloneChatMessage(msg))
			}
		}
	}
	return view
}

func sessionFromExport(session *exportSession) *Session {
	if session == nil {
		return nil
	}
	return &Session{
		ID:          session.ID,
		Name:        session.Name,
		CreatedAt:   session.CreatedAt,
		UpdatedAt:   session.UpdatedAt,
		IsActive:    session.IsActive,
		Description: session.Description,
	}
}

func cloneHostGroups(groups map[string][]string) map[string][]string {
	if groups == nil {
		return nil
	}
	cloned := make(map[string][]string, len(groups))
	for key, values := range groups {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}
