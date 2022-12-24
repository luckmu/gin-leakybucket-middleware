package middleware

import (
	"crypto/rand"
	"strings"

	"github.com/gin-gonic/gin"
)

func SetRequest(c *gin.Context) {
	// x-forwarded-for
	// x-real-ip
	// remote addr
	var xff, xrip string
	xff, xrip = c.GetHeader("X-Forwarded-For"), c.GetHeader("X-Real-IP")
	if xff != "" && xrip == "" {
		for _, ip := range strings.Split(xff, ",") {
			// maybe filter private ips
			c.Set("ip", ip)
			break
		}
	} else if xff == "" && xrip == "" {
		c.Set("ip", c.ClientIP())
	} else {
		c.Set("ip", xrip)
	}

	buf := make([]byte, 16)
	rand.Read(buf)
	c.Set("Request-ID", buf)
}
