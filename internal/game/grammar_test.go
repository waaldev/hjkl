package game

import "testing"

func lit(keys string) map[string]bool {
	got := map[string]bool{}
	for _, p := range LightFromKeys(keys) {
		got[p[0]+p[1]] = true
	}
	return got
}

func TestLightFromKeys(t *testing.T) {
	got := lit("dwci\"da(")
	for _, want := range []string{"dw", `ci"`, "da("} {
		if !got[want] {
			t.Fatalf("missing %s in %v", want, got)
		}
	}
}

func TestLightFromKeysCounts(t *testing.T) {
	for _, keys := range []string{"d2w", "2dw", "d10w"} {
		if got := lit(keys); !got["dw"] || len(got) != 1 {
			t.Fatalf("%s lit %v, want only dw", keys, got)
		}
	}
	if got := lit("d0"); !got["d0"] {
		t.Fatalf("d0 lit %v", got)
	}
}

func TestLightFromKeysSkipsArguments(t *testing.T) {
	// fd is "find d", so the following w is a plain motion, not d+w.
	// "a is register a. dd is a linewise op, not d+motion.
	for _, keys := range []string{"fdw", `"ayy`, "dd", "jj"} {
		if got := lit(keys); len(got) != 0 {
			t.Fatalf("%s lit %v, want nothing", keys, got)
		}
	}
	if got := lit("df:"); !got["df"] {
		t.Fatalf("df: lit %v", got)
	}
}
