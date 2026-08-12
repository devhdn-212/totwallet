package domain

import (
	"context"
	"database/sql"

	"github.com/devhdn-212/totwallet/dto"
)

// SlowQuery merepresentasikan tbl_slow_query: query database yang lebih lambat dari
// threshold, dicatat otomatis oleh pgx query tracer (internal/connection/tracer.go).
type SlowQuery struct {
	ID         int64        `db:"id"`
	Query      string       `db:"query"`
	DurationMs int64        `db:"duration_ms"`
	CreateAt   sql.NullTime `db:"create_at"`
}

type SlowQueryRepository interface {
	FindAll(ctx context.Context, limit, offset int) ([]SlowQuery, error)
	CountAll(ctx context.Context) (int, error)
}

type SlowQueryService interface {
	List(ctx context.Context, limit, offset int) ([]dto.SlowQueryData, int, error)
}
