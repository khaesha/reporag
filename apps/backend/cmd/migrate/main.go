package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
)

func main() {
	directory := flag.String("dir", "migrations", "directory containing ordered .sql migrations")
	reportOnly := flag.Bool("report", false, "print database versions and corpus counts without applying migrations")
	flag.Parse()
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if *reportOnly {
		report, err := databaseReport(context.Background(), databaseURL)
		if err == nil {
			fmt.Printf("postgresql=%s pgvector=%s documents=%d embeddings=%d\n", report.PostgreSQL, report.PGVector, report.Documents, report.Embeddings)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := run(context.Background(), databaseURL, *directory); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type report struct {
	PostgreSQL string
	PGVector   string
	Documents  int
	Embeddings int
}

func databaseReport(ctx context.Context, databaseURL string) (report, error) {
	if databaseURL == "" {
		return report{}, errors.New("DATABASE_URL is required")
	}
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return report{}, errors.New("database connection failed")
	}
	defer connection.Close(ctx)
	var result report
	if err := connection.QueryRow(ctx, "SHOW server_version").Scan(&result.PostgreSQL); err != nil {
		return report{}, errors.New("database version query failed")
	}
	if err := connection.QueryRow(ctx, "SELECT extversion FROM pg_extension WHERE extname = 'vector'").Scan(&result.PGVector); err != nil {
		return report{}, errors.New("pgvector version query failed")
	}
	if err := connection.QueryRow(ctx, "SELECT count(*), count(embedding) FROM documents").Scan(&result.Documents, &result.Embeddings); err != nil {
		return report{}, errors.New("document count query failed")
	}
	return result, nil
}

func run(ctx context.Context, databaseURL, directory string) error {
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	files, err := migrationFiles(directory)
	if err != nil {
		return err
	}
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return errors.New("database connection failed")
	}
	defer connection.Close(ctx)
	for _, file := range files {
		sql, err := os.ReadFile(filepath.Join(directory, file))
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
		if _, err := connection.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}
	}
	return nil
}

func migrationFiles(directory string) ([]string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			files = append(files, entry.Name())
		}
	}
	if len(files) == 0 {
		return nil, errors.New("no migration files found")
	}
	return files, nil
}
