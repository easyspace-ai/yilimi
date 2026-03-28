package plugins

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	database := &Database{db: db}
	err = database.createTables()
	if err != nil {
		return nil, err
	}

	return database, nil
}

func (d *Database) createTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS plugins (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			version TEXT NOT NULL,
			author TEXT NOT NULL,
			homepage TEXT,
			repository TEXT,
			license TEXT,
			tags TEXT DEFAULT '[]',
			main_file TEXT NOT NULL,
			asset_files TEXT DEFAULT '[]',
			manifest TEXT NOT NULL,
			installed BOOLEAN DEFAULT FALSE,
			enabled BOOLEAN DEFAULT FALSE,
			install_path TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS plugin_registries (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			url TEXT NOT NULL UNIQUE,
			description TEXT,
			enabled BOOLEAN DEFAULT TRUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS plugin_configs (
			plugin_id TEXT PRIMARY KEY,
			config TEXT NOT NULL DEFAULT '{}',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_plugins_enabled ON plugins(enabled)`,
		`CREATE INDEX IF NOT EXISTS idx_plugins_installed ON plugins(installed)`,
		`CREATE INDEX IF NOT EXISTS idx_registries_enabled ON plugin_registries(enabled)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	// Insert default registry if not exists
	d.insertDefaultRegistry()

	return nil
}

func (d *Database) insertDefaultRegistry() {
	query := `INSERT OR IGNORE INTO plugin_registries (id, name, url, description) 
			  VALUES ('default', 'Official Plugin Registry', 'https://plugins.renote.app/registry', 'Official ReNote plugin registry')`
	d.db.Exec(query)
}

// Plugin CRUD operations
func (d *Database) CreatePlugin(plugin *Plugin) error {
	plugin.CreatedAt = time.Now()
	plugin.UpdatedAt = time.Now()

	query := `INSERT INTO plugins (
		id, name, description, version, author, homepage, repository, license,
		tags, main_file, asset_files, manifest, installed, enabled, install_path,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := d.db.Exec(query,
		plugin.ID, plugin.Name, plugin.Description, plugin.Version, plugin.Author,
		plugin.Homepage, plugin.Repository, plugin.License, plugin.Tags,
		plugin.MainFile, plugin.AssetFiles, plugin.Manifest, plugin.Installed,
		plugin.Enabled, plugin.InstallPath, plugin.CreatedAt, plugin.UpdatedAt,
	)
	return err
}

func (d *Database) GetPlugin(id string) (*Plugin, error) {
	query := `SELECT id, name, description, version, author, homepage, repository, license,
			  tags, main_file, asset_files, manifest, installed, enabled, install_path,
			  created_at, updated_at FROM plugins WHERE id = ?`

	row := d.db.QueryRow(query, id)
	plugin := &Plugin{}

	err := row.Scan(
		&plugin.ID, &plugin.Name, &plugin.Description, &plugin.Version, &plugin.Author,
		&plugin.Homepage, &plugin.Repository, &plugin.License, &plugin.Tags,
		&plugin.MainFile, &plugin.AssetFiles, &plugin.Manifest, &plugin.Installed,
		&plugin.Enabled, &plugin.InstallPath, &plugin.CreatedAt, &plugin.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return plugin, nil
}

func (d *Database) UpdatePlugin(plugin *Plugin) error {
	plugin.UpdatedAt = time.Now()

	query := `UPDATE plugins SET 
		name = ?, description = ?, version = ?, author = ?, homepage = ?, 
		repository = ?, license = ?, tags = ?, main_file = ?, asset_files = ?,
		manifest = ?, installed = ?, enabled = ?, install_path = ?, updated_at = ?
		WHERE id = ?`

	_, err := d.db.Exec(query,
		plugin.Name, plugin.Description, plugin.Version, plugin.Author,
		plugin.Homepage, plugin.Repository, plugin.License, plugin.Tags,
		plugin.MainFile, plugin.AssetFiles, plugin.Manifest, plugin.Installed,
		plugin.Enabled, plugin.InstallPath, plugin.UpdatedAt, plugin.ID,
	)
	return err
}

func (d *Database) DeletePlugin(id string) error {
	query := `DELETE FROM plugins WHERE id = ?`
	_, err := d.db.Exec(query, id)
	return err
}

func (d *Database) ListPlugins() ([]Plugin, error) {
	query := `SELECT id, name, description, version, author, homepage, repository, license,
			  tags, main_file, asset_files, manifest, installed, enabled, install_path,
			  created_at, updated_at FROM plugins ORDER BY name`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plugins []Plugin
	for rows.Next() {
		plugin := Plugin{}
		err := rows.Scan(
			&plugin.ID, &plugin.Name, &plugin.Description, &plugin.Version, &plugin.Author,
			&plugin.Homepage, &plugin.Repository, &plugin.License, &plugin.Tags,
			&plugin.MainFile, &plugin.AssetFiles, &plugin.Manifest, &plugin.Installed,
			&plugin.Enabled, &plugin.InstallPath, &plugin.CreatedAt, &plugin.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}

func (d *Database) ListInstalledPlugins() ([]Plugin, error) {
	query := `SELECT id, name, description, version, author, homepage, repository, license,
			  tags, main_file, asset_files, manifest, installed, enabled, install_path,
			  created_at, updated_at FROM plugins WHERE installed = TRUE ORDER BY name`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plugins []Plugin
	for rows.Next() {
		plugin := Plugin{}
		err := rows.Scan(
			&plugin.ID, &plugin.Name, &plugin.Description, &plugin.Version, &plugin.Author,
			&plugin.Homepage, &plugin.Repository, &plugin.License, &plugin.Tags,
			&plugin.MainFile, &plugin.AssetFiles, &plugin.Manifest, &plugin.Installed,
			&plugin.Enabled, &plugin.InstallPath, &plugin.CreatedAt, &plugin.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}

func (d *Database) ListEnabledPlugins() ([]Plugin, error) {
	query := `SELECT id, name, description, version, author, homepage, repository, license,
			  tags, main_file, asset_files, manifest, installed, enabled, install_path,
			  created_at, updated_at FROM plugins WHERE enabled = TRUE ORDER BY name`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plugins []Plugin
	for rows.Next() {
		plugin := Plugin{}
		err := rows.Scan(
			&plugin.ID, &plugin.Name, &plugin.Description, &plugin.Version, &plugin.Author,
			&plugin.Homepage, &plugin.Repository, &plugin.License, &plugin.Tags,
			&plugin.MainFile, &plugin.AssetFiles, &plugin.Manifest, &plugin.Installed,
			&plugin.Enabled, &plugin.InstallPath, &plugin.CreatedAt, &plugin.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}

// Registry CRUD operations
func (d *Database) CreateRegistry(registry *PluginRegistry) error {
	registry.CreatedAt = time.Now()
	registry.UpdatedAt = time.Now()

	query := `INSERT INTO plugin_registries (id, name, url, description, enabled, created_at, updated_at) 
			  VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := d.db.Exec(query, registry.ID, registry.Name, registry.URL, registry.Description,
		registry.Enabled, registry.CreatedAt, registry.UpdatedAt)
	return err
}

func (d *Database) ListRegistries() ([]PluginRegistry, error) {
	query := `SELECT id, name, url, description, enabled, created_at, updated_at 
			  FROM plugin_registries ORDER BY name`

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var registries []PluginRegistry
	for rows.Next() {
		registry := PluginRegistry{}
		err := rows.Scan(&registry.ID, &registry.Name, &registry.URL, &registry.Description,
			&registry.Enabled, &registry.CreatedAt, &registry.UpdatedAt)
		if err != nil {
			return nil, err
		}
		registries = append(registries, registry)
	}
	return registries, nil
}

// Plugin config operations
func (d *Database) SetPluginConfig(pluginID string, config map[string]interface{}) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	query := `INSERT OR REPLACE INTO plugin_configs (plugin_id, config, updated_at) 
			  VALUES (?, ?, ?)`
	_, err = d.db.Exec(query, pluginID, string(configJSON), time.Now())
	return err
}

func (d *Database) GetPluginConfig(pluginID string) (map[string]interface{}, error) {
	query := `SELECT config FROM plugin_configs WHERE plugin_id = ?`

	var configJSON string
	err := d.db.QueryRow(query, pluginID).Scan(&configJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	var config map[string]interface{}
	err = json.Unmarshal([]byte(configJSON), &config)
	return config, err
}

func (d *Database) Close() error {
	return d.db.Close()
}
