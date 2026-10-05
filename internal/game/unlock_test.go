package game

import "testing"

func TestUnlocked(t *testing.T) {
	belts := []string{"white", "yellow", "orange"}
	challenges := map[string]string{
		"w1": "white", "w2": "white", "w3": "white", "w4": "white", "w5": "white",
		"y1": "yellow",
	}
	stars := map[string]int{"w1": 3, "w2": 3, "w3": 1, "w4": 2} // 4/5 = 0.8
	got := Unlocked(belts, challenges, stars)
	if !got["white"] || !got["yellow"] || got["orange"] {
		t.Fatalf("unlocked=%v", got)
	}

	stars["w4"] = 0 // 3/5 = 0.6
	got = Unlocked(belts, challenges, stars)
	if got["yellow"] {
		t.Fatalf("yellow should stay locked: %v", got)
	}
}

func TestStars(t *testing.T) {
	if Stars(false, 1, 5) != 0 {
		t.Fatal("fail should be 0 stars")
	}
	if Stars(true, 5, 5) != 3 {
		t.Fatal("at par should be 3")
	}
	if Stars(true, 8, 5) != 2 {
		t.Fatal("within 2x par should be 2")
	}
	if Stars(true, 20, 5) != 1 {
		t.Fatal("over 2x par should be 1")
	}
}
