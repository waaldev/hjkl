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

func TestHintCost(t *testing.T) {
	if HintCap(3, 0) != 3 || HintCap(3, 1) != 2 || HintCap(3, 5) != 1 || HintCap(0, 2) != 0 {
		t.Fatal("HintCap")
	}
	if HintQuality(5, 0, false) != 5 || HintQuality(5, 1, false) != 3 || HintQuality(5, 3, true) != 2 || HintQuality(1, 3, true) != 1 {
		t.Fatal("HintQuality")
	}
}

func TestQualityFluency(t *testing.T) {
	fast, slow := FluentMS(4)-1, FluentMS(4)+1
	if Quality(true, 3, fast, 4) != 5 || Quality(true, 3, slow, 4) != 4 {
		t.Fatal("three stars: fluent is 5, slow is 4")
	}
	if Quality(true, 2, fast, 4) != 3 || Quality(false, 0, fast, 4) != 1 {
		t.Fatal("stars still dominate")
	}
}
