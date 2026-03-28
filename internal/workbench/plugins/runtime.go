package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Runtime manages loaded plugins and their lifecycle
type Runtime struct {
	plugins  map[string]*LoadedPlugin
	hooks    map[string][]HookCallback
	commands map[string]CommandCallback
	menus    []PluginMenu
	panels   []PluginPanel
	eventBus *EventBus
	mu       sync.RWMutex
}

// LoadedPlugin represents a plugin that is currently loaded in memory
type LoadedPlugin struct {
	Plugin   *Plugin
	Manifest *PluginManifest
	Context  *PluginContext
	Hooks    map[string]HookCallback
	Commands map[string]CommandCallback
}

// PluginContext provides runtime context for plugins
type PluginContext struct {
	PluginID   string
	PluginPath string
	ConfigPath string
	DataPath   string
	EventBus   *EventBus
	Logger     *Logger
}

// HookCallback represents a plugin hook function
type HookCallback func(context *PluginContext, data interface{}) (interface{}, error)

// CommandCallback represents a plugin command function
type CommandCallback func(context *PluginContext, args []string) (interface{}, error)

// EventBus handles plugin events
type EventBus struct {
	subscribers map[string][]func(interface{})
	mu          sync.RWMutex
}

// Logger provides logging for plugins
type Logger struct {
	pluginID string
}

func NewRuntime() *Runtime {
	return &Runtime{
		plugins:  make(map[string]*LoadedPlugin),
		hooks:    make(map[string][]HookCallback),
		commands: make(map[string]CommandCallback),
		menus:    make([]PluginMenu, 0),
		eventBus: NewEventBus(),
	}
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]func(interface{})),
	}
}

func (r *Runtime) LoadPlugin(plugin *Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[plugin.ID]; exists {
		return fmt.Errorf("plugin %s is already loaded", plugin.ID)
	}

	// Parse manifest
	manifest, err := plugin.GetManifest()
	if err != nil {
		return fmt.Errorf("failed to parse manifest: %v", err)
	}

	// Create plugin context
	context := &PluginContext{
		PluginID:   plugin.ID,
		PluginPath: plugin.InstallPath,
		ConfigPath: filepath.Join(plugin.InstallPath, "config"),
		DataPath:   filepath.Join(plugin.InstallPath, "data"),
		EventBus:   r.eventBus,
		Logger:     &Logger{pluginID: plugin.ID},
	}

	// Ensure data directories exist
	os.MkdirAll(context.ConfigPath, 0755)
	os.MkdirAll(context.DataPath, 0755)

	// Load plugin main file (JavaScript/TypeScript for now)
	mainPath := filepath.Join(plugin.InstallPath, manifest.MainFile)
	if !fileExists(mainPath) {
		return fmt.Errorf("main file not found: %s", mainPath)
	}

	// Create loaded plugin
	loadedPlugin := &LoadedPlugin{
		Plugin:   plugin,
		Manifest: manifest,
		Context:  context,
		Hooks:    make(map[string]HookCallback),
		Commands: make(map[string]CommandCallback),
	}

	// Register plugin hooks and commands
	r.registerPluginHooks(loadedPlugin)
	r.registerPluginCommands(loadedPlugin)
	r.registerPluginMenus(loadedPlugin)
	r.registerPluginPanels(loadedPlugin)

	// Store loaded plugin
	r.plugins[plugin.ID] = loadedPlugin

	// Emit plugin loaded event
	r.eventBus.Emit("plugin:loaded", PluginEvent{
		Type:     "loaded",
		PluginID: plugin.ID,
	})

	return nil
}

func (r *Runtime) UnloadPlugin(pluginID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	loadedPlugin, exists := r.plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", pluginID)
	}

	// Unregister hooks
	r.unregisterPluginHooks(loadedPlugin)

	// Unregister commands
	r.unregisterPluginCommands(loadedPlugin)

	// Unregister menus
	r.unregisterPluginMenus(loadedPlugin)

	// Unregister panels
	r.unregisterPluginPanels(loadedPlugin)

	// Remove from loaded plugins
	delete(r.plugins, pluginID)

	// Emit plugin unloaded event
	r.eventBus.Emit("plugin:unloaded", PluginEvent{
		Type:     "unloaded",
		PluginID: pluginID,
	})

	return nil
}

func (r *Runtime) registerPluginHooks(plugin *LoadedPlugin) {
	// For now, hooks are defined in manifest
	// In a real implementation, these would be registered by the plugin code
	for hookName := range plugin.Manifest.Hooks {
		callback := r.createHookCallback(plugin, hookName)
		plugin.Hooks[hookName] = callback
		r.hooks[hookName] = append(r.hooks[hookName], callback)
	}
}

func (r *Runtime) registerPluginCommands(plugin *LoadedPlugin) {
	for _, cmd := range plugin.Manifest.Commands {
		callback := r.createCommandCallback(plugin, cmd.Callback)
		plugin.Commands[cmd.ID] = callback
		r.commands[cmd.ID] = callback
	}
}

func (r *Runtime) registerPluginMenus(plugin *LoadedPlugin) {
	for _, menu := range plugin.Manifest.Menus {
		menu.ID = plugin.Plugin.ID + ":" + menu.ID // Namespace menu IDs
		r.menus = append(r.menus, menu)
	}
}

func (r *Runtime) registerPluginPanels(plugin *LoadedPlugin) {
	for _, panel := range plugin.Manifest.Panels {
		panel.ID = plugin.Plugin.ID + ":" + panel.ID // Namespace panel IDs
		r.panels = append(r.panels, panel)
	}
}

