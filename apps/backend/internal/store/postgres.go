package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khaesha/reporag/apps/backend/internal/records"
)

const (
	insertDocument = `
		INSERT INTO documents (
			uri, source_year, title, abstract, authors, item_type, subjects,
			divisions, depositing_user, date_deposited, search_text
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (uri) DO NOTHING`
	updateDocument = `
		UPDATE documents SET
			source_year = $2, title = $3, abstract = $4, authors = $5,
			item_type = $6, subjects = $7, divisions = $8, depositing_user = $9,
			date_deposited = $10, search_text = $11, imported_at = now()
		WHERE uri = $1`
)

type Counts struct {
	Inserted int
	Updated  int
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (store *Store) ImportFile(ctx context.Context, documents []records.Document) (Counts, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return Counts{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck -- rollback is harmless after commit

	var counts Counts
	for index, document := range documents {
		arguments := []any{
			document.URI, document.SourceYear, document.Title, document.Abstract,
			document.Authors, document.ItemType, document.Subjects, document.Divisions,
			document.DepositingUser, document.DateDeposited, document.SearchText,
		}
		tag, err := tx.Exec(ctx, insertDocument, arguments...)
		if err != nil {
			return Counts{}, fmt.Errorf("record[%d]: %w", index, err)
		}
		if tag.RowsAffected() == 1 {
			counts.Inserted++
			continue
		}
		tag, err = tx.Exec(ctx, updateDocument, arguments...)
		if err != nil {
			return Counts{}, fmt.Errorf("record[%d]: %w", index, err)
		}
		if tag.RowsAffected() != 1 {
			return Counts{}, fmt.Errorf("record[%d]: existing URI disappeared during update", index)
		}
		counts.Updated++
	}

	if err := tx.Commit(ctx); err != nil {
		return Counts{}, err
	}
	return counts, nil
}
