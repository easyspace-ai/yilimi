package plugins

import (
	"encoding/json"
	"time"
)

// Plugin represents a plugin in the system
type Plugin struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Version     string    `json:"version" db:"version"`
	Author      string    `json:"author" db:"author"`
	Homepage    string    `json:"homepage" db:"homepage"`
	Repository  string    `json:"repository" db:"repository"`
	License     string    `json:"license" db:"license"`
	Tags        string    `json:"tags" db:"tags"` // JSON array as string
	MainFile    string    `json:"main_file" db:"main_file"`
	AssetFiles  string    `json:"asset_files" db:"asset_files"` // JSON array as string
	Manifest    string    `json:"manifest" db:"manifest"`       // Full manifest JSON
	Installed   bool      `json:"installed" db:"installed"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	InstallPath string    `json:"install_path" db:"install_path"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PluginManifest represents the plugin manifest structure
type PluginManifest struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Version      string                 `json:"version"`
	Author       string                 `json:"author"`
	Homepage     string                 `json:"homepage,omitempty"`
	Repository   string                 `json:"repository,omitempty"`
	License      string                 `json:"license,omitempty"`
	Tags         []string               `json:"tags,omitempty"`
	MainFile     string                 `json:"main"`
	AssetFiles   []string               `json:"assets,omitempty"`
	MinVersion   string                 `json:"minAppVersion,omitempty"`
	Dependencies map[string]string      `json:"dependencies,omitempty"`
	Permissions  []string               `json:"permissions,omitempty"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Hooks        map[string]interface{} `json:"hooks,omitempty"`
	Commands     []PluginCommand        `json:"commands,omitempty"`
	Menus        []PluginMenu           `json:"menus,omitempty"`
	Panels       []PluginPanel          `json:"panels,omitempty"`
}

// PluginCommand represents a command that can be registered by a plugin
type PluginCommand struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Callback string `json:"callback"`
	Hotkey   string `json:"hotkey,omitempty"`
}

// PluginMenu represents a menu item that can be added by a plugin
type PluginMenu struct {
	ID       string       `json:"id"`
	Label    string       `json:"label"`
	Icon     string       `json:"icon,omitempty"`
	Parent   string       `json:"parent,omitempty"`
	Position string       `json:"position,omitempty"`
	Action   string       `json:"action"`
	Children []PluginMenu `json:"children,omitempty"`
}

// PluginRegistry represents a plugin registry/repository
type PluginRegistry struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	URL         string    `json:"url" db:"url"`
	Description string    `json:"description" db:"description"`
	Enabled     bool      `json:"enabled" db:"enabled"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// PluginEvent represents plugin lifecycle events
type PluginEvent struct {
	Type      string      `json:"type"`
	PluginID  string      `json:"plugin_id"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// PluginPanel represents a plugin panel
type PluginPanel struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Icon     string `json:"icon,omitempty"`
	Position string `json:"position"` // left, right, bottom
	Content  string `json:"content"`  // HTML content or component reference
}

// PluginConfig represents plugin configuration
type PluginConfig struct {
	PluginID  string                 `json:"plugin_id" db:"plugin_id"`
	Config    map[string]interface{} `json:"config"`
	UpdatedAt time.Time              `json:"updated_at" db:"updated_at"`
}

// GetTagsArray converts tags JSON string to array
func (p *Plugin) GetTagsArray() []string {
	if p.Tags == "" {
		return []string{}
	}
	var tags []string
	json.Unmarshal([]byte(p.Tags), &tags)
	return tags
}

// SetTagsArray converts array to JSON string
func (p *Plugin) SetTagsArray(tags []string) {
	if tags == nil {
		p.Tags = "[]"
		return
	}
	data, _ := json.Marshal(tags)
	p.Tags = string(data)
}

// GetAssetFilesArray converts asset files JSON string to array
func (p *Plugin) GetAssetFilesArray() []string {
	if p.AssetFiles == "" {
		return []string{}
	}
	var files []string
	json.Unmarshal([]byte(p.AssetFiles), &files)
	return files
}

// SetAssetFilesArray converts array to JSON string
func (p *Plugin) SetAssetFilesArray(files []string) {
	if files == nil {
		p.AssetFiles = "[]"
		return
	}
	data, _ := json.Marshal(files)
	p.AssetFiles = string(data)
}

// GetManifest returns parsed manifest
func (p *Plugin) GetManifest() (*PluginManifest, error) {
	var manifest PluginManifest
	err := json.Unmarshal([]byte(p.Manifest), &manifest)
	return &manifest, err
}

// SetManifest sets manifest from struct
func (p *Plugin) SetManifest(manifest *PluginManifest) error {
	data, err := json.Marshal(manifest)
	if err != nil {
		return err
	}
	p.Manifest = string(data)
	return nil
}
