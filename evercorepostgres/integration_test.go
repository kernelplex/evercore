//zzgo:build integration

package evercorepostgres_test

import (
	"context"
	"database/sql"
	"embed"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kernelplex/evercore/enginetests"
	"github.com/kernelplex/evercore/evercorepostgres"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

//go:embed sql/migrations/*.sql
var EmbeddedPostgresMigrations embed.FS

const migrationsDir = "sql/migrations"

func TestPostgrtesDatastore(t *testing.T) {
	ctx := context.Background()

	postgresContainer, err := postgres.Run(
		ctx,
		"postgres:16-alpine",
		/*
			postgres.WithDatabase("test"),
			postgres.WithUsername("user"),
			postgres.WithPassword("password"),
		*/

		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %s", err)
		return
	}
	defer func() {
		err := postgresContainer.Terminate(ctx)
		if err != nil {
			t.Fatalf("failed to terminate postgres container: %s", err)
		}
	}()

	err = postgresContainer.Start(ctx)
	if err != nil {
		t.Fatalf("failed to start postgres container: %s", err)
		return
	}

	connectionString, err := postgresContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get postgres connection string: %s", err)
		return
	}

	t.Logf("Using postgres connection string: %s", connectionString)

	// maybeLoadDotenv()
	goose.SetBaseFS(EmbeddedPostgresMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		panic(err)
	}

	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		panic(err)
	}

	// Initial up migration - ensure tables there
	t.Log("Clearing any existing migrations.")
	err = evercorepostgres.MigrateUp(db)
	if err != nil {
		t.Errorf("MigrateUp failed: %s", err)
		return
	}

	// Clear out migrations for any previous failed runs
	err = evercorepostgres.MigrateDown(db)
	if err != nil {
		t.Errorf("MigrateDown failed: %s", err)
		return
	}

	// Migrate up again
	t.Log("Running migrations.")
	err = evercorepostgres.MigrateUp(db)
	if err != nil {
		t.Errorf("MigrateDown failed: %s", err)
		return
	}

	// Defer cleanup migrations
	defer func() {
		err := evercorepostgres.MigrateDown(db)
		if err != nil {
			t.Errorf("MigrateDown failed: %s", err)
			return
		}
	}()

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
