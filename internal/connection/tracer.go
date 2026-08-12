package connection

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// SlowQueryThreshold: query yang makan waktu lebih lama dari ini otomatis kecatat ke
// tbl_slow_query (lihat sql/schema.sql) — dipakai halaman "Slow Query" dashboard admin
// buat ngedeteksi masalah performa (mis. lock contention, latensi jaringan ke DB) tanpa
// harus manual query pg_stat_activity kayak yang biasa dilakuin pas debug incident.
const SlowQueryThreshold = 500 * time.Millisecond

type slowQueryTraceKey struct{}

type slowQueryTraceData struct {
	sql   string
	start time.Time
}

// SlowQueryTracer implementasi pgx.QueryTracer. Pool di-set belakangan (SetPool) karena
// tracer harus udah nempel ke ConnConfig SEBELUM pool-nya sendiri kebentuk (chicken-egg) —
// lihat GetDatabase di database.go.
type SlowQueryTracer struct {
	pool *pgxpool.Pool
}

func NewSlowQueryTracer() *SlowQueryTracer {
	return &SlowQueryTracer{}
}

func (t *SlowQueryTracer) SetPool(pool *pgxpool.Pool) {
	t.pool = pool
}

func (t *SlowQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, slowQueryTraceKey{}, &slowQueryTraceData{sql: data.SQL, start: time.Now()})
}

func (t *SlowQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	td, ok := ctx.Value(slowQueryTraceKey{}).(*slowQueryTraceData)
	if !ok || t.pool == nil {
		return
	}

	duration := time.Since(td.start)
	if duration < SlowQueryThreshold {
		return
	}
	// Cegah rekursi tak berujung: INSERT logging di bawah ini sendiri juga lewat tracer.
	if strings.Contains(td.sql, "tbl_slow_query") {
		return
	}

	sql, durationMs, pool := td.sql, duration.Milliseconds(), t.pool
	go func() {
		insertCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := pool.Exec(insertCtx,
			`INSERT INTO tbl_slow_query (query, duration_ms) VALUES ($1, $2)`,
			sql, durationMs,
		); err != nil {
			Log.Warn("Failed to save slow query log", zap.Error(err))
		}
	}()
}
