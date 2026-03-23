package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type clientData struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	mu      sync.Mutex
	clients = map[string]*clientData{}
)

func getLimiter(clientID string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	if c, ok := clients[clientID]; ok {
		c.lastSeen = time.Now()
		return c.limiter
	}

	limiter := rate.NewLimiter(1, 2)
	clients[clientID] = &clientData{
		limiter:  limiter,
		lastSeen: time.Now(),
	}

	fmt.Println("RateLimiter: ", clientID, clients[clientID].lastSeen, clients[clientID].limiter)
	return limiter
}

func RateLimiterMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientID := c.GetHeader("X-Client-ID")
		fmt.Println("Here: clientID:", clientID)
		if clientID == "" {
			clientID = c.ClientIP()
		}

		if !getLimiter(clientID).Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
