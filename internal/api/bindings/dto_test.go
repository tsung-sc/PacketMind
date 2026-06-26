package bindings

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestBytesToTransportString_PlainText(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
		headers     map[string][]string
		want        string
	}{
		{
			name:        "html stays readable",
			data:        []byte("<html><body>ok</body></html>"),
			contentType: "text/html; charset=utf-8",
			headers:     nil,
			want:        "<html><body>ok</body></html>",
		},
		{
			name:        "javascript stays readable",
			data:        []byte("console.log('ok')"),
			contentType: "application/javascript",
			headers:     nil,
			want:        "console.log('ok')",
		},
		{
			name:        "json stays readable even with control-like whitespace",
			data:        []byte("{\n  \"a\": 1\n}"),
			contentType: "application/json",
			headers:     nil,
			want:        "{\n  \"a\": 1\n}",
		},
		{
			name:        "xml stays readable",
			data:        []byte("<?xml version=\"1.0\"?><root><item>test</item></root>"),
			contentType: "application/xml",
			headers:     nil,
			want:        "<?xml version=\"1.0\"?><root><item>test</item></root>",
		},
		{
			name:        "svg stays readable",
			data:        []byte("<svg><circle cx=\"50\" cy=\"50\" r=\"40\"/></svg>"),
			contentType: "image/svg+xml",
			headers:     nil,
			want:        "<svg><circle cx=\"50\" cy=\"50\" r=\"40\"/></svg>",
		},
		{
			name:        "form-urlencoded stays readable",
			data:        []byte("name=test&value=123"),
			contentType: "application/x-www-form-urlencoded",
			headers:     nil,
			want:        "name=test&value=123",
		},
		{
			name:        "empty data returns empty string",
			data:        []byte{},
			contentType: "text/html",
			headers:     nil,
			want:        "",
		},
		{
			name:        "nil data returns empty string",
			data:        nil,
			contentType: "text/html",
			headers:     nil,
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesToTransportString(tt.data, tt.contentType, tt.headers)
			if got != tt.want {
				t.Fatalf("bytesToTransportString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBytesToTransportString_Binary(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x01}
	want := base64.StdEncoding.EncodeToString(pngHeader)

	got := bytesToTransportString(pngHeader, "image/png", nil)
	if got != want {
		t.Fatalf("binary image should be base64, got %q, want %q", got, want)
	}
}

func TestBytesToTransportString_GzipCompressedText(t *testing.T) {
	originalHTML := "<html><body>Hello, World! This is a test page with some content.</body></html>"

	// Compress with gzip
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	_, err := gzWriter.Write([]byte(originalHTML))
	if err != nil {
		t.Fatalf("failed to write gzip: %v", err)
	}
	gzWriter.Close()

	compressed := buf.Bytes()

	tests := []struct {
		name        string
		contentType string
		headers     map[string][]string
		wantText    string // Expected decoded text
		wantBase64  bool   // If true, expect base64 output
	}{
		{
			name:        "gzip compressed html with Content-Encoding header",
			contentType: "text/html; charset=utf-8",
			headers:     map[string][]string{"Content-Encoding": {"gzip"}},
			wantText:    originalHTML,
			wantBase64:  false,
		},
		{
			name:        "gzip compressed html without Content-Encoding header (should be base64)",
			contentType: "text/html; charset=utf-8",
			headers:     nil,
			wantBase64:  true, // Compressed bytes without encoding hint -> base64
		},
		{
			name:        "gzip compressed json with Content-Encoding header",
			contentType: "application/json",
			headers:     map[string][]string{"Content-Encoding": {"gzip"}},
			wantText:    originalHTML, // We're using HTML content but json content-type
			wantBase64:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesToTransportString(compressed, tt.contentType, tt.headers)

			if tt.wantBase64 {
				expected := base64.StdEncoding.EncodeToString(compressed)
				if got != expected {
					t.Fatalf("expected base64, got %q, want %q", got, expected)
				}
			} else {
				if got != tt.wantText {
					t.Fatalf("expected decoded text, got %q, want %q", got, tt.wantText)
				}
			}
		})
	}
}

func TestBytesToTransportString_DeflateCompressedText(t *testing.T) {
	originalText := "This is some text content that will be compressed with deflate."

	// Compress with deflate
	var buf bytes.Buffer
	flateWriter, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		t.Fatalf("failed to create flate writer: %v", err)
	}
	_, err = flateWriter.Write([]byte(originalText))
	if err != nil {
		t.Fatalf("failed to write flate: %v", err)
	}
	flateWriter.Close()

	compressed := buf.Bytes()

	// Test with Content-Encoding header
	got := bytesToTransportString(compressed, "text/plain", map[string][]string{"Content-Encoding": {"deflate"}})
	if got != originalText {
		t.Fatalf("deflate compressed text with header should decode, got %q, want %q", got, originalText)
	}

	// Test without Content-Encoding header (should be base64)
	gotNoHeader := bytesToTransportString(compressed, "text/plain", nil)
	expected := base64.StdEncoding.EncodeToString(compressed)
	if gotNoHeader != expected {
		t.Fatalf("deflate compressed text without header should be base64, got length %d, want length %d", len(gotNoHeader), len(expected))
	}
}

func TestBytesToTransportString_InvalidGzip(t *testing.T) {
	// Invalid gzip data
	invalidGzip := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff} // Truncated gzip

	got := bytesToTransportString(invalidGzip, "text/html", map[string][]string{"Content-Encoding": {"gzip"}})
	expected := base64.StdEncoding.EncodeToString(invalidGzip)

	if got != expected {
		t.Fatalf("invalid gzip should fall back to base64, got %q, want %q", got, expected)
	}
}

