package services

import (
	"fmt"
	"time"

	"github.com/easyspace-ai/yilimi/internal/workbench/database"
	"github.com/easyspace-ai/yilimi/internal/workbench/utils"
)

type NotebookService struct {
	db *database.DB
}

func NewNotebookService(db *database.DB) *NotebookService {
	return &NotebookService{db: db}
}

// ListNotebooks 获取所有笔记本列表
func (s *NotebookService) ListNotebooks() ([]*database.Notebook, error) {
	query := `SELECT id, name, icon, sort, closed, created, updated 
			  FROM notebooks 
			  ORDER BY sort ASC, created ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询笔记本失败: %w", err)
	}
	defer rows.Close()

	var notebooks []*database.Notebook
	for rows.Next() {
		notebook := &database.Notebook{}
		var closed int
		var created, updated string

		err := rows.Scan(
			&notebook.ID,
			&notebook.Name,
			&notebook.Icon,
			&notebook.Sort,
			&closed,
			&created,
			&updated,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描笔记本数据失败: %w", err)
		}

		notebook.Closed = closed == 1
		notebook.Created, _ = time.Parse(time.RFC3339, created)
		notebook.Updated, _ = time.Parse(time.RFC3339, updated)

		notebooks = append(notebooks, notebook)
	}

	return notebooks, nil
}

// GetNotebook 根据ID获取笔记本
func (s *NotebookService) GetNotebook(id string) (*database.Notebook, error) {
	query := `SELECT id, name, icon, sort, closed, created, updated 
			  FROM notebooks 
			  WHERE id = ?`

	row := s.db.QueryRow(query, id)

	notebook := &database.Notebook{}
	var closed int
	var created, updated string

	err := row.Scan(
		&notebook.ID,
		&notebook.Name,
		&notebook.Icon,
		&notebook.Sort,
		&closed,
		&created,
		&updated,
	)
	if err != nil {
		return nil, fmt.Errorf("获取笔记本失败: %w", err)
	}

	notebook.Closed = closed == 1
	notebook.Created, _ = time.Parse(time.RFC3339, created)
	notebook.Updated, _ = time.Parse(time.RFC3339, updated)

	return notebook, nil
}

// CreateNotebook 创建新笔记本
func (s *NotebookService) CreateNotebook(name, icon string) (*database.Notebook, error) {
	// 生成ID
	id := utils.GenerateID()
	now := time.Now()

	// 获取下一个排序值
	var maxSort int
	err := s.db.QueryRow("SELECT COALESCE(MAX(sort), 0) FROM notebooks").Scan(&maxSort)
	if err != nil {
		return nil, fmt.Errorf("获取排序值失败: %w", err)
	}

	notebook := &database.Notebook{
		ID:      id,
		Name:    name,
		Icon:    icon,
		Sort:    maxSort + 1,
		Closed:  false,
		Created: now,
		Updated: now,
	}

	// 插入数据库
	query := `INSERT INTO notebooks (id, name, icon, sort, closed, created, updated) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = s.db.Exec(query,
		notebook.ID,
		notebook.Name,
		notebook.Icon,
		notebook.Sort,
		0, // false
		notebook.Created.Format(time.RFC3339),
		notebook.Updated.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("创建笔记本失败: %w", err)
	}

	return notebook, nil
}

// UpdateNotebook 更新笔记本
func (s *NotebookService) UpdateNotebook(id, name, icon string) (*database.Notebook, error) {
	now := time.Now()

	query := `UPDATE notebooks 
			  SET name = ?, icon = ?, updated = ? 
			  WHERE id = ?`

	result, err := s.db.Exec(query, name, icon, now.Format(time.RFC3339), id)
	if err != nil {
		return nil, fmt.Errorf("更新笔记本失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("笔记本不存在")
	}

	// 返回更新后的笔记本
	return s.GetNotebook(id)
}

// RenameNotebook 重命名笔记本
func (s *NotebookService) RenameNotebook(id, name string) (*database.Notebook, error) {
	now := time.Now()

	query := `UPDATE notebooks 
			  SET name = ?, updated = ? 
			  WHERE id = ?`

	result, err := s.db.Exec(query, name, now.Format(time.RFC3339), id)
	if err != nil {
		return nil, fmt.Errorf("重命名笔记本失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("笔记本不存在")
	}

	return s.GetNotebook(id)
}

// SetNotebookIcon 设置笔记本图标
func (s *NotebookService) SetNotebookIcon(id, icon string) (*database.Notebook, error) {
	now := time.Now()

	query := `UPDATE notebooks 
			  SET icon = ?, updated = ? 
			  WHERE id = ?`

	result, err := s.db.Exec(query, icon, now.Format(time.RFC3339), id)
	if err != nil {
		return nil, fmt.Errorf("设置笔记本图标失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("笔记本不存在")
	}

	return s.GetNotebook(id)
}

// OpenNotebook 打开笔记本
func (s *NotebookService) OpenNotebook(id string) error {
	query := `UPDATE notebooks 
			  SET closed = 0, updated = ? 
			  WHERE id = ?`

	result, err := s.db.Exec(query, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("打开笔记本失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("笔记本不存在")
	}

	return nil
}

// CloseNotebook 关闭笔记本
func (s *NotebookService) CloseNotebook(id string) error {
	query := `UPDATE notebooks 
			  SET closed = 1, updated = ? 
			  WHERE id = ?`

	result, err := s.db.Exec(query, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("关闭笔记本失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("笔记本不存在")
	}

	return nil
}

// DeleteNotebook 删除笔记本
func (s *NotebookService) DeleteNotebook(id string) error {
	tx, err := s.db.BeginTx()
	if err != nil {
		return fmt.Errorf("开始事务失败: %w", err)
	}
	defer tx.Rollback()

	// 删除笔记本下的所有块
	_, err = tx.Exec("DELETE FROM blocks WHERE box = ?", id)
	if err != nil {
		return fmt.Errorf("删除笔记本下的块失败: %w", err)
	}

	// 删除笔记本
	result, err := tx.Exec("DELETE FROM notebooks WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除笔记本失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("笔记本不存在")
	}

	return tx.Commit()
}

// ChangeSortNotebook 更改笔记本排序
func (s *NotebookService) ChangeSortNotebook(id string, sort int) error {
	query := `UPDATE notebooks 
			  SET sort = ?, updated = ? 
			  WHERE id = ?`

	result, err := s.db.Exec(query, sort, time.Now().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("更改笔记本排序失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("笔记本不存在")
	}

	return nil
}

// GetNotebookCount 获取笔记本数量
func (s *NotebookService) GetNotebookCount() (int, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM notebooks").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("获取笔记本数量失败: %w", err)
	}
	return count, nil
}

// IsNotebookExists 检查笔记本是否存在
func (s *NotebookService) IsNotebookExists(id string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM notebooks WHERE id = ?", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查笔记本是否存在失败: %w", err)
	}
	return count > 0, nil
}
