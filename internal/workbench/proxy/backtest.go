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

// BacktestReverseProxyHandlerWithPrefix 将指定前缀从路径上剥掉后代理到回测服务（使用 /api/v1/backtest）。
func BacktestReverseProxyHandlerWithPrefix(stripPrefix string) gin.HandlerFunc {
	return backtestProxy(stripPrefix)
}

func backtestAPIBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("BACKTESTAPI_URL")); v != "" {
		return strings.TrimRight(v, "/")
	}
	if p := strings.TrimSpace(os.Getenv("BACKTESTAPI_PORT")); p != "" {
		host := strings.TrimSpace(os.Getenv("BACKTESTAPI_HOST"))
		if host == "" {
			host = "127.0.0.1"
		}
		return fmt.Sprintf("http://%s:%s", host, p)
	}
	// 与 backtestapi 本地开发默认一致；生产/PM2 请在 .env 中设置 BACKTESTAPI_PORT 或 BACKTESTAPI_URL。
	return "http://127.0.0.1:8001"
}

func backtestProxy(stripPrefix string) gin.HandlerFunc {
	target, err := url.Parse(backtestAPIBaseURL())
	if err != nil {
		panic(fmt.Sprintf("invalid BACKTESTAPI_URL: %v", err))
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
			"detail": fmt.Sprintf("backtest proxy error: %v", err),
		})
	}

	return gin.WrapH(proxy)
}
