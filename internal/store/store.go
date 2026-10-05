package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Progress struct {
	ChallengeID  string
	BestStars    int
	BestKeyCount int
	FirstClearAt *time.Time
	LastClearAt  *time.Time
	Attempts     int
	XPEarned     int
}

type Attempt struct {
	ChallengeID string
	OK          bool
	Keys        string
	KeyCount    int
	Par         int
	Stars       int
	XP          int
	DurationMS  int
	At          time.Time
}

type SRSCard struct {
	Skill        string
	Easiness     float64
	IntervalDays float64
	Repetitions  int
	DueAt        time.Time
	LastQuality  int
	LastReviewAt *time.Time
}

type CoachEvent struct {
	TS       time.Time
	Pattern  string
	Keys     string
	Skill    string
	Filetype string
	Count    int
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  challenge_id TEXT NOT NULL,
  completed_at TEXT NOT NULL,
  ok INTEGER NOT NULL,
  keys TEXT NOT NULL,
  key_count INTEGER NOT NULL,
  par INTEGER NOT NULL,
  stars INTEGER NOT NULL,
  xp INTEGER NOT NULL,
  duration_ms INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS progress (
  challenge_id TEXT PRIMARY KEY,
  best_stars INTEGER NOT NULL DEFAULT 0,
  best_key_count INTEGER NOT NULL DEFAULT 0,
  first_clear_at TEXT,
  last_clear_at TEXT,
  attempts INTEGER NOT NULL DEFAULT 0,
  xp_earned INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS srs (
  skill TEXT PRIMARY KEY,
  easiness REAL NOT NULL,
  interval_days REAL NOT NULL,
  repetitions INTEGER NOT NULL,
  due_at TEXT NOT NULL,
  last_quality INTEGER NOT NULL DEFAULT 0,
  last_review_at TEXT
);
CREATE TABLE IF NOT EXISTS coach_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts TEXT NOT NULL,
  pattern TEXT NOT NULL,
  keys TEXT NOT NULL,
  skill TEXT,
  filetype TEXT,
  count INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS grammar (
  operator TEXT NOT NULL,
  motion TEXT NOT NULL,
  count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (operator, motion)
);
CREATE TABLE IF NOT EXISTS streak (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  current INTEGER NOT NULL DEFAULT 0,
  best INTEGER NOT NULL DEFAULT 0,
  last_day TEXT
);
INSERT OR IGNORE INTO streak (id, current, best, last_day) VALUES (1, 0, 0, '');
`)
	return err
}

func (s *Store) RecordAttempt(ctx context.Context, a Attempt) (firstClear bool, err error) {
	if a.At.IsZero() {
		a.At = time.Now()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
INSERT INTO attempts (challenge_id, completed_at, ok, keys, key_count, par, stars, xp, duration_ms)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ChallengeID, a.At.Format(time.RFC3339Nano), boolInt(a.OK), a.Keys, a.KeyCount, a.Par, a.Stars, a.XP, a.DurationMS)
	if err != nil {
		return false, err
	}

	var bestStars, attempts, xp int
	var firstStr, lastStr sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT best_stars, first_clear_at, last_clear_at, attempts, xp_earned FROM progress WHERE challenge_id = ?`, a.ChallengeID).
		Scan(&bestStars, &firstStr, &lastStr, &attempts, &xp)
	if err == sql.ErrNoRows {
		err = nil
	} else if err != nil {
		return false, err
	}
	firstClear = a.OK && !firstStr.Valid
	if a.OK && a.Stars > bestStars {
		bestStars = a.Stars
	}
	bestKeys := a.KeyCount
	if a.OK {
		var existingKeys int
		_ = tx.QueryRowContext(ctx, `SELECT best_key_count FROM progress WHERE challenge_id = ?`, a.ChallengeID).Scan(&existingKeys)
		if existingKeys > 0 && (bestKeys == 0 || existingKeys < bestKeys) {
			bestKeys = existingKeys
		}
	}
	first := firstStr.String
	if firstClear {
		first = a.At.Format(time.RFC3339Nano)
	}
	last := lastStr.String
	if a.OK {
		last = a.At.Format(time.RFC3339Nano)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO progress (challenge_id, best_stars, best_key_count, first_clear_at, last_clear_at, attempts, xp_earned)
VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), 1, ?)
ON CONFLICT(challenge_id) DO UPDATE SET
  best_stars = excluded.best_stars,
  best_key_count = CASE
    WHEN excluded.best_key_count > 0 AND (progress.best_key_count = 0 OR excluded.best_key_count < progress.best_key_count)
    THEN excluded.best_key_count ELSE progress.best_key_count END,
  first_clear_at = COALESCE(progress.first_clear_at, excluded.first_clear_at),
  last_clear_at = COALESCE(excluded.last_clear_at, progress.last_clear_at),
  attempts = progress.attempts + 1,
  xp_earned = progress.xp_earned + excluded.xp_earned
