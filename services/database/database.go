package database

import (
	"context"
	"errors"
	"rba/util"

	"gopkg.in/yaml.v3"
)

var dbType string

type DB interface {
	CreateEvent(ctx context.Context, score float64, details []util.RiskResult) error
	AddDenylistNetwork(ctx context.Context, network string) error
	RemoveDenylistNetwork(ctx context.Context, network string) error
	IPDenied(ctx context.Context, ip string) (bool, error)
	GetDenylistNetworks(ctx context.Context) ([]string, error)
}

func ConnectDatabase(db string, connection yaml.Node) error {
	if dbType != "" {
		return errors.New("Database already initialized")
	}
	switch db {
	case util.Databases.Postgres:
		if err := connectPostgres(connection); err != nil {
			return err
		}
		if err := migratePostgres(pgdb.db, "./migrations/postgres"); err != nil {
			return err
		}
	default:
		return errors.New("unsupported database type")
	}
	dbType = db
	return nil
}

func GetDB() (DB, error) {
	switch dbType {
	case util.Databases.Postgres:
		if pgdb.db == nil {
			return nil, errors.New("postgres not initialized")
		}
		return &pgdb, nil
	case "":
		return nil, errors.New("db not intilized")
	default:
		return nil, errors.New("unsupported database type")
	}
}
