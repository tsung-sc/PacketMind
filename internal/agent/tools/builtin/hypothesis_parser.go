package builtin

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// hypothesisExpr represents a parsed hypothesis expression node.
type hypothesisExpr interface {
	Eval(ctx *evalContext) (string, error)
}

// evalContext provides request data for EXTRACT operations.
type evalContext struct {
	req      requestAccessor
	reqIndex int
}

type requestAccessor interface {
	BodyText() string
	HeaderText(name string) string
	QueryText(key string) string
	RespBodyText() string
	RespHeaderText(name string) string
	CookieText(key string) string
}

// parser state
type hypothesisParser struct {
	input string
	pos   int
}

func parseHypothesis(input string) (hypothesisExpr, error) {
	p := &hypothesisParser{input: strings.TrimSpace(input)}
	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos < len(p.input) {
		return nil, fmt.Errorf("unexpected trailing characters at position %d: %s", p.pos, p.input[p.pos:])
	}
	return expr, nil
}

func (p *hypothesisParser) parseExpr() (hypothesisExpr, error) {
	p.skipWhitespace()
	if p.pos >= len(p.input) {
		return nil, fmt.Errorf("unexpected end of input")
	}

	// Check for function call or quoted literal
	if p.input[p.pos] == '"' {
		return p.parseLiteral()
	}

	return p.parseFunction()
}

func (p *hypothesisParser) parseFunction() (hypothesisExpr, error) {
	name, err := p.parseIdentifier()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if !p.consume('(') {
		return nil, fmt.Errorf("expected '(' after %s", name)
	}

	args, err := p.parseArgs()
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if !p.consume(')') {
		return nil, fmt.Errorf("expected ')' after arguments to %s", name)
	}

	switch strings.ToUpper(name) {
	case "EXTRACT":
		return p.buildExtract(args)
	case "CONCAT":
		return &concatExpr{parts: args}, nil
	case "CONCAT_WITH":
		return p.buildConcatWith(args)
	case "LOWER":
		return &unaryExpr{name: "LOWER", arg: args[0], fn: strings.ToLower}, nil
	case "UPPER":
		return &unaryExpr{name: "UPPER", arg: args[0], fn: strings.ToUpper}, nil
	case "MD5":
		return &hashExpr{name: "MD5", arg: args[0], fn: md5Hash}, nil
	case "SHA1":
		return &hashExpr{name: "SHA1", arg: args[0], fn: sha1Hash}, nil
	case "SHA256":
		return &hashExpr{name: "SHA256", arg: args[0], fn: sha256Hash}, nil
	case "HMAC_SHA256":
		return p.buildHMAC(args)
	case "BASE64":
		return &unaryExpr{name: "BASE64", arg: args[0], fn: base64Encode}, nil
	case "URLENCODE":
		return &unaryExpr{name: "URLENCODE", arg: args[0], fn: urlEncode}, nil
	default:
		return nil, fmt.Errorf("unknown function: %s", name)
	}
}

func (p *hypothesisParser) parseIdentifier() (string, error) {
	start := p.pos
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' {
			p.pos++
		} else {
			break
		}
	}
	if start == p.pos {
		return "", fmt.Errorf("expected identifier at position %d", p.pos)
	}
	return p.input[start:p.pos], nil
}

func (p *hypothesisParser) parseArgs() ([]hypothesisExpr, error) {
	args := make([]hypothesisExpr, 0)
	for {
		p.skipWhitespace()
		if p.pos >= len(p.input) {
			return args, nil
		}
		if p.input[p.pos] == ')' {
			return args, nil
		}
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		p.skipWhitespace()
		if p.consume(',') {
			continue
		}
		if p.pos < len(p.input) && p.input[p.pos] == ')' {
			return args, nil
		}
		return nil, fmt.Errorf("expected ',' or ')' at position %d", p.pos)
	}
}

