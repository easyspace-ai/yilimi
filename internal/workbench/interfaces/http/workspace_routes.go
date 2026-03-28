package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/easyspace-ai/yilimi/internal/workbench/filesystem"
	"github.com/easyspace-ai/yilimi/internal/workbench/tags"
	"github.com/easyspace-ai/yilimi/internal/workbench/ws"

	"github.com/gin-gonic/gin"
)

type filePayload struct {
	Path    string `json:"path"` // "/folder/file.md"
	Content string `json:"content"`
}

type movePayload struct {
	From string `json:"from"`
	To   string `json:"to"`
}

func RegisterWorkspaceRoutes(r *gin.RouterGroup, fsSvc *filesystem.Service, hub *ws.Hub, tagIndexer *tags.Indexer, root string) {
	r.GET("/tree", func(c *gin.Context) {
		p := c.Query("path")
		node, err := fsSvc.ListTree(p, 8)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, node)
	})

	r.GET("/file", func(c *gin.Context) {
		p := c.Query("path")
		content, err := fsSvc.ReadFile(p)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"path": p, "content": content})
	})

	r.POST("/file", func(c *gin.Context) {
		var req filePayload
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := fsSvc.CreateFile(req.Path, req.Content); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Update tags indexer
		if tagIndexer != nil {
			abs, _ := filepath.Abs(filepath.Join(root, strings.TrimPrefix(filepath.FromSlash(req.Path), "/")))
			tagIndexer.OnFsEvent("created", abs)
		}
		hub.Broadcast(ws.Event{Type: "fs", Action: "created", Path: req.Path})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.PUT("/file", func(c *gin.Context) {
		var req filePayload
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := fsSvc.WriteFile(req.Path, req.Content); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if tagIndexer != nil {
			abs, _ := filepath.Abs(filepath.Join(root, strings.TrimPrefix(filepath.FromSlash(req.Path), "/")))
			tagIndexer.OnFsEvent("modified", abs)
		}
		hub.Broadcast(ws.Event{Type: "fs", Action: "modified", Path: req.Path})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.POST("/folder", func(c *gin.Context) {
		var req struct {
			Path string `json:"path"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := fsSvc.CreateFolder(req.Path); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hub.Broadcast(ws.Event{Type: "fs", Action: "created", Path: req.Path})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.DELETE("/path", func(c *gin.Context) {
		p := c.Query("path")
		if err := fsSvc.DeletePath(p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if tagIndexer != nil {
			abs, _ := filepath.Abs(filepath.Join(root, strings.TrimPrefix(filepath.FromSlash(p), "/")))
			tagIndexer.OnFsEvent("deleted", abs)
		}
		hub.Broadcast(ws.Event{Type: "fs", Action: "deleted", Path: p})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.POST("/move", func(c *gin.Context) {
		var req movePayload
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err := fsSvc.RenamePath(req.From, req.To); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if tagIndexer != nil {
			absFrom, _ := filepath.Abs(filepath.Join(root, strings.TrimPrefix(filepath.FromSlash(req.From), "/")))
			absTo, _ := filepath.Abs(filepath.Join(root, strings.TrimPrefix(filepath.FromSlash(req.To), "/")))
			// Treat as delete+create to preserve counts under new path
			tagIndexer.OnFsEvent("deleted", absFrom)
			tagIndexer.OnFsEvent("created", absTo)
		}
		hub.Broadcast(ws.Event{Type: "fs", Action: "renamed", Path: req.To, From: req.From, To: req.To})
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// Tags endpoints
	r.GET("/tags", func(c *gin.Context) {
		if tagIndexer == nil {
			c.JSON(http.StatusOK, []any{})
			return
		}
		c.JSON(http.StatusOK, tagIndexer.ListTags())
	})

	r.GET("/tags/:name", func(c *gin.Context) {
		if tagIndexer == nil {
			c.JSON(http.StatusOK, []any{})
			return
		}
		name := c.Param("name")
		c.JSON(http.StatusOK, tagIndexer.FilesForTag(name))
	})

	r.GET("/file/tags", func(c *gin.Context) {
		if tagIndexer == nil {
			c.JSON(http.StatusOK, []string{})
			return
		}
		p := c.Query("path")
		c.JSON(http.StatusOK, tagIndexer.TagsForFile(p))
	})
}
