package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"blog-api/pkg/settings"
)

// types
type txKey struct{}

// Config
const (
	MaxOpenConnections     = 25
	MaxIdleConnections     = 10
	ConnMaxLifetimeMinutes = 30
)

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  bool
}

func (c *DatabaseConfig) Setup() []settings.EnvLoadable {
	return []settings.EnvLoadable{
		settings.Item[string]{Name: "POSTGRES_HOST", Default: "localhost", Field: &c.Host},
		settings.Item[int]{Name: "POSTGRES_PORT", Default: 5432, Field: &c.Port},
		settings.Item[string]{Name: "POSTGRES_USER", Default: settings.NoDefault, Field: &c.User},
		settings.Item[string]{Name: "POSTGRES_PASSWORD", Default: settings.NoDefault, Field: &c.Password},
		settings.Item[string]{Name: "POSTGRES_DB", Default: settings.NoDefault, Field: &c.DBName},
		settings.Item[bool]{Name: "POSTGRES_SSL", Default: settings.NoDefault, Field: &c.SSLMode},
	}
}

// Manager
type DatabaseManager struct {
	connection *sql.DB
	ORM        *gorm.DB
}

func NewDatabaseManager(config *DatabaseConfig) (*DatabaseManager, error) {
	conn, err := sql.Open(
		"postgres",
		fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			config.Host,
			config.Port,
			config.User,
			config.Password,
			config.DBName,
			func() string {
				if config.SSLMode {
					return "require"
				}
				return "disable"
			}(),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed create db connection: %w", err)
	}

	conn.SetMaxOpenConns(MaxOpenConnections)
	conn.SetMaxIdleConns(MaxIdleConnections)
	conn.SetConnMaxLifetime(ConnMaxLifetimeMinutes * time.Minute)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("db connection is not functional: %w", err)
	}

	return &DatabaseManager{connection: conn}, nil
}

func (dm *DatabaseManager) InitORM() error {
	var err error
	dm.ORM, err = gorm.Open(
		postgres.New(
			postgres.Config{Conn: dm.connection},
		),
		&gorm.Config{},
	)
	return err
}

func (dm *DatabaseManager) Dispose() error {
	if dm.connection == nil {
		return fmt.Errorf("no db connection was ever established")
	}
	if err := dm.connection.Close(); err != nil {
		return fmt.Errorf("failed to close the db connection: %w", err)
	}
	return nil
}

func (dm *DatabaseManager) TestResponsiveness() error {
	row := dm.connection.QueryRow("SELECT 1")
	var result int
	if err := row.Scan(&result); err != nil {
		return fmt.Errorf("database is not responsive: %w", err)
	}
	return nil
}

func (dm *DatabaseManager) WithinTransaction(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	return dm.ORM.WithContext(ctx).Transaction(
		func(tx *gorm.DB) error {
			txCtx := context.WithValue(ctx, txKey{}, tx)
			return fn(txCtx)
		},
	)
}

func (dm *DatabaseManager) TxDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx // inside a transaction
	}
	return dm.ORM.WithContext(ctx) // outside transaction
}
