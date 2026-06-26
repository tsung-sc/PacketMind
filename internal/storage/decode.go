package storage

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"strings"

	"github.com/andybalholm/brotli"
)

// DecodeBodyBytes decompresses body bytes based on Content-Encoding header.
// If Content-Encoding is absent or decompression fails, returns the original bytes unchanged.
func DecodeBodyBytes(body []byte, headers Headers) []byte {
	if len(body) == 0 || headers == nil {
		return body
	}
	encoding := getHeaderFromHeaders(headers, "Content-Encoding")
	if encoding == "" {
		return body
	}
	decoded, ok := decodeContentBody(body, encoding)
	if !ok || len(decoded) == 0 {
		return body
	}
	return decoded
}

// GetContentEncoding returns the Content-Encoding header value (case-insensitive).
func GetContentEncoding(headers Headers) string {
	return getHeaderFromHeaders(headers, "Content-Encoding")
}

func getHeaderFromHeaders(headers Headers, key string) string {
	if headers == nil {
		return ""
	}
	if values, ok := headers[key]; ok && len(values) > 0 {
		return values[0]
	}
	lowerKey := strings.ToLower(key)
	for k, values := range headers {
		if strings.ToLower(k) == lowerKey && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

func decodeContentBody(data []byte, contentEncoding string) ([]byte, bool) {
	encoding := strings.ToLower(strings.TrimSpace(contentEncoding))
	switch {
	case encoding == "gzip" || encoding == "x-gzip":
		return decodeGzipBody(data)
	case encoding == "deflate":
		return decodeDeflateBody(data)
	case encoding == "br" || encoding == "brotli":
		return decodeBrotliBody(data)
	case encoding == "identity" || encoding == "":
		return data, true
	default:
		return nil, false
	}
}

func decodeGzipBody(data []byte) ([]byte, bool) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, false
	}
	defer r.Close()
	decoded, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return decoded, true
}

func decodeDeflateBody(data []byte) ([]byte, bool) {
	zr, err := zlib.NewReader(bytes.NewReader(data))
	if err == nil {
		defer zr.Close()
		decoded, readErr := io.ReadAll(zr)
		if readErr == nil {
			return decoded, true
		}
	}
	r := flate.NewReader(bytes.NewReader(data))
	defer r.Close()
	decoded, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return decoded, true
}

func decodeBrotliBody(data []byte) ([]byte, bool) {
	r := brotli.NewReader(bytes.NewReader(data))
	decoded, err := io.ReadAll(r)
	if err != nil {
		return nil, false
	}
	return decoded, true
}
