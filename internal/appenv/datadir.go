package appenv

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// defaultAI_DATA_DIR 当 .env 未配置 AI_DATA_DIR 时使用（并已写入环境变量）。
const defaultAI_DATA_DIR = "./data"

// Init 在进程入口调用一次：加载 .env，并保证 AI_DATA_DIR 在环境中非空且为规范化路径。
func Init() {
	LoadDotenv()
	EnsureUnifiedDataDir()
}

// LoadDotenv 从当前工作目录加载 .env、backend/.env（与仓库根 / backend 两种启动方式兼容）。
func LoadDotenv() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("backend/.env")
	// 从 backend/ / backend/cmd/datainit 等子目录启动时，向上找到仓库侧配置
	_ = godotenv.Load("../.env")
}

// EnsureUnifiedDataDir 若 AI_DATA_DIR 未设置则设为 defaultAI_DATA_DIR，并始终将规范化路径写回环境变量。
// 可重复调用；与 Init 拆开便于仅需保证目录、不加载 .env 的测试（一般仍应使用 Init）。
func EnsureUnifiedDataDir() {
	v := strings.TrimSpace(os.Getenv("AI_DATA_DIR"))
	if v == "" {
		v = filepath.Clean(defaultAI_DATA_DIR)
		log.Printf("[appenv] AI_DATA_DIR unset; using default %s (set in .env to override)", v)
	} else {
		v = filepath.Clean(v)
	}
	_ = os.Setenv("AI_DATA_DIR", v)
}

// DataRootDir 统一数据根目录（环境变量 AI_DATA_DIR）。
// 启动时须先调用 Init；若未调用且变量仍为空，则回退 defaultAI_DATA_DIR（仅此路径下可能未写回 os.Environ）。
func DataRootDir() string {
	v := strings.TrimSpace(os.Getenv("AI_DATA_DIR"))
	if v == "" {
		return filepath.Clean(defaultAI_DATA_DIR)
	}
	return filepath.Clean(v)
}

// StockDatabaseDir 与 DataRootDir 相同；stock.db 位于该目录下。
func StockDatabaseDir() string {
	return DataRootDir()
}

// WorkspaceRoot 文档工作区根路径（如 workspace/data、.plugins），位于 ${AI_DATA_DIR}/workspace。
func WorkspaceRoot() string {
	return filepath.Clean(filepath.Join(DataRootDir(), "workspace"))
}
