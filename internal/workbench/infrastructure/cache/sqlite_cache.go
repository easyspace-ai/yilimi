package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteCache SQLite 缓存
type SQLiteCache struct {
	db     *sql.DB
	dbPath string
}

// CachedData 缓存数据结构
type CachedData struct {
	Key        string
	Value      string
	Expiration int64
	CreatedAt  int64
}

// NewSQLiteCache 创建 SQLite 缓存
func NewSQLiteCache(dataDir string) (*SQLiteCache, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(dataDir, "cache.db")
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	c := &SQLiteCache{db: db, dbPath: dbPath}
	if err := c.initSchema(); err != nil {
		return nil, err
	}

	return c, nil
}

// initSchema 初始化数据库表
func (c *SQLiteCache) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS cache (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		expiration INTEGER NOT NULL,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_cache_expiration ON cache(expiration);
	`

	_, err := c.db.Exec(schema)
	return err
}

// Set 设置缓存
func (c *SQLiteCache) Set(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	expiration := time.Now().Add(ttl).Unix()
	createdAt := time.Now().Unix()

	query := `
	INSERT OR REPLACE INTO cache (key, value, expiration, created_at)
	VALUES (?, ?, ?, ?)
	`

	_, err = c.db.Exec(query, key, string(data), expiration, createdAt)
	return err
}

// Get 获取缓存
func (c *SQLiteCache) Get(key string, out interface{}) (bool, error) {
	query := `SELECT value, expiration FROM cache WHERE key = ?`

	var value string
	var expiration int64

	err := c.db.QueryRow(query, key).Scan(&value, &expiration)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	if time.Now().Unix() > expiration {
		go c.Delete(key)
		return false, nil
	}

	return true, json.Unmarshal([]byte(value), out)
}

// Delete 删除缓存
func (c *SQLiteCache) Delete(key string) error {
	_, err := c.db.Exec(`DELETE FROM cache WHERE key = ?`, key)
	return err
}

// Clear 清空缓存
func (c *SQLiteCache) Clear() error {
	_, err := c.db.Exec(`DELETE FROM cache`)
	return err
}

// Cleanup 清理过期缓存
func (c *SQLiteCache) Cleanup() (int64, error) {
	result, err := c.db.Exec(`DELETE FROM cache WHERE expiration < ?`, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Close 关闭数据库
func (c *SQLiteCache) Close() error {
	return c.db.Close()
}

// GetStats 获取缓存统计
func (c *SQLiteCache) GetStats() (total int64, expired int64, err error) {
	err = c.db.QueryRow(`SELECT COUNT(*) FROM cache`).Scan(&total)
	if err != nil {
		return
	}
	err = c.db.QueryRow(`SELECT COUNT(*) FROM cache WHERE expiration < ?`, time.Now().Unix()).Scan(&expired)
	return
}

// MultiLayerCache 多层缓存（内存 + SQLite）
type MultiLayerCache struct {
	memory *MemoryCache
	sqlite *SQLiteCache
}

// NewMultiLayerCache 创建多层缓存
func NewMultiLayerCache(dataDir string) (*MultiLayerCache, error) {
	sqliteCache, err := NewSQLiteCache(dataDir)
	if err != nil {
		return nil, err
	}

	return &MultiLayerCache{
		memory: NewMemoryCache(),
		sqlite: sqliteCache,
	}, nil
}

// Set 设置缓存
func (c *MultiLayerCache) Set(key string, value interface{}, ttl time.Duration) error {
	c.memory.Set(key, value, ttl)
	return c.sqlite.Set(key, value, ttl)
}

// Get 获取缓存
func (c *MultiLayerCache) Get(key string, out interface{}) bool {
	if val, ok := c.memory.Get(key); ok {
		if out != nil {
			outData, _ := json.Marshal(val)
			json.Unmarshal(outData, out)
		}
		return true
	}

	if ok, _ := c.sqlite.Get(key, out); ok {
		var val interface{} = out
		c.memory.Set(key, val, 5*time.Minute)
		return true
	}

	return false
}

// Delete 删除缓存
func (c *MultiLayerCache) Delete(key string) {
	c.memory.Delete(key)
	c.sqlite.Delete(key)
}

// Clear 清空缓存
func (c *MultiLayerCache) Clear() {
	c.memory.Clear()
	c.sqlite.Clear()
}

// Close 关闭
func (c *MultiLayerCache) Close() error {
	return c.sqlite.Close()
}

// GenerateKey 生成缓存键
func GenerateKey(prefix string, parts ...interface{}) string {
	key := prefix
	for _, part := range parts {
		key += fmt.Sprintf(":%v", part)
	}
	return key
}
