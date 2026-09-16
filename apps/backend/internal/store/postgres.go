package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
			date_deposited = $10, search_text = $11, imported_at = now(),
			embedding = CASE WHEN ROW(title, abstract, authors, subjects, divisions) IS DISTINCT FROM ROW($3, $4, $5, $7, $8) THEN NULL ELSE embedding END,
			embedding_model = CASE WHEN ROW(title, abstract, authors, subjects, divisions) IS DISTINCT FROM ROW($3, $4, $5, $7, $8) THEN NULL ELSE embedding_model END,
			embedding_input_hash = CASE WHEN ROW(title, abstract, authors, subjects, divisions) IS DISTINCT FROM ROW($3, $4, $5, $7, $8) THEN NULL ELSE embedding_input_hash END,
			embedding_dimensions = CASE WHEN ROW(title, abstract, authors, subjects, divisions) IS DISTINCT FROM ROW($3, $4, $5, $7, $8) THEN NULL ELSE embedding_dimensions END,
			embedded_at = CASE WHEN ROW(title, abstract, authors, subjects, divisions) IS DISTINCT FROM ROW($3, $4, $5, $7, $8) THEN NULL ELSE embedded_at END
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

var (
	ErrRelatedNotFound         = errors.New("related source not found")
	ErrRelatedEmbeddingMissing = errors.New("related source embedding unavailable")
)

type Counts struct {
	Inserted int
	Updated  int
}

type SearchParams struct {
	Query       string
	Mode        string
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
	Total      int
	Documents  []SearchDocument
	Degraded   bool
	ModelTime  time.Duration
	SearchTime time.Duration
}

type SearchCandidate struct {
	Document SearchDocument
	Rank     int
	Exact    bool
}

type RelatedParams struct {
	URI      string
	Division string
	Limit    int
}

type RelatedResult struct {
	SourceURI string
	Documents []SearchDocument
}

type TrendParams struct {
	Year     *int
	Division string
}

type TrendBucket struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type TrendResult struct {
	Total            int           `json:"total"`
	MissingAbstracts int           `json:"missing_abstracts"`
	MissingDivisions int           `json:"missing_divisions"`
	MissingItemTypes int           `json:"missing_item_types"`
	MissingSubjects  int           `json:"missing_subjects"`
	ByYear           []TrendBucket `json:"by_year"`
	ByDivision       []TrendBucket `json:"by_division"`
	ByItemType       []TrendBucket `json:"by_item_type"`
	BySubject        []TrendBucket `json:"by_subject"`
}

type FilterValues struct {
	Years     []int32  `json:"years"`
	Divisions []string `json:"divisions"`
	ItemTypes []string `json:"item_types"`
}

type EmbeddingDocument struct {
	Document            records.Document
	HasEmbedding        bool
	EmbeddingModel      *string
	EmbeddingInputHash  *string
	EmbeddingDimensions *int16
}

