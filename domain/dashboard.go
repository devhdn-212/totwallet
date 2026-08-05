package domain

import (
	"context"

	"github.com/devhdn-212/totwallet/dto"
)

type DashboardService interface {
	Summary(ctx context.Context) (dto.DashboardData, error)
}
