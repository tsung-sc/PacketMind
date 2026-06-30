package builtin

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
	"github.com/packetmind/packetmind/internal/agent/tools"
	"github.com/packetmind/packetmind/internal/storage"
)

func newSummarizeSessionHandler(store *storage.Storage) tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		sessionID = strings.TrimSpace(sessionID)
		if sessionID == "" {
			sessionID = tools.GetStringArg(args, "session_id", "")
		}

		opts := storage.RequestListOptions{SessionID: sessionID, SortBy: "created_at", SortOrder: "asc"}
		requests, err := store.ListRequests(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list requests: %w", err)
		}

		totalRequests := len(requests)
		if totalRequests == 0 {
			content := mustMarshalJSON(map[string]any{
				"ok":                    true,
				"session_id":            sessionID,
				"total_requests":        0,
				"unique_hosts":          0,
				"hosts":                 []any{},
				"critical_observations": []string{"No requests found in this session."},
			})
			return &agentruntime.ToolExecutionResult{
				Content: content,
				Summary: "Session summary: no requests found",
			}, nil
		}

		hostMap := make(map[string]*hostSummary)
		var firstTime, lastTime time.Time

		for _, req := range requests {
			if req == nil {
				continue
			}
			if firstTime.IsZero() || req.CreatedAt.Before(firstTime) {
				firstTime = req.CreatedAt
			}
			if lastTime.IsZero() || req.CreatedAt.After(lastTime) {
				lastTime = req.CreatedAt
			}

			hs, ok := hostMap[req.Host]
			if !ok {
				hs = newHostSummary(req.Host)
				hostMap[req.Host] = hs
			}
			hs.add(req)
		}

		hosts := make([]*hostSummary, 0, len(hostMap))
		for _, hs := range hostMap {
			hs.finalize()
			hosts = append(hosts, hs)
		}
		sort.Slice(hosts, func(i, j int) bool {
			return hosts[i].RequestCount > hosts[j].RequestCount
		})

		durationSec := int(lastTime.Sub(firstTime).Seconds())
		if durationSec <= 0 {
			durationSec = 1
		}
		rpm := float64(totalRequests) / (float64(durationSec) / 60.0)

		observations := generateCriticalObservations(hosts, totalRequests)

		hostOutputs := make([]map[string]any, 0, len(hosts))
		for _, hs := range hosts {
			hostOutputs = append(hostOutputs, hs.toMap())
		}

		content := mustMarshalJSON(map[string]any{
			"ok":             true,
			"session_id":     sessionID,
			"total_requests": totalRequests,
			"unique_hosts":   len(hostMap),
			"time_range": map[string]any{
				"first_request_at": firstTime.Format(time.RFC3339),
				"last_request_at":  lastTime.Format(time.RFC3339),
				"duration_seconds": durationSec,
			},
			"hosts":                 hostOutputs,
			"critical_observations": observations,
			"timeline_density": map[string]any{
				"requests_per_minute": fmt.Sprintf("%.1f", rpm),
			},
		})

		return &agentruntime.ToolExecutionResult{
			Content:    content,
			Summary:    fmt.Sprintf("Session summary: %d requests across %d hosts", totalRequests, len(hostMap)),
			RequestIDs: nil,
		}, nil
	}
}

type hostSummary struct {
	Host               string
	RequestCount       int
	IsAPI              bool
	HasAuthHeaders     bool
	HasSignedParams    bool
	ContentTypes       map[string]int
	StatusDistribution map[string]int
	Endpoints          map[string]*endpointStat
}

type endpointStat struct {
	Path   string
	Method string
	Count  int
}

func newHostSummary(host string) *hostSummary {
	return &hostSummary{
		Host:               host,
		ContentTypes:       make(map[string]int),
		StatusDistribution: make(map[string]int),
		Endpoints:          make(map[string]*endpointStat),
	}
}

func (h *hostSummary) add(req *storage.Request) {
	h.RequestCount++

	ct := strings.ToLower(req.ContentType)
	if ct != "" {
		h.ContentTypes[ct]++
	}

	statusBucket := statusBucket(req.StatusCode)
	h.StatusDistribution[statusBucket]++

	key := fmt.Sprintf("%s %s", req.Method, req.Path)
	if ep, ok := h.Endpoints[key]; ok {
		ep.Count++
	} else {
		h.Endpoints[key] = &endpointStat{Path: req.Path, Method: req.Method, Count: 1}
	}

	if !h.IsAPI {
		h.IsAPI = isAPIRequest(req)
	}
	if !h.HasAuthHeaders {
		h.HasAuthHeaders = hasAuthHeaders(req)
	}
	if !h.HasSignedParams {
		h.HasSignedParams = hasSignedParams(req)
	}
}

