package service

import (
	"context"

	"github.com/devhdn-212/totwallet/domain"
	"github.com/devhdn-212/totwallet/dto"
	"github.com/devhdn-212/totwallet/internal/util"
)

type slowQueryService struct {
	repo domain.SlowQueryRepository
}

func NewSlowQueryService(repo domain.SlowQueryRepository) domain.SlowQueryService {
	return &slowQueryService{repo: repo}
}

func (s slowQueryService) List(ctx context.Context, limit, offset int) ([]dto.SlowQueryData, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	records, err := s.repo.FindAll(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	total, err := s.repo.CountAll(ctx)
	if err != nil {
		return nil, 0, err
	}

	res := make([]dto.SlowQueryData, 0, len(records))
	for _, r := range records {
		var createdAt string
		if r.CreateAt.Valid {
			createdAt = r.CreateAt.Time.In(util.LocJakarta).Format("2006-01-02 15:04:05")
		}
		res = append(res, dto.SlowQueryData{
			ID:         r.ID,
			Query:      r.Query,
			DurationMs: r.DurationMs,
			CreatedAt:  createdAt,
		})
	}
	return res, total, nil
}
