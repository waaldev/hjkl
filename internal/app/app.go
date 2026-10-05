package app

import (
	"context"

	"github.com/waaldev/hjkl/internal/coach"
	"github.com/waaldev/hjkl/internal/config"
	"github.com/waaldev/hjkl/internal/curriculum"
	"github.com/waaldev/hjkl/internal/store"
)

type App struct {
	Cfg   config.Config
	Cat   *curriculum.Catalog
	Store *store.Store
}

func Open() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	cat, err := curriculum.Load()
	if err != nil {
		return nil, err
	}
	dbPath, err := config.DBPath()
	if err != nil {
		return nil, err
	}
	st, err := store.Open(dbPath)
	if err != nil {
		return nil, err
	}
	a := &App{Cfg: cfg, Cat: cat, Store: st}
	if p, err := config.CoachEventsPath(); err == nil {
		_, _ = coach.IngestJSONL(context.Background(), st, p)
	}
	return a, nil
}

func (a *App) Close() error {
	if a.Store != nil {
		return a.Store.Close()
	}
	return nil
}
