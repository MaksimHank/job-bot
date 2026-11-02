package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"job-bot/internal/cases"
	"job-bot/internal/entities"
)

type ApplicationRepo struct{ pool *pgxpool.Pool }

func NewApplicationRepo(pool *pgxpool.Pool) *ApplicationRepo {
	return &ApplicationRepo{pool: pool}
}

func (r *ApplicationRepo) Save(ctx context.Context, a *entities.Application) error {
	const q = `
        INSERT INTO applications
            (source, external_id, url, title, company, letter, status, error, applied_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
        ON CONFLICT (source, external_id) DO UPDATE SET
            status=EXCLUDED.status, error=EXCLUDED.error,
            letter=EXCLUDED.letter, applied_at=EXCLUDED.applied_at
        RETURNING id`
	return r.pool.QueryRow(ctx, q,
		a.Vacancy.Source, a.Vacancy.ExternalID, a.Vacancy.URL,
		a.Vacancy.Title, a.Vacancy.Company, a.Letter,
		a.Status, a.Error, a.AppliedAt,
	).Scan(&a.ID)
}

func (r *ApplicationRepo) Exists(ctx context.Context, src entities.Source, id string) (bool, error) {
	var ok bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM applications
                       WHERE source=$1 AND external_id=$2 AND status='applied')`,
		src, id).Scan(&ok)
	return ok, err
}

func (r *ApplicationRepo) CountToday(ctx context.Context, src entities.Source) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM applications
         WHERE source=$1 AND status='applied' AND applied_at >= CURRENT_DATE`,
		src).Scan(&n)
	return n, err
}

func (r *ApplicationRepo) ListBySource(ctx context.Context, src entities.Source) ([]*entities.Application, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, source, external_id, url, title, company, letter, status, error, applied_at
         FROM applications WHERE source=$1 ORDER BY applied_at DESC`, src)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*entities.Application
	for rows.Next() {
		a := &entities.Application{}
		if err := rows.Scan(
			&a.ID, &a.Vacancy.Source, &a.Vacancy.ExternalID, &a.Vacancy.URL,
			&a.Vacancy.Title, &a.Vacancy.Company, &a.Letter,
			&a.Status, &a.Error, &a.AppliedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Проверка на этапе компиляции, что адаптер реализует порт из cases.
var _ cases.ApplicationRepository = (*ApplicationRepo)(nil)
