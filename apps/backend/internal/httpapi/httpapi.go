package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/khaesha/reporag/apps/backend/internal/search"
	"github.com/khaesha/reporag/apps/backend/internal/store"
)

type searchRequest struct {
	Query       string  `form:"q"`
	Mode        string  `form:"mode,default=hybrid"`
	Year        *int    `form:"year"`
	Division    *string `form:"division"`
	ItemType    *string `form:"item_type"`
	HasAbstract *string `form:"has_abstract"`
	Sort        string  `form:"sort,default=relevance"`
	Page        int     `form:"page,default=1"`
	Limit       int     `form:"limit,default=10"`
}

type searchResponse struct {
	Query   string                 `json:"query"`
	Page    int                    `json:"page"`
	Limit   int                    `json:"limit"`
	Total   int                    `json:"total"`
	Results []store.SearchDocument `json:"results"`
}

type relatedRequest struct {
	URI      string  `form:"uri"`
	Division *string `form:"division"`
	Limit    int     `form:"limit,default=6"`
}

type relatedResponse struct {
	SourceURI string                 `json:"source_uri"`
	Results   []store.SearchDocument `json:"results"`
}

type trendsRequest struct {
	Year     *int    `form:"year"`
	Division *string `form:"division"`
}

func New(
	ping func(context.Context) error,
	search func(context.Context, store.SearchParams) (store.SearchResult, error),
	related func(context.Context, store.RelatedParams) (store.RelatedResult, error),
	trends func(context.Context, store.TrendParams) (store.TrendResult, error),
	filters func(context.Context) (store.FilterValues, error),
	frontendOrigin string,
) http.Handler {
	router := gin.New()
	router.Use(requestID(), requestLog(), gin.Recovery(), cors(frontendOrigin))
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", func(c *gin.Context) {
		if err := ping(c.Request.Context()); err != nil {
			slog.Warn("database readiness check failed", "request_id", c.GetString("request_id"))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": gin.H{
				"code": "not_ready", "message": "database unavailable",
			}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	})
	router.GET("/api/v1/search", searchHandler(search))
	router.GET("/api/v1/related", relatedHandler(related))
	router.GET("/api/v1/trends", trendsHandler(trends))
	router.GET("/api/v1/filters", filtersHandler(filters))
	return router
}

func relatedHandler(related func(context.Context, store.RelatedParams) (store.RelatedResult, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request relatedRequest
		if err := c.ShouldBindQuery(&request); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_query", "query parameters must use valid types")
			return
		}
		params, err := request.params()
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
			return
		}
		result, err := related(c.Request.Context(), params)
		if errors.Is(err, store.ErrRelatedNotFound) {
			writeError(c, http.StatusNotFound, "not_found", "source record not found")
			return
		}
		if errors.Is(err, store.ErrRelatedEmbeddingMissing) {
			writeError(c, http.StatusConflict, "embedding_unavailable", "source record has no current embedding")
			return
		}
		if err != nil {
			slog.Error("related search failed", "request_id", c.GetString("request_id"), "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "related theses unavailable")
			return
		}
		c.JSON(http.StatusOK, relatedResponse{SourceURI: result.SourceURI, Results: result.Documents})
	}
}

func trendsHandler(trends func(context.Context, store.TrendParams) (store.TrendResult, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request trendsRequest
		if err := c.ShouldBindQuery(&request); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_query", "query parameters must use valid types")
			return
		}
		params, err := request.params()
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
			return
		}
		result, err := trends(c.Request.Context(), params)
		if err != nil {
			slog.Error("trend lookup failed", "request_id", c.GetString("request_id"), "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "trends unavailable")
			return
		}
		c.JSON(http.StatusOK, result)
	}
}

func searchHandler(execute func(context.Context, store.SearchParams) (store.SearchResult, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request searchRequest
		if err := c.ShouldBindQuery(&request); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_query", "query parameters must use valid types")
			return
		}
		params, err := request.params()
		if err != nil {
			writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
			return
		}
		result, err := execute(c.Request.Context(), params)
		if err != nil {
			slog.Error("search failed", "request_id", c.GetString("request_id"), "error", err)
			if errors.Is(err, search.ErrSemanticUnavailable) {
				writeError(c, http.StatusServiceUnavailable, "semantic_unavailable", "semantic search unavailable")
				return
			}
			writeError(c, http.StatusInternalServerError, "internal_error", "search unavailable")
			return
		}
		if result.Degraded {
			slog.Warn("search degraded", "request_id", c.GetString("request_id"), "mode", params.Mode)
		}
		slog.Info("search timing", "request_id", c.GetString("request_id"), "mode", params.Mode, "model_duration", result.ModelTime, "retrieval_duration", result.SearchTime)
		c.JSON(http.StatusOK, searchResponse{
			Query: params.Query, Page: params.Page, Limit: params.Limit,
			Total: result.Total, Results: result.Documents,
		})
	}
}

