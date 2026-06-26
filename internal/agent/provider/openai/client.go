package openai

import (
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	openaisdk "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/packetmind/packetmind/internal/agent/llmtypes"
)

type Config struct {
	AppendV1ToBaseURL        bool
	InjectThinkingOptions    bool
	ExtractReasoningInStream bool
	ProviderName             string
	ProviderID               string
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	oc         *openaisdk.Client
	modelID    string
	config     Config
}

func NewClient(apiKey, baseURL string, config Config) *Client {
	client := newClient(apiKey, baseURL, config)
	return &client
}

func newClient(apiKey, baseURL string, config Config) Client {
	client := Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		config: config,
	}
	client.rebuildClient()
	return client
}

func (c *Client) rebuildClient() {
	opts := []option.RequestOption{
		option.WithAPIKey(c.apiKey),
		option.WithMaxRetries(0),
		option.WithRequestTimeout(120 * time.Second),
	}
	if normalizedBaseURL := c.normalizeBaseURL(c.baseURL); normalizedBaseURL != "" {
		opts = append(opts, option.WithBaseURL(normalizedBaseURL))
	}
	if c.httpClient != nil {
		opts = append(opts, option.WithHTTPClient(c.httpClient))
	}
	client := openaisdk.NewClient(opts...)
	c.oc = &client
}

func (c *Client) normalizeBaseURL(baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return ""
	}
	if c.config.AppendV1ToBaseURL {
		if strings.HasSuffix(strings.ToLower(trimmed), "/v1") {
			return trimmed + "/"
		}
		return trimmed + "/v1/"
	}
	return trimmed + "/"
}

func (c *Client) providerName() string {
	if name := strings.TrimSpace(c.config.ProviderName); name != "" {
		return name
	}
	return "Provider"
}

func (c *Client) providerID() string {
	if id := strings.TrimSpace(c.config.ProviderID); id != "" {
		return id
	}
	return "provider"
}

var _ llmtypes.LLMClient = (*Client)(nil)
