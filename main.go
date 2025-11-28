package main

import (
	"bytes"
	"flag"
	"fmt"
	"strings"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/cmd/profile"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/helpers"
	"github.com/cmd-tools/aws-commander/logger"
	commandParser "github.com/cmd-tools/aws-commander/parser"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var App *tview.Application

var Search *tview.InputField
var Body tview.Primitive
var AutoCompletionWordList []string
var ProfileList profile.Profiles
var LogView *tview.TextView
var LogViewTextBuffer bytes.Buffer

var IsLogViewEnabled bool

func main() {
	flag.BoolVar(&IsLogViewEnabled, "logview", false, "Enable log view while using the tool.")
	flag.Parse()

	logger.InitLog(IsLogViewEnabled)

	logger.Logger.Info().Msg("Starting aws-commander")

	logger.Logger.Debug().Msg("Loading configurations")

	cmd.Init()

	App = tview.NewApplication()

	Search = createSearchBar()

	ProfileList = profile.GetList()

	Body = createBody()

	mainFlexPanel := updateRootView(nil)

	if IsLogViewEnabled {
		go startLogViewListener()
	}

	if err := App.SetRoot(mainFlexPanel, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func createHeader(keyCombs []ui.CustomShortCut) *tview.Flex {

	flex := tview.NewFlex()

	header := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	header.SetCell(0, 0, tview.NewTableCell("Aws CLI Rev:").SetTextColor(tcell.ColorGold))
	header.SetCell(0, 1, tview.NewTableCell(helpers.GetAWSClientVersion()).SetTextColor(tcell.ColorWhite))
	header.SetCell(1, 0, tview.NewTableCell("Aws Commander Rev:").SetTextColor(tcell.ColorGold))
	// TODO: get AWS commander version from somewhere else?
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

func createBody() *tview.Table {
	cmd.UiState.Breadcrumbs = []string{}
	return ui.CreateCustomTableView(ui.CustomTableViewProperties{
		Title: fmt.Sprintf(" Profiles [%d] ", len(ProfileList)),
		Columns: []ui.Column{
			{Name: "NAME", Width: 0},
			{Name: "REGION", Width: 0},
			{Name: "SSO_REGION", Width: 0},
			{Name: "SSO_ROLE_NAME", Width: 0},
			{Name: "SSO_ACCOUNT_ID", Width: 0},
			{Name: "SSO_START_URL", Width: 0},
		},
		Rows: ProfileList.AsMatrix(),
		Handler: func(selectedProfileName string) {
			cmd.UiState.Profile = selectedProfileName
			cmd.UiState.SelectedItems = make(map[string]string)
			resources := cmd.GetAvailableResourceNames()
			AutoCompletionWordList = []string{constants.Profiles}
			Body = createResources(resources)

			cmd.UiState.Breadcrumbs = []string{constants.Profiles, selectedProfileName}
			updateRootView(nil)
		},
	})
}

func createSearchBar() *tview.InputField {
	searchBar := tview.NewInputField()
	searchBar.SetLabel("️🔍 > ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetBorder(true).
		SetBackgroundColor(tcell.ColorDefault)

	searchBar.SetAutocompleteFunc(func(currentText string) (entries []string) {
		if len(currentText) == 0 {
			return
		}
		for _, word := range AutoCompletionWordList {
			if strings.HasPrefix(strings.ToLower(word), strings.ToLower(currentText)) {
				entries = append(entries, word)
			}
		}
		return
	})

	searchBar.SetAutocompletedFunc(func(text string, index, source int) bool {
		if source != tview.AutocompletedNavigate {
			Search.SetText(text)
		}
		return source == tview.AutocompletedEnter || source == tview.AutocompletedClick
	})

	searchBar.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc && Search.HasFocus() {
			logger.Logger.Debug().Msg("[Search section] Got ESC")
			cmd.UiState.CommandBarVisible = false
			Search.SetText(constants.EmptyString)
			updateRootView(nil)
			return nil
		}

		if event.Key() == tcell.KeyEnter && Search.HasFocus() {
			logger.Logger.Debug().Msg("[Search section] Got ENTER")
			text := Search.GetText()
			logger.Logger.Debug().Msg(fmt.Sprintf("[Search section] Got text: %s", Search.GetText()))
			switch text {
			case constants.Profiles:
				Body = createBody()
			case constants.Resources:
				Body = createResources(cmd.GetAvailableResourceNames())
			}
			cmd.UiState.CommandBarVisible = false
			updateRootView(nil)
			Search.SetText(constants.EmptyString)
			return nil
		}
		return event
	})

	return searchBar
}

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

func createResources(resources []string) tview.Primitive {
	cmd.UiState.Breadcrumbs = []string{constants.Profiles, cmd.UiState.Profile}
	return ui.CreateCustomListView(ui.ListViewBoxProperties{
		Title:   fmt.Sprintf(" Resources [%d] ", len(resources)),
		Options: resources,
		Handler: func(selectedResourceName string) {

			cmd.UiState.Resource = cmd.Resources[selectedResourceName]
			cmd.UiState.Breadcrumbs = []string{constants.Profiles, cmd.UiState.Profile, selectedResourceName}
			cmd.UiState.SelectedItems = make(map[string]string)
			AutoCompletionWordList = append(resources, constants.Profiles)
			if cmd.UiState.Resource.DefaultCommand == constants.EmptyString {
				Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
			} else {
				cmd.UiState.Command = cmd.UiState.Resource.GetCommand(cmd.UiState.Resource.DefaultCommand)
				cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, cmd.UiState.Command.Name)
				var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
				Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, cmd.UiState.Command, itemHandler, App, func() {
					updateRootView(nil)
				}, func() *tview.Flex { return createHeader(nil) }, createFooter, LogView, IsLogViewEnabled)
			}

			updateRootView(nil)
		},
	})
}

func createCommandView(commandNames []string) tview.Primitive {
	return ui.CreateCustomListView(ui.ListViewBoxProperties{
		Title:   fmt.Sprintf(" Commands [%d] ", len(commandNames)),
		Options: commandNames,
		Handler: createExecuteCommandView,
	})
}

func createExecuteCommandView(selectedCommandName string) {
	cmd.UiState.Command = cmd.UiState.Resource.GetCommand(selectedCommandName)
	AutoCompletionWordList = append(cmd.UiState.Resource.GetCommandNames(), constants.Profiles)

	cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, cmd.UiState.Command.Name)

	var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
	Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, cmd.UiState.Command, itemHandler, App, func() {
		updateRootView(nil)
	}, func() *tview.Flex { return createHeader(nil) }, createFooter, LogView, IsLogViewEnabled)

	updateRootView(nil)
}