func TestBytesToTransportString_InvalidUTF8(t *testing.T) {
	// Invalid UTF-8 sequence in text content-type
	invalidUTF8 := []byte{0xff, 0xfe, 0xfd}

	got := bytesToTransportString(invalidUTF8, "text/html", nil)
	expected := base64.StdEncoding.EncodeToString(invalidUTF8)

	if got != expected {
		t.Fatalf("invalid UTF-8 in text content-type should be base64, got %q, want %q", got, expected)
	}
}

func TestBytesToTransportString_BrotliEncoding(t *testing.T) {
	// Brotli is now supported — compressed text should be decompressed
	html := "<!DOCTYPE html><html><head><title>Brotli Test</title></head><body>Hello Brotli</body></html>"
	compressed := brotliCompress([]byte(html))

	got := bytesToTransportString(compressed, "text/html; charset=utf-8", map[string][]string{
		"Content-Encoding": {"br"},
	})

	if got != html {
		t.Errorf("brotli encoding: expected decompressed text, got base64 or wrong content (len=%d)", len(got))
	}
}

func TestBytesToTransportString_IdentityEncoding(t *testing.T) {
	text := "plain text with identity encoding"

	got := bytesToTransportString([]byte(text), "text/plain", map[string][]string{"Content-Encoding": {"identity"}})

	if got != text {
		t.Fatalf("identity encoding should pass through, got %q, want %q", got, text)
	}
}

func TestBytesToTransportString_CaseInsensitiveHeaders(t *testing.T) {
	originalText := "test content"

	// Compress with gzip
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write([]byte(originalText))
	gzWriter.Close()

	tests := []struct {
		name    string
		headers map[string][]string
	}{
		{"lowercase", map[string][]string{"content-encoding": {"gzip"}}},
		{"uppercase", map[string][]string{"CONTENT-ENCODING": {"gzip"}}},
		{"mixed case", map[string][]string{"Content-Encoding": {"gzip"}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bytesToTransportString(buf.Bytes(), "text/plain", tt.headers)
			if got != originalText {
				t.Fatalf("case-insensitive header lookup failed, got %q, want %q", got, originalText)
			}
		})
	}
}

func TestBytesToTransportString_HeuristicFallback(t *testing.T) {
	// Text-like content without explicit content-type should be detected by heuristics
	text := "This looks like plain text without a content-type"

	got := bytesToTransportString([]byte(text), "", nil)

	if got != text {
		t.Fatalf("text-like content without content-type should pass heuristics, got %q, want %q", got, text)
	}
}

func TestIsTextLikeContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"text/html", true},
		{"text/plain", true},
		{"text/css", true},
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"application/javascript", true},
		{"application/x-javascript", true},
		{"application/ecmascript", true},
		{"application/xml", true},
		{"text/xml", true},
		{"application/xml+soap", true},
		{"application/x-www-form-urlencoded", true},
		{"application/graphql", true},
		{"image/svg+xml", true},
		{"image/png", false},
		{"image/jpeg", false},
		{"video/mp4", false},
		{"audio/mpeg", false},
		{"application/octet-stream", false},
		{"application/pdf", false},
		{"application/zip", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			if got := isTextLikeContentType(tt.contentType); got != tt.want {
				t.Fatalf("isTextLikeContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestIsValidUTF8OrASCII(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{"valid ASCII", []byte("hello world"), true},
		{"valid UTF-8", []byte("你好世界"), true},
		{"valid UTF-8 with emoji", []byte("Hello 🌍"), true},
		{"invalid UTF-8", []byte{0xff, 0xfe, 0xfd}, false},
		{"incomplete UTF-8 sequence", []byte{0xc3, 0x28}, false}, // Invalid start byte
		{"empty", []byte{}, true},
		{"nil", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidUTF8OrASCII(tt.data); got != tt.want {
				t.Fatalf("isValidUTF8OrASCII() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetHeader(t *testing.T) {
	headers := map[string][]string{
		"Content-Type":     {"text/html"},
		"content-encoding": {"gzip"},
		"X-Custom":         {"value1", "value2"},
	}

	tests := []struct {
		key  string
		want string
	}{
		{"Content-Type", "text/html"},
		{"content-type", "text/html"},
		{"CONTENT-TYPE", "text/html"},
		{"Content-Encoding", "gzip"},
		{"content-encoding", "gzip"},
		{"X-Custom", "value1"},
		{"x-custom", "value1"},
		{"Non-Existent", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := getHeader(headers, tt.key); got != tt.want {
				t.Fatalf("getHeader(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}

	// Test nil headers
	if got := getHeader(nil, "Content-Type"); got != "" {
		t.Fatalf("getHeader(nil, ...) should return empty string, got %q", got)
	}
}

// --- test helpers ---

// brotliCompress compresses data with brotli for test fixtures.
func brotliCompress(data []byte) []byte {
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	_, _ = w.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

// gzipCompress compresses data with gzip for test fixtures.
func gzipCompress(data []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, _ = w.Write(data)
	_ = w.Close()
	return buf.Bytes()
}

// gbkEncode encodes a UTF-8 string to GBK bytes.
func gbkEncode(s string) []byte {
	data, _ := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(s))
	return data
}

// iso88591Encode encodes a UTF-8 string to ISO-8859-1 bytes.
func iso88591Encode(s string) []byte {
	data, _ := charmap.ISO8859_1.NewEncoder().Bytes([]byte(s))
	return data
}

// --- Brotli + charset tests ---

func TestBytesToTransportString_BrotliJSON(t *testing.T) {
	jsonBody := `{"message":"hello brotli","status":"ok"}`
	compressed := brotliCompress([]byte(jsonBody))

	result := bytesToTransportString(compressed, "application/json", map[string][]string{
		"Content-Encoding": {"br"},
	})

	if result != jsonBody {
		t.Errorf("brotli JSON: expected decompressed text\ngot: %s", result)
	}
}

func TestBytesToTransportString_GBKText(t *testing.T) {
	original := "你好世界"
	gbkBytes := gbkEncode(original)

	result := bytesToTransportString(gbkBytes, "text/html; charset=gbk", map[string][]string{})

	if result == base64.StdEncoding.EncodeToString(gbkBytes) {
		t.Error("GBK text: should have been transcoded, not base64'd")
	}
	if !strings.Contains(result, original) {
		t.Errorf("GBK text: expected UTF-8 output containing %q, got %q", original, result)
	}
}

func TestBytesToTransportString_GB2312Text(t *testing.T) {
	// GB2312 is a subset of GBK; htmlindex resolves "gb2312" to the GBK encoder.
	original := "中文内容测试"
	gbkBytes := gbkEncode(original)

	result := bytesToTransportString(gbkBytes, "text/plain; charset=gb2312", map[string][]string{})

	if result == base64.StdEncoding.EncodeToString(gbkBytes) {
		t.Error("GB2312 text: should have been transcoded, not base64'd")
	}
	if !strings.Contains(result, original) {
		t.Errorf("GB2312 text: expected UTF-8 output containing %q, got %q", original, result)
	}
}

func TestBytesToTransportString_ISO8859_1Text(t *testing.T) {
	original := "café résumé naïve"
	latinBytes := iso88591Encode(original)

	result := bytesToTransportString(latinBytes, "text/html; charset=iso-8859-1", map[string][]string{})

	if result == base64.StdEncoding.EncodeToString(latinBytes) {
		t.Error("ISO-8859-1 text: should have been transcoded, not base64'd")
	}
	if !strings.Contains(result, original) {
		t.Errorf("ISO-8859-1 text: expected UTF-8 output containing %q, got %q", original, result)
	}
}

func TestBytesToTransportString_UnsupportedCharsetFallback(t *testing.T) {
	// Non-UTF8 data with a fake charset that htmlindex won't resolve.
	// Should fall back to base64 using the decoded bytes.
	nonUTF8 := gbkEncode("你好") // GBK bytes, invalid UTF-8
	ct := "text/html; charset=fake-charset-xyz"

	result := bytesToTransportString(nonUTF8, ct, map[string][]string{})

	decoded, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("unsupported charset: expected base64 output, got error: %v (result=%q)", err, result)
	}
	if !bytes.Equal(decoded, nonUTF8) {
		t.Error("unsupported charset fallback: base64 should decode to original data")
	}
}

func TestBytesToTransportString_BinaryImage(t *testing.T) {
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00}

	result := bytesToTransportString(data, "image/png", map[string][]string{})

	decoded, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("binary image: expected base64, got decode error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Error("binary image: base64 should round-trip to original bytes")
	}
}

func TestBytesToTransportString_BinaryZIP(t *testing.T) {
	data := []byte{0x50, 0x4B, 0x03, 0x04, 0x00, 0x00, 0x00, 0x00}

	result := bytesToTransportString(data, "application/zip", map[string][]string{})

	decoded, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("binary zip: expected base64, got decode error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Error("binary zip: base64 should round-trip to original bytes")
	}
}

func TestBytesToTransportString_UTF8Passthrough(t *testing.T) {
	text := "Hello, 世界! Привет! 🌍"
	result := bytesToTransportString([]byte(text), "text/plain; charset=utf-8", map[string][]string{})
	if result != text {
		t.Errorf("UTF-8 passthrough: expected %q, got %q", text, result)
	}
}

func TestBytesToTransportString_BrotliGBKHTML(t *testing.T) {
	// Brotli-compressed GBK HTML: should decompress then transcode
	original := "<html><body>你好世界</body></html>"
	gbkBytes := gbkEncode(original)
	compressed := brotliCompress(gbkBytes)

	result := bytesToTransportString(compressed, "text/html; charset=gbk", map[string][]string{
		"Content-Encoding": {"br"},
	})

	if result == base64.StdEncoding.EncodeToString(compressed) {
		t.Error("brotli+GBK: should have decompressed and transcoded, not base64'd compressed bytes")
	}
	if result == base64.StdEncoding.EncodeToString(gbkBytes) {
		t.Error("brotli+GBK: should have transcoded, not base64'd decoded GBK bytes")
	}
	if !strings.Contains(result, "你好世界") {
		t.Errorf("brotli+GBK: expected UTF-8 text containing '你好世界', got %q", result)
	}
}

func TestBytesToTransportString_GzipGBKHTML(t *testing.T) {
	// Gzip-compressed GBK text: should decompress then transcode
	original := "测试gzip中文"
	gbkBytes := gbkEncode(original)
	compressed := gzipCompress(gbkBytes)

	result := bytesToTransportString(compressed, "text/html; charset=gbk", map[string][]string{
		"Content-Encoding": {"gzip"},
	})

	if result == base64.StdEncoding.EncodeToString(compressed) {
		t.Error("gzip+GBK: should have decompressed and transcoded, not base64'd compressed bytes")
	}
	if !strings.Contains(result, original) {
		t.Errorf("gzip+GBK: expected UTF-8 text containing %q, got %q", original, result)
	}
}

func TestBytesToTransportString_FallbackUsesDecodedBytes(t *testing.T) {
	// Text content-type, no charset hint, non-UTF8 decoded bytes.
	// Should base64-encode the decoded bytes (same as input here, no compression).
	nonUTF8 := gbkEncode("你好")

	result := bytesToTransportString(nonUTF8, "text/html", map[string][]string{})

	decoded, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("expected base64 output, got error: %v", err)
	}
	if !bytes.Equal(decoded, nonUTF8) {
		t.Error("fallback should use decoded bytes, not some other byte sequence")
	}
}

func TestBytesToTransportString_UnknownEncoding_Base64(t *testing.T) {
	data := []byte("some data")
	result := bytesToTransportString(data, "text/html", map[string][]string{
		"Content-Encoding": {"zstd"},
	})
	decoded, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		t.Fatalf("unknown encoding: expected base64, got error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Error("unknown encoding: base64 should encode original bytes")
	}
}

// --- extractCharset tests ---

func TestExtractCharset(t *testing.T) {
	tests := []struct {
		ct       string
		expected string
	}{
		{"text/html; charset=gbk", "gbk"},
		{"text/html; charset=UTF-8", ""},
		{"text/html; charset=utf8", ""},
		{"text/html; charset=\"iso-8859-1\"", "iso-8859-1"},
		{"text/html; charset=gb2312; other=param", "gb2312"},
		{"text/html", ""},
		{"application/json; charset= shift_jis ", "shift_jis"},
		{"text/plain;Charset=windows-1252", "windows-1252"},
	}

	for _, tc := range tests {
		got := extractCharset(tc.ct)
		if got != tc.expected {
			t.Errorf("extractCharset(%q) = %q, want %q", tc.ct, got, tc.expected)
		}
	}
}

// --- decodeBrotli tests ---

func TestDecodeBrotli(t *testing.T) {
	original := []byte("brotli decompression test")
	compressed := brotliCompress(original)

	decoded, ok := decodeBrotli(compressed)
	if !ok {
		t.Fatal("decodeBrotli: expected ok=true")
	}
	if !bytes.Equal(decoded, original) {
		t.Errorf("decodeBrotli: got %q, want %q", decoded, original)
	}
}

func TestDecodeBrotli_InvalidData(t *testing.T) {
	_, ok := decodeBrotli([]byte("not brotli data"))
	if ok {
		t.Error("decodeBrotli: expected ok=false for invalid data")
	}
}

// --- transcodeToUTF8 tests ---

func TestTranscodeToUTF8_GBK(t *testing.T) {
	original := "你好"
	gbkBytes := gbkEncode(original)

	result, err := transcodeToUTF8(gbkBytes, "gbk")
	if err != nil {
		t.Fatalf("transcodeToUTF8 gbk: %v", err)
	}
	if string(result) != original {
		t.Errorf("transcodeToUTF8 gbk: got %q, want %q", string(result), original)
	}
}

func TestTranscodeToUTF8_ISO8859_1(t *testing.T) {
	original := "café"
	latinBytes := iso88591Encode(original)

	result, err := transcodeToUTF8(latinBytes, "iso-8859-1")
	if err != nil {
		t.Fatalf("transcodeToUTF8 iso-8859-1: %v", err)
	}
	if string(result) != original {
		t.Errorf("transcodeToUTF8 iso-8859-1: got %q, want %q", string(result), original)
	}
}

func TestTranscodeToUTF8_UnsupportedCharset(t *testing.T) {
	_, err := transcodeToUTF8([]byte("data"), "totally-fake-charset")
	if err == nil {
		t.Error("transcodeToUTF8: expected error for unsupported charset")
	}
}
