package plugins

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	db         *Database
	pluginsDir string
	registries map[string]*PluginRegistry
	runtime    *Runtime
}

func NewService(dbPath string, pluginsDir string) (*Service, error) {
	db, err := NewDatabase(dbPath)
	if err != nil {
		return nil, err
	}

	// Ensure plugins directory exists
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return nil, err
	}

	service := &Service{
		db:         db,
		pluginsDir: pluginsDir,
		registries: make(map[string]*PluginRegistry),
		runtime:    NewRuntime(),
	}

	// Load registries
	err = service.loadRegistries()
	if err != nil {
		return nil, err
	}

	// Load installed plugins
	err = service.loadInstalledPlugins()
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *Service) loadRegistries() error {
	registries, err := s.db.ListRegistries()
	if err != nil {
		return err
	}

	for _, registry := range registries {
		s.registries[registry.ID] = &registry
	}
	return nil
}

func (s *Service) loadInstalledPlugins() error {
	plugins, err := s.db.ListInstalledPlugins()
	if err != nil {
		return err
	}

	for _, plugin := range plugins {
		if plugin.Enabled {
			err := s.runtime.LoadPlugin(&plugin)
			if err != nil {
				// Log error but don't fail startup
				fmt.Printf("Failed to load plugin %s: %v\n", plugin.ID, err)
			}
		}
	}
	return nil
}

// Plugin management methods
func (s *Service) ListPlugins() ([]Plugin, error) {
	return s.db.ListPlugins()
}

func (s *Service) GetPlugin(id string) (*Plugin, error) {
	return s.db.GetPlugin(id)
}

func (s *Service) InstallPlugin(pluginURL string) (*Plugin, error) {
	// Download plugin archive
	resp, err := http.Get(pluginURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download plugin: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download plugin: status %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "plugin-*.zip")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy response to temporary file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return nil, err
	}

	// Extract and install
	return s.installFromZip(tmpFile.Name())
}

func (s *Service) InstallPluginFromFile(zipPath string) (*Plugin, error) {
	return s.installFromZip(zipPath)
}

func (s *Service) installFromZip(zipPath string) (*Plugin, error) {
	// Open zip file
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	fmt.Println("r.File", r.File)

	// Find and read manifest
	var manifestFile *zip.File
	for _, f := range r.File {
		fmt.Println("f.Name", f.Name)

		if f.Name == "manifest.json" || f.Name == "plugin.json" {
			manifestFile = f
			break
		}
	}

	if manifestFile == nil {
		return nil, fmt.Errorf("no manifest.json found in plugin archive")
	}

	// Read manifest
	rc, err := manifestFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	manifestData, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	var manifest PluginManifest
	err = json.Unmarshal(manifestData, &manifest)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest.json: %v", err)
	}

	// Validate manifest
	if manifest.ID == "" || manifest.Name == "" || manifest.Version == "" {
		return nil, fmt.Errorf("manifest missing required fields (id, name, version)")
	}

	// Check if plugin already exists
	existingPlugin, _ := s.db.GetPlugin(manifest.ID)
	if existingPlugin != nil {
		return nil, fmt.Errorf("plugin %s is already installed", manifest.ID)
	}

	// Create plugin directory
	pluginDir := filepath.Join(s.pluginsDir, manifest.ID)
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, err
	}

	// Extract files
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// Create directory structure
		destPath := filepath.Join(pluginDir, f.Name)
		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return nil, err
		}

		// Extract file
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			rc.Close()
			return nil, err
		}

		_, err = io.Copy(destFile, rc)
		rc.Close()
		destFile.Close()

		if err != nil {
			return nil, err
		}
	}

	// Create plugin record
	plugin := &Plugin{
		ID:          manifest.ID,
		Name:        manifest.Name,
		Description: manifest.Description,
		Version:     manifest.Version,
		Author:      manifest.Author,
		Homepage:    manifest.Homepage,
		Repository:  manifest.Repository,
		License:     manifest.License,
		MainFile:    manifest.MainFile,
		Installed:   true,
		Enabled:     false,
		InstallPath: pluginDir,
	}

	plugin.SetTagsArray(manifest.Tags)
	plugin.SetAssetFilesArray(manifest.AssetFiles)
	plugin.SetManifest(&manifest)

	err = s.db.CreatePlugin(plugin)
	if err != nil {
		// Cleanup on error
		os.RemoveAll(pluginDir)
		return nil, err
	}

	return plugin, nil
}