func createLogView() *tview.TextView {
	logview := tview.NewTextView()
	logview.SetTitle(" Logs ")
	logview.
		ScrollToEnd().
		SetScrollable(true).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return logview
}

func itemHandler(selectedItemName string) {
	resourceName := cmd.VariablePlaceHolderPrefix + strings.ToUpper(cmd.UiState.Command.ResourceName)
	cmd.UiState.SelectedItems[resourceName] = selectedItemName

	AutoCompletionWordList = append(cmd.UiState.Resource.GetCommandNames(), constants.Profiles)

	// Find command that depends on the current command
	var dependentCommand *cmd.Command
	for _, c := range cmd.UiState.Resource.Commands {
		if c.DependsOn == cmd.UiState.Command.Name {
			dependentCommand = &c
			break
		}
	}

	if dependentCommand == nil {
		// No dependent command found, show command list
		Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
	} else {
		// Add the selected item name and command name to breadcrumbs
		cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, selectedItemName)

		cmd.UiState.Command = *dependentCommand
		cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, cmd.UiState.Command.Name)

		var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
		Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, cmd.UiState.Command, itemHandler, App, func() {
			updateRootView(nil)
		}, func() *tview.Flex { return createHeader(nil) }, createFooter, LogView, IsLogViewEnabled)
	}

	updateRootView(nil)
}

