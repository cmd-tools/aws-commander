package cmd

import "github.com/rivo/tview"

type UIState struct {
	Profile            string            `yaml:"profile"`
	Resource           Resource          `yaml:"resource"`
	Command            Command           `yaml:"command"`
	SelectedItems      map[string]string `yaml:"selectedItems"`
	CommandBarVisible  bool              `yaml:"commandBarVisible"`
	Breadcrumbs        []string          `yaml:"breadcrumbs"`
	ViewStack          []tview.Primitive
	ProcessedJsonData  interface{} // Stores processed JSON data (parsed or decompressed)
	JsonViewerCallback func()      // Callback to rebuild JSON viewer
	SelectedNodeText   string      // Stores the text of the selected node for focus restoration
}

var UiState UIState = UIState{SelectedItems: make(map[string]string), Breadcrumbs: []string{}}
