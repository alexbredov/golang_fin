package sqldb

import (
	"antibf/helpers"
	storageData "antibf/internal/storage/storageData"
	"context"
	"database/sql"
	"errors"
)

type Storage struct {
	DB *sql.DB
}

func New() *Storage {
	return &Storage{}
}
func (storage *Storage) Init(ctx context.Context, logger storageData.Logger, config storageData.Config) error {
	err := storage.Connect(ctx, logger, config)
	if err != nil {
		logger.Error("SQL connection failed: " + err.Error())
		return err
	}
	err = storage.DB.PingContext(ctx)
	if err != nil {
		logger.Error("SQL DB ping failed: " + err.Error())
		return err
	}
	return err
}
func (storage *Storage) Connect(ctx context.Context, logger storageData.Logger, config storageData.Config) error {
	select {
	case <-ctx.Done():
		return storageData.ErrStorageTimeout
	default:
		dsn := helpers.StringBuild(config.GetDBUser(), ":", config.GetDBPassword(), "@tcp(",
			config.GetDBAddress(), ":", config.GetDBPort(), ")/", config.GetDBName())
		var err error
		storage.DB, err = sql.Open("pgx", dsn)
		if err != nil {
			logger.Error("SQL opening connection failed: " + err.Error())
			return err
		}
		storage.DB.SetConnMaxLifetime(config.GetDBMaxConnectionLifetime())
		storage.DB.SetMaxIdleConns(config.GetDBMaxIdleConnections())
		storage.DB.SetMaxOpenConns(config.GetDBMaxOpenConnections())
		return nil
	}
}
func (storage *Storage) Close(ctx context.Context, logger storageData.Logger) error {
	select {
	case <-ctx.Done():
		return storageData.ErrStorageTimeout
	default:
		err := storage.DB.Close()
		if err != nil {
			logger.Error("SQL closing connection failed: " + err.Error())
			return err
		}
	}
	return nil
}
func (storage *Storage) IPAddToList(ctx context.Context, listname string, logger storageData.Logger, ipData storageData.StorageIPData) (int, error) { //nolint:lll
	script := "INSERT INTO " + listname + "(IP, mask) VALUES (?,?)"
	result, err := storage.DB.ExecContext(ctx, script, ipData.IP, ipData.Mask)
	if err != nil {
		logger.Error("SQL IPAddToList script failed: " + err.Error() + ", SQL script: " + script)
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		logger.Error("SQL IPAddToList LastInsertId failed: " + err.Error() + ", SQL script: " + script)
		return 0, err
	}
	return int(id), nil
}
func (storage *Storage) IPRemoveFromList(ctx context.Context, listname string, logger storageData.Logger, ipData storageData.StorageIPData) error { //nolint:lll
	script := "DELETE FROM " + listname + "WHERE IP = ? AND Mask = ?"
	result, err := storage.DB.ExecContext(ctx, script, ipData.IP, ipData.Mask)
	if err != nil {
		logger.Error("SQL IPRemoveFromList script failed: " + err.Error() + ", SQL script: " + script)
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		logger.Error("SQL IPRemoveFromList script failed: " + err.Error() + ", SQL script: " + script)
		return err
	}
	if count == int64(0) {
		logger.Error("SQL IPRemoveFromList script failed: " + storageData.ErrNoRecord.Error() + ", SQL script: " + script)
		return storageData.ErrNoRecord
	}
	return nil
}
func (storage *Storage) IPIsInList(ctx context.Context, listname string, logger storageData.Logger, ipData storageData.StorageIPData) (bool, error) { //nolint:lll
	script := "SELECT id, IP FROM " + listname + "WHERE IP = ? AND Mask = ?"
	row := storage.DB.QueryRowContext(ctx, script, ipData.IP, ipData.Mask)
	storageDataIP := &storageData.StorageIPData{}
	err := row.Scan(&storageDataIP.ID, &storageDataIP.IP)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		logger.Error("SQL IPIsInList row script failed: " + err.Error() + ", SQL script: " + script)
		return false, err
	}
	return true, nil
}
func (storage *Storage) IPGetAllFromList(ctx context.Context, listname string, logger storageData.Logger) ([]storageData.StorageIPData, error) { //nolint:lll
	resultIP := make([]storageData.StorageIPData, 0)
	script := "SELECT id, mask, IP FROM " + listname
	rows, err := storage.DB.QueryContext(ctx, script)
	if err != nil {
		logger.Error("SQL IPGetAllFromList query failed: " + err.Error() + ", SQL script: " + script)
		return nil, err
	}
	defer rows.Close()
	storageDataIP := &storageData.StorageIPData{}
	for rows.Next() {
		err = rows.Scan(&storageDataIP.ID, &storageDataIP.Mask, &storageDataIP.IP)
		if err != nil {
			logger.Error("SQL IPGetAllFromList rows scan failed")
			return nil, err
		}
		resultIP = append(resultIP, *storageDataIP)
		storageDataIP = &storageData.StorageIPData{}
	}
	if err := rows.Err(); err != nil {
		logger.Error("SQL IPGetAllFromList row scan failed: " + err.Error())
		return nil, err
	}
	return resultIP, nil
}
