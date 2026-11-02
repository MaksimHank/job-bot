package cases

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"job-bot/internal/entities"
)

// Limit — ограничения для одной площадки.
type Limit struct {
	MaxPerDay     int
	ApplyInterval time.Duration
}

type Client struct {
	repo    ApplicationRepository
	letter  LetterGenerator
	sources map[entities.Source]VacancySource
	limits  map[entities.Source]Limit
	logger  *slog.Logger
}

func NewClient(
	repo ApplicationRepository,
	letter LetterGenerator,
	limits map[entities.Source]Limit,
	logger *slog.Logger,
	sources ...VacancySource,
) *Client {
	m := make(map[entities.Source]VacancySource, len(sources))
	for _, s := range sources {
		m[s.Source()] = s
	}
	return &Client{repo: repo, letter: letter, sources: m, limits: limits, logger: logger}
}

// Run ищет вакансии по каждому запросу и откликается,
// соблюдая дневные лимиты и таймауты между откликами.
func (c *Client) Run(ctx context.Context, queries []entities.SearchQuery) error {
	// Счётчики «сколько уже откликов сегодня» по источникам — кэшируем,
	// чтобы не дёргать БД на каждую вакансию.
	today := make(map[entities.Source]int, len(c.sources))
	for src := range c.sources {
		n, err := c.repo.CountToday(ctx, src)
		if err != nil {
			return fmt.Errorf("init counter %s: %w", src, err)
		}
		today[src] = n
	}

	for _, q := range queries {
		src, ok := c.sources[q.Source]
		if !ok {
			c.logger.Warn("unknown source", "source", q.Source)
			continue
		}

		limit := c.limits[q.Source]
		if today[q.Source] >= limit.MaxPerDay {
			c.logger.Warn("daily limit reached before search",
				"source", q.Source, "used", today[q.Source], "limit", limit.MaxPerDay)
			continue
		}

		vacancies, err := src.Search(ctx, q)
		if err != nil {
			c.logger.Error("search failed", "source", q.Source, "query", q.Text, "err", err)
			continue
		}
		c.logger.Info("search done",
			"source", q.Source, "query", q.Text, "found", len(vacancies))

		for _, v := range vacancies {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if today[v.Source] >= limit.MaxPerDay {
				c.logger.Warn("daily limit reached", "source", v.Source)
				break
			}

			applied, err := c.applyOne(ctx, v)
			if err != nil {
				c.logger.Error("apply", "vacancy", v.ExternalID, "err", err)
			}
			if !applied {
				continue
			}
			today[v.Source]++

			// Таймаут между откликами — «уступаем» сайту.
			c.logger.Info("waiting before next apply", "duration", limit.ApplyInterval)
			select {
			case <-time.After(limit.ApplyInterval):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
	return nil
}

func (c *Client) applyOne(ctx context.Context, v entities.Vacancy) (bool, error) {
	exists, err := c.repo.Exists(ctx, v.Source, v.ExternalID)
	if err != nil {
		return false, fmt.Errorf("exists: %w", err)
	}
	if exists {
		c.logger.Debug("already applied, skip", "source", v.Source, "id", v.ExternalID)
		return false, nil
	}

	letter, err := c.letter.Generate(ctx, v)
	if err != nil {
		return false, fmt.Errorf("letter: %w", err)
	}

	app := &entities.Application{
		Vacancy:   v,
		Letter:    letter,
		Status:    entities.StatusApplied,
		AppliedAt: time.Now().UTC(),
	}

	if err := c.sources[v.Source].Apply(ctx, v, letter); err != nil {
		app.Status = entities.StatusFailed
		app.Error = err.Error()
		_ = c.repo.Save(ctx, app)
		return false, fmt.Errorf("apply: %w", err)
	}

	if err := c.repo.Save(ctx, app); err != nil {
		return false, fmt.Errorf("save: %w", err)
	}
	c.logger.Info("applied", "source", v.Source, "id", v.ExternalID, "url", v.URL)
	return true, nil
}
