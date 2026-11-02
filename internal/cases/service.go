package cases

import (
	"context"
	"job-bot/internal/entities"
)

// ApplicationRepository — хранилище откликов.
// Определён здесь, потому что нужен use case-логике.
type ApplicationRepository interface {
	Save(ctx context.Context, app *entities.Application) error
	Exists(ctx context.Context, source entities.Source, externalID string) (bool, error)
	CountToday(ctx context.Context, source entities.Source) (int, error)
	ListBySource(ctx context.Context, source entities.Source) ([]*entities.Application, error)
}

// LetterGenerator — генератор сопроводительного письма.
type LetterGenerator interface {
	Generate(ctx context.Context, v entities.Vacancy) (string, error)
}

// VacancySource — платформа, умеющая искать вакансии и откликаться.
type VacancySource interface {
	Source() entities.Source
	Search(ctx context.Context, q entities.SearchQuery) ([]entities.Vacancy, error)
	Apply(ctx context.Context, v entities.Vacancy, letter string) error
}
