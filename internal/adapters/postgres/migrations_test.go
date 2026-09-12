package postgres

import "testing"

func TestMigrationVersion(t *testing.T) {
	version, err := migrationVersion("0007_example.sql")
	if err != nil || version != 7 {
		t.Fatalf("expected version 7, got %d, err=%v", version, err)
	}
}

func TestMigrationVersionRejectsUnversionedFile(t *testing.T) {
	if _, err := migrationVersion("initial.sql"); err == nil {
		t.Fatal("expected unversioned migration to fail")
	}
}
