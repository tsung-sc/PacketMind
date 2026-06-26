package bindings

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/packetmind/packetmind/internal/proxy"
	"github.com/packetmind/packetmind/internal/storage"
)

type RequestAPI struct {
	onRequest  func(*storage.Request)
	onComplete func(*storage.Request)
}

// NewRequestAPI 创建 RequestAPI 实例。
func NewRequestAPI() *RequestAPI {
	return &RequestAPI{}
}

// SetOnRequest 设置请求录制回调。
func (r *RequestAPI) SetOnRequest(fn func(*storage.Request)) {
	r.onRequest = fn
}

// SetOnComplete 设置请求完成回调。
func (r *RequestAPI) SetOnComplete(fn func(*storage.Request)) {
	r.onComplete = fn
}

// emitRequest 触发请求录制回调。
func (r *RequestAPI) emitRequest(req *storage.Request) {
	if r.onRequest != nil {
		r.onRequest(req)
	}
}

// emitComplete 触发请求完成回调。
func (r *RequestAPI) emitComplete(req *storage.Request) {
	if r.onComplete != nil {
		r.onComplete(req)
	}
}

type RequestListOptions struct {
	SessionID  string `json:"session_id"`
	Host       string `json:"host"`
	Method     string `json:"method"`
	Search     string `json:"search"`
	SortBy     string `json:"sort_by"`
	SortOrder  string `json:"sort_order"`
	StatusCode int    `json:"status_code"`
}

type PaginatedRequests struct {
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Items    []*RequestDTO `json:"items"`
}

