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
