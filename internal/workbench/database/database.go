package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

var instance *DB

// InitDatabase 初始化数据库
func InitDatabase(dataDir string) (*DB, error) {
	if instance != nil {
		return instance, nil
	}

	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	// 数据库文件路径
	dbPath := filepath.Join(dataDir, "siyuan.db")

	// 打开数据库连接
	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	// 设置连接池参数
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(time.Hour)

	db := &DB{conn: conn}
	instance = db

	// 初始化数据库结构
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("初始化数据库结构失败: %w", err)
	}

	return db, nil
}

// GetInstance 获取数据库实例
func GetInstance() *DB {
	return instance
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// initSchema 初始化数据库结构
func (db *DB) initSchema() error {
	schemas := []string{
		// notebooks 表
		`CREATE TABLE IF NOT EXISTS notebooks (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			icon TEXT DEFAULT '',
			sort INTEGER DEFAULT 0,
			closed INTEGER DEFAULT 0,
			created TEXT NOT NULL,
			updated TEXT NOT NULL
		)`,

		// blocks 表
		`CREATE TABLE IF NOT EXISTS blocks (
			id TEXT PRIMARY KEY,
			parent_id TEXT,
			root_id TEXT NOT NULL,
			type TEXT NOT NULL,
			subtype TEXT DEFAULT '',
			content TEXT NOT NULL,
			markdown TEXT NOT NULL,
			path TEXT NOT NULL,
			hpath TEXT DEFAULT '',
			name TEXT DEFAULT '',
			alias TEXT DEFAULT '',
			memo TEXT DEFAULT '',
			tag TEXT DEFAULT '',
			ial TEXT DEFAULT '{}',
			sort INTEGER DEFAULT 0,
			box TEXT NOT NULL,
			created TEXT NOT NULL,
			updated TEXT NOT NULL,
			FOREIGN KEY (root_id) REFERENCES blocks(id) ON DELETE CASCADE
		)`,

		// attributes 表
		`CREATE TABLE IF NOT EXISTS attributes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			block_id TEXT NOT NULL,
			name TEXT NOT NULL,
			value TEXT NOT NULL,
			type TEXT DEFAULT 'text',
			created TEXT NOT NULL,
			updated TEXT NOT NULL,
			FOREIGN KEY (block_id) REFERENCES blocks(id) ON DELETE CASCADE,
			UNIQUE(block_id, name)
		)`,

		// refs 表
		`CREATE TABLE IF NOT EXISTS refs (
			id TEXT PRIMARY KEY,
			def_block_id TEXT NOT NULL,
			def_block_path TEXT NOT NULL,
			def_block_content TEXT,
			def_block_subtype TEXT,
			block_id TEXT NOT NULL,
			root_id TEXT NOT NULL,
			box TEXT NOT NULL,
			path TEXT NOT NULL,
			content TEXT NOT NULL,
			markdown TEXT NOT NULL,
			type TEXT NOT NULL,
			created TEXT NOT NULL,
			updated TEXT NOT NULL,
			FOREIGN KEY (def_block_id) REFERENCES blocks(id) ON DELETE CASCADE,
			FOREIGN KEY (block_id) REFERENCES blocks(id) ON DELETE CASCADE
		)`,

		// spans 表
		`CREATE TABLE IF NOT EXISTS spans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			block_id TEXT NOT NULL,
			root_id TEXT NOT NULL,
			box TEXT NOT NULL,
			path TEXT NOT NULL,
			content TEXT NOT NULL,
			markdown TEXT NOT NULL,
			type TEXT NOT NULL,
			ial TEXT DEFAULT '{}',
			created TEXT NOT NULL,
			updated TEXT NOT NULL,
			FOREIGN KEY (block_id) REFERENCES blocks(id) ON DELETE CASCADE
		)`,

		// file_annotation_refs 表
		`CREATE TABLE IF NOT EXISTS file_annotation_refs (
			id TEXT PRIMARY KEY,
			file_path TEXT NOT NULL,
			annotation_id TEXT NOT NULL,
			block_id TEXT NOT NULL,
			root_id TEXT NOT NULL,
			box TEXT NOT NULL,
			path TEXT NOT NULL,
			content TEXT NOT NULL,
			type TEXT NOT NULL,
			created TEXT NOT NULL,
			updated TEXT NOT NULL,
			FOREIGN KEY (block_id) REFERENCES blocks(id) ON DELETE CASCADE
		)`,
	}

	// 创建表
	for _, schema := range schemas {
		if _, err := db.conn.Exec(schema); err != nil {
			return fmt.Errorf("创建表失败: %w", err)
		}
	}

	// 创建索引
	indexes := []string{
		// blocks 表索引
		`CREATE INDEX IF NOT EXISTS idx_blocks_parent_id ON blocks(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_root_id ON blocks(root_id)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_type ON blocks(type)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_path ON blocks(path)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_box ON blocks(box)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_updated ON blocks(updated)`,
		`CREATE INDEX IF NOT EXISTS idx_blocks_created ON blocks(created)`,

		// refs 表索引
		`CREATE INDEX IF NOT EXISTS idx_refs_def_block_id ON refs(def_block_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refs_block_id ON refs(block_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refs_root_id ON refs(root_id)`,
		`CREATE INDEX IF NOT EXISTS idx_refs_box ON refs(box)`,

		// spans 表索引
		`CREATE INDEX IF NOT EXISTS idx_spans_block_id ON spans(block_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spans_root_id ON spans(root_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spans_box ON spans(box)`,
		`CREATE INDEX IF NOT EXISTS idx_spans_content ON spans(content)`,

		// attributes 表索引
		`CREATE INDEX IF NOT EXISTS idx_attributes_block_id ON attributes(block_id)`,
		`CREATE INDEX IF NOT EXISTS idx_attributes_name ON attributes(name)`,
	}

	for _, index := range indexes {
		if _, err := db.conn.Exec(index); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}

	return nil
}

// BeginTx 开始事务
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.conn.Begin()
}

// Exec 执行SQL语句
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// Query 查询
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

// QueryRow 查询单行
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

// Prepare 预编译SQL语句
func (db *DB) Prepare(query string) (*sql.Stmt, error) {
	return db.conn.Prepare(query)
}

// GetConn 获取底层数据库连接（用于高级操作）
func (db *DB) GetConn() *sql.DB {
	return db.conn
}

// Ping 测试数据库连接
func (db *DB) Ping() error {
	return db.conn.Ping()
}
