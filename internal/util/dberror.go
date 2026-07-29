package util

import (
	"errors"
)

var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("duplicate entry")
var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrInvalidAmount = errors.New("amount must be greater than zero")