type EmbeddingUpdate struct {
	URI        string
	Vector     []float32
	Model      string
	InputHash  string
	Dimensions int
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

func (store *Store) LexicalCandidates(ctx context.Context, params SearchParams, limit int) ([]SearchCandidate, error) {
	arguments := []any{params.Query}
	conditions := []string{"search_vector @@ plainto_tsquery('simple', $1)"}
	where := appendSearchFilters(&arguments, conditions, params)
	query := fmt.Sprintf(`
		SELECT title, abstract, authors, item_type, subjects, divisions,
		       date_deposited, source_year, uri,
		       ts_rank_cd(search_vector, plainto_tsquery('simple', $1)) AS score,
		       (lower(title) = lower($1) OR $1 = ANY(authors)) AS exact
		FROM documents
		WHERE %s
		ORDER BY score DESC, lower(title), uri
		LIMIT $%d`, where, len(arguments)+1)
	arguments = append(arguments, limit)
	return scanCandidates(ctx, store.pool, query, arguments)
}

func (store *Store) SemanticCandidates(ctx context.Context, params SearchParams, vector []float32, model string, dimensions, limit int) ([]SearchCandidate, error) {
	arguments := []any{vectorLiteral(vector), model, dimensions, params.Query}
	conditions := []string{
		"embedding IS NOT NULL",
		"embedding_input_hash IS NOT NULL",
		"embedding_model = $2",
		"embedding_dimensions = $3",
	}
	where := appendSearchFilters(&arguments, conditions, params)
	query := fmt.Sprintf(`
		SELECT title, abstract, authors, item_type, subjects, divisions,
		       date_deposited, source_year, uri,
		       1 - (embedding OPERATOR(extensions.<=>) $1::extensions.vector) AS score,
		       (lower(title) = lower($4) OR $4 = ANY(authors)) AS exact
		FROM documents
		WHERE %s
		ORDER BY embedding OPERATOR(extensions.<=>) $1::extensions.vector, lower(title), uri
		LIMIT $%d`, where, len(arguments)+1)
	arguments = append(arguments, limit)
	return scanCandidates(ctx, store.pool, query, arguments)
}

func (store *Store) Related(ctx context.Context, params RelatedParams, model string, dimensions int) (RelatedResult, error) {
	var current bool
	err := store.pool.QueryRow(ctx, `
		SELECT embedding IS NOT NULL AND embedding_input_hash IS NOT NULL
			AND embedding_model = $2 AND embedding_dimensions = $3
		FROM documents WHERE uri = $1`, params.URI, model, dimensions).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return RelatedResult{}, ErrRelatedNotFound
	}
	if err != nil {
		return RelatedResult{}, fmt.Errorf("find related source: %w", err)
	}
	if !current {
		return RelatedResult{}, ErrRelatedEmbeddingMissing
	}

	rows, err := store.pool.Query(ctx, `
		WITH source AS (SELECT embedding FROM documents WHERE uri = $1)
		SELECT title, abstract, authors, item_type, subjects, divisions,
		       date_deposited, source_year, uri,
		       1 - (documents.embedding OPERATOR(extensions.<=>) source.embedding) AS score
		FROM documents CROSS JOIN source
		WHERE uri <> $1 AND documents.embedding IS NOT NULL AND embedding_input_hash IS NOT NULL
		  AND embedding_model = $2 AND embedding_dimensions = $3
		  AND ($4 = '' OR divisions = $4)
		ORDER BY documents.embedding OPERATOR(extensions.<=>) source.embedding, lower(title), uri
		LIMIT $5`, params.URI, model, dimensions, params.Division, params.Limit)
	if err != nil {
		return RelatedResult{}, fmt.Errorf("find related documents: %w", err)
	}
	defer rows.Close()

	result := RelatedResult{SourceURI: params.URI, Documents: make([]SearchDocument, 0, params.Limit)}
	for rows.Next() {
		var document SearchDocument
		if err := rows.Scan(
			&document.Title, &document.Abstract, &document.Authors, &document.ItemType,
			&document.Subjects, &document.Divisions, &document.DateDeposited,
			&document.SourceYear, &document.URI, &document.Score,
		); err != nil {
			return RelatedResult{}, fmt.Errorf("scan related document: %w", err)
		}
		result.Documents = append(result.Documents, document)
	}
	if err := rows.Err(); err != nil {
		return RelatedResult{}, fmt.Errorf("read related documents: %w", err)
	}
	return result, nil
}

