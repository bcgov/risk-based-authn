package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"rba/util"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gopkg.in/yaml.v3"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type PostgresConfig struct {
	Port         uint   `yaml:"port" env:"DATABASE_CONNECTION_PORT" validate:"required,port"`
	Host         string `yaml:"host" env:"DATABASE_CONNECTION_HOST" validate:"required,hostname_rfc1123"`
	Username     string `yaml:"username" env:"DATABASE_CONNECTION_USERNAME" validate:"required"`
	Password     string `yaml:"password" env:"DATABASE_CONNECTION_PASSWORD" validate:"required"`
	DatabaseName string `yaml:"databaseName" env:"DATABASE_CONNECTION_DATABASE_NAME" validate:"required"`
	SSLMode      string `yaml:"sslMode" env:"DATABASE_CONNECTION_SSL_MODE" validate:"required,oneof=enable disable"`
}

type postgresDB struct {
	db *sql.DB
}

var pgdb postgresDB

func pqCodeToStatusCode(err error) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "23505":
			return util.ErrNetworkAlreadyExists
		case "22P02":
			return util.ErrInvalidNetwork
		default:
			return util.ErrUnknown
		}
	} else {
		return util.ErrUnknown
	}
}

func (p *postgresDB) CreateEvent(ctx context.Context, score float64, details []util.RiskResult) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx,
		`INSERT INTO events (score, details) VALUES ($1, $2)`,
		score,
		detailsJSON,
	)
	return err
}

func (p *postgresDB) AddDenylistNetwork(ctx context.Context, network string) error {
	_, err := p.db.ExecContext(ctx,
		`INSERT INTO denylist (network) VALUES ($1)`,
		network,
	)
	return pqCodeToStatusCode(err)
}

func (p *postgresDB) RemoveDenylistNetwork(ctx context.Context, network string) error {
	_, err := p.db.ExecContext(ctx,
		`DELETE FROM denylist WHERE network = $1`,
		network,
	)
	return pqCodeToStatusCode(err)
}

func (p *postgresDB) GetDenylistNetworks(ctx context.Context) ([]string, error) {
	rows, err := p.db.QueryContext(ctx,
		`SELECT network FROM denylist`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var networks []string
	for rows.Next() {
		var network string
		if err := rows.Scan(&network); err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return networks, nil
}

func (p *postgresDB) IPDenied(ctx context.Context, ip string) (bool, error) {
	var exists bool
	err := p.db.QueryRowContext(
		ctx,
		`SELECT 1 FROM denylist WHERE network >>= $1 LIMIT 1`,
		ip,
	).Scan(&exists)
	return exists, err
}

func connectPostgres(connection yaml.Node) error {
	if pgdb.db != nil {
		return errors.New("postgres already initialized")
	}
	var cfg PostgresConfig

	if err := connection.Decode(&cfg); err != nil {
		return fmt.Errorf("failed to decode YAML: %w", err)
	}
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to parse env vars: %w", err)
	}

	if err := validator.New(validator.WithRequiredStructEnabled()).Struct(cfg); err != nil {
		return fmt.Errorf("error parsing postgres connection details: %w", err)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.DatabaseName,
		cfg.SSLMode,
	)

	var err error
	pgdb.db, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	pgdb.db.SetMaxOpenConns(25)
	pgdb.db.SetMaxIdleConns(25)
	pgdb.db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pgdb.db.PingContext(ctx); err != nil {
		return fmt.Errorf("postgres ping failed: %w", err)
	}

	return nil
}

func migratePostgres(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
