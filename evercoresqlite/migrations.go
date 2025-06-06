//go:build !evercorenomigrations

package evercoresqlite

import (
	"database/sql"
	"embed"
	"github.com/pressly/goose/v3"
)

//go:embed sql/migrations/*.sql
var EmbeddedSqliteMigrations embed.FS

const migrationsDir = "sql/migrations"
const migrationsTable = "evercore_migrations"

func MigrateUp(db *sql.DB) error {
	goose.SetDialect("sqlite3")
	goose.SetBaseFS(EmbeddedSqliteMigrations)
	goose.SetTableName("evercore_migrations")
	return goose.Up(db, migrationsDir)
}

func MigrateDown(db *sql.DB) error {
	goose.SetDialect("sqlite3")
	goose.SetBaseFS(EmbeddedSqliteMigrations)
	goose.SetTableName("evercore_migrations")
	return goose.Down(db, migrationsDir)
}
