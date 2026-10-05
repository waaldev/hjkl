package cli

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/waaldev/hjkl/internal/app"
	"github.com/waaldev/hjkl/internal/config"
	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/store"
)

func TestSuggestEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("nvim"); err != nil {
		t.Skip("nvim not installed")
	}
	t.Setenv("HJKL_DATA_DIR", t.TempDir())
	t.Setenv("HJKL_TEST_KEY", "k")

	// An OpenAI-compatible endpoint that answers with a correct shorter way.
	answer := `{"keys":"ciwbar<Esc>j.j.","explanation":"ciw then dot.","principle":"dot","skills":["text-objects","dot"]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reply, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": answer}}}})
		_, _ = w.Write(reply)
	}))
	defer srv.Close()

	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cat, _ := curriculum.Load()
	a := &app.App{Cat: cat, Store: st, Cfg: config.Config{Nvim: "nvim"}}
	a.Cfg.AI.Provider = "openai"
	a.Cfg.AI.OpenAI = config.OpenAIConfig{BaseURL: srv.URL, APIKeyEnv: "HJKL_TEST_KEY", Model: "m"}

	in := filepath.Join(t.TempDir(), "edit.json")
	edit := `{"before":"foo := 1\nfoo := 2\nfoo := 3","after":"bar := 1\nbar := 2\nbar := 3","cursor":[1,1],"keys":"xxxibar<Esc>jhhxxxibar<Esc>jhhxxxibar<Esc>","filetype":"go"}`
	if err := os.WriteFile(in, []byte(edit), 0o600); err != nil {
		t.Fatal(err)
	}

	// Off by default: nothing is sent.
	if res := suggest(a, in); res.OK || res.Error != suggestDisabled {
		t.Fatalf("disabled: %+v", res)
	}

	a.Cfg.AI.Suggest = true
	res := suggest(a, in)
	if !res.OK || res.KeyCount != 11 || res.YourKeyCount != 30 || res.Drill == "" {
		t.Fatalf("suggest: %+v", res)
	}

	// Save it, and it loads as a personal drill that reviews can pick.
	saved := saveDrill(res.Drill)
	if saved["ok"] != true {
		t.Fatalf("save: %v", saved)
	}
	dir, _ := config.DrillsDir()
	mine, err := curriculum.LoadPersonal(dir)
	if err != nil || len(mine) != 1 || mine[0].Solution != "ciwbar<Esc>j.j." {
		t.Fatalf("personal drills: %+v %v", mine, err)
	}
}
