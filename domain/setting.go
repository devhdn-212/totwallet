package domain

import (
	"context"
	"database/sql"

	"github.com/devhdn-212/totmaster_api/dto"
	"github.com/jackc/pgx/v5/pgtype"
)

type Setting struct {
	ID               int            `db:"idsetting"`
	Appversion       string         `db:"appversion"`
	Startmaintenance pgtype.Time    `db:"startmaintenance"`
	Endmaintenance   pgtype.Time    `db:"endmaintenance"`
	Shio_parent      int            `db:"shio_parent"`
	Created          string         `db:"create_by"`
	CreatedAt        sql.NullTime   `db:"create_at"`
	Update           sql.NullString `db:"update_by"`
	UpdateAt         sql.NullTime   `db:"update_at"`
}

type SettingRepository interface {
	FindAll(ctx context.Context) ([]Setting, error)
	FindByID(ctx context.Context, id int) (Setting, error)
	Save(ctx context.Context, cur *Setting) error
	Update(ctx context.Context, cur *Setting) error
}
type SettingService interface {
	All(ctx context.Context) ([]dto.SettingData, error)
	Save(ctx context.Context, req dto.SettingSave, client string) error
}