func (p *hypothesisParser) parseLiteral() (hypothesisExpr, error) {
	if !p.consume('"') {
		return nil, fmt.Errorf("expected %q at position %d", "\"", p.pos)
	}
	var sb strings.Builder
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == '"' {
			p.pos++
			return &literalExpr{value: sb.String()}, nil
		}
		if c == '\\' && p.pos+1 < len(p.input) {
			p.pos++
			switch p.input[p.pos] {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '"', '\\':
				sb.WriteByte(p.input[p.pos])
			default:
				sb.WriteByte(p.input[p.pos])
			}
			p.pos++
		} else {
			sb.WriteByte(c)
			p.pos++
		}
	}
	return nil, fmt.Errorf("unterminated string literal")
}

func (p *hypothesisParser) buildExtract(args []hypothesisExpr) (hypothesisExpr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("EXTRACT requires at least 2 arguments: source, path")
	}
	sourceLit, ok := args[0].(*literalExpr)
	if !ok {
		return nil, fmt.Errorf("EXTRACT first argument must be a literal source name")
	}
	pathLit, ok := args[1].(*literalExpr)
	if !ok {
		return nil, fmt.Errorf("EXTRACT second argument must be a literal path")
	}
	return &extractExpr{source: strings.ToLower(sourceLit.value), path: pathLit.value}, nil
}

func (p *hypothesisParser) buildConcatWith(args []hypothesisExpr) (hypothesisExpr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("CONCAT_WITH requires at least 2 arguments: separator, ...parts")
	}
	sepLit, ok := args[0].(*literalExpr)
	if !ok {
		return nil, fmt.Errorf("CONCAT_WITH first argument must be a literal separator")
	}
	return &concatWithExpr{separator: sepLit.value, parts: args[1:]}, nil
}

func (p *hypothesisParser) buildHMAC(args []hypothesisExpr) (hypothesisExpr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("HMAC_SHA256 requires exactly 2 arguments: data, key")
	}
	// key may be specified as key=EXPR or just EXPR
	keyExpr := args[1]
	if lit, ok := args[1].(*literalExpr); ok && strings.HasPrefix(lit.value, "secret=") {
		// re-parse the inner expression after "secret="
		inner := strings.TrimPrefix(lit.value, "secret=")
		innerParser := &hypothesisParser{input: inner}
		parsed, err := innerParser.parseExpr()
		if err != nil {
			return nil, fmt.Errorf("failed to parse secret expression: %w", err)
		}
		keyExpr = parsed
	}
	return &hmacExpr{data: args[0], key: keyExpr}, nil
}

func (p *hypothesisParser) skipWhitespace() {
	for p.pos < len(p.input) {
		c := p.input[p.pos]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			p.pos++
		} else {
			break
		}
	}
}

func (p *hypothesisParser) consume(expected byte) bool {
	if p.pos < len(p.input) && p.input[p.pos] == expected {
		p.pos++
		return true
	}
	return false
}

// expression implementations

type literalExpr struct {
	value string
}

func (e *literalExpr) Eval(ctx *evalContext) (string, error) {
	return e.value, nil
}

type extractExpr struct {
	source string
	path   string
}

func (e *extractExpr) Eval(ctx *evalContext) (string, error) {
	switch e.source {
	case "body":
		if e.path == "body_raw" {
			return ctx.req.BodyText(), nil
		}
		return extractJSONPath(ctx.req.BodyText(), e.path), nil
	case "header":
		return ctx.req.HeaderText(e.path), nil
	case "query":
		return ctx.req.QueryText(e.path), nil
	case "response_body":
		return extractJSONPath(ctx.req.RespBodyText(), e.path), nil
	case "response_header":
		return ctx.req.RespHeaderText(e.path), nil
	case "cookie":
		return ctx.req.CookieText(e.path), nil
	default:
		return "", fmt.Errorf("unknown extract source: %s", e.source)
	}
}

type concatExpr struct {
	parts []hypothesisExpr
}