func filtersHandler(filters func(context.Context) (store.FilterValues, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		values, err := filters(c.Request.Context())
		if err != nil {
			slog.Error("filter lookup failed", "request_id", c.GetString("request_id"), "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "filters unavailable")
			return
		}
		c.JSON(http.StatusOK, values)
	}
}

func (request searchRequest) params() (store.SearchParams, error) {
	query := strings.TrimSpace(request.Query)
	if query == "" {
		return store.SearchParams{}, errors.New("q is required")
	}
	if utf8.RuneCountInString(query) > 200 {
		return store.SearchParams{}, errors.New("q must be at most 200 characters")
	}
	if request.Year != nil && (*request.Year < 1900 || *request.Year > 2100) {
		return store.SearchParams{}, errors.New("year must be between 1900 and 2100")
	}
	division, err := filterValue("division", request.Division)
	if err != nil {
		return store.SearchParams{}, err
	}
	itemType, err := filterValue("item_type", request.ItemType)
	if err != nil {
		return store.SearchParams{}, err
	}
	var hasAbstract *bool
	if request.HasAbstract != nil {
		value := *request.HasAbstract == "true"
		if !value && *request.HasAbstract != "false" {
			return store.SearchParams{}, errors.New("has_abstract must be true or false")
		}
		hasAbstract = &value
	}
	if request.Sort != "relevance" && request.Sort != "title" && request.Sort != "date" {
		return store.SearchParams{}, errors.New("sort must be relevance, title, or date")
	}
	if request.Mode != "lexical" && request.Mode != "semantic" && request.Mode != "hybrid" {
		return store.SearchParams{}, errors.New("mode must be lexical, semantic, or hybrid")
	}
	if request.Page < 1 {
		return store.SearchParams{}, errors.New("page must be a positive integer")
	}
	if request.Limit < 1 || request.Limit > 50 {
		return store.SearchParams{}, errors.New("limit must be between 1 and 50")
	}
	if int64(request.Page) > math.MaxInt64/int64(request.Limit) {
		return store.SearchParams{}, errors.New("page is too large")
	}
	return store.SearchParams{
		Query: query, Mode: request.Mode, Year: request.Year, Division: division, ItemType: itemType,
		HasAbstract: hasAbstract, Sort: request.Sort, Page: request.Page, Limit: request.Limit,
	}, nil
}

func (request relatedRequest) params() (store.RelatedParams, error) {
	uri := strings.TrimSpace(request.URI)
	if uri == "" {
		return store.RelatedParams{}, errors.New("uri is required")
	}
	if utf8.RuneCountInString(uri) > 2048 {
		return store.RelatedParams{}, errors.New("uri must be at most 2048 characters")
	}
	division, err := filterValue("division", request.Division)
	if err != nil {
		return store.RelatedParams{}, err
	}
	if request.Limit < 1 || request.Limit > 20 {
		return store.RelatedParams{}, errors.New("limit must be between 1 and 20")
	}
	return store.RelatedParams{URI: uri, Division: division, Limit: request.Limit}, nil
}

func (request trendsRequest) params() (store.TrendParams, error) {
	if request.Year != nil && (*request.Year < 1900 || *request.Year > 2100) {
		return store.TrendParams{}, errors.New("year must be between 1900 and 2100")
	}
	division, err := filterValue("division", request.Division)
	if err != nil {
		return store.TrendParams{}, err
	}
	return store.TrendParams{Year: request.Year, Division: division}, nil
}

func filterValue(name string, value *string) (string, error) {
	if value == nil {
		return "", nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}
	if utf8.RuneCountInString(trimmed) > 200 {
		return "", fmt.Errorf("%s must be at most 200 characters", name)
	}
	return trimmed, nil
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		value := make([]byte, 16)
		if _, err := rand.Read(value); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{
				"code": "internal_error", "message": "request ID unavailable",
			}})
			return
		}
		id := hex.EncodeToString(value)
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func requestLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		slog.Info("request",
			"request_id", c.GetString("request_id"),
			"method", c.Request.Method,
			"route", c.FullPath(),
			"status", c.Writer.Status(),
			"duration", time.Since(started),
		)
	}
}

func cors(frontendOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Origin") == frontendOrigin {
			c.Header("Access-Control-Allow-Origin", frontendOrigin)
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
