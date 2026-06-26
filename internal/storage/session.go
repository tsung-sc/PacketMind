package storage

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/hashicorp/go-memdb"
)

// Storage stores sessions/requests in go-memdb.
type Storage struct {
	db *memdb.MemDB
}

// NewStorage 创建内存数据库。
func NewStorage(_ ...string) (*Storage, error) {
	db, err := memdb.NewMemDB(newSchema())
	if err != nil {
		return nil, fmt.Errorf("failed to create memdb: %w", err)
	}

	return &Storage{
		db: db,
	}, nil
}

// Close 关闭存储。
func (d *Storage) Close() error {
	return nil
}

// CreateSession 创建新会话。若 session.ID 为空则自动生成；
// 若当前无任何会话则自动激活；若 session.IsActive=true 则切换激活态。
func (d *Storage) CreateSession(session *Session) error {
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	if session.ID == "" {
		session.ID = generateID("sess")
	}

	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	txn := d.db.Txn(true)
	defer txn.Abort()

	hasSessions, err := d.hasAnySessionTxn(txn)
	if err != nil {
		return err
	}

	shouldActivate := !hasSessions || session.IsActive
	session.IsActive = shouldActivate
	if shouldActivate {
		if err := d.setAllSessionsActiveStateTxn(txn, session.ID, now); err != nil {
			return err
		}
	}

	if err := txn.Insert(tableSession, sessionToRecord(session)); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	txn.Commit()
	return nil
}

type SessionView struct {
	Session
	Requests   []*Request          `json:"requests,omitempty"`
	HostGroups map[string][]string `json:"host_groups,omitempty"`
}

// GetSession 按 ID 获取会话。
func (d *Storage) GetSession(id string) (*Session, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	rec, err := d.getSessionRecordTxn(txn, id)
	if err != nil {
		return nil, err
	}

	return d.buildSessionTxn(rec)
}

func (d *Storage) GetSessionView(id string) (*SessionView, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	rec, err := d.getSessionRecordTxn(txn, id)
	if err != nil {
		return nil, err
	}

	return d.buildSessionViewTxn(txn, rec)
}

// ListSessions 返回所有会话列表（按创建时间倒序）。
func (d *Storage) ListSessions() ([]*Session, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	records, err := d.listSessionRecordsTxn(txn)
	if err != nil {
		return nil, err
	}

	sessions := make([]*Session, 0, len(records))
	for _, rec := range records {
		session, err := d.buildSessionTxn(rec)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].CreatedAt.Equal(sessions[j].CreatedAt) {
			return sessions[i].ID > sessions[j].ID
		}
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

func (d *Storage) ListSessionViews() ([]*SessionView, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	records, err := d.listSessionRecordsTxn(txn)
	if err != nil {
		return nil, err
	}

	sessions := make([]*SessionView, 0, len(records))
	for _, rec := range records {
		session, err := d.buildSessionViewTxn(txn, rec)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].CreatedAt.Equal(sessions[j].CreatedAt) {
			return sessions[i].ID > sessions[j].ID
		}
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// DeleteSession 按 ID 删除会话。若删除的是当前激活会话，自动激活最新的会话。
func (d *Storage) DeleteSession(id string) error {
	txn := d.db.Txn(true)
	defer txn.Abort()

	rec, err := d.getSessionRecordTxn(txn, id)
	if err != nil {
		return err
	}

	if _, err := txn.DeleteAll(tableRequest, "session_id", id); err != nil {
		return fmt.Errorf("failed to delete session requests: %w", err)
	}
	if _, err := txn.DeleteAll(tableChat, "session_id", id); err != nil {
		return fmt.Errorf("failed to delete session chat messages: %w", err)
	}
	if err := txn.Delete(tableSession, rec); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	remaining, err := d.listSessionRecordsTxn(txn)
	if err != nil {
		return err
	}

	if rec.IsActive {
		newest := latestSessionRecord(remaining)
		if newest != nil {
			now := time.Now()
			if err := d.setAllSessionsActiveStateTxn(txn, newest.ID, now); err != nil {
				return err
			}
		}
	}

	txn.Commit()
	return nil
}

// GetActiveSession 返回当前激活的会话。
func (d *Storage) GetActiveSession() (*Session, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	rec, err := d.activeSessionRecordTxn(txn)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	return d.buildSessionTxn(rec)
}

// SetActiveSession 将指定会话设为激活状态，同时取消其他会话的激活态。
func (d *Storage) SetActiveSession(id string) error {
	txn := d.db.Txn(true)
	defer txn.Abort()

	if _, err := d.getSessionRecordTxn(txn, id); err != nil {
		return err
	}

	now := time.Now()
	if err := d.setAllSessionsActiveStateTxn(txn, id, now); err != nil {
		return err
	}

	txn.Commit()
	return nil
}

// UpdateSession 更新会话的名称与描述。
func (d *Storage) UpdateSession(id, name, description string) (*Session, error) {
	txn := d.db.Txn(true)
	defer txn.Abort()

	rec, err := d.getSessionRecordTxn(txn, id)
	if err != nil {
		return nil, err
	}

	updated := recordToSession(rec)
	updated.Name = name
	updated.Description = description
	updated.UpdatedAt = time.Now()
	record := sessionToRecord(updated)

	if err := txn.Insert(tableSession, record); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	built, err := d.buildSessionTxn(record)
	if err != nil {
		return nil, err
	}

	txn.Commit()
	return built, nil
}

func (d *Storage) getSessionRecordTxn(txn *memdb.Txn, id string) (*sessionRecord, error) {
	rec, err := d.getSessionRecordByIDTxn(txn, id)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("session not found")
	}
	return rec, nil
}

func (d *Storage) getSessionRecordByIDTxn(txn *memdb.Txn, id string) (*sessionRecord, error) {
	obj, err := txn.First(tableSession, "id", id)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup session: %w", err)
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(*sessionRecord), nil
}

func (d *Storage) activeSessionRecordTxn(txn *memdb.Txn) (*sessionRecord, error) {
	obj, err := txn.First(tableSession, "is_active", true)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup active session: %w", err)
	}
	if obj == nil {
		return nil, nil
	}
	return obj.(*sessionRecord), nil
}

