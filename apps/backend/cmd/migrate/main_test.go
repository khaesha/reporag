package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMigrationFiles(t *testing.T) {
	directory := t.TempDir()
	for _, name := range []string{"010_last.sql", "002_first.sql", "notes.txt"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(directory, "003_ignored.sql"), 0o700); err != nil {
		t.Fatal(err)
	}
	files, err := migrationFiles(directory)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"002_first.sql", "010_last.sql"}; !reflect.DeepEqual(files, want) {
		t.Fatalf("files = %v, want %v", files, want)
	}
}

func TestInitialMigrationIsRepeatable(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("../../migrations", "001_init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"CREATE TABLE IF NOT EXISTS documents",
		"CREATE INDEX IF NOT EXISTS documents_search_idx",
		"CREATE INDEX IF NOT EXISTS documents_year_idx",
		"CREATE INDEX IF NOT EXISTS documents_item_type_idx",
	} {
		if !strings.Contains(string(contents), statement) {
			t.Errorf("migration missing %q", statement)
		}
	}
}