func (h *hostSummary) finalize() {
	if h.ContentTypes == nil {
		h.ContentTypes = make(map[string]int)
	}
	if h.StatusDistribution == nil {
		h.StatusDistribution = make(map[string]int)
	}
	if h.Endpoints == nil {
		h.Endpoints = make(map[string]*endpointStat)
	}
}

func (h *hostSummary) toMap() map[string]any {
	cts := make([]string, 0, len(h.ContentTypes))
	for ct := range h.ContentTypes {
		cts = append(cts, ct)
	}

	// top endpoints by frequency
	eps := make([]*endpointStat, 0, len(h.Endpoints))
	for _, ep := range h.Endpoints {
		eps = append(eps, ep)
	}
	sort.Slice(eps, func(i, j int) bool {
		return eps[i].Count > eps[j].Count
	})
	if len(eps) > 10 {
		eps = eps[:10]
	}
	keyEndpoints := make([]map[string]any, 0, len(eps))
	for _, ep := range eps {
		keyEndpoints = append(keyEndpoints, map[string]any{
			"path":   ep.Path,
			"method": ep.Method,
			"count":  ep.Count,
		})
	}

	return map[string]any{
		"host":                h.Host,
		"request_count":       h.RequestCount,
		"is_api":              h.IsAPI,
		"has_auth_headers":    h.HasAuthHeaders,
		"has_signed_params":   h.HasSignedParams,
		"content_types":       cts,
		"status_distribution": h.StatusDistribution,
		"key_endpoints":       keyEndpoints,
	}
}

func statusBucket(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "other"
	}
}

func isAPIRequest(req *storage.Request) bool {
	ct := strings.ToLower(req.ContentType)
	if strings.Contains(ct, "json") || strings.Contains(ct, "xml") || strings.Contains(ct, "form") {
		return true
	}
	path := strings.ToLower(req.Path)
	staticExts := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot", ".mp4", ".webp"}
	for _, ext := range staticExts {
		if strings.HasSuffix(path, ext) {
			return false
		}
	}
	return true
}

func hasAuthHeaders(req *storage.Request) bool {
	authHeaders := []string{"authorization", "x-auth-token", "x-api-key", "cookie"}
	for key := range req.Headers {
		lower := strings.ToLower(key)
		for _, ah := range authHeaders {
			if lower == ah {
				return true
			}
		}
	}
	return false
}

func hasSignedParams(req *storage.Request) bool {
	// query
	if req.QueryString != "" {
		values, err := url.ParseQuery(req.QueryString)
		if err == nil {
			for key := range values {
				if isSignedParamName(key) {
					return true
				}
			}
		}
	}
	// body
	bodyPreview := previewBody(req.Body, req.ContentType, req.Headers)
	if isSignedParamName(bodyPreview) {
		return true
	}
	// headers
	for key := range req.Headers {
		if isSignedParamName(key) {
			return true
		}
	}
	return false
}

func isSignedParamName(s string) bool {
	lower := strings.ToLower(s)
	signatures := []string{"sign", "signature", "hmac", "md5", "sha1", "sha256", "sha512"}
	for _, sig := range signatures {
		if strings.Contains(lower, sig) {
			return true
		}
	}
	return false
}

func generateCriticalObservations(hosts []*hostSummary, totalRequests int) []string {
	obs := make([]string, 0, 8)
	if len(hosts) == 0 {
		return obs
	}

	// Top host
	top := hosts[0]
	pct := float64(top.RequestCount) * 100.0 / float64(totalRequests)
	obs = append(obs, fmt.Sprintf("Host %s accounts for %.0f%% of traffic (%d requests) — likely the main API", top.Host, pct, top.RequestCount))

	for _, hs := range hosts {
		if hs.HasAuthHeaders && hs.IsAPI {
			obs = append(obs, fmt.Sprintf("Host %s has auth headers — potential auth flow target", hs.Host))
		}
		if hs.HasSignedParams {
			obs = append(obs, fmt.Sprintf("Host %s has signed parameters in requests — signature mechanism suspected", hs.Host))
		}
		// check for static
		if !hs.IsAPI && hs.RequestCount > 5 {
			obs = append(obs, fmt.Sprintf("Host %s has %d non-API requests — likely static resources or CDN", hs.Host, hs.RequestCount))
		}
	}

	if len(obs) > 6 {
		obs = obs[:6]
	}
	return obs
}
