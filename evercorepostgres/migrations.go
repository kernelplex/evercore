//go:build !evercorenomigrations

package evercorepostgres

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed sql/migrations/*.sql
var EmbeddedSqliteMigrations embed.FS

const migrationsDir = "sql/migrations"
const migrationsTable = "evercore_migrations"

func MigrateUp(db *sql.DB) error {
	goose.SetDialect("postgres")
	goose.SetBaseFS(EmbeddedSqliteMigrations)
	goose.SetTableName("evercore_migrations")
	if err := goose.Up(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to migrate up: %w", err)
	}
	return nil
}

func MigrateDown(db *sql.DB) error {
	goose.SetDialect("postgres")
	goose.SetBaseFS(EmbeddedSqliteMigrations)
	goose.SetTableName("evercore_migrations")
	if err := goose.Down(db, migrationsDir); err != nil {
		return fmt.Errorf("failed to migrate down: %w", err)
	}
	return nil
}
