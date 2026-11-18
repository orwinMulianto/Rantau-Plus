package database

import (
	"path"
	"slices"
	"testing"
)

func TestEmbeddedMigrationVersions(t *testing.T) {
	versions, err := migrationVersions()
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{initialVersion, "000002_transaction_integrity.up.sql"} {
		if !slices.Contains(versions, version) {
			t.Errorf("migration history version %q is missing", version)
		}
	}
	for _, version := range versions {
		if version != path.Base(version) {
			t.Errorf("version %q must match the filename stored in schema_migrations", version)
		}
		if _, err := scripts.ReadFile("migrations/" + version); err != nil {
			t.Errorf("migration %q could not be read: %v", version, err)
		}
	}
}