func (store *Store) Trends(ctx context.Context, params TrendParams) (TrendResult, error) {
	arguments, where := trendFilter(params)
	var result TrendResult
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*),
			count(*) FILTER (WHERE abstract IS NULL),
			count(*) FILTER (WHERE divisions IS NULL),
			count(*) FILTER (WHERE item_type IS NULL),
			count(*) FILTER (WHERE subjects IS NULL)
		FROM documents WHERE `+where, arguments...).Scan(
		&result.Total, &result.MissingAbstracts, &result.MissingDivisions,
		&result.MissingItemTypes, &result.MissingSubjects,
	); err != nil {
		return TrendResult{}, fmt.Errorf("count trends: %w", err)
	}
	var err error
	if result.ByYear, err = store.trendBuckets(ctx, `source_year::text`, where, arguments, `value DESC`); err != nil {
		return TrendResult{}, err
	}
	if result.ByDivision, err = store.trendBuckets(ctx, `divisions`, where+` AND divisions IS NOT NULL`, arguments, `count(*) DESC, lower(value), value`); err != nil {
		return TrendResult{}, err
	}
	if result.ByItemType, err = store.trendBuckets(ctx, `item_type`, where+` AND item_type IS NOT NULL`, arguments, `count(*) DESC, lower(value), value`); err != nil {
		return TrendResult{}, err
	}
	if result.BySubject, err = store.trendBuckets(ctx, subjectClassSQL, where+` AND subjects IS NOT NULL`, arguments, `count(*) DESC, lower(value), value`); err != nil {
		return TrendResult{}, err
	}
	return result, nil
}

const subjectClassSQL = `CASE split_part(subjects, ' > ', 1)
	WHEN 'B Philosophy' THEN 'B Philosophy, Psychology, Religion'
	WHEN 'G Geography' THEN 'G Geography, Anthropology, Recreation'
	WHEN 'H Social sciences' THEN 'H Social Sciences'
	WHEN 'J Political science' THEN 'J Political Science'
	WHEN 'K Law' THEN 'K Law'
	WHEN 'L Education' THEN 'L Education'
	WHEN 'M Music' THEN 'M Music'
	WHEN 'P Language' THEN 'P Language and Literature'
	WHEN 'Q Science' THEN 'Q Science'
	WHEN 'S Agriculture' THEN 'S Agriculture'
	WHEN 'T Technology' THEN 'T Technology'
	WHEN 'V Naval science' THEN 'V Naval Science'
	WHEN 'Z Bibliography' THEN 'Z Bibliography and Information Resources'
END`

func trendFilter(params TrendParams) ([]any, string) {
	arguments := make([]any, 0, 2)
	conditions := make([]string, 0, 2)
	if params.Year != nil {
		arguments = append(arguments, *params.Year)
		conditions = append(conditions, fmt.Sprintf("source_year = $%d", len(arguments)))
	}
	if params.Division != "" {
		arguments = append(arguments, params.Division)
		conditions = append(conditions, fmt.Sprintf("divisions = $%d", len(arguments)))
	}
	if len(conditions) == 0 {
		return arguments, "TRUE"
	}
	return arguments, strings.Join(conditions, " AND ")
}

func (store *Store) trendBuckets(ctx context.Context, expression, where string, arguments []any, orderBy string) ([]TrendBucket, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT value, count(*)
		FROM (SELECT `+expression+` AS value FROM documents WHERE `+where+`) values
		WHERE value IS NOT NULL
		GROUP BY value
		ORDER BY `+orderBy+`
		LIMIT 20`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list trend buckets: %w", err)
	}
	defer rows.Close()
	buckets := make([]TrendBucket, 0)
	for rows.Next() {
		var bucket TrendBucket
		if err := rows.Scan(&bucket.Value, &bucket.Count); err != nil {
			return nil, fmt.Errorf("scan trend bucket: %w", err)
		}
		buckets = append(buckets, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read trend buckets: %w", err)
	}
	return buckets, nil
}

func appendSearchFilters(arguments *[]any, conditions []string, params SearchParams) string {
	add := func(expression string, value any) {
		*arguments = append(*arguments, value)
		conditions = append(conditions, fmt.Sprintf(expression, len(*arguments)))
	}
	if params.Year != nil {
		add("source_year = $%d", *params.Year)
	}
	if params.Division != "" {
		add("divisions = $%d", params.Division)
	}
	if params.ItemType != "" {
		add("item_type = $%d", params.ItemType)
	}
	if params.HasAbstract != nil {
		add("(abstract IS NOT NULL) = $%d", *params.HasAbstract)
	}
	return strings.Join(conditions, " AND ")
}

