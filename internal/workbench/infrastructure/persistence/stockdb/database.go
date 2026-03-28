package stockdb

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/easyspace-ai/yilimi/internal/workbench/domain/ai"
	"github.com/easyspace-ai/yilimi/internal/workbench/domain/market"
	"github.com/easyspace-ai/yilimi/internal/workbench/domain/stock"
)

var DB *gorm.DB

// InitStockDatabase 初始化股票数据库
func InitStockDatabase(dataDir string) (*gorm.DB, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	// 数据库文件路径
	dbPath := filepath.Join(dataDir, "stock.db")
	dsn := dbPath + "?_busy_timeout=10000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-524288"

	// 配置日志
	dbLogger := logger.New(
		log.New(os.Stdout, "[stock-db] ", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second * 3,
			Colorful:                  false,
			IgnoreRecordNotFoundError: true,
			ParameterizedQueries:      false,
			LogLevel:                  logger.Info,
		},
	)

	// 打开数据库
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger:                                   dbLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		PrepareStmt:                              true,
	})
	if err != nil {
		return nil, err
	}

	// 确保 PRAGMA 设置生效
	_ = db.Exec("PRAGMA busy_timeout=10000").Error
	_ = db.Exec("PRAGMA journal_mode=WAL").Error
	_ = db.Exec("PRAGMA synchronous=NORMAL").Error

	// 自动迁移数据库表
	if err := autoMigrate(db); err != nil {
		return nil, err
	}

	// 配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	return db, nil
}

// autoMigrate 自动迁移数据表
func autoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		// Stock 领域模型
		&stock.StockInfo{},
		&stock.FollowedStock{},
		&stock.StockAlarm{},
		&stock.AllStockInfo{},
		&stock.StockGroup{},
		&stock.StockGroupItem{},
		&stock.DailyKLineCache{},

		// Market 领域模型
		&market.LongTigerRank{},
		&market.MarketNews{},
		&market.Telegraph{},
		&market.ResearchReport{},
		&market.StockNotice{},
		&market.BKDict{},
		&market.InteractiveAnswer{},

		// AI 领域模型
		&ai.PromptTemplate{},
		&ai.AiAssistantSession{},
		&ai.AIResponseResult{},
		&ai.AiRecommendStocks{},
		&ai.CronTask{},
		&ai.Settings{},
		&ai.VersionInfo{},
	)
}

// GetStockDB 获取股票数据库实例
func GetStockDB() *gorm.DB {
	return DB
}