func (s *Service) UninstallPlugin(id string) error {
	plugin, err := s.db.GetPlugin(id)
	if err != nil {
		return err
	}

	if !plugin.Installed {
		return fmt.Errorf("plugin %s is not installed", id)
	}

	// Disable plugin first
	if plugin.Enabled {
		err = s.DisablePlugin(id)
		if err != nil {
			return err
		}
	}

	// Remove plugin directory
	if plugin.InstallPath != "" {
		err = os.RemoveAll(plugin.InstallPath)
		if err != nil {
			return err
		}
	}

	// Remove from database
	err = s.db.DeletePlugin(id)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) EnablePlugin(id string) error {
	plugin, err := s.db.GetPlugin(id)
	if err != nil {
		return err
	}

	if !plugin.Installed {
		return fmt.Errorf("plugin %s is not installed", id)
	}

	if plugin.Enabled {
		return nil // Already enabled
	}

	// Load plugin in runtime
	err = s.runtime.LoadPlugin(plugin)
	if err != nil {
		return err
	}

	// Update database
	plugin.Enabled = true
	err = s.db.UpdatePlugin(plugin)
	if err != nil {
		// Unload on error
		s.runtime.UnloadPlugin(id)
		return err
	}

	return nil
}

func (s *Service) DisablePlugin(id string) error {
	plugin, err := s.db.GetPlugin(id)
	if err != nil {
		return err
	}

	if !plugin.Enabled {
		return nil // Already disabled
	}

	// Unload from runtime
	s.runtime.UnloadPlugin(id)

	// Update database
	plugin.Enabled = false
	err = s.db.UpdatePlugin(plugin)
	if err != nil {
		return err
	}

	return nil
}

// Registry management
func (s *Service) AddRegistry(name, url, description string) error {
	registry := &PluginRegistry{
		ID:          uuid.New().String(),
		Name:        name,
		URL:         url,
		Description: description,
		Enabled:     true,
	}

	err := s.db.CreateRegistry(registry)
	if err != nil {
		return err
	}

	s.registries[registry.ID] = registry
	return nil
}

func (s *Service) ListRegistries() ([]PluginRegistry, error) {
	return s.db.ListRegistries()
}

func (s *Service) SearchPlugins(query string) ([]Plugin, error) {
	// For now, search in local database
	// TODO: Implement remote registry search
	plugins, err := s.db.ListPlugins()
	if err != nil {
		return nil, err
	}

	if query == "" {
		return plugins, nil
	}

	var filtered []Plugin
	query = strings.ToLower(query)
	for _, plugin := range plugins {
		if strings.Contains(strings.ToLower(plugin.Name), query) ||
			strings.Contains(strings.ToLower(plugin.Description), query) {
			filtered = append(filtered, plugin)
		}
	}

	return filtered, nil
}

// Plugin configuration
func (s *Service) SetPluginConfig(pluginID string, config map[string]interface{}) error {
	return s.db.SetPluginConfig(pluginID, config)
}

func (s *Service) GetPluginConfig(pluginID string) (map[string]interface{}, error) {
	return s.db.GetPluginConfig(pluginID)
}

// Runtime access
func (s *Service) GetRuntime() *Runtime {
	return s.runtime
}

func (s *Service) Close() error {
	s.runtime.Stop()
	return s.db.Close()
}
