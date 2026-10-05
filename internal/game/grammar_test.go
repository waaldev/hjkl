package game

import "testing"

func TestLightFromKeys(t *testing.T) {
	pairs := LightFromKeys("dwci\"da(")
	got := map[string]bool{}
	for _, p := range pairs {
		got[p[0]+p[1]] = true
	}
	for _, want := range []string{"dw", `ci"`, "da("} {
		if !got[want] {
			t.Fatalf("missing %s in %v", want, pairs)
		}
	}
}
