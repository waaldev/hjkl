package ai

import (
	"context"
	"strings"
	"testing"
)

const bossCode = "func load() error {\n\tlog.Println(\"debug\")\n\terr := read()\n\treturn err\n}"

func TestGenerateBoss(t *testing.T) {
	good := `{"title":"rename and clean","brief":"Rename err to e and drop the debug line.",
"target":"func load() error {\n\te := read()\n\treturn e\n}",
"solution":"jdd/err<CR>cgne<Esc>.","skills":["gn","dot"],"principle":"gn","hints":["two jobs"]}`
	p := &scripted{replies: []string{good}}
	ch, err := GenerateBoss(context.Background(), p, BossRequest{Code: bossCode, Language: "go"}, nvimVerify(t))
	if err != nil {
		t.Fatal(err)
	}
	if ch.Start != bossCode || ch.Type != "boss" || !strings.HasPrefix(ch.ID, "boss-") || ch.Belt != "personal" {
		t.Fatalf("boss=%+v", ch)
	}
	if last := ch.Hints[len(ch.Hints)-1]; last != "answer: "+ch.Solution {
		t.Fatalf("hint ladder should end in the answer: %q", ch.Hints)
	}
	if !strings.Contains(p.prompts[0], "  2| \tlog.Println") {
		t.Fatalf("prompt should carry the numbered code:\n%s", p.prompts[0])
	}
}

func TestGenerateBossRejectsSmallOrSingleSkill(t *testing.T) {
	tooSmall := `{"target":"func load() error {\n\terr := read()\n\treturn err\n}","solution":"jdd","skills":["dd-yy-p","hjkl"]}`
	oneSkill := `{"target":"func load() error {\n\terr := read()\n\treturn err\n}","solution":"jddjjjjjjjjk","skills":["dd-yy-p"]}`
	p := &scripted{replies: []string{tooSmall, oneSkill, oneSkill}}
	_, err := GenerateBoss(context.Background(), p, BossRequest{Code: bossCode}, nvimVerify(t))
	if err == nil {
		t.Fatal("a one-step or one-skill boss must be rejected")
	}
	if !strings.Contains(p.prompts[1], "several steps") || !strings.Contains(p.prompts[2], "at least two skills") {
		t.Fatalf("rejections should be fed back:\n%s\n---\n%s", p.prompts[1], p.prompts[2])
	}
}

func TestGenerateBossLimits(t *testing.T) {
	p := &scripted{replies: []string{"{}"}}
	big := strings.Repeat("x\n", MaxBossLines+1)
	if _, err := GenerateBoss(context.Background(), p, BossRequest{Code: big}, nil); err == nil {
		t.Fatal("too many lines must be refused")
	}
	if len(p.prompts) != 0 {
		t.Fatal("refused code must never reach the provider")
	}
}

func TestIsSecretPath(t *testing.T) {
	for path, want := range map[string]bool{
		"main.go": false, "internal/store/store.go": false,
		".env": true, "prod.env": true, ".env.local": true, "certs/server.pem": true,
		"id_rsa.key": true, "config/secrets.yaml": true, "secrets/config.go": true,
		"aws_credentials": true, "PASSWORDS.txt": true,
	} {
		if got := IsSecretPath(path); got != want {
			t.Errorf("IsSecretPath(%q) = %v, want %v", path, got, want)
		}
	}
}
