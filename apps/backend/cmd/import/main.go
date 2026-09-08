package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khaesha/reporag/apps/backend/internal/records"
	"github.com/khaesha/reporag/apps/backend/internal/store"
)

type summary struct {
	Inserted int
	Updated  int
	Rejected int
	Total    int
}

func main() {
	corpus := flag.String("corpus", "", "directory containing repository_*_data.json files")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	result, err := run(ctx, *corpus)
	fmt.Printf("inserted=%d updated=%d rejected=%d total=%d\n", result.Inserted, result.Updated, result.Rejected, result.Total)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, corpus string) (summary, error) {
	if strings.TrimSpace(corpus) == "" {
		return summary{}, errors.New("-corpus is required")
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return summary{}, errors.New("DATABASE_URL is required")
	}

	files, err := records.Discover(corpus)
	if err != nil {
		return summary{}, err
	}
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return summary{}, errors.New("DATABASE_URL is invalid")
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return summary{}, errors.New("database pool initialization failed")
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return summary{}, errors.New("database unavailable")
	}

	postgres := store.New(pool)
	var result summary
	for _, file := range files {
		documents, recordCount, err := records.Read(file)
		result.Total += recordCount
		if err != nil {
			result.Rejected += recordCount
			return result, err
		}
		counts, err := postgres.ImportFile(ctx, documents)
		if err != nil {
			result.Rejected += recordCount
			return result, fmt.Errorf("%s: %w", file.Path, err)
		}
		result.Inserted += counts.Inserted
		result.Updated += counts.Updated
	}
	return result, nil
}
