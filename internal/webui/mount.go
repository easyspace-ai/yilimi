package webui

import (
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// Mount 掛載編譯進二進位的前端（Vite dist）。設 AISTOCK_SERVE_WEB=0 可僅啟 API、不托管頁面。
func Mount(engine *gin.Engine) {
	if strings.TrimSpace(os.Getenv("AISTOCK_SERVE_WEB")) == "0" {
		return
	}
	sub, err := fs.Sub(webDist, "webdist")
	if err != nil {
		panic(err)
	}
	httpFS := http.FS(sub)
	fileServer := http.FileServer(httpFS)

	engine.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		rel := strings.TrimPrefix(c.Request.URL.Path, "/")
		if rel != "" {
			if fi, err := fs.Stat(sub, rel); err == nil && !fi.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
			// 目錄請求：嘗試 index.html
			if fi, err := fs.Stat(sub, rel+"/index.html"); err == nil && !fi.IsDir() {
				c.Request.URL.Path = "/" + rel + "/index.html"
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}