`, a.ChallengeID, bestStars, bestKeys, first, last, a.XP)
	if err != nil {
		return false, err
	}
	if a.OK {
		if err := bumpStreak(ctx, tx, a.At); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return firstClear, nil
}

func bumpStreak(ctx context.Context, tx *sql.Tx, at time.Time) error {
	day := at.Local().Format("2006-01-02")
	var current, best int
	var last string
	if err := tx.QueryRowContext(ctx, `SELECT current, best, last_day FROM streak WHERE id = 1`).Scan(&current, &best, &last); err != nil {
		return err
	}
	if last == day {
		return nil
	}
	yesterday := at.Local().AddDate(0, 0, -1).Format("2006-01-02")
	if last == yesterday {
		current++
	} else {
		current = 1
	}
	if current > best {
		best = current
	}
	_, err := tx.ExecContext(ctx, `UPDATE streak SET current = ?, best = ?, last_day = ? WHERE id = 1`, current, best, day)
	return err
}

func (s *Store) Progress(ctx context.Context) (map[string]Progress, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT challenge_id, best_stars, best_key_count, first_clear_at, last_clear_at, attempts, xp_earned FROM progress`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]Progress{}
	for rows.Next() {
		var p Progress
		var first, last sql.NullString
		if err := rows.Scan(&p.ChallengeID, &p.BestStars, &p.BestKeyCount, &first, &last, &p.Attempts, &p.XPEarned); err != nil {
			return nil, err
		}
		p.FirstClearAt = parseTime(first)
		p.LastClearAt = parseTime(last)
		out[p.ChallengeID] = p
	}
	return out, rows.Err()
}

func (s *Store) Stars(ctx context.Context) (map[string]int, error) {
	p, err := s.Progress(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(p))
	for id, rec := range p {
		out[id] = rec.BestStars
	}
	return out, nil
}

func (s *Store) TotalXP(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(xp_earned), 0) FROM progress`).Scan(&n)
	return n, err
}

type Streak struct {
	Current int
	Best    int
	LastDay string
}

func (s *Store) Streak(ctx context.Context) (Streak, error) {
	var st Streak
	err := s.db.QueryRowContext(ctx, `SELECT current, best, last_day FROM streak WHERE id = 1`).Scan(&st.Current, &st.Best, &st.LastDay)
	return st, err
}

func (s *Store) UpsertSRS(ctx context.Context, c SRSCard) error {
	var last any
	if c.LastReviewAt != nil {
		last = c.LastReviewAt.Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO srs (skill, easiness, interval_days, repetitions, due_at, last_quality, last_review_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(skill) DO UPDATE SET
  easiness = excluded.easiness,
  interval_days = excluded.interval_days,
  repetitions = excluded.repetitions,
  due_at = excluded.due_at,
  last_quality = excluded.last_quality,
  last_review_at = excluded.last_review_at
`, c.Skill, c.Easiness, c.IntervalDays, c.Repetitions, c.DueAt.Format(time.RFC3339Nano), c.LastQuality, last)
	return err
}

