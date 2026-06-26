package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type harFile struct {
	Log harLog `json:"log"`
}

type harLog struct {
	Version string     `json:"version"`
	Creator harCreator `json:"creator"`
	Entries []harEntry `json:"entries"`
}

type harCreator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type harEntry struct {
	StartedDateTime string      `json:"startedDateTime"`
	Time            int64       `json:"time"`
	Request         harRequest  `json:"request"`
	Response        harResponse `json:"response"`
}

type harRequest struct {
	Method      string       `json:"method"`
	URL         string       `json:"url"`
	HTTPVersion string       `json:"httpVersion"`
	Headers     []harHeader  `json:"headers"`
	QueryString []harQuery   `json:"queryString"`
	Cookies     []harCookie  `json:"cookies"`
	BodySize    int64        `json:"bodySize"`
	HeadersSize int          `json:"headersSize"`
	PostData    *harPostData `json:"postData,omitempty"`
}

type harResponse struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []harHeader `json:"headers"`
	Cookies     []harCookie `json:"cookies"`
	Content     harContent  `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	BodySize    int64       `json:"bodySize"`
	HeadersSize int         `json:"headersSize"`
}

type harContent struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text,omitempty"`
	Encoding string `json:"encoding,omitempty"`
}

type harHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harQuery struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type harPostData struct {
	MimeType string     `json:"mimeType"`
	Text     string     `json:"text,omitempty"`
	Params   []harParam `json:"params,omitempty"`
}

type harParam struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ExportHAR 将指定请求导出为 HAR 1.2 格式。
func (d *Storage) ExportHAR(requestID string) ([]byte, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	req, err := d.getRequestByIDTxn(txn, requestID)
	if err != nil {
		return nil, err
	}

	har := harFile{
		Log: harLog{
			Version: "1.2",
			Creator: harCreator{Name: "PacketMind", Version: "dev"},
			Entries: []harEntry{
				{
					StartedDateTime: req.CreatedAt.Format(time.RFC3339Nano),
					Time:            req.Duration,
					Request: harRequest{
						Method:      req.Method,
						URL:         req.URL,
						HTTPVersion: req.HTTPVersion,
						Headers:     convertHARHeaders(req.Headers),
						QueryString: parseHARQuery(req.QueryString),
						Cookies:     convertHARCookies(req.Cookies),
						BodySize:    req.BodySize,
						HeadersSize: -1,
						PostData:    buildHARPostData(req),
					},
					Response: harResponse{
						Status:      req.StatusCode,
						StatusText:  req.StatusReason,
						HTTPVersion: req.HTTPVersion,
						Headers:     convertHARHeaders(req.RespHeaders),
						Cookies:     []harCookie{},
						Content:     buildHARContent(req),
						RedirectURL: "",
						BodySize:    req.RespBodySize,
						HeadersSize: -1,
					},
				},
			},
		},
	}

	data, err := json.Marshal(har)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal har: %w", err)
	}

	return data, nil
}

func convertHARHeaders(headers Headers) []harHeader {
	result := make([]harHeader, 0)
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values := headers[key]
		for _, value := range values {
			result = append(result, harHeader{Name: key, Value: value})
		}
	}
	return result
}

func parseHARQuery(query string) []harQuery {
	if strings.TrimSpace(query) == "" {
		return []harQuery{}
	}

	parts := strings.Split(query, "&")
	result := make([]harQuery, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		segments := strings.SplitN(part, "=", 2)
		if len(segments) == 1 {
			result = append(result, harQuery{Name: segments[0], Value: ""})
			continue
		}
		result = append(result, harQuery{Name: segments[0], Value: segments[1]})
	}
	return result
}

func convertHARCookies(cookies Cookies) []harCookie {
	if len(cookies) == 0 {
		return []harCookie{}
	}
	keys := make([]string, 0, len(cookies))
	for key := range cookies {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]harCookie, 0, len(cookies))
	for _, key := range keys {
		out = append(out, harCookie{Name: key, Value: cookies[key]})
	}
	return out
}

func buildHARPostData(req *Request) *harPostData {
	if len(req.Body) == 0 {
		return nil
	}
	return &harPostData{
		MimeType: req.ContentType,
		Text:     string(req.Body),
	}
}

func buildHARContent(req *Request) harContent {
	content := harContent{
		Size:     req.RespBodySize,
		MimeType: req.RespContentType,
	}

	if len(req.RespBody) == 0 {
		return content
	}

	if isTextLike(req.RespContentType) {
		content.Text = string(req.RespBody)
		return content
	}

	content.Text = base64.StdEncoding.EncodeToString(req.RespBody)
	content.Encoding = "base64"
	return content
}

func isTextLike(contentType string) bool {
	ct := strings.ToLower(contentType)
	return strings.HasPrefix(ct, "text/") ||
		strings.Contains(ct, "json") ||
		strings.Contains(ct, "xml") ||
		strings.Contains(ct, "javascript") ||
		strings.Contains(ct, "x-www-form-urlencoded")
}
