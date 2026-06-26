package bindings

import (
	"encoding/base64"

	"github.com/packetmind/packetmind/internal/storage"
)

type SessionAPI struct {
	onDeleteCallbacks []func(sessionID string)
}

// NewSessionAPI 创建 SessionAPI 实例。
func NewSessionAPI() *SessionAPI {
	return &SessionAPI{}
}

// RegisterSessionDeleteHook registers a callback invoked when a session is deleted.
// It is intentionally a package-level helper so internal lifecycle wiring does not
// leak onto the Wails-bound SessionAPI surface.
func RegisterSessionDeleteHook(api *SessionAPI, fn func(sessionID string)) {
	if api == nil || fn == nil {
		return
	}
	api.onDeleteCallbacks = append(api.onDeleteCallbacks, fn)
}

type Sessions struct {
	Total int64         `json:"total"`
	Items []*SessionDTO `json:"items"`
}

type CreateSessionRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateSessionRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// ListSessions 列出会话。
func (s *SessionAPI) ListSessions() SessionResponse {
	sessions, err := storage.Default.ListSessions()
	if err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{
		Code: 0,
		Data: Sessions{
			Total: int64(len(sessions)),
			Items: toSessionDTOs(sessions),
		},
	}
}

// GetSession 获取单个会话。
func (s *SessionAPI) GetSession(id string) SessionResponse {
	session, err := storage.Default.GetSession(id)
	if err != nil {
		return SessionResponse{Code: 40001, Message: "Session not found"}
	}

	return SessionResponse{Code: 0, Data: toSessionDTO(session)}
}

// CreateSession 创建会话。
func (s *SessionAPI) CreateSession(req CreateSessionRequest) SessionResponse {
	session := &storage.Session{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    false,
	}

	if err := storage.Default.CreateSession(session); err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Data: toSessionDTO(session)}
}

// UpdateSession 更新会话。
func (s *SessionAPI) UpdateSession(id string, req UpdateSessionRequest) SessionResponse {
	session, err := storage.Default.GetSession(id)
	if err != nil {
		return SessionResponse{Code: 40001, Message: "Session not found"}
	}

	name := session.Name
	description := session.Description
	if req.Name != nil {
		name = *req.Name
	}
	if req.Description != nil {
		description = *req.Description
	}

	updated, err := storage.Default.UpdateSession(id, name, description)
	if err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Data: toSessionDTO(updated)}
}

// DeleteSession 删除会话。
func (s *SessionAPI) DeleteSession(id string) SessionResponse {
	if err := storage.Default.DeleteSession(id); err != nil {
		if err.Error() == "session not found" {
			return SessionResponse{Code: 40001, Message: err.Error()}
		}
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	for _, fn := range s.onDeleteCallbacks {
		fn(id)
	}

	return SessionResponse{Code: 0, Message: "success"}
}

// ActivateSession 激活会话。
func (s *SessionAPI) ActivateSession(id string) SessionResponse {
	if err := storage.Default.SetActiveSession(id); err != nil {
		if err.Error() == "session not found" {
			return SessionResponse{Code: 40001, Message: err.Error()}
		}
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Message: "success"}
}

// ExportSessions 导出当前全部会话数据。
func (s *SessionAPI) ExportSessions() SessionResponse {
	data, err := storage.Default.ExportAll()
	if err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{
		Code: 0,
		Data: map[string]any{
			"filename": "packetmind-session.json",
			"content":  base64.StdEncoding.EncodeToString(data),
		},
	}
}

// ImportSessions 导入会话数据。
func (s *SessionAPI) ImportSessions(contentBase64 string) SessionResponse {
	if contentBase64 == "" {
		return SessionResponse{Code: 40002, Message: "import content is empty"}
	}

	data, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return SessionResponse{Code: 40002, Message: "invalid import content"}
	}

	if err := storage.Default.ImportAll(data); err != nil {
		return SessionResponse{Code: 40002, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Message: "sessions imported"}
}
