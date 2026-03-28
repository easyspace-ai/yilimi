package httpapi

import (
	"net/http"

	"github.com/easyspace-ai/yilimi/internal/workbench/database"
	"github.com/easyspace-ai/yilimi/internal/workbench/services"

	"github.com/gin-gonic/gin"
)

type NotebookAPI struct {
	notebookService *services.NotebookService
}

func NewNotebookAPI(notebookService *services.NotebookService) *NotebookAPI {
	return &NotebookAPI{
		notebookService: notebookService,
	}
}

// ListNotebooks 获取所有笔记本
// POST /api/notebook/lsNotebooks
func (api *NotebookAPI) ListNotebooks(c *gin.Context) {
	notebooks, err := api.notebookService.ListNotebooks()
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"notebooks": notebooks,
		},
	})
}

// CreateNotebook 创建笔记本
// POST /api/notebook/createNotebook
func (api *NotebookAPI) CreateNotebook(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Icon string `json:"icon"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	// 设置默认图标
	if req.Icon == "" {
		req.Icon = "📔"
	}

	notebook, err := api.notebookService.CreateNotebook(req.Name, req.Icon)
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"notebook": notebook,
		},
	})
}

// RenameNotebook 重命名笔记本
// POST /api/notebook/renameNotebook
func (api *NotebookAPI) RenameNotebook(c *gin.Context) {
	var req struct {
		Notebook string `json:"notebook" binding:"required"`
		Name     string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	notebook, err := api.notebookService.RenameNotebook(req.Notebook, req.Name)
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"notebook": notebook,
		},
	})
}

// SetNotebookIcon 设置笔记本图标
// POST /api/notebook/setNotebookIcon
func (api *NotebookAPI) SetNotebookIcon(c *gin.Context) {
	var req struct {
		Notebook string `json:"notebook" binding:"required"`
		Icon     string `json:"icon" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	notebook, err := api.notebookService.SetNotebookIcon(req.Notebook, req.Icon)
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"notebook": notebook,
		},
	})
}

// OpenNotebook 打开笔记本
// POST /api/notebook/openNotebook
func (api *NotebookAPI) OpenNotebook(c *gin.Context) {
	var req struct {
		Notebook string `json:"notebook" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	err := api.notebookService.OpenNotebook(req.Notebook)
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
	})
}

// CloseNotebook 关闭笔记本
// POST /api/notebook/closeNotebook
func (api *NotebookAPI) CloseNotebook(c *gin.Context) {
	var req struct {
		Notebook string `json:"notebook" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	err := api.notebookService.CloseNotebook(req.Notebook)
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
	})
}

// RemoveNotebook 删除笔记本
// POST /api/notebook/removeNotebook
func (api *NotebookAPI) RemoveNotebook(c *gin.Context) {
	var req struct {
		Notebook string `json:"notebook" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	err := api.notebookService.DeleteNotebook(req.Notebook)
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
	})
}

// ChangeSortNotebook 更改笔记本排序
// POST /api/notebook/changeSortNotebook
func (api *NotebookAPI) ChangeSortNotebook(c *gin.Context) {
	var req struct {
		Notebook string `json:"notebook" binding:"required"`
		Sort     int    `json:"sort" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	err := api.notebookService.ChangeSortNotebook(req.Notebook, req.Sort)
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
	})
}

// GetNotebookInfo 获取笔记本信息
// POST /api/notebook/getNotebookInfo
func (api *NotebookAPI) GetNotebookInfo(c *gin.Context) {
	var req struct {
		Notebook string `json:"notebook" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  "参数错误: " + err.Error(),
		})
		return
	}

	notebook, err := api.notebookService.GetNotebook(req.Notebook)
	if err != nil {
		c.JSON(http.StatusOK, database.APIResponse{
			Code: -1,
			Msg:  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, database.APIResponse{
		Code: 0,
		Msg:  "success",
		Data: map[string]interface{}{
			"notebook": notebook,
		},
	})
}
