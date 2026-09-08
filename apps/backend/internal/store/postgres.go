package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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
	filterValuesQuery = `
		SELECT
			ARRAY(SELECT value FROM (SELECT DISTINCT source_year::integer AS value FROM documents) years ORDER BY value DESC),
			ARRAY(SELECT value FROM (SELECT DISTINCT divisions AS value FROM documents WHERE divisions IS NOT NULL) divisions ORDER BY lower(value), value),
			ARRAY(SELECT value FROM (SELECT DISTINCT item_type AS value FROM documents WHERE item_type IS NOT NULL) item_types ORDER BY lower(value), value)`
)

var searchSorts = map[string]string{
	"relevance": "score DESC, lower(title), uri",
	"title":     "lower(title), title, uri",
	"date":      "date_deposited DESC NULLS LAST, lower(title), uri",
}

type Counts struct {
	Inserted int
	Updated  int
}

type SearchParams struct {
	Query       string
	Year        *int
	Division    string
	ItemType    string
	HasAbstract *bool
	Sort        string
	Page        int
	Limit       int
}

type SearchDocument struct {
	Title         string     `json:"title"`
	Abstract      *string    `json:"abstract"`
	Authors       []string   `json:"authors"`
	ItemType      *string    `json:"item_type"`
	Subjects      *string    `json:"subjects"`
	Divisions     *string    `json:"divisions"`
	DateDeposited *time.Time `json:"date_deposited"`
	SourceYear    int        `json:"source_year"`
	URI           string     `json:"uri"`
	Score         float32    `json:"score"`
}

type SearchResult struct {
	Total     int
	Documents []SearchDocument
}

type FilterValues struct {
	Years     []int32  `json:"years"`
	Divisions []string `json:"divisions"`
	ItemTypes []string `json:"item_types"`
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

func (store *Store) Search(ctx context.Context, params SearchParams) (SearchResult, error) {
	orderBy, ok := searchSorts[params.Sort]
	if !ok {
		return SearchResult{}, errors.New("unsupported search sort")
	}

	arguments := []any{params.Query}
	conditions := []string{"search_vector @@ plainto_tsquery('simple', $1)"}
	addCondition := func(expression string, value any) {
		arguments = append(arguments, value)
		conditions = append(conditions, fmt.Sprintf(expression, len(arguments)))
	}
	if params.Year != nil {
		addCondition("source_year = $%d", *params.Year)
	}
	if params.Division != "" {
		addCondition("divisions = $%d", params.Division)
	}
	if params.ItemType != "" {
		addCondition("item_type = $%d", params.ItemType)
	}
	if params.HasAbstract != nil {
		addCondition("(abstract IS NOT NULL) = $%d", *params.HasAbstract)
	}
	where := strings.Join(conditions, " AND ")

	var result SearchResult
	if err := store.pool.QueryRow(ctx, "SELECT count(*) FROM documents WHERE "+where, arguments...).Scan(&result.Total); err != nil {
		return SearchResult{}, fmt.Errorf("count search results: %w", err)
	}

	arguments = append(arguments, params.Limit, (int64(params.Page)-1)*int64(params.Limit))
	query := fmt.Sprintf(`
		SELECT title, abstract, authors, item_type, subjects, divisions,
		       date_deposited, source_year, uri,
		       ts_rank_cd(search_vector, plainto_tsquery('simple', $1)) AS score
		FROM documents
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, where, orderBy, len(arguments)-1, len(arguments))
	rows, err := store.pool.Query(ctx, query, arguments...)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search documents: %w", err)
	}
	defer rows.Close()

	result.Documents = make([]SearchDocument, 0, params.Limit)
	for rows.Next() {
		var document SearchDocument
		if err := rows.Scan(
			&document.Title, &document.Abstract, &document.Authors, &document.ItemType,
			&document.Subjects, &document.Divisions, &document.DateDeposited,
			&document.SourceYear, &document.URI, &document.Score,
		); err != nil {
			return SearchResult{}, fmt.Errorf("scan search result: %w", err)
		}
		result.Documents = append(result.Documents, document)
	}
	if err := rows.Err(); err != nil {
		return SearchResult{}, fmt.Errorf("read search results: %w", err)
	}
	return result, nil
}

func (store *Store) Filters(ctx context.Context) (FilterValues, error) {
	var filters FilterValues
	if err := store.pool.QueryRow(ctx, filterValuesQuery).Scan(&filters.Years, &filters.Divisions, &filters.ItemTypes); err != nil {
		return FilterValues{}, fmt.Errorf("list filter values: %w", err)
	}
	return filters, nil
}