func (s *Store) SRS(ctx context.Context, skill string) (SRSCard, bool, error) {
	var c SRSCard
	var last sql.NullString
	var due string
	err := s.db.QueryRowContext(ctx, `SELECT skill, easiness, interval_days, repetitions, due_at, last_quality, last_review_at FROM srs WHERE skill = ?`, skill).
		Scan(&c.Skill, &c.Easiness, &c.IntervalDays, &c.Repetitions, &due, &c.LastQuality, &last)
	if err == sql.ErrNoRows {
		return SRSCard{}, false, nil
	}
	if err != nil {
		return SRSCard{}, false, err
	}
	c.DueAt, _ = time.Parse(time.RFC3339Nano, due)
	c.LastReviewAt = parseTime(last)
	return c, true, nil
}

func (s *Store) DueSkills(ctx context.Context, now time.Time, limit int) ([]SRSCard, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT skill, easiness, interval_days, repetitions, due_at, last_quality, last_review_at FROM srs WHERE due_at <= ? ORDER BY due_at ASC LIMIT ?`, now.Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SRSCard
	for rows.Next() {
		var c SRSCard
		var last sql.NullString
		var due string
		if err := rows.Scan(&c.Skill, &c.Easiness, &c.IntervalDays, &c.Repetitions, &due, &c.LastQuality, &last); err != nil {
			return nil, err
		}
		c.DueAt, _ = time.Parse(time.RFC3339Nano, due)
		c.LastReviewAt = parseTime(last)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) AllSRS(ctx context.Context) ([]SRSCard, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT skill, easiness, interval_days, repetitions, due_at, last_quality, last_review_at FROM srs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SRSCard
	for rows.Next() {
		var c SRSCard
		var last sql.NullString
		var due string
		if err := rows.Scan(&c.Skill, &c.Easiness, &c.IntervalDays, &c.Repetitions, &due, &c.LastQuality, &last); err != nil {
			return nil, err
		}
		c.DueAt, _ = time.Parse(time.RFC3339Nano, due)
		c.LastReviewAt = parseTime(last)
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) AddCoachEvent(ctx context.Context, e CoachEvent) error {
	if e.TS.IsZero() {
		e.TS = time.Now()
	}
	if e.Count == 0 {
		e.Count = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO coach_events (ts, pattern, keys, skill, filetype, count) VALUES (?, ?, ?, ?, ?, ?)`,
		e.TS.Format(time.RFC3339Nano), e.Pattern, e.Keys, e.Skill, e.Filetype, e.Count)
	return err
}

func (s *Store) CoachSummary(ctx context.Context, since time.Time) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT pattern, SUM(count) FROM coach_events WHERE ts >= ? GROUP BY pattern ORDER BY SUM(count) DESC`, since.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var p string
		var n int
		if err := rows.Scan(&p, &n); err != nil {
			return nil, err
		}
		out[p] = n
	}
	return out, rows.Err()
}

func (s *Store) LightGrammar(ctx context.Context, op, motion string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO grammar (operator, motion, count) VALUES (?, ?, 1)
ON CONFLICT(operator, motion) DO UPDATE SET count = grammar.count + 1
`, op, motion)
	return err
}

func (s *Store) GrammarGrid(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT operator, motion, count FROM grammar`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var op, mo string
		var n int
		if err := rows.Scan(&op, &mo, &n); err != nil {
			return nil, err
		}
		out[op+"\t"+mo] = n
	}
	return out, rows.Err()
}

func (s *Store) Meta(ctx context.Context, key string) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM meta WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func (s *Store) SetMeta(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO meta (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func parseTime(s sql.NullString) *time.Time {
	if !s.Valid || s.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, s.String)
	if err != nil {
		return nil
	}
	return &t
}
