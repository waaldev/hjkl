package srs

import "time"

// Card is an SM-2 skill record.
type Card struct {
	Skill        string
	Easiness     float64
	IntervalDays float64
	Repetitions  int
	DueAt        time.Time
	LastQuality  int
	LastReviewAt *time.Time
}

func NewCard(skill string, now time.Time) Card {
	return Card{
		Skill:        skill,
		Easiness:     2.5,
		IntervalDays: 0,
		Repetitions:  0,
		DueAt:        now,
	}
}

// Counts reports whether an attempt now should move the schedule.
//
// Only the first attempt per skill per day is a review; replays later that
// day are practice. Without this, three replays in a row push a skill out
// 1 → 6 → 15 days after half a minute of work. A success before the card is
// due says nothing about retention at the interval, so it doesn't count
// either; an early failure still does (it resets the card).
func Counts(card Card, quality int, now time.Time) bool {
	if card.LastReviewAt != nil && sameDay(*card.LastReviewAt, now) {
		return false
	}
	if quality >= 3 && now.Before(card.DueAt) {
		return false
	}
	return true
}

func sameDay(a, b time.Time) bool {
	a, b = a.Local(), b.Local()
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// Review applies SM-2. quality is 0–5.
func Review(card Card, quality int, now time.Time) Card {
	if quality < 0 {
		quality = 0
	}
	if quality > 5 {
		quality = 5
	}
	if quality < 3 {
		card.Repetitions = 0
		card.IntervalDays = 1
	} else {
		switch card.Repetitions {
		case 0:
			card.IntervalDays = 1
		case 1:
			card.IntervalDays = 6
		default:
			card.IntervalDays = card.IntervalDays * card.Easiness
			if card.IntervalDays < 1 {
				card.IntervalDays = 1
			}
		}
		card.Repetitions++
	}
	ef := card.Easiness + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))
	if ef < 1.3 {
		ef = 1.3
	}
	card.Easiness = ef
	card.LastQuality = quality
	t := now
	card.LastReviewAt = &t
	hours := card.IntervalDays * 24
	card.DueAt = now.Add(time.Duration(hours * float64(time.Hour)))
	return card
}
