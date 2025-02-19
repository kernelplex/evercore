// go:build integration
package evercoresqlite_test

import (
	"testing"

	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/evercoresqlite"
)

// Ensure the SqliteStorageEngine implements StorageEngine
func TestSqliteStorageEngine_ImplementsStorageEngine(_ *testing.T) {
	var _ evercore.StorageEngine = &evercoresqlite.SqliteStorageEngine{}
}
