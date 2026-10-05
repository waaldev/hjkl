package cli

import (
	"strings"
	"testing"
)

func TestBossRange(t *testing.T) {
	file := strings.Split("package x\n\nfunc a() {\n\treturn\n}\n\nfunc b() {\n\tx := 1\n\ty := 2\n\treturn\n}\n", "\n")
	first := func(int) int { return 0 }
	// Blocks between blank lines of 4+ lines: only func b qualifies.
	if lo, hi, err := bossRange(file, "", first); err != nil || lo != 7 || hi != 11 {
		t.Fatalf("auto block = %d-%d, %v", lo, hi, err)
	}
	if lo, hi, err := bossRange(file, "3-5", first); err != nil || lo != 3 || hi != 5 {
		t.Fatalf("--lines 3-5 = %d-%d, %v", lo, hi, err)
	}
	for _, bad := range []string{"0-3", "5-3", "1-99", "x", "3"} {
		if _, _, err := bossRange(file, bad, first); err == nil {
			t.Errorf("--lines %q should be rejected", bad)
		}
	}
	long := strings.Split(strings.Repeat("x\n", 50), "\n")
	if _, _, err := bossRange(long, "1-40", first); err == nil {
		t.Fatal("a 40-line range is over the limit")
	}
	if lo, hi, _ := bossRange(long, "", first); lo != 1 || hi != 30 {
		t.Fatalf("no blocks: fall back to the first 30 lines, got %d-%d", lo, hi)
	}
}

func TestBossRefusesSecretFiles(t *testing.T) {
	cmd := newBossCmd()
	cmd.SetArgs([]string{"config/.env.production"})
	cmd.SetOut(new(strings.Builder))
	cmd.SetErr(new(strings.Builder))
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "never sent") {
		t.Fatalf("want a refusal, got %v", err)
	}
}

func TestLanguageFor(t *testing.T) {
	for path, want := range map[string]string{"main.go": "go", "a/b.PY": "python", "x.tsx": "typescriptreact", "Makefile": ""} {
		if got := languageFor(path); got != want {
			t.Errorf("languageFor(%q) = %q, want %q", path, got, want)
		}
	}
}
