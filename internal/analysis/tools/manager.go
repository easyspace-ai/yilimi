package tools

import (
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/easyspace-ai/stock_api/pkg/tsdb"
)

var (
	globalTools        []tool.BaseTool
	globalUnified      *tsdb.UnifiedClient
	globalToolsOnce    sync.Once
)

// InitGlobalTools 初始化全局工具集
func InitGlobalTools(dataDir string) error {
	var err error
	globalToolsOnce.Do(func() {
		stockTools, initErr := NewStockTools(dataDir)
		if initErr != nil {
			err = initErr
			return
		}
		globalTools = stockTools.GetAllTools()
		globalUnified = stockTools.UnifiedClient()
	})
	return err
}

// GetGlobalTools 获取全局工具集
func GetGlobalTools() []tool.BaseTool {
	return globalTools
}

// GlobalUnifiedClient 获取与数据工具共享的 tsdb 客户端（未调用 InitGlobalTools 时为 nil）。
func GlobalUnifiedClient() *tsdb.UnifiedClient {
	return globalUnified
}
