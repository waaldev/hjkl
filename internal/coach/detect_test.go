package coach

import "testing"

func TestDetectRepeatJ(t *testing.T) {
	hits := Detect("jjjjjjj")
	if !has(hits, "repeat-j") {
		t.Fatalf("expected repeat-j, got %#v", hits)
	}
	if Detect("jjj") != nil && has(Detect("jjj"), "repeat-j") {
		t.Fatal("3 j should not trip the 5-j detector")
	}
}

func TestDetectArrows(t *testing.T) {
	hits := Detect("<Down><Down><Down>")
	if !has(hits, "arrow-keys") {
		t.Fatalf("expected arrow-keys, got %#v", hits)
	}
}

func TestDetectVisualIW(t *testing.T) {
	hits := Detect("viwd")
	if !has(hits, "visual-iw-delete") {
		t.Fatalf("expected visual-iw-delete, got %#v", hits)
	}
	if has(Detect("diw"), "visual-iw-delete") {
		t.Fatal("diw is the recommended form")
	}
}

func TestDetectXxxxx(t *testing.T) {
	if !has(Detect("xxxxx"), "repeat-x") {
		t.Fatal("expected repeat-x")
	}
}

func TestDetectInsertArrows(t *testing.T) {
	if !has(Detect("i<Left><Left>"), "insert-arrows") {
		t.Fatal("expected insert-arrows")
	}
}

func has(hits []Hit, pattern string) bool {
	for _, h := range hits {
		if h.Pattern == pattern {
			return true
		}
	}
	return false
}
