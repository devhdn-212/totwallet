package util

import (
	"errors"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("duplicate entry")
var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrInvalidAmount = errors.New("amount must be greater than zero")

// ErrDuplicateTransaction ditandain kalau refno (mis. playerinvoice) udah pernah diproses
// buat username yang sama — lihat internal/service/wallet_transaction.go fungsi process().
var ErrDuplicateTransaction = errors.New("duplicate transaction: already processed")