func defaultKeyCombinations() []ui.CustomShortCut {
	return []ui.CustomShortCut{
		{
			Name:        "esc",
			Key:         tcell.KeyEsc,
			Description: "Back",
			Rune:        -1,
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				if Search.HasFocus() {
					return event
				}
				breadcrumbsLen := len(cmd.UiState.Breadcrumbs)

				logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Breadcrumbs: %v (length: %d)", cmd.UiState.Breadcrumbs, breadcrumbsLen))

				// Handle navigation based on breadcrumb length
				switch breadcrumbsLen {
				case 0, 1:
					// At profiles or empty - show profiles
					Body = createBody()
					cmd.UiState.Breadcrumbs = []string{}
				case 2:
					// At resources list - go back to profiles
					cmd.UiState.Breadcrumbs = cmd.UiState.Breadcrumbs[:1]
					Body = createBody()
				case 3:
					// At resource view (command list or default command) - go back to resources
					profileName := cmd.UiState.Breadcrumbs[1]
					cmd.UiState.Breadcrumbs = []string{constants.Profiles, profileName}
					Body = createResources(cmd.GetAvailableResourceNames())
				case 4:
					// At command result - go back to resource view
					resourceName := cmd.UiState.Breadcrumbs[2]
					cmd.UiState.Breadcrumbs = cmd.UiState.Breadcrumbs[:3]
					cmd.UiState.Resource = cmd.Resources[resourceName]

					// If the current command is the default command, skip it and go to resources list
					if cmd.UiState.Resource.DefaultCommand != constants.EmptyString &&
						cmd.UiState.Command.Name == cmd.UiState.Resource.DefaultCommand {
						// Go back to resources list instead
						profileName := cmd.UiState.Breadcrumbs[1]
						cmd.UiState.Breadcrumbs = []string{constants.Profiles, profileName}
						Body = createResources(cmd.GetAvailableResourceNames())
					} else if cmd.UiState.Resource.DefaultCommand == constants.EmptyString {
						Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
					} else {
						// Re-run the default command
						cmd.UiState.Command = cmd.UiState.Resource.GetCommand(cmd.UiState.Resource.DefaultCommand)
						cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, cmd.UiState.Command.Name)
						var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
						Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, cmd.UiState.Command, itemHandler, App, func() {
							updateRootView(nil)
						}, func() *tview.Flex { return createHeader(nil) }, createFooter, LogView, IsLogViewEnabled)
					}
				default:
					// Length 5+: Could be at dependent command, JSON viewer, or processed JSON
					lastBreadcrumb := cmd.UiState.Breadcrumbs[breadcrumbsLen-1]

					// Check if we're in a processed JSON view (parsed or decompressed)
					if lastBreadcrumb == "Parsed JSON" || strings.HasPrefix(lastBreadcrumb, "Decompressed") {
						// Remove the processed JSON breadcrumb
						cmd.UiState.Breadcrumbs = cmd.UiState.Breadcrumbs[:breadcrumbsLen-1]
						cmd.UiState.ProcessedJsonData = nil // Clear processed data

						// Call the JSON viewer callback to rebuild the view with original data
						if cmd.UiState.JsonViewerCallback != nil {
							cmd.UiState.JsonViewerCallback()
							// The callback already sets the focus correctly, so just return
							return nil
						}
					} else if len(lastBreadcrumb) >= 9 && lastBreadcrumb[:9] == "JSON View" {
						// Remove JSON viewer breadcrumb and restore the command view
						cmd.UiState.Breadcrumbs = cmd.UiState.Breadcrumbs[:breadcrumbsLen-1]
						cmd.UiState.ProcessedJsonData = nil  // Clear any processed data
						cmd.UiState.JsonViewerCallback = nil // Clear callback

						// Re-run current command to restore table view
						var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
						Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, cmd.UiState.Command, itemHandler, App, func() {
							updateRootView(nil)
						}, func() *tview.Flex { return createHeader(nil) }, createFooter, LogView, IsLogViewEnabled)
					} else {
						// At dependent command (length 5+) - go back to parent command
						logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Going back from dependent command. Last breadcrumb: %s", lastBreadcrumb))

						// Remove selected item name and dependent command name (last 2 breadcrumbs)
						cmd.UiState.Breadcrumbs = cmd.UiState.Breadcrumbs[:breadcrumbsLen-2]

						logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] After removing 2 breadcrumbs: %v", cmd.UiState.Breadcrumbs))

						// Get the parent command name from breadcrumbs
						parentCommandName := cmd.UiState.Breadcrumbs[len(cmd.UiState.Breadcrumbs)-1]
						logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Parent command name: %s", parentCommandName))

						cmd.UiState.Command = cmd.UiState.Resource.GetCommand(parentCommandName)

						// Re-run parent command
						var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
						Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, cmd.UiState.Command, itemHandler, App, func() {
							updateRootView(nil)
						}, func() *tview.Flex { return createHeader(nil) }, createFooter, LogView, IsLogViewEnabled)
					}
				}

				updateRootView(nil)
				App.SetFocus(Body)
				return nil
			},
		},
		{
			Rune:        ':',
			Description: "Search",
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				cmd.UiState.CommandBarVisible = true
				updateRootView(nil)
				App.SetFocus(Search)
				return nil
			},
		},
		{
			Rune:        '?',
			Description: "Help",
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				return nil
			},
		},
	}
}

func startLogViewListener() {
	for {
		select {
		case logMessage := <-logger.LogChannel:
			LogViewTextBuffer.WriteString(logMessage)
			if nil != LogView {
				App.QueueUpdateDraw(func() {
					if nil != LogView {
						LogView.SetText(LogViewTextBuffer.String())
					}
				})
			}
		}
	}
}
