package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// DailyAPIReverseProxyHandlerWithPrefix strips the given prefix and forwards to the DailyAPI (FastAPI) service.
func DailyAPIReverseProxyHandlerWithPrefix(stripPrefix string) gin.HandlerFunc {
	return dailyAPIProxy(stripPrefix)
}

func dailyAPIBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("DAILYAPI_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if p := strings.TrimSpace(os.Getenv("DAILYAPI_PORT")); p != "" {
		host := strings.TrimSpace(os.Getenv("DAILYAPI_HOST"))
		if host == "" {
			host = "127.0.0.1"
		}
		return fmt.Sprintf("http://%s:%s", host, p)
	}
	// 与 ecosystem.config.js / backend/.env 默认一致
	return "http://127.0.0.1:7220"
}

func dailyAPIProxy(stripPrefix string) gin.HandlerFunc {
	target, err := url.Parse(dailyAPIBaseURL())
	if err != nil {
		panic(fmt.Sprintf("invalid DAILYAPI_URL: %v", err))
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		req.URL.RawPath = req.URL.Path
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Expose-Headers")
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"detail": fmt.Sprintf("dailyapi proxy error: %v", err),
		})
	}

	return gin.WrapH(proxy)
}
