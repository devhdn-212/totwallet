package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/devhdn-212/totmaster_api/domain"
	"github.com/devhdn-212/totmaster_api/dto"
	"github.com/devhdn-212/totmaster_api/internal/config"
	"github.com/devhdn-212/totmaster_api/internal/connection"
	"github.com/devhdn-212/totmaster_api/internal/repository"
	"github.com/devhdn-212/totmaster_api/internal/util"

	"github.com/gofiber/fiber/v2/log"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RedissettingKey = "master:setting:all"
)

type settingService struct {
	db   *pgxpool.Pool
	repo domain.SettingRepository
}

func NewSettingService(db *pgxpool.Pool, repo domain.SettingRepository) domain.SettingService {
	return &settingService{
		db:   db,
		repo: repo,
	}
}

func (b settingService) All(ctx context.Context) ([]dto.SettingData, error) {
	cached, found, err := connection.GetRedis(RedissettingKey)
	if err != nil {
		return nil, err
	}

	if found {
		var data []dto.SettingData
		if err := json.Unmarshal([]byte(cached), &data); err == nil {
			connection.Log.Info("Returning data from Redis - Setting")
			return data, nil
		}
	}

	setting, err := b.repo.FindAll(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var settingData []dto.SettingData
	for _, v := range setting {
		var createdAt, updatedAt string
		if v.CreatedAt.Valid {
			createdAt = v.Created + ", " + v.CreatedAt.Time.In(util.LocJakarta).Format("2006-01-02 15:04:05")
		}
		if v.UpdateAt.Valid && v.Update.Valid && v.Update.String != "" {
			updatedAt = v.Update.String + ", " + v.UpdateAt.Time.In(util.LocJakarta).Format("2006-01-02 15:04:05")
		} else {
			updatedAt = ""
		}

		settingData = append(settingData, dto.SettingData{
			ID:               v.ID,
			Appversion:       v.Appversion,
			Startmaintenance: util.PgTimeToString(v.Startmaintenance),
			Endmaintenance:   util.PgTimeToString(v.Endmaintenance),
			Shio_parent:      v.Shio_parent,
			Created:          createdAt,
			Update:           updatedAt,
		})
	}
	go connection.SetRedis(RedissettingKey, settingData, 60*time.Minute)
	connection.Log.Info("Returning data Database - Setting")
	return settingData, nil
}

func (b settingService) Save(ctx context.Context, req dto.SettingSave, client_admin string) error {
	// Start transaction pgx
	tx, err := b.db.Begin(ctx)
	if err != nil {
		return err
	}

	// Defer rollback
	defer tx.Rollback(ctx)

	// Executor transaksi
	txExec := repository.NewPGXTxExecutor(tx)
	txRepo := repository.NewSettingRepository(txExec)

	flag, err := txRepo.FindByID(ctx, req.ID)
	if err != nil {
		return err
	}

	now := util.GetNowJakarta()

	if req.Type == "New" {
		existing, err := txRepo.FindAll(ctx)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return errors.New("Setting already exists, please use update")
		}

		idcounter, err := util.GetNextCounterManualTx(ctx, tx, config.DB_tbl_mst_setting)
		if err != nil {
			return err
		}

		setting := domain.Setting{
			ID:               int(idcounter),
			Appversion:       req.Appversion,
			Startmaintenance: util.StringToPgTime(req.Start_maintenance),
			Endmaintenance:   util.StringToPgTime(req.End_maintenance),
			Shio_parent:      req.Shioparent,
			Created:          client_admin,
			CreatedAt:        sql.NullTime{Valid: true, Time: now},
		}
		err = txRepo.Save(ctx, &setting)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				return util.ErrDuplicate
			}
			return err
		}
	} else {
		if flag.ID == 0 {
			return errors.New("Setting not found")
		}

		flag.ID = req.ID
		flag.Appversion = req.Appversion
		flag.Startmaintenance = util.StringToPgTime(req.Start_maintenance)
		flag.Endmaintenance = util.StringToPgTime(req.End_maintenance)
		flag.Shio_parent = req.Shioparent
		flag.Update = sql.NullString{Valid: client_admin != "", String: client_admin}
		flag.UpdateAt = sql.NullTime{Valid: true, Time: now}

		// Pakai txRepo biar masuk ke transaksi yang sama
		if err = txRepo.Update(ctx, &flag); err != nil {
			return err
		}
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	go connection.DeleteRedis(RedissettingKey)
	return nil
}
