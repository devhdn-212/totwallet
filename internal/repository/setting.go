package repository

import (
	"context"
	"errors"

	"github.com/devhdn-212/totmaster_api/domain"
	"github.com/devhdn-212/totmaster_api/internal/config"
	"github.com/jackc/pgx/v5"
)

type settingRepository struct {
	db DBExecutor
}

func NewSettingRepository(db DBExecutor) domain.SettingRepository {
	return &settingRepository{
		db: db,
	}
}
func (b settingRepository) FindAll(ctx context.Context) ([]domain.Setting, error) {
	query := `SELECT *
				FROM ` + config.DB_tbl_mst_setting + `
				ORDER BY idsetting ASC LIMIT 1`

	rows, err := b.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res, err := pgx.CollectRows(rows, pgx.RowToStructByName[domain.Setting])
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (b settingRepository) FindByID(ctx context.Context, id int) (domain.Setting, error) {
	var setting domain.Setting
	query := `SELECT idsetting FROM ` + config.DB_tbl_mst_setting + ` WHERE idsetting = $1 LIMIT 1`

	err := b.db.QueryRow(ctx, query, id).Scan(&setting.ID)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Setting{}, nil
		}
		return setting, err
	}
	return setting, nil
}

func (b settingRepository) Save(ctx context.Context, setting *domain.Setting) error {
	query := `INSERT INTO ` + config.DB_tbl_mst_setting + ` 
                (idsetting, appversion, 
				startmaintenance, endmaintenance, shio_parent, 
				create_by, create_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := b.db.Exec(ctx, query,
		setting.ID,
		setting.Appversion,
		setting.Startmaintenance,
		setting.Endmaintenance,
		setting.Shio_parent,
		setting.Created, // Sesuaikan nama field di struct domain Anda
		setting.CreatedAt,
	)
	return err
}

func (b settingRepository) Update(ctx context.Context, setting *domain.Setting) error {
	query := `UPDATE ` + config.DB_tbl_mst_setting + ` SET 
                appversion = $1, 
                startmaintenance = $2, 
                endmaintenance = $3, 
                shio_parent = $4, 
                update_by = $5, 
                update_at = $6  
              WHERE idsetting = $7`

	res, err := b.db.Exec(ctx, query,
		setting.Appversion,
		setting.Startmaintenance,
		setting.Endmaintenance,
		setting.Shio_parent,
		setting.Update,
		setting.UpdateAt,
		setting.ID,
	)
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
