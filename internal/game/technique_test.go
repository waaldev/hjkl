package game

import "testing"

func TestCheckTechnique(t *testing.T) {
	cases := []struct {
		keys            string
		require, forbid []string
		ok              bool
	}{
		{"dd..", []string{`\.`}, []string{`[2-9]dd`}, true},
		{"3dd", []string{`\.`}, []string{`[2-9]dd`}, false}, // orange-14: right result, wrong technique
		{"dd.3dd", []string{`\.`}, []string{`[2-9]dd`}, false},
		{"diw", nil, []string{"v"}, true},
		{"viwd", nil, []string{"v"}, false},
		{"anything", nil, nil, true},
	}
	for _, c := range cases {
		ok, msg, err := CheckTechnique(c.keys, c.require, c.forbid)
		if err != nil {
			t.Fatal(err)
		}
		if ok != c.ok {
			t.Fatalf("%q: ok=%v want %v (%s)", c.keys, ok, c.ok, msg)
		}
		if !ok && msg == "" {
			t.Fatalf("%q: failed rule needs a message", c.keys)
		}
	}
	if _, _, err := CheckTechnique("x", []string{"("}, nil); err == nil {
		t.Fatal("bad regexp should error")
	}
}
