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

	"github.com/khaesha/reporag/apps/backend/internal/store"
)

type searchRequest struct {
	Query       string  `form:"q"`
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

func New(
	ping func(context.Context) error,
	search func(context.Context, store.SearchParams) (store.SearchResult, error),
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
	router.GET("/api/v1/filters", filtersHandler(filters))
	return router
}

func searchHandler(search func(context.Context, store.SearchParams) (store.SearchResult, error)) gin.HandlerFunc {
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
		result, err := search(c.Request.Context(), params)
		if err != nil {
			slog.Error("search failed", "request_id", c.GetString("request_id"), "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "search unavailable")
			return
		}
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
		Query: query, Year: request.Year, Division: division, ItemType: itemType,
		HasAbstract: hasAbstract, Sort: request.Sort, Page: request.Page, Limit: request.Limit,
	}, nil
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
