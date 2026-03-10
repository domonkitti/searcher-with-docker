package internal

import (
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func attachNextProxy(r *gin.Engine, nextBase string) {
	u, _ := url.Parse(nextBase)
	proxy := httputil.NewSingleHostReverseProxy(u)

	// ส่งต่อทุกอย่างที่ไม่ใช่ /api ไป Next
	r.NoRoute(func(c *gin.Context) {
		// ถ้าเป็น /api ให้ 404 ตามปกติ (กันเผลอ)
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(404, gin.H{"detail": "Not Found"})
			return
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	})
}
