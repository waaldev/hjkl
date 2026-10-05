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

func TestCountsOncePerDay(t *testing.T) {
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.Local)
	c := NewCard("dw", now)
	if !Counts(c, 5, now) {
		t.Fatal("a new card's first attempt is a review")
	}
	c = Review(c, 5, now)
	// Replaying the drill an hour later is practice, not a review.
	later := now.Add(time.Hour)
	if Counts(c, 5, later) || Counts(c, 1, later) {
		t.Fatal("same-day replay must not move the schedule")
	}
	if c.IntervalDays != 1 {
		t.Fatalf("interval=%v", c.IntervalDays)
	}
}

func TestCountsEarlyReview(t *testing.T) {
	now := time.Date(2026, 10, 5, 9, 0, 0, 0, time.Local)
	c := Review(Review(NewCard("dw", now), 5, now), 5, now.Add(24*time.Hour)) // due in 6 days
	early := now.Add(3 * 24 * time.Hour)
	if Counts(c, 5, early) {
		t.Fatal("an early success says nothing about retention at the interval")
	}
	if !Counts(c, 1, early) {
		t.Fatal("an early failure should still reset the card")
	}
	if !Counts(c, 5, c.DueAt.Add(time.Hour)) {
		t.Fatal("a due card counts")
	}
}