func scanCandidates(ctx context.Context, pool *pgxpool.Pool, query string, arguments []any) ([]SearchCandidate, error) {
	rows, err := pool.Query(ctx, query, arguments...)
	if err != nil {
		return nil, fmt.Errorf("search candidates: %w", err)
	}
	defer rows.Close()
	candidates := make([]SearchCandidate, 0)
	for rows.Next() {
		var candidate SearchCandidate
		if err := rows.Scan(
			&candidate.Document.Title, &candidate.Document.Abstract, &candidate.Document.Authors,
			&candidate.Document.ItemType, &candidate.Document.Subjects, &candidate.Document.Divisions,
			&candidate.Document.DateDeposited, &candidate.Document.SourceYear, &candidate.Document.URI,
			&candidate.Document.Score, &candidate.Exact,
		); err != nil {
			return nil, fmt.Errorf("scan search candidate: %w", err)
		}
		candidate.Rank = len(candidates) + 1
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read search candidates: %w", err)
	}
	return candidates, nil
}

func (store *Store) Filters(ctx context.Context) (FilterValues, error) {
	var filters FilterValues
	if err := store.pool.QueryRow(ctx, filterValuesQuery).Scan(&filters.Years, &filters.Divisions, &filters.ItemTypes); err != nil {
		return FilterValues{}, fmt.Errorf("list filter values: %w", err)
	}
	return filters, nil
}

func (store *Store) EmbeddingDocuments(ctx context.Context) ([]EmbeddingDocument, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT uri, source_year, title, abstract, authors, item_type, subjects,
		       divisions, depositing_user, date_deposited, search_text,
		       embedding IS NOT NULL, embedding_model, embedding_input_hash, embedding_dimensions
		FROM documents
		ORDER BY uri`)
	if err != nil {
		return nil, fmt.Errorf("list embedding documents: %w", err)
	}
	defer rows.Close()

	documents := make([]EmbeddingDocument, 0)
	for rows.Next() {
		var document EmbeddingDocument
		if err := rows.Scan(
			&document.Document.URI, &document.Document.SourceYear, &document.Document.Title,
			&document.Document.Abstract, &document.Document.Authors, &document.Document.ItemType,
			&document.Document.Subjects, &document.Document.Divisions, &document.Document.DepositingUser,
			&document.Document.DateDeposited, &document.Document.SearchText, &document.HasEmbedding,
			&document.EmbeddingModel, &document.EmbeddingInputHash, &document.EmbeddingDimensions,
		); err != nil {
			return nil, fmt.Errorf("scan embedding document: %w", err)
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read embedding documents: %w", err)
	}
	return documents, nil
}

func (store *Store) UpdateEmbeddings(ctx context.Context, updates []EmbeddingUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck -- rollback is harmless after commit

	for _, update := range updates {
		if len(update.Vector) != update.Dimensions {
			return errors.New("embedding dimensions are invalid")
		}
		tag, err := tx.Exec(ctx, `
			UPDATE documents
			SET embedding = $2::extensions.vector,
			    embedding_model = $3,
			    embedding_input_hash = $4,
			    embedding_dimensions = $5,
			    embedded_at = now()
			WHERE uri = $1`, update.URI, vectorLiteral(update.Vector), update.Model, update.InputHash, update.Dimensions)
		if err != nil {
			return fmt.Errorf("update %s: %w", update.URI, err)
		}
		if tag.RowsAffected() != 1 {
			return fmt.Errorf("update %s: document not found", update.URI)
		}
	}
	return tx.Commit(ctx)
}

func vectorLiteral(vector []float32) string {
	values := make([]string, len(vector))
	for index, value := range vector {
		values[index] = strconv.FormatFloat(float64(value), 'g', -1, 32)
	}
	return "[" + strings.Join(values, ",") + "]"
}
