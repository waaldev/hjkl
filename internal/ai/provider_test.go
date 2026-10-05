package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/waaldev/hjkl/internal/curriculum"
)

func TestExtractJSON(t *testing.T) {
	raw, err := ExtractJSON("sure\n```json\n{\"a\":1}\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	if raw != `{"a":1}` {
		t.Fatalf("got %s", raw)
	}
}

func TestAnthropicAndOpenAIContract(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("content-type", "application/json")
		switch {
		case strings.Contains(r.URL.Path, "messages"):
			_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"pong"}]}`))
		default:
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"pong"}}]}`))
		}
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	a := Anthropic{APIKey: "test", Model: "claude-test", BaseURL: srv.URL, Client: srv.Client()}
	resp, err := a.Complete(ctx, Request{Prompt: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "pong" || !strings.Contains(gotPath, "messages") {
		t.Fatalf("anthropic resp=%q path=%s", resp.Text, gotPath)
	}

	o := OpenAI{APIKey: "test", Model: "gpt-test", BaseURL: srv.URL, Client: srv.Client()}
	resp, err = o.Complete(ctx, Request{Prompt: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "pong" {
		t.Fatalf("openai resp=%q", resp.Text)
	}
}

func TestCLIAgent(t *testing.T) {
	p := CLIAgent{
		Command: "stub",
		Run: func(ctx context.Context, name string, args []string, stdin string) (string, error) {
			if !strings.Contains(stdin, "ping") {
				t.Fatalf("stdin=%q", stdin)
			}
			return "pong", nil
		},
	}
	resp, err := p.Complete(context.Background(), Request{Prompt: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "pong" {
		t.Fatalf("%q", resp.Text)
	}
}

func TestCompleteJSONRetry(t *testing.T) {
	n := 0
	p := CLIAgent{
		Command: "stub",
		Run: func(ctx context.Context, name string, args []string, stdin string) (string, error) {
			n++
			if n == 1 {
				return "not json", nil
			}
			return `{"ok":true}`, nil
		},
	}
	raw, err := CompleteJSON(context.Background(), p, Request{Prompt: "give json"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil || got["ok"] != true {
		t.Fatalf("%s", raw)
	}
	if n != 2 {
		t.Fatalf("retries=%d", n)
	}
}

func TestVerifyGateRejectsWrongChallenge(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	ch := curriculum.Challenge{
		ID:          "ai-bad",
		Title:       "bad",
		Type:        curriculum.TypeTransform,
		Start:       "ht",
		Target:      "hat",
		StartCursor: []int{1, 2},
		Par:         1,
		Solution:    "l",
	}
	err := VerifyGate(context.Background(), ch, "insert", "nvim")
	if err == nil {
		t.Fatal("expected reject")
	}
}

func TestCheckGeneratedRejectsDrillsThatTeachNothing(t *testing.T) {
	good := curriculum.Challenge{Type: curriculum.TypeTransform, Start: "say hello now", Target: "say  now", Solution: "diw"}
	if err := CheckGenerated(good, "text-objects"); err != nil {
		t.Fatalf("good drill rejected: %v", err)
	}
	cases := map[string]struct {
		ch    curriculum.Challenge
		focus string
	}{
		"skill not used": {curriculum.Challenge{Type: curriculum.TypeTransform, Start: "say hello now", Target: "say  now", Solution: "xxxxx"}, "text-objects"},
		"nothing to do":  {curriculum.Challenge{Type: curriculum.TypeTransform, Start: "a", Target: "a", Solution: "dw"}, "operators"},
		"already there":  {curriculum.Challenge{Type: curriculum.TypeNavigate, StartCursor: []int{1, 3}, TargetCursor: []int{1, 3}, Solution: "w"}, "words"},
	}
	for name, c := range cases {
		if err := CheckGenerated(c.ch, c.focus); err == nil {
			t.Errorf("%s: expected reject", name)
		}
	}
}

func TestGoldenAnthropic(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "anthropic_pong.json"))
	if os.IsNotExist(err) {
		t.Skip("golden missing")
	}
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()
	a := Anthropic{APIKey: "k", BaseURL: srv.URL, Client: srv.Client()}
	resp, err := a.Complete(context.Background(), Request{Prompt: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Text, "pong") {
		t.Fatalf("%q", resp.Text)
	}
}

func TestAnthropicDefaultModelRequest(t *testing.T) {
	var body map[string]any
	var beta string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		beta = r.Header.Get("anthropic-beta")
		body = nil
		_ = json.NewDecoder(r.Body).Decode(&body)
		// Thinking blocks come before the text and must not leak into it.
		_, _ = w.Write([]byte(`{"content":[{"type":"thinking","thinking":""},{"type":"text","text":"pong"}],"stop_reason":"end_turn"}`))
	}))
	defer srv.Close()

	a := Anthropic{APIKey: "k", BaseURL: srv.URL, Client: srv.Client()}
	resp, err := a.Complete(context.Background(), Request{Prompt: "ping"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "pong" || resp.Model != DefaultAnthropicModel {
		t.Fatalf("resp=%+v", resp)
	}
	if body["model"] != DefaultAnthropicModel || body["max_tokens"].(float64) < 8000 {
		t.Fatalf("body=%v", body)
	}
	if oc, _ := body["output_config"].(map[string]any); oc["effort"] != "low" {
		t.Fatalf("default model should use low effort: %v", body)
	}
	if body["fallbacks"] != "default" || beta != fallbackBeta {
		t.Fatalf("refusal fallback not enabled: body=%v beta=%q", body, beta)
	}

	// A configured older model gets neither effort nor fallbacks.
	a.Model = "claude-haiku-4-5"
	if _, err := a.Complete(context.Background(), Request{Prompt: "ping"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["output_config"]; ok || body["fallbacks"] != nil || beta != "" {
		t.Fatalf("older model got new parameters: body=%v beta=%q", body, beta)
	}
}

func TestAnthropicRefusal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":[],"stop_reason":"refusal","stop_details":{"type":"refusal","category":"cyber","explanation":"x"}}`))
	}))
	defer srv.Close()
	a := Anthropic{APIKey: "k", BaseURL: srv.URL, Client: srv.Client()}
	_, err := a.Complete(context.Background(), Request{Prompt: "ping"})
	if err == nil || !strings.Contains(err.Error(), "declined") {
		t.Fatalf("want a refusal error, got %v", err)
	}
}
