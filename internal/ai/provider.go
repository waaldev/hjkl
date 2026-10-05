package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Request struct {
	System   string
	Prompt   string
	JSONMode bool
}

type Response struct {
	Text     string
	Provider string
	Model    string
}

type Provider interface {
	Name() string
	Complete(ctx context.Context, req Request) (Response, error)
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func ExtractJSON(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```JSON")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return "", fmt.Errorf("no JSON object in model output")
	}
	return s[start : end+1], nil
}

func CompleteJSON(ctx context.Context, p Provider, req Request) (json.RawMessage, error) {
	req.JSONMode = true
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	raw, err := ExtractJSON(resp.Text)
	if err == nil && json.Valid([]byte(raw)) {
		return json.RawMessage(raw), nil
	}
	req.Prompt = req.Prompt + "\n\nYour previous reply was not valid JSON. Reply with a single JSON object only."
	resp, err = p.Complete(ctx, req)
	if err != nil {
		return nil, err
	}
	raw, err = ExtractJSON(resp.Text)
	if err != nil {
		return nil, err
	}
	if !json.Valid([]byte(raw)) {
		return nil, fmt.Errorf("model output is not JSON")
	}
	return json.RawMessage(raw), nil
}

// DefaultAnthropicModel is used when no model is configured.
const DefaultAnthropicModel = "claude-opus-5-5"

// fallbackModels accept server-side refusal fallback ("fallbacks":
// "default"): if a safety classifier declines a request, the API retries it
// on a suitable model instead of returning a refusal.
var fallbackModels = map[string]bool{
	"claude-fable-5-1": true, "claude-opus-5-5": true, "claude-opus-5": true, "claude-sonnet-5-5": true,
}

const fallbackBeta = "server-side-fallback-2026-07-01"

type Anthropic struct {
	APIKey  string
	Model   string
	Effort  string // output_config.effort; see config.ProviderKeys.Effort
	BaseURL string
	Client  HTTPDoer
}

func (a Anthropic) Name() string { return "anthropic" }

func (a Anthropic) Complete(ctx context.Context, req Request) (Response, error) {
	model := a.Model
	if model == "" {
		model = DefaultAnthropicModel
	}
	// Sensei answers are short; low effort keeps them quick and cheap on
	// the default model. Other models only get effort when configured,
	// because some older ones reject the parameter.
	effort := a.Effort
	if effort == "" && model == DefaultAnthropicModel {
		effort = "low"
	}
	base := a.BaseURL
	if base == "" {
		base = "https://api.anthropic.com"
	}
	key := a.APIKey
	if key == "" {
		key = os.Getenv("ANTHROPIC_API_KEY")
	}
	if key == "" {
		return Response{}, fmt.Errorf("anthropic: missing API key")
	}
	body := map[string]any{
		"model": model,
		// Current models think before answering and thinking counts toward
		// max_tokens, so a small cap can cut the answer off.
		"max_tokens": 16000,
		"messages":   []map[string]string{{"role": "user", "content": req.Prompt}},
	}
	if req.System != "" {
		body["system"] = req.System
	}
	if effort != "" {
		body["output_config"] = map[string]any{"effort": effort}
	}
	if fallbackModels[model] {
		body["fallbacks"] = "default"
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/v1/messages", bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("x-api-key", key)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	if fallbackModels[model] {
		httpReq.Header.Set("anthropic-beta", fallbackBeta)
	}
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer httpResp.Body.Close()
	payload, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("anthropic: %s: %s", httpResp.Status, truncate(payload, 400))
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		StopReason  string `json:"stop_reason"`
		StopDetails *struct {
			Category    string `json:"category"`
			Explanation string `json:"explanation"`
		} `json:"stop_details"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return Response{}, err
	}
	if parsed.StopReason == "refusal" {
		why := "no details"
		if d := parsed.StopDetails; d != nil {
			why = strings.TrimSpace(d.Category + " " + d.Explanation)
		}
		return Response{}, fmt.Errorf("anthropic: request declined (%s)", why)
	}
	var b strings.Builder
	for _, c := range parsed.Content {
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return Response{Text: b.String(), Provider: a.Name(), Model: model}, nil
}

type OpenAI struct {
	APIKey  string
	Model   string
	BaseURL string
	Client  HTTPDoer
}

func (o OpenAI) Name() string { return "openai" }

func (o OpenAI) Complete(ctx context.Context, req Request) (Response, error) {
	model := o.Model
	if model == "" {
		model = "gpt-4o"
	}
	base := o.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	key := o.APIKey
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	messages := []map[string]string{}
	if req.System != "" {
		messages = append(messages, map[string]string{"role": "system", "content": req.System})
	}
	messages = append(messages, map[string]string{"role": "user", "content": req.Prompt})
	body := map[string]any{"model": model, "messages": messages}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return Response{}, err
	}
	httpReq.Header.Set("content-type", "application/json")
	if key != "" {
		httpReq.Header.Set("authorization", "Bearer "+key)
	}
	client := o.Client
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return Response{}, err
	}
	defer httpResp.Body.Close()
	payload, _ := io.ReadAll(httpResp.Body)
	if httpResp.StatusCode >= 300 {
		return Response{}, fmt.Errorf("openai: %s: %s", httpResp.Status, truncate(payload, 400))
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return Response{}, err
	}
	text := ""
	if len(parsed.Choices) > 0 {
		text = parsed.Choices[0].Message.Content
	}
	return Response{Text: text, Provider: o.Name(), Model: model}, nil
}

type CLIAgent struct {
	Command string
	Args    []string
	Run     func(ctx context.Context, name string, args []string, stdin string) (string, error)
}

func (c CLIAgent) Name() string { return "cli-agent" }

func (c CLIAgent) Complete(ctx context.Context, req Request) (Response, error) {
	if c.Command == "" {
		return Response{}, fmt.Errorf("cli-agent: no command configured")
	}
	prompt := req.Prompt
	if req.System != "" {
		prompt = req.System + "\n\n" + prompt
	}
	run := c.Run
	if run == nil {
		run = func(ctx context.Context, name string, args []string, stdin string) (string, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdin = strings.NewReader(stdin)
			var out, errb bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &errb
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("cli-agent: %w: %s", err, errb.String())
			}
			return out.String(), nil
		}
	}
	text, err := run(ctx, c.Command, c.Args, prompt)
	if err != nil {
		return Response{}, err
	}
	return Response{Text: strings.TrimSpace(text), Provider: c.Name(), Model: c.Command}, nil
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