func (d *Storage) activeSessionIDTxn(txn *memdb.Txn) (string, error) {
	rec, err := d.activeSessionRecordTxn(txn)
	if err != nil {
		return "", err
	}
	if rec == nil {
		return "", nil
	}
	return rec.ID, nil
}

func (d *Storage) hasAnySessionTxn(txn *memdb.Txn) (bool, error) {
	it, err := txn.LowerBound(tableSession, "created_at", int64(math.MinInt64))
	if err != nil {
		return false, fmt.Errorf("failed to query sessions: %w", err)
	}
	return it.Next() != nil, nil
}

func (d *Storage) listSessionRecordsTxn(txn *memdb.Txn) ([]*sessionRecord, error) {
	it, err := txn.LowerBound(tableSession, "created_at", int64(math.MinInt64))
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}

	records := make([]*sessionRecord, 0)
	for obj := it.Next(); obj != nil; obj = it.Next() {
		records = append(records, obj.(*sessionRecord))
	}
	return records, nil
}

func (d *Storage) buildSessionTxn(rec *sessionRecord) (*Session, error) {
	if rec == nil {
		return nil, nil
	}
	return recordToSession(rec), nil
}

func (d *Storage) buildSessionViewTxn(txn *memdb.Txn, rec *sessionRecord) (*SessionView, error) {
	if rec == nil {
		return nil, nil
	}

	session := recordToSession(rec)
	requests, err := d.listRequestsBySessionTxn(txn, rec.ID)
	if err != nil {
		return nil, err
	}

	hostGroups := make(map[string][]string)
	for _, req := range requests {
		hostGroups[req.Host] = append(hostGroups[req.Host], req.ID)
	}

	view := &SessionView{Session: *session, Requests: requests}
	if len(hostGroups) > 0 {
		view.HostGroups = hostGroups
	}

	return view, nil
}

func (d *Storage) listRequestsBySessionTxn(txn *memdb.Txn, sessionID string) ([]*Request, error) {
	it, err := txn.Get(tableRequest, "session_id", sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query session requests: %w", err)
	}

	requests := make([]*Request, 0)
	for obj := it.Next(); obj != nil; obj = it.Next() {
		rec := obj.(*requestRecord)
		requests = append(requests, cloneRequest(recordToRequest(rec)))
	}

	d.sortRequests(requests, "created_at", "asc")
	return requests, nil
}

func (d *Storage) setAllSessionsActiveStateTxn(txn *memdb.Txn, activeID string, now time.Time) error {
	records, err := d.listSessionRecordsTxn(txn)
	if err != nil {
		return err
	}

	for _, rec := range records {
		updated := recordToSession(rec)
		updated.IsActive = updated.ID == activeID
		updated.UpdatedAt = now
		if err := txn.Insert(tableSession, sessionToRecord(updated)); err != nil {
			return fmt.Errorf("failed to update session active state: %w", err)
		}
	}

	return nil
}

func latestSessionRecord(records []*sessionRecord) *sessionRecord {
	var newest *sessionRecord
	for _, rec := range records {
		if rec == nil {
			continue
		}
		if newest == nil || rec.CreatedAtUnix > newest.CreatedAtUnix || (rec.CreatedAtUnix == newest.CreatedAtUnix && rec.ID > newest.ID) {
			newest = rec
		}
	}
	return newest
}
