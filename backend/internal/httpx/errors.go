package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Error is a client-safe error. Services return it; handlers pass every error to Fail.
type Error struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string // per-field validation messages
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

func E(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// Validation collects field errors; Err returns nil when there are none.
type Validation map[string]string

func (v Validation) Add(field, msg string) {
	if _, exists := v[field]; !exists {
		v[field] = msg
	}
}

func (v Validation) Err() error {
	if len(v) == 0 {
		return nil
	}
	return &Error{Status: http.StatusUnprocessableEntity, Code: "validation_failed", Message: "Please correct the highlighted fields.", Fields: v}
}

var (
	ErrNotFound  = E(http.StatusNotFound, "not_found", "Resource not found")
	ErrForbidden = E(http.StatusForbidden, "forbidden", "You do not have access to this resource")
	// ErrConflict is returned when an optimistic-lock version no longer matches.
	ErrConflict = E(http.StatusConflict, "conflict", "This record was changed by someone else. Reload and try again.")
)

// Fail writes err as JSON. Unknown errors are logged and reported as a generic 500.
func Fail(c *gin.Context, log *slog.Logger, err error) {
	var he *Error
	if errors.As(err, &he) {
		c.AbortWithStatusJSON(he.Status, ErrorBody{ErrorDetail{
			Code: he.Code, Message: he.Message, Fields: he.Fields, RequestID: c.GetString("request_id"),
		}})
		return
	}
	log.Error("request failed", slog.String("route", c.FullPath()), slog.String("request_id", c.GetString("request_id")), slog.String("error", err.Error()))
	Abort(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
}

// BindJSON strictly decodes a single JSON object (unknown fields rejected). On failure it aborts with 400.
func BindJSON(c *gin.Context, v any) bool {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		msg := "Invalid request body"
		var syntax *json.SyntaxError
		var typ *json.UnmarshalTypeError
		switch {
		case errors.As(err, &typ):
			msg = fmt.Sprintf("Invalid value for field %q", typ.Field)
		case strings.HasPrefix(err.Error(), "json: unknown field"):
			msg = "Unknown field in request body"
		case errors.As(err, &syntax), errors.Is(err, io.EOF):
		}
		Abort(c, http.StatusBadRequest, "invalid_request", msg)
		return false
	}
	if dec.More() {
		Abort(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return false
	}
	return true
}

// IDParam parses a positive integer path parameter; it aborts with 404 when invalid.
func IDParam(c *gin.Context, name string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		Abort(c, http.StatusNotFound, "not_found", "Resource not found")
		return 0, false
	}
	return id, true
}

// Page is a validated server-side page request.
type Page struct {
	Number int
	Size   int
}

func (p Page) Offset() int32 { return int32((p.Number - 1) * p.Size) }
func (p Page) Limit() int32  { return int32(p.Size) }

// PageParam reads ?page= and ?page_size= (clamped to [1, max]); invalid values fall back to page 1.
func PageParam(c *gin.Context, defSize, maxSize int) Page {
	p := Page{Number: 1, Size: defSize}
	if n, err := strconv.Atoi(c.Query("page")); err == nil && n >= 1 && n <= 100000 {
		p.Number = n
	}
	if s, err := strconv.Atoi(c.Query("page_size")); err == nil && s >= 1 {
		p.Size = min(s, maxSize)
	}
	return p
}

// Paged is the common list envelope.
type Paged[T any] struct {
	Items      []T   `json:"items"`
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

func NewPaged[T any](items []T, p Page, total int64) Paged[T] {
	if items == nil {
		items = []T{}
	}
	return Paged[T]{Items: items, Page: p.Number, PageSize: p.Size, TotalItems: total,
		TotalPages: int(math.Ceil(float64(total) / float64(p.Size)))}
}

// LikePattern builds a parameterized LIKE pattern, escaping wildcards in user input.
func LikePattern(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return "%" + r.Replace(s) + "%"
}
