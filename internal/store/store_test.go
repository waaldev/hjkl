package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordAndStreak(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	day1 := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	_, err = st.RecordAttempt(ctx, Attempt{
		ChallengeID: "w1", OK: true, Keys: "dw", KeyCount: 2, Par: 2, Stars: 3, XP: 15, At: day1,
	})
	if err != nil {
		t.Fatal(err)
	}
	day2 := day1.Add(24 * time.Hour)
	_, err = st.RecordAttempt(ctx, Attempt{
		ChallengeID: "w2", OK: true, Keys: "x", KeyCount: 1, Par: 1, Stars: 3, XP: 10, At: day2,
	})
	if err != nil {
		t.Fatal(err)
	}
	str, err := st.Streak(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if str.Current != 2 {
		t.Fatalf("streak=%d", str.Current)
	}
	xp, err := st.TotalXP(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if xp != 25 {
		t.Fatalf("xp=%d", xp)
	}
	stars, _ := st.Stars(ctx)
	if stars["w1"] != 3 {
		t.Fatalf("stars=%v", stars)
	}
}
