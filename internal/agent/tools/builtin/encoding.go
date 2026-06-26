package builtin

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/packetmind/packetmind/internal/agent/tools"
	agentruntime "github.com/packetmind/packetmind/internal/agent/runtime"
)

var base64Pattern = regexp.MustCompile(`^[A-Za-z0-9+/=_-]+$`)

type EncodingInfo struct {
	Chain          []string `json:"chain"`
	FinalType      string   `json:"final_type"`
	DecodedPreview string   `json:"decoded_preview,omitempty"`
	Errors         []string `json:"errors,omitempty"`
}

func newAnalyzeEncodingHandler() tools.BuiltinHandler {
	return func(args map[string]any, sessionID string) (*agentruntime.ToolExecutionResult, error) {
		_ = sessionID
		value, err := tools.GetRequiredStringArg(args, "value")
		if err != nil {
			return nil, err
		}
		info := encodingDetect(value)
		content := mustMarshalJSON(map[string]any{"ok": true, "encoding": info})

		summary := "No additional encoding detected"
		if len(info.Chain) > 0 {
			summary = fmt.Sprintf("Detected encoding chain: %s", strings.Join(info.Chain, " -> "))
		}

		return &agentruntime.ToolExecutionResult{Content: content, Summary: summary}, nil
	}
}

func encodingDetect(value string) EncodingInfo {
	current := strings.TrimSpace(value)
	info := EncodingInfo{Chain: make([]string, 0, 6), FinalType: "plain_text"}
	if current == "" {
		info.DecodedPreview = ""
		return info
	}

	seen := make(map[string]struct{})
	for i := 0; i < 8; i++ {
		signature := truncateForModel(current, 512)
		if _, exists := seen[signature]; exists {
			break
		}
		seen[signature] = struct{}{}

		next, step, advanced, err := decodeOneLayer(current)
		if err != nil {
			info.Errors = append(info.Errors, err.Error())
		}
		if !advanced {
			break
		}

		info.Chain = append(info.Chain, step...)
		current = next
	}

	trimmed := strings.TrimSpace(current)
	if json.Valid([]byte(trimmed)) {
		info.FinalType = "json"
	} else if utf8.ValidString(trimmed) {
		info.FinalType = "text"
	} else {
		info.FinalType = "binary"
	}
	info.DecodedPreview = truncateForModel(trimmed, 1000)
	return info
}

func decodeOneLayer(input string) (string, []string, bool, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return input, nil, false, nil
	}

	if decoded, ok, err := tryBase64Decode(trimmed); ok {
		steps := []string{"base64"}
		if unzipped, ok := tryGzipDecode(decoded); ok {
			decoded = unzipped
			steps = append(steps, "gzip")
		} else if inflated, ok := tryZlibDecode(decoded); ok {
			decoded = inflated
			steps = append(steps, "zlib")
		}

		result := string(decoded)
		if jsonLike := strings.TrimSpace(result); json.Valid([]byte(jsonLike)) {
			if embedded, ok := extractJSONEmbeddedString(jsonLike); ok {
				return embedded, append(steps, "json"), true, nil
			}
		}
		return result, steps, true, err
	}

	if decoded, ok := tryHexDecode(trimmed); ok {
		result := string(decoded)
		if jsonLike := strings.TrimSpace(result); json.Valid([]byte(jsonLike)) {
			if embedded, ok := extractJSONEmbeddedString(jsonLike); ok {
				return embedded, []string{"hex", "json"}, true, nil
			}
		}
		return result, []string{"hex"}, true, nil
	}

	if unescaped, err := url.QueryUnescape(trimmed); err == nil && unescaped != trimmed {
		return unescaped, []string{"url"}, true, nil
	}

	if json.Valid([]byte(trimmed)) {
		if embedded, ok := extractJSONEmbeddedString(trimmed); ok {
			return embedded, []string{"json"}, true, nil
		}
	}

	return input, nil, false, nil
}

func tryBase64Decode(value string) ([]byte, bool, error) {
	if len(value) < 8 || !base64Pattern.MatchString(value) {
		return nil, false, nil
	}

	encodings := []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding}
	var lastErr error
	for _, enc := range encodings {
		decoded, err := enc.DecodeString(value)
		if err != nil {
			lastErr = err
			continue
		}
		if len(decoded) == 0 || !looksLikeDecodedPayload(decoded) {
			continue
		}
		return decoded, true, nil
	}

	if lastErr != nil {
		return nil, false, fmt.Errorf("base64 decode failed: %w", lastErr)
	}
	return nil, false, nil
}

func looksLikeDecodedPayload(decoded []byte) bool {
	if len(decoded) == 0 {
		return false
	}
	if hasGzipMagic(decoded) || utf8.Valid(decoded) {
		return true
	}

	printable := 0
	for _, b := range decoded {
		if b == '\n' || b == '\r' || b == '\t' || (b >= 32 && b <= 126) {
			printable++
		}
	}
	return printable*100/len(decoded) >= 80
}

func tryHexDecode(value string) ([]byte, bool) {
	v := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "0x")
	if len(v) < 8 || len(v)%2 != 0 {
		return nil, false
	}
	for _, ch := range v {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return nil, false
		}
	}
	decoded, err := hex.DecodeString(v)
	if err != nil || len(decoded) == 0 || !looksLikeDecodedPayload(decoded) {
		return nil, false
	}
	return decoded, true
}

func hasGzipMagic(data []byte) bool { return len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b }

func tryGzipDecode(data []byte) ([]byte, bool) {
	if !hasGzipMagic(data) {
		return nil, false
	}
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil || len(decoded) == 0 {
		return nil, false
	}
	return decoded, true
}

func tryZlibDecode(data []byte) ([]byte, bool) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	defer reader.Close()
	decoded, err := io.ReadAll(reader)
	if err != nil || len(decoded) == 0 {
		return nil, false
	}
	return decoded, true
}

func extractJSONEmbeddedString(raw string) (string, bool) {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", false
	}

	candidates := make([]string, 0, 4)
	collectJSONStrings(value, &candidates)
	if len(candidates) == 0 {
		return "", false
	}

	sort.Slice(candidates, func(i, j int) bool { return scoreEmbeddedString(candidates[i]) > scoreEmbeddedString(candidates[j]) })
	best := strings.TrimSpace(candidates[0])
	if best == "" || best == strings.TrimSpace(raw) {
		return "", false
	}
	return best, true
}

func collectJSONStrings(value any, out *[]string) {
	switch v := value.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			*out = append(*out, trimmed)
		}
	case []any:
		for _, item := range v {
			collectJSONStrings(item, out)
		}
	case map[string]any:
		for _, item := range v {
			collectJSONStrings(item, out)
		}
	}
}

func scoreEmbeddedString(value string) int {
	v := strings.TrimSpace(value)
	score := len(v)
	if json.Valid([]byte(v)) {
		score += 200
	}
	if _, ok, _ := tryBase64Decode(v); ok {
		score += 150
	}
	if _, ok := tryHexDecode(v); ok {
		score += 80
	}
	if strings.Contains(v, "%") {
		score += 30
	}
	return score
}
