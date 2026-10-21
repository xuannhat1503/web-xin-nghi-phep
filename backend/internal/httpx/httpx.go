// Package httpx holds HTTP helpers shared by feature packages and the router.
package httpx

import (
	"github.com/gin-gonic/gin"
)

// ErrorBody is the only error shape clients ever see: no SQL, stack traces or internal details.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// Abort writes a JSON error and stops the handler chain.
func Abort(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, ErrorBody{ErrorDetail{Code: code, Message: msg, RequestID: c.GetString("request_id")}})
}
