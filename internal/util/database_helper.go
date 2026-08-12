package util

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// NextSequenceValue ambil nilai berikutnya dari PostgreSQL sequence (nextval).
// Dipakai gantiin pola counter manual (SELECT ... FOR UPDATE di tbl_counter) yang
// serialize SEMUA transaksi konkuren lewat satu row lock — nextval() atomic tanpa
// lock/antre, jadi burst transaksi konkuren gak saling nunggu. Konsekuensinya notrx
// bisa ada gap (skip angka) kalau ada transaksi yang rollback, itu normal & aman
// karena notrx cuma perlu UNIQUE, bukan gapless.
func NextSequenceValue(ctx context.Context, tx pgx.Tx, seqName string) (int64, error) {
	var next int64
	err := tx.QueryRow(ctx, `SELECT nextval($1::regclass)`, seqName).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("error nextval sequence: %w", err)
	}
	return next, nil
}
