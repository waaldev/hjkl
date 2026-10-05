package srs

import (
	"testing"
	"time"
)

func TestReviewLapseResetsInterval(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	c := NewCard("dw", now)
	c = Review(c, 5, now)
	if c.IntervalDays != 1 || c.Repetitions != 1 {
		t.Fatalf("first pass: %+v", c)
	}
	c = Review(c, 4, now.Add(24*time.Hour))
	if c.IntervalDays != 6 || c.Repetitions != 2 {
		t.Fatalf("second pass: %+v", c)
	}
	c = Review(c, 1, now.Add(8*24*time.Hour))
	if c.Repetitions != 0 || c.IntervalDays != 1 {
		t.Fatalf("lapse: %+v", c)
	}
}

func TestEasinessFloor(t *testing.T) {
	now := time.Now()
	c := NewCard("x", now)
	for i := 0; i < 20; i++ {
		c = Review(c, 0, now)
	}
	if c.Easiness < 1.3 {
		t.Fatalf("easiness fell through floor: %v", c.Easiness)
	}
}
