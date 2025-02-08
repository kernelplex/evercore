package evercorepostgres_test

import (
	"testing"

	"github.com/kernelplex/evercore/pkg/evercore"
	"github.com/kernelplex/evercore/pkg/evercorepostgres"
)

// Ensure the PostgresStorageEngine implements StorageEngine
func TestPostgresStorageEngine_ImplementsStorageEngine(_ *testing.T) {
	storageEngine := &evercorepostgres.PostgresStorageEngine{}
	var _ evercore.StorageEngine = storageEngine
}
