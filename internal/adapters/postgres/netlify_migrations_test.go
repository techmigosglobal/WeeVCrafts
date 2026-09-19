package postgres

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestNetlifyMigrationCopiesMatchApplicationMigrations(t *testing.T) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	netlifyDirectory := filepath.Join("..", "..", "..", "netlify", "database", "migrations")
	netlifyEntries, err := os.ReadDir(netlifyDirectory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(netlifyEntries) {
		t.Fatalf("Netlify has %d migrations; application has %d", len(netlifyEntries), len(entries))
	}
	for _, entry := range entries {
		name := entry.Name()
		applicationSQL, err := fs.ReadFile(migrationFiles, filepath.ToSlash(filepath.Join("migrations", name)))
		if err != nil {
			t.Fatal(err)
		}
		netlifySQL, err := os.ReadFile(filepath.Join(netlifyDirectory, name))
		if err != nil {
			t.Fatalf("Netlify migration %s is missing: %v", name, err)
		}
		if !bytes.Equal(applicationSQL, netlifySQL) {
			t.Errorf("Netlify migration %s differs from the application migration", name)
		}
	}
}