func (e *concatExpr) Eval(ctx *evalContext) (string, error) {
	var sb strings.Builder
	for _, part := range e.parts {
		val, err := part.Eval(ctx)
		if err != nil {
			return "", err
		}
		sb.WriteString(val)
	}
	return sb.String(), nil
}

type concatWithExpr struct {
	separator string
	parts     []hypothesisExpr
}

func (e *concatWithExpr) Eval(ctx *evalContext) (string, error) {
	vals := make([]string, 0, len(e.parts))
	for _, part := range e.parts {
		v, err := part.Eval(ctx)
		if err != nil {
			return "", err
		}
		vals = append(vals, v)
	}
	return strings.Join(vals, e.separator), nil
}

type unaryExpr struct {
	name string
	arg  hypothesisExpr
	fn   func(string) string
}

func (e *unaryExpr) Eval(ctx *evalContext) (string, error) {
	val, err := e.arg.Eval(ctx)
	if err != nil {
		return "", err
	}
	return e.fn(val), nil
}

type hashExpr struct {
	name string
	arg  hypothesisExpr
	fn   func(string) string
}

func (e *hashExpr) Eval(ctx *evalContext) (string, error) {
	val, err := e.arg.Eval(ctx)
	if err != nil {
		return "", err
	}
	return e.fn(val), nil
}

type hmacExpr struct {
	data hypothesisExpr
	key  hypothesisExpr
}

func (e *hmacExpr) Eval(ctx *evalContext) (string, error) {
	dataVal, err := e.data.Eval(ctx)
	if err != nil {
		return "", err
	}
	keyVal, err := e.key.Eval(ctx)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(keyVal))
	mac.Write([]byte(dataVal))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

// hash helpers

func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func sha1Hash(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func urlEncode(s string) string {
	return url.QueryEscape(s)
}

// JSON path extraction (simple form: $.data.userId or data.userId)
func extractJSONPath(jsonStr, path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return ""
	}

	// Use regex to find "key": "value" or "key": value patterns
	current := jsonStr
	for _, key := range parts {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		// Try array index
		if idx, err := parseInt(key); err == nil && idx >= 0 {
			current = extractJSONArrayItem(current, idx)
			continue
		}
		// Try object key
		pattern := fmt.Sprintf(`"%s"\s*:\s*("(?:[^"\\]|\\.)*"|\[[^\]]*\]|\{[^}]*\}|[^,\}\]]*)`, regexp.QuoteMeta(key))
		re := regexp.MustCompile(pattern)
		match := re.FindStringSubmatch(current)
		if len(match) < 2 {
			return ""
		}
		current = strings.TrimSpace(match[1])
		if strings.HasPrefix(current, "\"") && strings.HasSuffix(current, "\"") {
			current = current[1 : len(current)-1]
			current = strings.ReplaceAll(current, "\\\"", "\"")
			current = strings.ReplaceAll(current, "\\\\", "\\")
		}
	}
	return current
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

func extractJSONArrayItem(jsonStr string, idx int) string {
	jsonStr = strings.TrimSpace(jsonStr)
	if !strings.HasPrefix(jsonStr, "[") {
		return ""
	}
	jsonStr = jsonStr[1:]
	depth := 0
	inString := false
	currentIdx := 0
	start := 0
	for i, ch := range jsonStr {
		if ch == '"' && (i == 0 || jsonStr[i-1] != '\\') {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		if ch == '[' || ch == '{' {
			depth++
			continue
		}
		if ch == ']' || ch == '}' {
			depth--
			continue
		}
		if ch == ',' && depth == 0 {
			if currentIdx == idx {
				return strings.TrimSpace(jsonStr[start:i])
			}
			currentIdx++
			start = i + 1
		}
	}
	if currentIdx == idx {
		// handle trailing ] or }
		end := len(jsonStr)
		for end > 0 {
			last := jsonStr[end-1]
			if last == ']' || last == '}' || last == ' ' || last == '\t' || last == '\n' || last == '\r' {
				end--
			} else {
				break
			}
		}
		return strings.TrimSpace(jsonStr[start:end])
	}
	return ""
}
