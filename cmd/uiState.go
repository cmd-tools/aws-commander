package cmd

type UIState struct {
	Profile           string            `yaml:"profile"`
	Resource          Resource          `yaml:"resource"`
	Command           Command           `yaml:"command"`
	SelectedItems     map[string]string `yaml:"selectedItems"`
	CommandBarVisible bool              `yaml:"commandBarVisible"`
	Breadcrumbs       []string          `yaml:"breadcrumbs"`
}

var UiState UIState = UIState{SelectedItems: make(map[string]string), Breadcrumbs: []string{}}