func (r *Runtime) unregisterPluginHooks(plugin *LoadedPlugin) {
	for hookName, callback := range plugin.Hooks {
		hooks := r.hooks[hookName]
		for i, h := range hooks {
			if &h == &callback {
				r.hooks[hookName] = append(hooks[:i], hooks[i+1:]...)
				break
			}
		}
		if len(r.hooks[hookName]) == 0 {
			delete(r.hooks, hookName)
		}
	}
}

func (r *Runtime) unregisterPluginCommands(plugin *LoadedPlugin) {
	for cmdID := range plugin.Commands {
		delete(r.commands, cmdID)
	}
}

func (r *Runtime) unregisterPluginMenus(plugin *LoadedPlugin) {
	prefix := plugin.Plugin.ID + ":"
	filtered := make([]PluginMenu, 0)
	for _, menu := range r.menus {
		if !startsWith(menu.ID, prefix) {
			filtered = append(filtered, menu)
		}
	}
	r.menus = filtered
}

func (r *Runtime) unregisterPluginPanels(plugin *LoadedPlugin) {
	prefix := plugin.Plugin.ID + ":"
	filtered := make([]PluginPanel, 0)
	for _, panel := range r.panels {
		if !startsWith(panel.ID, prefix) {
			filtered = append(filtered, panel)
		}
	}
	r.panels = filtered
}

func (r *Runtime) createHookCallback(plugin *LoadedPlugin, hookName string) HookCallback {
	return func(context *PluginContext, data interface{}) (interface{}, error) {
		// In a real implementation, this would call into the plugin's JavaScript runtime
		// For now, we'll just return the data unchanged
		return data, nil
	}
}

func (r *Runtime) createCommandCallback(plugin *LoadedPlugin, callbackName string) CommandCallback {
	return func(context *PluginContext, args []string) (interface{}, error) {
		// In a real implementation, this would call into the plugin's JavaScript runtime
		// For now, we'll just return a success message
		return map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Command %s executed", callbackName),
			"plugin":  plugin.Plugin.ID,
		}, nil
	}
}

// Public methods for runtime interaction
func (r *Runtime) ExecuteHook(hookName string, data interface{}) []interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	callbacks, exists := r.hooks[hookName]
	if !exists {
		return nil
	}

	results := make([]interface{}, 0)
	for _, callback := range callbacks {
		result, err := callback(nil, data)
		if err != nil {
			// Log error but continue with other callbacks
			fmt.Printf("Hook %s error: %v\n", hookName, err)
			continue
		}
		results = append(results, result)
	}

	return results
}

func (r *Runtime) ExecuteCommand(commandID string, args []string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	callback, exists := r.commands[commandID]
	if !exists {
		return nil, fmt.Errorf("command %s not found", commandID)
	}

	return callback(nil, args)
}

func (r *Runtime) GetMenus() []PluginMenu {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	menus := make([]PluginMenu, len(r.menus))
	copy(menus, r.menus)
	return menus
}

func (r *Runtime) GetPanels() []PluginPanel {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	panels := make([]PluginPanel, len(r.panels))
	copy(panels, r.panels)
	return panels
}

func (r *Runtime) AddPanel(panel PluginPanel) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if panel already exists
	for i, existing := range r.panels {
		if existing.ID == panel.ID {
			r.panels[i] = panel
			return
		}
	}

	// Add new panel
	r.panels = append(r.panels, panel)
}

func (r *Runtime) RemovePanel(panelID string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, panel := range r.panels {
		if panel.ID == panelID {
			r.panels = append(r.panels[:i], r.panels[i+1:]...)
			return
		}
	}
}

func (r *Runtime) GetLoadedPlugins() map[string]*LoadedPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	plugins := make(map[string]*LoadedPlugin)
	for k, v := range r.plugins {
		plugins[k] = v
	}
	return plugins
}

func (r *Runtime) IsPluginLoaded(pluginID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.plugins[pluginID]
	return exists
}

func (r *Runtime) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Unload all plugins
	for pluginID := range r.plugins {
		r.unloadPluginUnsafe(pluginID)
	}
}

func (r *Runtime) unloadPluginUnsafe(pluginID string) {
	// Internal method that doesn't lock mutex (unsafe for external use)
	if loadedPlugin, exists := r.plugins[pluginID]; exists {
		r.unregisterPluginHooks(loadedPlugin)
		r.unregisterPluginCommands(loadedPlugin)
		r.unregisterPluginMenus(loadedPlugin)
		r.unregisterPluginPanels(loadedPlugin)
		delete(r.plugins, pluginID)
	}
}

// EventBus methods
func (e *EventBus) Subscribe(event string, callback func(interface{})) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.subscribers[event] = append(e.subscribers[event], callback)
}

func (e *EventBus) Emit(event string, data interface{}) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if callbacks, exists := e.subscribers[event]; exists {
		for _, callback := range callbacks {
			go callback(data) // Execute asynchronously
		}
	}
}

// Logger methods
func (l *Logger) Info(message string, args ...interface{}) {
	fmt.Printf("[%s] INFO: %s\n", l.pluginID, fmt.Sprintf(message, args...))
}

func (l *Logger) Error(message string, args ...interface{}) {
	fmt.Printf("[%s] ERROR: %s\n", l.pluginID, fmt.Sprintf(message, args...))
}

func (l *Logger) Debug(message string, args ...interface{}) {
	fmt.Printf("[%s] DEBUG: %s\n", l.pluginID, fmt.Sprintf(message, args...))
}

// Utility functions
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func startsWith(str, prefix string) bool {
	return len(str) >= len(prefix) && str[0:len(prefix)] == prefix
}
