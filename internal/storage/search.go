package storage

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// FindInSession 在指定会话中执行全文检索，支持 literal/regex、大小写与 whole-word 匹配。
func (d *Storage) FindInSession(opts FindInSessionOptions) ([]FindInSessionMatch, error) {
	matcher, err := newSessionFindMatcher(opts)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(opts.SessionID) == "" {
		return []FindInSessionMatch{}, nil
	}

	txn := d.db.Txn(false)
	defer txn.Abort()

	matches := make([]FindInSessionMatch, 0)
	sid := opts.SessionID
	requests, err := d.collectRequestCandidatesTxn(txn, RequestListOptions{SessionID: sid})
	if err != nil {
		return nil, err
	}
	for _, req := range requests {
		if req == nil {
			continue
		}

		if opts.IncludeReqURL {
			if match, ok := buildSessionFindMatch(req, sid, "request_url", "", req.URL, matcher); ok {
				matches = append(matches, match)
			}
		}

		if opts.IncludeReqHeader {
			for _, match := range buildHeaderMatches(req, sid, req.Headers, "request_header", matcher) {
				matches = append(matches, match)
			}
		}

		if opts.IncludeReqBody {
			decodedBody := string(DecodeBodyBytes(req.Body, req.Headers))
			if match, ok := buildSessionFindMatch(req, sid, "request_body", "", decodedBody, matcher); ok {
				matches = append(matches, match)
			}
		}

		if opts.IncludeRespHeader {
			for _, match := range buildHeaderMatches(req, sid, req.RespHeaders, "response_header", matcher) {
				matches = append(matches, match)
			}
		}

		if opts.IncludeRespBody {
			decodedRespBody := string(DecodeBodyBytes(req.RespBody, req.RespHeaders))
			if match, ok := buildSessionFindMatch(req, sid, "response_body", "", decodedRespBody, matcher); ok {
				matches = append(matches, match)
			}
		}

		if opts.IncludeNotes {
			if match, ok := buildSessionFindMatch(req, sid, "notes", "", req.Notes, matcher); ok {
				matches = append(matches, match)
			}
		}

		if opts.IncludeError {
			if match, ok := buildSessionFindMatch(req, sid, "error", "", req.Error, matcher); ok {
				matches = append(matches, match)
			}
		}
	}

	return matches, nil
}

// SearchRequestsByHeader searches requests containing a specific header value
func (d *Storage) SearchRequestsByHeader(sessionID, headerName, headerValue string, limit int) ([]*Request, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	var requests []*Request

	candidates, err := d.collectRequestCandidatesTxn(txn, RequestListOptions{SessionID: sessionID})
	if err != nil {
		return nil, err
	}
	for _, req := range candidates {
		if values, ok := req.Headers[headerName]; ok {
			for _, v := range values {
				if strings.Contains(strings.ToLower(v), strings.ToLower(headerValue)) {
					requests = append(requests, req)
					break
				}
			}
		}
		if limit > 0 && len(requests) >= limit {
			break
		}
	}

	return requests, nil
}

// SearchRequestsByBodyContent searches requests containing a value in the request body
func (d *Storage) SearchRequestsByBodyContent(sessionID, value string, limit int) ([]*Request, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	var requests []*Request

	candidates, err := d.collectRequestCandidatesTxn(txn, RequestListOptions{SessionID: sessionID})
	if err != nil {
		return nil, err
	}
	for _, req := range candidates {
		decodedBody := string(DecodeBodyBytes(req.Body, req.Headers))
		if strings.Contains(strings.ToLower(decodedBody), strings.ToLower(value)) {
			requests = append(requests, req)
		}
		if limit > 0 && len(requests) >= limit {
			break
		}
	}

	return requests, nil
}

// SearchRequestsByResponseBody searches requests containing a value in the response body
func (d *Storage) SearchRequestsByResponseBody(sessionID, value string, limit int) ([]*Request, error) {
	txn := d.db.Txn(false)
	defer txn.Abort()

	var requests []*Request

	candidates, err := d.collectRequestCandidatesTxn(txn, RequestListOptions{SessionID: sessionID})
	if err != nil {
		return nil, err
	}
	for _, req := range candidates {
		decodedBody := string(DecodeBodyBytes(req.RespBody, req.RespHeaders))
		if strings.Contains(strings.ToLower(decodedBody), strings.ToLower(value)) {
			requests = append(requests, req)
		}
		if limit > 0 && len(requests) >= limit {
			break
		}
	}

	return requests, nil
}

