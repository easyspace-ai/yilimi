package webui

import (
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func hasIndex(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !st.IsDir()
}

func resolveWebRoot() string {
	if d := strings.TrimSpace(os.Getenv("AISTOCK_WEB_DIR")); d != "" {
		d = filepath.Clean(d)
		if hasIndex(d) {
			return d
		}
		log.Printf("webui: AISTOCK_WEB_DIR=%s has no index.html; skipping static hosting", d)
		return ""
	}

	var exeDir string
	exe, err := os.Executable()
	if err == nil {
		exe, _ = filepath.EvalSymlinks(exe)
		exeDir = filepath.Dir(exe)
		candidate := filepath.Join(exeDir, "web")
		if hasIndex(candidate) {
			return candidate
		}
	}

	if wd, err := os.Getwd(); err == nil {
		candidate := filepath.Join(wd, "web")
		if hasIndex(candidate) {
			if exeDir != "" {
				log.Printf("webui: using %s (executable-adjacent web/ missing index.html; CWD fallback for go run / dev)", candidate)
			}
			return candidate
		}
	}

	if exeDir != "" {
		log.Printf("webui: no static root (expected %s with index.html); API-only. Build: cd frontend && npm run build, deploy web/ beside the binary.", filepath.Join(exeDir, "web"))
	} else {
		log.Printf("webui: no static root; set AISTOCK_WEB_DIR or place ./web with index.html")
	}
	return ""
}

// joinUnderRoot builds an absolute path under webRoot and rejects path traversal.
func joinUnderRoot(webRoot, rel string) (full string, ok bool) {
	webRoot = filepath.Clean(webRoot)
	cleanURL := path.Clean("/" + rel)
	if cleanURL == "/" || cleanURL == "." {
		return webRoot, true
	}
	rel = strings.TrimPrefix(cleanURL, "/")
	full = filepath.Join(webRoot, filepath.FromSlash(rel))
	rootAbs, err := filepath.Abs(webRoot)
	if err != nil {
		return "", false
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", false
	}
	sep := string(filepath.Separator)
	if fullAbs != rootAbs && !strings.HasPrefix(fullAbs, rootAbs+sep) {
		return "", false
	}
	return fullAbs, true
}

// Mount 掛載與可執行檔同目錄下 web/ 的前端靜態資源（Vite 產物）。
// AISTOCK_WEB_DIR 可覆蓋目錄；AISTOCK_SERVE_WEB=0 僅啟 API、不托管頁面。
func Mount(engine *gin.Engine) {
	if strings.TrimSpace(os.Getenv("AISTOCK_SERVE_WEB")) == "0" {
		return
	}
	webRoot := resolveWebRoot()
	if webRoot == "" {
		return
	}
	var err error
	webRoot, err = filepath.Abs(filepath.Clean(webRoot))
	if err != nil {
		log.Printf("webui: %v", err)
		return
	}
	log.Printf("webui: serving static files from %s", webRoot)

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
		rel = strings.Trim(rel, "/")

		if rel != "" {
			full, ok := joinUnderRoot(webRoot, rel)
			if !ok {
				c.Status(http.StatusNotFound)
				return
			}
			if fi, err := os.Stat(full); err == nil {
				if !fi.IsDir() {
					c.File(full)
					return
				}
				idx := filepath.Join(full, "index.html")
				if fi2, err := os.Stat(idx); err == nil && !fi2.IsDir() {
					c.File(idx)
					return
				}
			}
		}

		idx := filepath.Join(webRoot, "index.html")
		if _, err := os.Stat(idx); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.File(idx)
	})
}
