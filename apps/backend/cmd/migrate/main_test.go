package main

import (
	"os"
	"path/filepath"
	"reflect"
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
