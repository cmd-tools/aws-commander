package cmd

import "github.com/rivo/tview"

type BreadcrumbType string

const (
	BreadcrumbProfiles      BreadcrumbType = "profiles"
	BreadcrumbProfile       BreadcrumbType = "profile"
	BreadcrumbResource      BreadcrumbType = "resource"
	BreadcrumbCommand       BreadcrumbType = "command"
	BreadcrumbSelectedItem  BreadcrumbType = "selected_item"
	BreadcrumbDependentCmds BreadcrumbType = "dependent_commands" // List of dependent commands to choose from
	BreadcrumbDependentCmd  BreadcrumbType = "dependent_command"
	BreadcrumbJsonView      BreadcrumbType = "json_view"
	BreadcrumbProcessedJson BreadcrumbType = "processed_json"
)

type NavigationState struct {
	Type          BreadcrumbType
	Value         string
	CachedResult  string          // Cached command result for this navigation level
	CachedBody    tview.Primitive // Cached UI body for this navigation level
	ProcessedData interface{}     // Processed JSON data at this level (for nested JSON)
}

type UIState struct {
	Profile            string            `yaml:"profile"`
	Resource           Resource          `yaml:"resource"`
	Command            Command           `yaml:"command"`
	SelectedItems      map[string]string `yaml:"selectedItems"`
	CommandBarVisible  bool              `yaml:"commandBarVisible"`
	Breadcrumbs        []string          `yaml:"breadcrumbs"`
	NavigationStack    []NavigationState // Enhanced navigation tracking
	CommandCache       map[string]string // Cache of command results: "resource:command:params" -> output
	ViewStack          []tview.Primitive
	ProcessedJsonData  interface{} // Stores processed JSON data (parsed or decompressed)
	JsonViewerCallback func()      // Callback to rebuild JSON viewer
	SelectedNodeText   string      // Stores the text of the selected node for focus restoration
}

var UiState UIState = UIState{SelectedItems: make(map[string]string), Breadcrumbs: []string{}, NavigationStack: []NavigationState{}, CommandCache: make(map[string]string)}