// --- FindInSession matcher ---

type sessionFindMatcher struct {
	regex         *regexp.Regexp
	query         string
	queryFolded   string
	wholeWord     bool
	caseSensitive bool
}

func newSessionFindMatcher(opts FindInSessionOptions) (*sessionFindMatcher, error) {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return &sessionFindMatcher{}, nil
	}

	matcher := &sessionFindMatcher{
		query:         query,
		queryFolded:   strings.ToLower(query),
		wholeWord:     opts.IsWholeWord,
		caseSensitive: opts.IsCaseSensitive,
	}

	if !opts.IsRegex {
		return matcher, nil
	}

	pattern := query
	if opts.IsWholeWord {
		pattern = `\b(?:` + pattern + `)\b`
	}
	if !opts.IsCaseSensitive {
		pattern = `(?i)` + pattern
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid find regex: %w", err)
	}
	matcher.regex = re
	return matcher, nil
}

func buildHeaderMatches(req *Request, sessionID string, headers Headers, field string, matcher *sessionFindMatcher) []FindInSessionMatch {
	if len(headers) == 0 {
		return nil
	}

	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	matches := make([]FindInSessionMatch, 0)
	for _, key := range keys {
		values := headers[key]
		for _, value := range values {
			combined := key + ": " + value
			if match, ok := buildSessionFindMatch(req, sessionID, field, key, combined, matcher); ok {
				matches = append(matches, match)
			}
		}
	}

	return matches
}

func buildSessionFindMatch(req *Request, sessionID, field, fieldKey, value string, matcher *sessionFindMatcher) (FindInSessionMatch, bool) {
	if matcher == nil || strings.TrimSpace(matcher.query) == "" || strings.TrimSpace(value) == "" {
		return FindInSessionMatch{}, false
	}

	matchText, start, end, ok := matcher.find(value)
	if !ok {
		return FindInSessionMatch{}, false
	}

	return FindInSessionMatch{
		RequestID:        req.ID,
		SessionID:        sessionID,
		RequestCreatedAt: req.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"),
		Method:           req.Method,
		Host:             req.Host,
		Path:             req.Path,
		URL:              req.URL,
		StatusCode:       req.StatusCode,
		Field:            field,
		FieldKey:         fieldKey,
		Preview:          buildSessionFindPreview(value, start, end),
		MatchText:        matchText,
	}, true
}

func (m *sessionFindMatcher) find(value string) (string, int, int, bool) {
	if m == nil || strings.TrimSpace(m.query) == "" || value == "" {
		return "", 0, 0, false
	}

	if m.regex != nil {
		loc := m.regex.FindStringIndex(value)
		if loc == nil {
			return "", 0, 0, false
		}
		return value[loc[0]:loc[1]], loc[0], loc[1], true
	}

	if m.caseSensitive {
		return findLiteralMatch(value, m.query, m.wholeWord)
	}
	return findLiteralMatch(strings.ToLower(value), m.queryFolded, m.wholeWord, value)
}

func findLiteralMatch(searchValue, query string, wholeWord bool, original ...string) (string, int, int, bool) {
	if query == "" || searchValue == "" {
		return "", 0, 0, false
	}

	source := searchValue
	resultSource := searchValue
	if len(original) > 0 {
		resultSource = original[0]
	}

	if !wholeWord {
		idx := strings.Index(source, query)
		if idx < 0 {
			return "", 0, 0, false
		}
		return resultSource[idx : idx+len(query)], idx, idx + len(query), true
	}

	pattern := `\b` + regexp.QuoteMeta(query) + `\b`
	re := regexp.MustCompile(pattern)
	loc := re.FindStringIndex(source)
	if loc == nil {
		return "", 0, 0, false
	}
	return resultSource[loc[0]:loc[1]], loc[0], loc[1], true
}

func buildSessionFindPreview(value string, start, end int) string {
	if value == "" {
		return ""
	}
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}

	const radius = 80
	previewStart := max(start-radius, 0)
	previewEnd := min(end+radius, len(value))
	preview := value[previewStart:previewEnd]
	if previewStart > 0 {
		preview = "..." + preview
	}
	if previewEnd < len(value) {
		preview += "..."
	}
	return preview
}
