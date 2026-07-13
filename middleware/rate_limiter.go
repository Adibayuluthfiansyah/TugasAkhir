package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	ginlimiter "github.com/ulule/limiter/v3/drivers/middleware/gin"
	memorystore "github.com/ulule/limiter/v3/drivers/store/memory"
)

func RateLimiter() gin.HandlerFunc {
	rate, err := limiter.NewRateFromFormatted("100000-M") // unlimited untuk load testing
	// rate, err := limiter.NewRateFromFormatted("60-M") // 60 request per menit — kembalikan setelah testing
	if err != nil {
		panic(err)
	}

	// In-memory store
	store := memorystore.NewStore()

	return ginlimiter.NewMiddleware(limiter.New(store, rate))
}