type ReplayRequestOptions struct {
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

type ComposeRequestOptions struct {
	Method  string              `json:"method"`
	URL     string              `json:"url"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

type ReplayResult struct {
	RequestID      string              `json:"request_id"`
	StatusCode     int                 `json:"status_code"`
	DurationMs     int64               `json:"duration_ms"`
	ResponseSize   int                 `json:"response_size"`
	ResponseHeader map[string][]string `json:"response_header"`
}

func (r *RequestAPI) ensureActiveSession() (*storage.Session, error) {
	session, err := storage.Default.GetActiveSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get active session: %w", err)
	}
	if session != nil {
		return session, nil
	}

	session = &storage.Session{
		ID:       "default",
		Name:     "Default Session",
		IsActive: true,
	}
	if err := storage.Default.CreateSession(session); err != nil {
		return nil, fmt.Errorf("failed to create default session: %w", err)
	}

	return session, nil
}

// ListRequests 列出请求。
func (r *RequestAPI) ListRequests(opts RequestListOptions) SessionResponse {
	if opts.SortBy == "" {
		opts.SortBy = "created_at"
	}
	if opts.SortOrder == "" {
		opts.SortOrder = "desc"
	}

	listOpts := storage.RequestListOptions{
		SessionID:  opts.SessionID,
		Host:       opts.Host,
		Method:     opts.Method,
		Search:     opts.Search,
		SortBy:     opts.SortBy,
		SortOrder:  opts.SortOrder,
		StatusCode: opts.StatusCode,
	}

	requests, err := storage.Default.ListRequests(listOpts)
	if err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{
		Code: 0,
		Data: PaginatedRequests{
			Total: int64(len(requests)),
			Items: toRequestDTOs(requests, opts.SessionID),
		},
	}
}

// GetRequest 获取单个请求。
func (r *RequestAPI) GetRequest(sessionID, id string) SessionResponse {
	req, err := storage.Default.GetRequest(sessionID, id)
	if err != nil {
		return SessionResponse{Code: 40001, Message: "Request not found"}
	}

	return SessionResponse{Code: 0, Data: toRequestDTO(req, sessionID)}
}

// FindInSession 在指定会话内执行全文检索。
func (r *RequestAPI) FindInSession(opts storage.FindInSessionOptions) SessionResponse {
	matches, err := storage.Default.FindInSession(opts)
	if err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Data: matches}
}

// DeleteRequest 删除请求。
func (r *RequestAPI) DeleteRequest(sessionID string, id string) SessionResponse {
	if strings.TrimSpace(sessionID) == "" {
		return SessionResponse{Code: 40001, Message: "session_id is required"}
	}
	if strings.TrimSpace(id) == "" {
		return SessionResponse{Code: 40001, Message: "id is required"}
	}
	if err := storage.Default.DeleteRequest(sessionID, id); err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Message: "success"}
}

// ClearRequests 清空请求。
func (r *RequestAPI) ClearRequests(sessionID string) SessionResponse {
	if err := storage.Default.ClearRequests(sessionID); err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	return SessionResponse{Code: 0, Message: "success"}
}

// ExportRequest 导出单个请求。
func (r *RequestAPI) ExportRequest(sessionID, id, format string) SessionResponse {
	req, err := storage.Default.GetRequest(sessionID, id)
	if err != nil {
		return SessionResponse{Code: 40001, Message: "Request not found"}
	}

	var data interface{}
	switch format {
	case "curl":
		data = r.exportCurl(req)
	default:
		data = r.exportHAR(req)
	}

	return SessionResponse{Code: 0, Data: data}
}

// ReplayRequest 重放请求。
func (r *RequestAPI) ReplayRequest(sessionID, id string, opts ReplayRequestOptions) SessionResponse {
	originalReq, err := storage.Default.GetRequest(sessionID, id)
	if err != nil {
		return SessionResponse{Code: 40001, Message: "Request not found"}
	}

	_, err = url.Parse(originalReq.URL)
	if err != nil {
		return SessionResponse{Code: 40002, Message: "Invalid URL: " + err.Error()}
	}

	startTime := time.Now()

	var body []byte
	if opts.Body != "" {
		body = []byte(opts.Body)
	} else {
		body = originalReq.Body
	}

	httpReq, err := http.NewRequest(originalReq.Method, originalReq.URL, bytes.NewReader(body))
	if err != nil {
		return SessionResponse{Code: 50001, Message: "Failed to create request: " + err.Error()}
	}
	httpReq.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	if len(opts.Headers) > 0 {
		for k, values := range opts.Headers {
			for _, v := range values {
				httpReq.Header.Add(k, v)
			}
		}
	} else {
		for k, values := range originalReq.Headers {
			for _, v := range values {
				httpReq.Header.Add(k, v)
			}
		}
	}
	httpReq.Header.Del("Content-Length")

	// Phase 1: Save pending request (no response data yet) and emit request:new
	replayedReq := &storage.Request{
		ID:                uuid.New().String(),
		CreatedAt:         time.Now(),
		Method:            originalReq.Method,
		Scheme:            originalReq.Scheme,
		Host:              originalReq.Host,
		Port:              originalReq.Port,
		Path:              originalReq.Path,
		URL:               originalReq.URL,
		QueryString:       originalReq.QueryString,
		HTTPVersion:       originalReq.HTTPVersion,
		Headers:           storage.Headers(httpReq.Header),
		ContentType:       originalReq.ContentType,
		Body:              body,
		BodySize:          int64(len(body)),
		RemoteAddr:        originalReq.RemoteAddr,
		RequestStartTime:  startTime,
		Notes:             "Replayed from " + originalReq.ID,
		TLSClientHelloRaw: append([]byte(nil), originalReq.TLSClientHelloRaw...),
	}
	if err := storage.Default.SaveRequest(replayedReq); err != nil {
		fmt.Printf("[Handler] Failed to save replayed request start: %v\n", err)
	}
	r.emitRequest(replayedReq)

	resp, err := proxy.Default.ExecuteRequestWithClientHello(httpReq, originalReq.TLSClientHelloRaw)

	// Update record with response or error
	replayedReq.Duration = time.Since(startTime).Milliseconds()
	replayedReq.RequestEndTime = time.Now()

	if err != nil {
		replayedReq.StatusCode = 502
		replayedReq.Error = err.Error()
		storage.Default.SaveRequest(replayedReq)
		r.emitComplete(replayedReq)
		return SessionResponse{Code: 50001, Message: "Request failed: " + err.Error()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	replayedReq.ResponseStartTime = time.Now()
	replayedReq.ResponseEndTime = time.Now()
	replayedReq.StatusCode = resp.StatusCode
	replayedReq.StatusReason = resp.Status
	replayedReq.RespHeaders = make(storage.Headers)
	replayedReq.RespContentType = resp.Header.Get("Content-Type")
	replayedReq.RespBody = respBody
	replayedReq.RespBodySize = int64(len(respBody))

	for k, v := range resp.Header {
		replayedReq.RespHeaders[k] = v
	}

	storage.Default.SaveRequest(replayedReq)
	r.emitComplete(replayedReq)

	return SessionResponse{
		Code: 0,
		Data: ReplayResult{
			RequestID:      replayedReq.ID,
			StatusCode:     resp.StatusCode,
			DurationMs:     replayedReq.Duration,
			ResponseSize:   len(respBody),
			ResponseHeader: resp.Header,
		},
	}
}

// ComposeRequest 发送自定义请求并按两阶段记录结果。
func (r *RequestAPI) ComposeRequest(opts ComposeRequestOptions) SessionResponse {
	method := strings.TrimSpace(opts.Method)
	if method == "" {
		return SessionResponse{Code: 40002, Message: "Method is required"}
	}

	parsedURL, err := url.Parse(strings.TrimSpace(opts.URL))
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		if err == nil {
			err = fmt.Errorf("scheme and host are required")
		}
		return SessionResponse{Code: 40002, Message: "Invalid URL: " + err.Error()}
	}

	_, err = r.ensureActiveSession()
	if err != nil {
		return SessionResponse{Code: 50001, Message: err.Error()}
	}

	body := []byte(opts.Body)
	httpReq, err := http.NewRequest(method, parsedURL.String(), bytes.NewReader(body))
	if err != nil {
		return SessionResponse{Code: 50001, Message: "Failed to create request: " + err.Error()}
	}
	httpReq.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	for k, values := range opts.Headers {
		for _, v := range values {
			httpReq.Header.Add(k, v)
		}
	}
	httpReq.Header.Del("Content-Length")

	startTime := time.Now()
	port := requestURLPort(parsedURL)
	composedReq := &storage.Request{
		ID:               uuid.New().String(),
		CreatedAt:        startTime,
		Method:           method,
		Scheme:           parsedURL.Scheme,
		Host:             parsedURL.Hostname(),
		Port:             port,
		Path:             requestURLPath(parsedURL),
		URL:              parsedURL.String(),
		QueryString:      parsedURL.RawQuery,
		HTTPVersion:      "HTTP/1.1",
		Headers:          storage.Headers(httpReq.Header),
		ContentType:      httpReq.Header.Get("Content-Type"),
		Body:             body,
		BodySize:         int64(len(body)),
		RequestStartTime: startTime,
		Notes:            "Composed request",
	}

	if err := storage.Default.SaveRequest(composedReq); err != nil {
		fmt.Printf("[Handler] Failed to save composed request start: %v\n", err)
	}
	r.emitRequest(composedReq)

	resp, err := proxy.Default.ExecuteRequest(httpReq)
	composedReq.Duration = time.Since(startTime).Milliseconds()
	composedReq.RequestEndTime = time.Now()

	if err != nil {
		composedReq.StatusCode = 502
		composedReq.Error = err.Error()
		if saveErr := storage.Default.SaveRequest(composedReq); saveErr != nil {
			fmt.Printf("[Handler] Failed to save composed request error: %v\n", saveErr)
		}
		r.emitComplete(composedReq)
		return SessionResponse{Code: 50001, Message: "Request failed: " + err.Error()}
	}
	defer resp.Body.Close()

	composedReq.ResponseStartTime = time.Now()
	respBody, readErr := io.ReadAll(resp.Body)
	composedReq.ResponseEndTime = time.Now()
	if readErr != nil {
		composedReq.StatusCode = 502
		composedReq.Error = "Failed to read response body: " + readErr.Error()
		if saveErr := storage.Default.SaveRequest(composedReq); saveErr != nil {
			fmt.Printf("[Handler] Failed to save composed request read error: %v\n", saveErr)
		}
		r.emitComplete(composedReq)
		return SessionResponse{Code: 50001, Message: composedReq.Error}
	}

	composedReq.StatusCode = resp.StatusCode
	composedReq.StatusReason = resp.Status
	composedReq.RespHeaders = make(storage.Headers)
	composedReq.RespContentType = resp.Header.Get("Content-Type")
	composedReq.RespBody = respBody
	composedReq.RespBodySize = int64(len(respBody))
	for k, v := range resp.Header {
		composedReq.RespHeaders[k] = v
	}

	if err := storage.Default.SaveRequest(composedReq); err != nil {
		fmt.Printf("[Handler] Failed to save composed request complete: %v\n", err)
	}
	r.emitComplete(composedReq)

	return SessionResponse{
		Code: 0,
		Data: ReplayResult{
			RequestID:      composedReq.ID,
			StatusCode:     resp.StatusCode,
			DurationMs:     composedReq.Duration,
			ResponseSize:   len(respBody),
			ResponseHeader: resp.Header,
		},
	}
}

func requestURLPath(parsedURL *url.URL) string {
	if parsedURL == nil || parsedURL.Path == "" {
		return "/"
	}
	return parsedURL.EscapedPath()
}

func requestURLPort(parsedURL *url.URL) int {
	if parsedURL == nil {
		return 0
	}
	if port := parsedURL.Port(); port != "" {
		if parsed, err := strconv.Atoi(port); err == nil {
			return parsed
		}
	}

	switch strings.ToLower(parsedURL.Scheme) {
	case "https":
		return 443
	case "http":
		return 80
	default:
		return 0
	}
}

func (r *RequestAPI) exportHAR(req *storage.Request) map[string]interface{} {
	return map[string]interface{}{
		"log": map[string]interface{}{
			"version": "1.2",
			"entries": []map[string]interface{}{
				{
					"startedDateTime": req.CreatedAt.Format("2006-01-02T15:04:05.000Z"),
					"request": map[string]interface{}{
						"method":   req.Method,
						"url":      req.URL,
						"headers":  r.convertHeaders(req.Headers),
						"bodySize": req.BodySize,
					},
					"response": map[string]interface{}{
						"status":   req.StatusCode,
						"headers":  r.convertHeaders(req.RespHeaders),
						"bodySize": req.RespBodySize,
					},
					"time": req.Duration,
				},
			},
		},
	}
}

func (r *RequestAPI) exportCurl(req *storage.Request) string {
	cmd := "curl -X " + req.Method + " '" + req.URL + "'"
	for k, values := range req.Headers {
		for _, v := range values {
			cmd += " \\\n  -H '" + k + ": " + v + "'"
		}
	}
	if len(req.Body) > 0 {
		cmd += " \\\n  -d '" + string(req.Body) + "'"
	}
	return cmd
}

func (r *RequestAPI) convertHeaders(headers storage.Headers) []map[string]string {
	result := make([]map[string]string, 0)
	for k, values := range headers {
		for _, v := range values {
			result = append(result, map[string]string{
				"name":  k,
				"value": v,
			})
		}
	}
	return result
}
