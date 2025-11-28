package main

import (
	"fmt"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/helpers"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// createHeader creates the header component with version info and shortcuts
func createHeader(keyCombs []ui.CustomShortCut) *tview.Flex {
	flex := tview.NewFlex()

	header := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	header.SetCell(0, 0, tview.NewTableCell("Aws CLI Rev:").SetTextColor(tcell.ColorGold))
	header.SetCell(0, 1, tview.NewTableCell(helpers.GetAWSClientVersion()).SetTextColor(tcell.ColorWhite))
	header.SetCell(1, 0, tview.NewTableCell("Aws Commander Rev:").SetTextColor(tcell.ColorGold))
	header.SetCell(1, 1, tview.NewTableCell("v0.0.1").SetTextColor(tcell.ColorWhite))

	header.SetBorderPadding(0, 1, 1, 1)

	shortcuts := ui.CreateCustomShortCutsView(App, ui.CustomShortCutProperties{
		Shortcuts: append(keyCombs, defaultKeyCombinations()...),
	})

	flex.AddItem(header, 0, 2, false).
		AddItem(shortcuts, 0, 4, false).
		AddItem(createLogo(), 0, 2, false)

	return flex
}

// createFooter creates the footer component showing breadcrumbs
func createFooter(sections []string) *tview.Table {
	sectionBackgroundColorTags := []tcell.Color{
		tcell.ColorGold,
		tcell.ColorDarkMagenta,
		tcell.ColorAquaMarine,
		tcell.ColorWhite,
	}

	sectionForegroundColorTags := []tcell.Color{
		tcell.ColorBlack,
		tcell.ColorWhite,
		tcell.ColorBlack,
		tcell.ColorBlack,
	}

	header := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	for i, section := range sections {
		colorScheme := i
		if colorScheme > 3 {
			colorScheme = 3
		}

		header.SetCell(0, i*2, tview.NewTableCell(fmt.Sprintf(" <%s>", section)).
			SetBackgroundColor(sectionBackgroundColorTags[colorScheme]).
			SetTextColor(sectionForegroundColorTags[colorScheme]).
			SetAlign(tview.AlignCenter))
	}

	header.SetBorderPadding(0, 1, 1, 1)
	return header
}

// createLogo creates the AWS Commander logo text view
func createLogo() *tview.TextView {
	l := `
	 _____  _ _ _  _____
	|  _  || | | ||   __|
	|     || | | ||__   |
	|__|__||_____||_____|
	  C O M M A N D E R
	`
	t1 := tview.NewTextView()
	t1.SetBorder(false).SetBorderPadding(0, 1, 1, 1)
	t1.SetText(l).SetTextAlign(tview.AlignRight).SetTextColor(tcell.ColorGold)

	return t1
}

// updateRootView assembles and displays the main UI layout
func updateRootView(shortcuts []ui.CustomShortCut) *tview.Flex {
	view := tview.
		NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(createHeader(shortcuts), 7, 2, false)

	if cmd.UiState.CommandBarVisible {
		view.AddItem(Search, 3, 2, false)
	}

	view.AddItem(Body, 0, 1, true).
		AddItem(createFooter(cmd.UiState.Breadcrumbs), 2, 2, false)

	if IsLogViewEnabled {
		if nil == LogView {
			LogView = createLogView()
		}
		view.AddItem(LogView, 8, 3, false)
	}

	App.SetRoot(view, true)
	return view
}

// createBody creates the initial profile selection view
func createBody() *tview.Table {
	cmd.UiState.Breadcrumbs = []string{constants.Profiles}
	cmd.UiState.NavigationStack = []cmd.NavigationState{
		{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
	}

	columns := []ui.Column{
		{Name: "Name", Width: 0},
	}

	return ui.CreateCustomTableView(ui.CustomTableViewProperties{
		Title:   fmt.Sprintf(" Profiles [%d] ", len(ProfileList)),
		Columns: columns,
		Rows:    ProfileList.AsMatrix(),
		Handler: func(selectedProfileName string) {
			cmd.UiState.Profile = selectedProfileName
			cmd.UiState.Breadcrumbs = []string{constants.Profiles, selectedProfileName}
			cmd.UiState.NavigationStack = []cmd.NavigationState{
				{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
				{Type: cmd.BreadcrumbProfile, Value: selectedProfileName},
			}

			AutoCompletionWordList = append(cmd.GetAvailableResourceNames(), constants.Profiles)
			Body = createResources(cmd.GetAvailableResourceNames())

			updateRootView(nil)
		},
		App: App,
	})
}

// createResources creates the resource selection view
func createResources(resources []string) tview.Primitive {
	cmd.UiState.Breadcrumbs = []string{constants.Profiles, cmd.UiState.Profile}
	cmd.UiState.NavigationStack = []cmd.NavigationState{
		{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
		{Type: cmd.BreadcrumbProfile, Value: cmd.UiState.Profile},
	}
	return ui.CreateCustomListView(ui.ListViewBoxProperties{
		Title:   fmt.Sprintf(" Resources [%d] ", len(resources)),
		Options: resources,
		Handler: resourceSelectionHandler,
	})
}

// resourceSelectionHandler handles resource selection
func resourceSelectionHandler(selectedResourceName string) {
	cmd.UiState.Resource = cmd.Resources[selectedResourceName]
	cmd.UiState.Breadcrumbs = []string{constants.Profiles, cmd.UiState.Profile, selectedResourceName}
	cmd.UiState.NavigationStack = []cmd.NavigationState{
		{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
		{Type: cmd.BreadcrumbProfile, Value: cmd.UiState.Profile},
		{Type: cmd.BreadcrumbResource, Value: selectedResourceName},
	}
	cmd.UiState.SelectedItems = make(map[string]string)
	AutoCompletionWordList = append(cmd.GetAvailableResourceNames(), constants.Profiles)
	
	if cmd.UiState.Resource.DefaultCommand == constants.EmptyString {
		Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
	} else {
		cmd.UiState.Command = cmd.UiState.Resource.GetCommand(cmd.UiState.Resource.DefaultCommand)
		pushNavigation(cmd.BreadcrumbCommand, cmd.UiState.Command.Name)
		_, body := executeCommand(cmd.UiState.Command)
		Body = body
	}

	updateRootView(nil)
}

// createCommandView creates the command selection view
func createCommandView(commandNames []string) tview.Primitive {
	return ui.CreateCustomListView(ui.ListViewBoxProperties{
		Title:   fmt.Sprintf(" Commands [%d] ", len(commandNames)),
		Options: commandNames,
		Handler: createExecuteCommandView,
	})
}

// createDependentCommandView creates a view for selecting dependent commands
func createDependentCommandView(commandNames []string) tview.Primitive {
	return ui.CreateCustomListView(ui.ListViewBoxProperties{
		Title:   fmt.Sprintf(" Dependent Commands [%d] ", len(commandNames)),
		Options: commandNames,
		Handler: executeDependentCommand,
	})
}

// createLogView creates the log viewer component
func createLogView() *tview.TextView {
	logview := tview.NewTextView()
	logview.SetTitle(" Logs ")
	logview.ScrollToEnd()
	logview.SetScrollable(true)
	logview.SetBorder(true)
	logview.SetBorderPadding(0, 0, 1, 1)

	return logview
}
