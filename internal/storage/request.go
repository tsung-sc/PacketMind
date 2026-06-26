package storage

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-memdb"
)

// SaveRequest 将请求保存到当前激活会话中。
// 若无激活会话则返回错误；若 req.ID 为空则自动生成。
func (d *Storage) SaveRequest(req *Request) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	txn := d.db.Txn(true)
	defer txn.Abort()

	stored := cloneRequest(req)
	if stored.SessionID == "" {
		activeSessionID, err := d.activeSessionIDTxn(txn)
		if err != nil {
			return err
		}
		stored.SessionID = activeSessionID
	}
	if stored.SessionID == "" {
		return fmt.Errorf("no active session")
	}
	if stored.ID == "" {
		stored.ID = generateID("req")
	}

	now := time.Now()
	if stored.CreatedAt.IsZero() {
		stored.CreatedAt = now
	}
	stored.UpdatedAt = now

	if _, err := d.getSessionRecordTxn(txn, stored.SessionID); err != nil {
		return err
	}

	if err := txn.Insert(tableRequest, requestToRecord(stored)); err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}

	req.ID = stored.ID
	req.SessionID = stored.SessionID
	req.CreatedAt = stored.CreatedAt
	req.UpdatedAt = stored.UpdatedAt

	txn.Commit()
	return nil
}

// GetRequest 在指定会话中按 ID 查找请求。
func (d *Storage) GetRequest(sessionID, id string) (*Request, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session id required")
	}

	txn := d.db.Txn(false)
	defer txn.Abort()

	obj, err := txn.First(tableRequest, "id", id)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup request: %w", err)
	}
	if obj == nil {
		return nil, fmt.Errorf("request not found")
	}

	rec := obj.(*requestRecord)
	if rec.SessionID != sessionID {
		return nil, fmt.Errorf("request not found")
	}

	return cloneRequest(recordToRequest(rec)), nil
}

// GetRequestByID 按请求 ID 查找请求。
func (d *Storage) GetRequestByID(id string) (*Request, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	return d.getRequestByIDTxn(txn, id)
}

func (d *Storage) getRequestByIDTxn(txn *memdb.Txn, id string) (*Request, error) {
	obj, err := txn.First(tableRequest, "id", id)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup request: %w", err)
	}
	if obj == nil {
		return nil, fmt.Errorf("request not found")
	}

	return cloneRequest(recordToRequest(obj.(*requestRecord))), nil
}

// ListRequests 返回请求列表，支持按会话/Host/Method/状态码/关键字/时间范围过滤与排序。
func (d *Storage) ListRequests(opts RequestListOptions) ([]*Request, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	candidates, err := d.collectRequestCandidatesTxn(txn, opts)
	if err != nil {
		return nil, err
	}

	filtered := make([]*Request, 0, len(candidates))
	for _, req := range candidates {
		if !d.matchRequestFilters(req, opts) {
			continue
		}
		filtered = append(filtered, req)
	}

	d.sortRequests(filtered, opts.SortBy, opts.SortOrder)
	return filtered, nil
}

// DeleteRequest 在指定会话中按 ID 删除请求，同时清理 HostGroups 索引。
func (d *Storage) DeleteRequest(sessionID, id string) error {
	txn := d.db.Txn(true)
	defer txn.Abort()

	rec, err := d.getRequestRecordTxn(txn, id)
	if err != nil {
		return err
	}
	if rec.SessionID != sessionID {
		return fmt.Errorf("request not found")
	}

	if _, err := d.getSessionRecordTxn(txn, sessionID); err != nil {
		return err
	}

	if err := txn.Delete(tableRequest, rec); err != nil {
		return fmt.Errorf("failed to delete request: %w", err)
	}

	txn.Commit()
	return nil
}

// ClearRequests 清空指定会话中的所有请求与 HostGroups 索引。
func (d *Storage) ClearRequests(sessionID string) error {
	txn := d.db.Txn(true)
	defer txn.Abort()

	if _, err := d.getSessionRecordTxn(txn, sessionID); err != nil {
		return err
	}

	if _, err := txn.DeleteAll(tableRequest, "session_id", sessionID); err != nil {
		return fmt.Errorf("failed to clear session requests: %w", err)
	}

	txn.Commit()
	return nil
}

