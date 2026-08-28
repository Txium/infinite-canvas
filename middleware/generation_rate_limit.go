package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/handler"
	"github.com/tigerowo/infinite-canvas/service"
)

type generationWindow struct {
	startedAt time.Time
	count     int
}

var generationWindows = struct {
	sync.Mutex
	items map[string]generationWindow
}{items: map[string]generationWindow{}}

// GenerationRateLimit protects the shared upstream provider accounts from one
// platform user flooding every model route. Billing checks remain authoritative;
// this is an additional abuse-control boundary.
func GenerationRateLimit(c *gin.Context) {
	user, ok := service.UserFromContext(c.Request.Context())
	if !ok || user.ID == "" {
		handler.FailWithStatus(c.Writer, http.StatusUnauthorized, "未登录或权限不足")
		c.Abort()
		return
	}
	limit := config.Cfg.GenerationRPM
	if limit <= 0 {
		c.Next()
		return
	}
	now := time.Now()
	generationWindows.Lock()
	window := generationWindows.items[user.ID]
	if window.startedAt.IsZero() || now.Sub(window.startedAt) >= time.Minute {
		window = generationWindow{startedAt: now}
	}
	if window.count >= limit {
		generationWindows.Unlock()
		c.Header("Retry-After", "60")
		handler.FailWithStatus(c.Writer, http.StatusTooManyRequests, "生成请求过于频繁，请稍后再试")
		c.Abort()
		return
	}
	window.count++
	generationWindows.items[user.ID] = window
	if len(generationWindows.items) > 10000 {
		for id, item := range generationWindows.items {
			if now.Sub(item.startedAt) >= 2*time.Minute {
				delete(generationWindows.items, id)
			}
		}
	}
	generationWindows.Unlock()
	c.Next()
}
