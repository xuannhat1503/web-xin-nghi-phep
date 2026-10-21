// Package httpapi wires the Gin router, cross-cutting middleware and HTTP-level helpers.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/analytics"
	"github.com/g-asiapacific/elm/backend/internal/auth"
	"github.com/g-asiapacific/elm/backend/internal/config"
	"github.com/g-asiapacific/elm/backend/internal/employee"
	"github.com/g-asiapacific/elm/backend/internal/holiday"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/leave"
	"github.com/g-asiapacific/elm/backend/internal/notification"
	"github.com/g-asiapacific/elm/backend/internal/review"
	"github.com/g-asiapacific/elm/backend/internal/settings"
	"github.com/g-asiapacific/elm/backend/internal/system"
)

type Deps struct {
	Config    *config.Config
	Logger    *slog.Logger
	Health    *Health
	Auth      *auth.Service // nil only in router unit tests
	Employees *employee.Handlers
	Leave     *leave.Handlers
	Calendar  *holiday.Handlers
	Settings  *settings.Handlers
	Review    *review.Handlers
	Notify    *notification.Center
	Outbox    *notification.Admin
	Analytics *analytics.Handlers
	System    *system.Handlers
}

// MaxBodyBytes caps request bodies; individual routes may lower it.
const MaxBodyBytes = 64 << 10

func NewRouter(d Deps) *gin.Engine {
	if d.Config.Env != config.EnvDevelopment {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	// Only the local SvelteKit proxy may set X-Forwarded-For.
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})
	r.HandleMethodNotAllowed = true

	r.Use(
		requestID(),
		recovery(d.Logger),
		accessLog(d.Logger),
		securityHeaders(),
		bodyLimit(MaxBodyBytes),
	)

	r.GET("/health/live", d.Health.Live)
	r.GET("/health/ready", d.Health.Ready)

	if a := d.Auth; a != nil {
		authPerMin, authBurst, apiPerMin, apiBurst := 20, 10, 300, 60
		authLimit := NewRateLimiter(authPerMin, authBurst).Middleware()
		apiLimit := NewRateLimiter(apiPerMin, apiBurst).Middleware()

		ag := r.Group("/auth", authLimit, a.LoadSession(), a.CSRF())
		ag.GET("/google/login", a.Login)
		ag.GET("/google/callback", a.Callback)
		ag.POST("/logout", a.Logout)

		api := r.Group("/api/v1", apiLimit, a.LoadSession(), a.CSRF())
		api.GET("/me", auth.RequireAuth(), a.Me)

		active := api.Group("", auth.RequireActive())
		admin := api.Group("/admin", auth.RequireAdmin())
		if d.Employees != nil {
			d.Employees.Register(admin)
		}
		if d.Calendar != nil {
			d.Calendar.RegisterAdmin(admin)
			d.Calendar.RegisterActive(active)
		}
		if d.Settings != nil {
			d.Settings.RegisterAdmin(admin)
			d.Settings.RegisterActive(active)
		}
		if d.Leave != nil {
			d.Leave.Register(active)
		}
		if d.Review != nil {
			d.Review.RegisterActive(active)
		}
		if d.Notify != nil {
			d.Notify.Register(active)
		}
		if d.Outbox != nil {
			d.Outbox.Register(admin)
		}
		if d.Analytics != nil {
			d.Analytics.RegisterAdmin(admin)
		}
		if d.System != nil {
			d.System.RegisterAdmin(admin)
		}
	}

	r.NoRoute(func(c *gin.Context) { httpx.Abort(c, http.StatusNotFound, "not_found", "Resource not found") })
	r.NoMethod(func(c *gin.Context) {
		httpx.Abort(c, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	})
	return r
}
