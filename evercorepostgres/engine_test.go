package evercorepostgres_test

import (
	"testing"

	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/evercorepostgres"
)

// Ensure the PostgresStorageEngine implements StorageEngine
func TestPostgresStorageEngine_ImplementsStorageEngine(_ *testing.T) {
	storageEngine := &evercorepostgres.PostgresStorageEngine{}
	var _ evercore.StorageEngine = storageEngine
}
