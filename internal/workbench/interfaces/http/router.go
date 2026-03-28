package httpapi

import (
	"github.com/gin-gonic/gin"
)

// RegisterAllRoutes 注册所有 API 路由
func RegisterAllRoutes(r *gin.RouterGroup) {
	// 健康检查（独立）
	r.GET("/health", Health)
	r.GET("/vip-status", GetVipStatus)

	// 注意：StockHandler、MarketHandler、AIHandler 需要依赖注入
	// 这里只定义路由结构，具体注册在 main.go 中进行
}

// RouteConfig 路由配置
type RouteConfig struct {
	StockHandler  *StockHandler
	MarketHandler *MarketHandler
	AIHandler     *AIHandler
}

// RegisterStockRoutes 注册股票路由
func (cfg *RouteConfig) RegisterStockRoutes(r *gin.RouterGroup) {
	if cfg.StockHandler != nil {
		cfg.StockHandler.RegisterRoutes(r)
	}
}

// RegisterMarketRoutes 注册市场路由
func (cfg *RouteConfig) RegisterMarketRoutes(r *gin.RouterGroup) {
	if cfg.MarketHandler != nil {
		cfg.MarketHandler.RegisterRoutes(r)
	}
}

// RegisterAIRoutes 注册 AI 路由
func (cfg *RouteConfig) RegisterAIRoutes(r *gin.RouterGroup) {
	if cfg.AIHandler != nil {
		cfg.AIHandler.RegisterRoutes(r)
	}
}
