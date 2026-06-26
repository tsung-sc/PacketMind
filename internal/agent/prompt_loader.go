package agent

import (
	"embed"
	"strings"
	"sync"
	"text/template"
)

//go:embed prompts/*.txt
var promptFS embed.FS

var (
	promptCache     = make(map[string]string)
	promptCacheMu   sync.RWMutex
	promptTemplates = make(map[string]*template.Template)
	promptTmplMu    sync.RWMutex
)

// LoadPrompt loads a static prompt file by name (without .txt extension).
// The file is read from the embedded prompts/ directory.
// Returns the trimmed file content, or empty string if not found.
func LoadPrompt(name string) string {
	promptCacheMu.RLock()
	if cached, ok := promptCache[name]; ok {
		promptCacheMu.RUnlock()
		return cached
	}
	promptCacheMu.RUnlock()

	data, err := promptFS.ReadFile("prompts/" + name + ".txt")
	if err != nil {
		return ""
	}

	content := strings.TrimSpace(string(data))

	promptCacheMu.Lock()
	promptCache[name] = content
	promptCacheMu.Unlock()

	return content
}

// ExecutePrompt loads a template prompt by name and executes it with the
// given data map. Template files use Go text/template syntax with {{.Key}}
// placeholders.
//
// Returns the rendered string. If template parsing or execution fails,
// falls back to the raw prompt text with no substitution.
func ExecutePrompt(name string, data map[string]interface{}) string {
	promptTmplMu.RLock()
	tmpl, ok := promptTemplates[name]
	promptTmplMu.RUnlock()

	if !ok {
		raw := LoadPrompt(name)
		if raw == "" {
			return ""
		}
		var err error
		tmpl, err = template.New(name).Parse(raw)
		if err != nil {
			return raw
		}
		promptTmplMu.Lock()
		promptTemplates[name] = tmpl
		promptTmplMu.Unlock()
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return LoadPrompt(name)
	}
	return buf.String()
}
