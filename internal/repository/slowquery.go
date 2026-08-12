package repository

import (
	"context"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/internal/config"

	"github.com/jackc/pgx/v5"
)

type slowQueryRepository struct {
	db DBExecutor
}

func NewSlowQueryRepository(db DBExecutor) domain.SlowQueryRepository {
	return &slowQueryRepository{db: db}
}

func (r slowQueryRepository) FindAll(ctx context.Context, limit, offset int) ([]domain.SlowQuery, error) {
	query := `SELECT id, query, duration_ms, create_at FROM ` + config.DB_tbl_slow_query + `
	          ORDER BY create_at DESC LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.SlowQuery])
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r slowQueryRepository) CountAll(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM ` + config.DB_tbl_slow_query
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}
