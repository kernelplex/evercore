//go:build integration

package evercorepostgres_test

import (
	"database/sql"
	"embed"
	"os"
	"path/filepath"
	"slices"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/kernelplex/evercore/enginetests"
	"github.com/kernelplex/evercore/evercorepostgres"
	"github.com/pressly/goose/v3"
)

//go:embed sql/migrations/*.sql
var EmbeddedPostgresMigrations embed.FS

const migrationsDir = "sql/migrations"

func maybeLoadDotenv() error {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	envFiles := make([]string, 0, 10)

	for {
		envFilename := filepath.Join(wd, ".env")

		_, statErr := os.Stat(envFilename)
		if statErr == nil {
			envFiles = append(envFiles, envFilename)
		}

		if wd == "/" {
			break
		}
		newWd := filepath.Dir(wd)
		if newWd == wd {
			break
		}
		wd = newWd
	}

	// Reverse the found .env files so we load the ones higher in the path first.
	slices.Reverse(envFiles)

	// Iterate and load the found .env files.
	for _, envFile := range envFiles {
		err := godotenv.Load(envFile)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestPostgrtesDatastore(t *testing.T) {
	maybeLoadDotenv()
	goose.SetBaseFS(EmbeddedPostgresMigrations)
	connectionString := os.Getenv("PG_TEST_RUNNER_CONNECTION")
	if connectionString == "" {
		panic("PG_TEST_RUNNER_CONNECTION environment variable not set.")
	}

	// TODO: Should use TestContainers
	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		panic(err)
	}

	// Initial up migration - ensure tables there
	evercorepostgres.MigrateUp(db)

	// Clear out migrations for any previous failed runs
	evercorepostgres.MigrateDown(db)

	// Migrate up again
	evercorepostgres.MigrateUp(db)

	// Defer cleanup migrations
	defer evercorepostgres.MigrateDown(db)

	iut := evercorepostgres.NewPostgresStorageEngine(db)
	testSuite := evercoreenginetests.NewStorageEngineTestSuite(iut)
	testSuite.RunTests(t)
}

func TestNewPostgresStorageEngineWithConnection(t *testing.T) {
	connectionString := os.Getenv("PG_TEST_RUNNER_CONNECTION")
	_, err := evercorepostgres.NewPostgresStorageEngineWithConnection(connectionString)
	if err != nil {
		t.Errorf("NewPostgresStorageEngineWithConnection failed: %s", err)
	}
}
