package tools

import (
	"sync"

	"github.com/cloudwego/eino/components/tool"
)

var (
	globalTools     []tool.BaseTool
	globalToolsOnce sync.Once
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
	})
	return err
}

// GetGlobalTools 获取全局工具集
func GetGlobalTools() []tool.BaseTool {
	return globalTools
}
