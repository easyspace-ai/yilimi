package tdxapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/easyspace-ai/yilimi/internal/appenv"
)

// activeService 由异步初始化写入；供合并进程内直接取 K 线/分时，无需 HTTP 回环。
var activeService atomic.Pointer[Service]

// ActiveService 返回已就绪的 TDX 服务；未初始化或失败时为 nil。
func ActiveService() *Service {
	return activeService.Load()
}

// NewLazyHandler 立即返回可挂载的 Handler，在后台 goroutine 中执行 NewService（拨号、代码表、连接池等），
// 避免阻塞进程其余启动路径。就绪前请求返回 503，便于客户端轮询或退避重试。
// dataRoot 为统一数据根（通常即 .env 中 AI_DATA_DIR）；空字符串则使用 appenv.DataRootDir()。
func NewLazyHandler(dataRoot string) http.Handler {
	if strings.TrimSpace(dataRoot) == "" {
		dataRoot = appenv.DataRootDir()
	}
	var mu sync.RWMutex

	starting := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(Response{
			Code:    -1,
			Message: "通达信服务正在初始化（代码表与连接池），请稍后重试",
			Data:    nil,
		})
	})

	var real http.Handler = starting
	initRoot := dataRoot

	go func() {
		start := time.Now()
		svc, err := NewService(initRoot)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			activeService.Store(nil)
			log.Printf("tdx: async init failed: %v", err)
			errMsg := err.Error()
			real = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(Response{
					Code:    -1,
					Message: "通达信服务不可用: " + errMsg,
					Data:    nil,
				})
			})
			return
		}
		activeService.Store(svc)
		real = svc
		log.Printf("tdx: async init finished in %s", time.Since(start).Round(time.Millisecond))
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		h := real
		mu.RUnlock()
		h.ServeHTTP(w, r)
	})
}