func (d *Storage) collectRequestCandidatesTxn(txn *memdb.Txn, opts RequestListOptions) ([]*Request, error) {
	sessionID := strings.TrimSpace(opts.SessionID)
	if sessionID == "" {
		activeSessionID, err := d.activeSessionIDTxn(txn)
		if err != nil {
			return nil, err
		}
		sessionID = activeSessionID
	}
	if sessionID == "" {
		return nil, nil
	}

	iterator, err := d.requestCandidatesIterator(txn, sessionID, opts)
	if err != nil {
		return nil, nil
	}

	requests := make([]*Request, 0)
	for obj := iterator.Next(); obj != nil; obj = iterator.Next() {
		rec := obj.(*requestRecord)
		if rec.SessionID != sessionID {
			continue
		}
		requests = append(requests, cloneRequest(recordToRequest(rec)))
	}

	if opts.Host != "" && len(requests) == 0 {
		fallback, err := txn.Get(tableRequest, "session_id", sessionID)
		if err == nil {
			for obj := fallback.Next(); obj != nil; obj = fallback.Next() {
				rec := obj.(*requestRecord)
				requests = append(requests, cloneRequest(recordToRequest(rec)))
			}
		}
	}

	return requests, nil
}

func (d *Storage) requestCandidatesIterator(txn *memdb.Txn, sessionID string, opts RequestListOptions) (memdb.ResultIterator, error) {
	if opts.Host != "" {
		return txn.Get(tableRequest, "session_host", sessionID, opts.Host)
	}
	return txn.Get(tableRequest, "session_id", sessionID)
}

// matchRequestFilters 检查请求是否匹配过滤条件。
func (d *Storage) matchRequestFilters(req *Request, opts RequestListOptions) bool {
	if opts.Host != "" && !strings.Contains(strings.ToLower(req.Host), strings.ToLower(opts.Host)) {
		return false
	}
	if opts.Method != "" && req.Method != opts.Method {
		return false
	}
	if opts.StatusCode > 0 && req.StatusCode != opts.StatusCode {
		return false
	}
	if opts.Search != "" {
		needle := strings.ToLower(opts.Search)
		if !strings.Contains(strings.ToLower(req.Path), needle) &&
			!strings.Contains(strings.ToLower(req.Host), needle) {
			return false
		}
	}
	if opts.StartTime != nil && req.CreatedAt.Before(*opts.StartTime) {
		return false
	}
	if opts.EndTime != nil && req.CreatedAt.After(*opts.EndTime) {
		return false
	}

	return true
}

// sortRequests 按指定字段和排序方向对请求列表进行排序。
func (d *Storage) sortRequests(requests []*Request, sortBy, sortOrder string) {
	field := normalizeSortField(sortBy)
	order := strings.ToLower(sortOrder)
	if order != "asc" {
		order = "desc"
	}

	less := func(i, j int) bool {
		ai := requests[i]
		aj := requests[j]

		switch field {
		case "created_at":
			if ai.CreatedAt.Equal(aj.CreatedAt) {
				if order == "asc" {
					return ai.ID < aj.ID
				}
				return ai.ID > aj.ID
			}
			if order == "asc" {
				return ai.CreatedAt.Before(aj.CreatedAt)
			}
			return ai.CreatedAt.After(aj.CreatedAt)
		case "status_code":
			if ai.StatusCode == aj.StatusCode {
				if order == "asc" {
					return ai.ID < aj.ID
				}
				return ai.ID > aj.ID
			}
			if order == "asc" {
				return ai.StatusCode < aj.StatusCode
			}
			return ai.StatusCode > aj.StatusCode
		case "host":
			hi := strings.ToLower(ai.Host)
			hj := strings.ToLower(aj.Host)
			if hi == hj {
				if order == "asc" {
					return ai.ID < aj.ID
				}
				return ai.ID > aj.ID
			}
			if order == "asc" {
				return hi < hj
			}
			return hi > hj
		case "method":
			if ai.Method == aj.Method {
				if order == "asc" {
					return ai.ID < aj.ID
				}
				return ai.ID > aj.ID
			}
			if order == "asc" {
				return ai.Method < aj.Method
			}
			return ai.Method > aj.Method
		case "duration":
			if ai.Duration == aj.Duration {
				if order == "asc" {
					return ai.ID < aj.ID
				}
				return ai.ID > aj.ID
			}
			if order == "asc" {
				return ai.Duration < aj.Duration
			}
			return ai.Duration > aj.Duration
		default:
			if ai.CreatedAt.Equal(aj.CreatedAt) {
				if order == "asc" {
					return ai.ID < aj.ID
				}
				return ai.ID > aj.ID
			}
			if order == "asc" {
				return ai.CreatedAt.Before(aj.CreatedAt)
			}
			return ai.CreatedAt.After(aj.CreatedAt)
		}
	}

	sort.Slice(requests, less)
}

func (d *Storage) getRequestRecordTxn(txn *memdb.Txn, id string) (*requestRecord, error) {
	obj, err := txn.First(tableRequest, "id", id)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup request: %w", err)
	}
	if obj == nil {
		return nil, fmt.Errorf("request not found")
	}
	return obj.(*requestRecord), nil
}
