package plugins

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterPluginRoutes registers all plugin-related API routes
func RegisterPluginRoutes(r *gin.RouterGroup, service *Service) {
	plugins := r.Group("/plugins")
	{
		// Plugin management
		plugins.GET("", listPlugins(service))
		plugins.GET("/:id", getPlugin(service))
		plugins.POST("/install", installPlugin(service))
		plugins.POST("/install-file", installPluginFromFile(service))
		plugins.DELETE("/:id", uninstallPlugin(service))
		plugins.PUT("/:id/enable", enablePlugin(service))
		plugins.PUT("/:id/disable", disablePlugin(service))
		plugins.GET("/search", searchPlugins(service))

		// Plugin configuration
		plugins.GET("/:id/config", getPluginConfig(service))
		plugins.PUT("/:id/config", setPluginConfig(service))

		// Plugin registries
		plugins.GET("/registries", listRegistries(service))
		plugins.POST("/registries", addRegistry(service))

		// Plugin runtime
		plugins.GET("/runtime/menus", getRuntimeMenus(service))
		plugins.GET("/runtime/panels", getRuntimePanels(service))
		plugins.POST("/runtime/commands/:id/execute", executeCommand(service))
		plugins.GET("/runtime/hooks/:name/execute", executeHook(service))
	}
}

// Plugin management handlers
func listPlugins(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		installedOnly := c.Query("installed") == "true"
		enabledOnly := c.Query("enabled") == "true"

		var plugins []Plugin
		var err error

		if enabledOnly {
			plugins, err = service.db.ListEnabledPlugins()
		} else if installedOnly {
			plugins, err = service.db.ListInstalledPlugins()
		} else {
			plugins, err = service.ListPlugins()
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"plugins": plugins})
	}
}

func getPlugin(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		plugin, err := service.GetPlugin(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"plugin": plugin})
	}
}

func installPlugin(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			URL string `json:"url" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		plugin, err := service.InstallPlugin(req.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Plugin installed successfully",
			"plugin":  plugin,
		})
	}
}

func installPluginFromFile(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		file, err := c.FormFile("plugin")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No plugin file provided"})
			return
		}

		// Save uploaded file temporarily
		tempPath := "/tmp/" + file.Filename
		if err := c.SaveUploadedFile(file, tempPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save uploaded file"})
			return
		}

		plugin, err := service.InstallPluginFromFile(tempPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Plugin installed successfully",
			"plugin":  plugin,
		})
	}
}

func uninstallPlugin(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		err := service.UninstallPlugin(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Plugin uninstalled successfully"})
	}
}

func enablePlugin(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		err := service.EnablePlugin(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Plugin enabled successfully"})
	}
}

func disablePlugin(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		err := service.DisablePlugin(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Plugin disabled successfully"})
	}
}

func searchPlugins(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		plugins, err := service.SearchPlugins(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"plugins": plugins})
	}
}

// Configuration handlers
func getPluginConfig(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		config, err := service.GetPluginConfig(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"config": config})
	}
}

func setPluginConfig(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var config map[string]interface{}

		if err := c.ShouldBindJSON(&config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := service.SetPluginConfig(id, config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
	}
}

// Registry handlers
func listRegistries(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		registries, err := service.ListRegistries()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"registries": registries})
	}
}

func addRegistry(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name        string `json:"name" binding:"required"`
			URL         string `json:"url" binding:"required"`
			Description string `json:"description"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err := service.AddRegistry(req.Name, req.URL, req.Description)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Registry added successfully"})
	}
}

// Runtime handlers
func getRuntimeMenus(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		menus := service.GetRuntime().GetMenus()
		c.JSON(http.StatusOK, gin.H{"menus": menus})
	}
}

func getRuntimePanels(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		panels := service.GetRuntime().GetPanels()
		c.JSON(http.StatusOK, gin.H{"panels": panels})
	}
}

func executeCommand(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		commandID := c.Param("id")
		var req struct {
			Args []string `json:"args"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := service.GetRuntime().ExecuteCommand(commandID, req.Args)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"result": result})
	}
}

func executeHook(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		hookName := c.Param("name")
		var data interface{}

		if err := c.ShouldBindJSON(&data); err != nil {
			// If no JSON body, use query parameters as data
			data = c.Request.URL.Query()
		}

		results := service.GetRuntime().ExecuteHook(hookName, data)
		c.JSON(http.StatusOK, gin.H{"results": results})
	}
}
