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

// pushNavigation adds a new navigation state to the stack
func pushNavigation(navType cmd.BreadcrumbType, value string) {
	cmd.UiState.NavigationStack = append(cmd.UiState.NavigationStack, cmd.NavigationState{
		Type:  navType,
		Value: value,
	})
	cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, value)
}

// pushNavigationWithCache adds a new navigation state with cached result
func pushNavigationWithCache(navType cmd.BreadcrumbType, value string, cachedResult string, cachedBody tview.Primitive) {
	cmd.UiState.NavigationStack = append(cmd.UiState.NavigationStack, cmd.NavigationState{
		Type:         navType,
		Value:        value,
		CachedResult: cachedResult,
		CachedBody:   cachedBody,
	})
	cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, value)
}

// updateNavigationCache updates the cache for the current navigation level
func updateNavigationCache(cachedResult string, cachedBody tview.Primitive) {
	if len(cmd.UiState.NavigationStack) > 0 {
		lastIndex := len(cmd.UiState.NavigationStack) - 1
		cmd.UiState.NavigationStack[lastIndex].CachedResult = cachedResult
		cmd.UiState.NavigationStack[lastIndex].CachedBody = cachedBody
	}
}

// executeCommand runs a command and optionally caches the result based on command configuration
func executeCommand(command cmd.Command) (string, tview.Primitive) {
	// Check if we should use pagination token
	paginationToken := cmd.UiState.CurrentPageToken

	// Run the command with pagination support
	var commandOutput string
	if command.Pagination != nil && command.Pagination.Enabled && paginationToken != "" {
		commandOutput = command.RunWithPaginationToken(cmd.UiState.Resource.Name, cmd.UiState.Profile, paginationToken)
	} else {
		commandOutput = command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile)
	}

	// Extract next page token if pagination is enabled
	if command.Pagination != nil && command.Pagination.Enabled {
		nextToken := cmd.ExtractPaginationToken(commandOutput, command)
		// Store in navigation state
		currentNav := peekNavigation()
		if currentNav != nil {
			currentNav.PaginationToken = nextToken
		}
	}

	commandParsed := commandParser.ParseCommand(command, commandOutput)
	body := commandParser.ParseToObject(command.View, commandParsed, command, itemHandler, App, func() {
		updateRootView(nil)
	}, func() *tview.Flex { return createHeader(nil) }, createFooter, LogView, IsLogViewEnabled)

	// Cache the result if rerunOnBack is false
	if !command.RerunOnBack {
		updateNavigationCache(commandOutput, body)
	}

	return commandOutput, body
}

// popNavigation removes the last navigation state from the stack
func popNavigation() *cmd.NavigationState {
	if len(cmd.UiState.NavigationStack) == 0 {
		return nil
	}

	lastIndex := len(cmd.UiState.NavigationStack) - 1
	state := cmd.UiState.NavigationStack[lastIndex]
	cmd.UiState.NavigationStack = cmd.UiState.NavigationStack[:lastIndex]

	if len(cmd.UiState.Breadcrumbs) > 0 {
		cmd.UiState.Breadcrumbs = cmd.UiState.Breadcrumbs[:len(cmd.UiState.Breadcrumbs)-1]
	}

	return &state
}

// peekNavigation returns the last navigation state without removing it
func peekNavigation() *cmd.NavigationState {
	if len(cmd.UiState.NavigationStack) == 0 {
		return nil
	}
	return &cmd.UiState.NavigationStack[len(cmd.UiState.NavigationStack)-1]
}

func createBody() *tview.Table {
	cmd.UiState.Breadcrumbs = []string{}
	cmd.UiState.NavigationStack = []cmd.NavigationState{}
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
			cmd.UiState.NavigationStack = []cmd.NavigationState{
				{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
				{Type: cmd.BreadcrumbProfile, Value: selectedProfileName},
			}
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

	searchBar.SetChangedFunc(func(text string) {
		// Filter table rows if Body is a table
		if table, ok := Body.(*tview.Table); ok {
			if text != "" {
				// Filter the table
				filterTableRows(table, text)
			} else {
				// Restore all rows when search is cleared
				restoreTableRows(table)
			}
		}
	})

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

func filterTableRows(table *tview.Table, filter string) {
	filter = strings.ToLower(filter)
	rowCount := table.GetRowCount()

	// Start from row 1 to skip the header
	for row := 1; row < rowCount; row++ {
		// Check all cells in the row
		rowMatches := false
		colCount := table.GetColumnCount()

		for col := 0; col < colCount; col++ {
			cell := table.GetCell(row, col)
			if cell != nil {
				cellText := strings.ToLower(cell.Text)
				if strings.Contains(cellText, filter) {
					rowMatches = true
					break
				}
			}
		}

		// Show or hide the row based on match
		// Note: tview doesn't support hiding rows, so we'll use a workaround
		// by setting the row's cells to selectable/non-selectable
		for col := 0; col < colCount; col++ {
			cell := table.GetCell(row, col)
			if cell != nil {
				if rowMatches {
					cell.SetTextColor(tcell.ColorWhite)
					cell.SetSelectable(true)
				} else {
					cell.SetTextColor(tcell.ColorGray)
					cell.SetSelectable(false)
				}
			}
		}
	}
}

func restoreTableRows(table *tview.Table) {
	rowCount := table.GetRowCount()

	// Restore all rows to visible/selectable (skip header row 0)
	for row := 1; row < rowCount; row++ {
		colCount := table.GetColumnCount()
		for col := 0; col < colCount; col++ {
			cell := table.GetCell(row, col)
			if cell != nil {
				cell.SetTextColor(tcell.ColorWhite)
				cell.SetSelectable(true)
			}
		}
	}
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
	cmd.UiState.NavigationStack = []cmd.NavigationState{
		{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
		{Type: cmd.BreadcrumbProfile, Value: cmd.UiState.Profile},
	}
	return ui.CreateCustomListView(ui.ListViewBoxProperties{
		Title:   fmt.Sprintf(" Resources [%d] ", len(resources)),
		Options: resources,
		Handler: func(selectedResourceName string) {

			cmd.UiState.Resource = cmd.Resources[selectedResourceName]
			cmd.UiState.Breadcrumbs = []string{constants.Profiles, cmd.UiState.Profile, selectedResourceName}
			cmd.UiState.NavigationStack = []cmd.NavigationState{
				{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
				{Type: cmd.BreadcrumbProfile, Value: cmd.UiState.Profile},
				{Type: cmd.BreadcrumbResource, Value: selectedResourceName},
			}
			cmd.UiState.SelectedItems = make(map[string]string)
			AutoCompletionWordList = append(resources, constants.Profiles)
			if cmd.UiState.Resource.DefaultCommand == constants.EmptyString {
				Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
			} else {
				cmd.UiState.Command = cmd.UiState.Resource.GetCommand(cmd.UiState.Resource.DefaultCommand)
				pushNavigation(cmd.BreadcrumbCommand, cmd.UiState.Command.Name)
				_, body := executeCommand(cmd.UiState.Command)
				Body = body
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

func createDependentCommandView(commandNames []string) tview.Primitive {
	return ui.CreateCustomListView(ui.ListViewBoxProperties{
		Title:   fmt.Sprintf(" Dependent Commands [%d] ", len(commandNames)),
		Options: commandNames,
		Handler: executeDependentCommand,
	})
}

func executeDependentCommand(selectedCommandName string) {
	cmd.UiState.Command = cmd.UiState.Resource.GetCommand(selectedCommandName)
	AutoCompletionWordList = append(cmd.UiState.Resource.GetCommandNames(), constants.Profiles)

	pushNavigation(cmd.BreadcrumbDependentCmd, cmd.UiState.Command.Name)

	_, body := executeCommand(cmd.UiState.Command)
	Body = body

	updateRootView(nil)
}

func createExecuteCommandView(selectedCommandName string) {
	cmd.UiState.Command = cmd.UiState.Resource.GetCommand(selectedCommandName)
	AutoCompletionWordList = append(cmd.UiState.Resource.GetCommandNames(), constants.Profiles)

	// Reset pagination state for new command
	cmd.UiState.CurrentPageToken = ""
	cmd.UiState.PageHistory = []string{}

	pushNavigation(cmd.BreadcrumbCommand, cmd.UiState.Command.Name)

	_, body := executeCommand(cmd.UiState.Command)
	Body = body

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

	// Find all commands that depend on the current command
	var dependentCommands []cmd.Command
	for _, c := range cmd.UiState.Resource.Commands {
		if c.DependsOn == cmd.UiState.Command.Name {
			dependentCommands = append(dependentCommands, c)
		}
	}

	if len(dependentCommands) == 0 {
		// No dependent command found, show command list
		Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
	} else if len(dependentCommands) == 1 {
		// Only one dependent command, execute it directly
		pushNavigation(cmd.BreadcrumbSelectedItem, selectedItemName)

		cmd.UiState.Command = dependentCommands[0]
		pushNavigation(cmd.BreadcrumbDependentCmd, cmd.UiState.Command.Name)

		_, body := executeCommand(cmd.UiState.Command)
		Body = body
	} else {
		// Multiple dependent commands, show selection list
		pushNavigation(cmd.BreadcrumbSelectedItem, selectedItemName)

		var commandNames []string
		for _, c := range dependentCommands {
			commandNames = append(commandNames, c.Name)
		}

		pushNavigation(cmd.BreadcrumbDependentCmds, "Select Command")
		Body = createDependentCommandView(commandNames)
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

				currentState := peekNavigation()
				if currentState == nil {
					// At root, nothing to do
					return nil
				}

				logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Current state: %s = %s, Stack length: %d", currentState.Type, currentState.Value, len(cmd.UiState.NavigationStack)))

				// Handle navigation based on the current state type
				switch currentState.Type {
				case cmd.BreadcrumbProfiles:
					// At profiles list, nothing to go back to
					return nil

				case cmd.BreadcrumbProfile:
					// Go back to profiles list
					popNavigation()
					Body = createBody()

				case cmd.BreadcrumbResource:
					// Go back to resources list
					popNavigation() // Remove resource
					Body = createResources(cmd.GetAvailableResourceNames())

				case cmd.BreadcrumbCommand:
					// Go back from command result
					popNavigation() // Remove command
					prevState := peekNavigation()

					if prevState != nil && prevState.Type == cmd.BreadcrumbResource {
						// We're at a command under a resource
						resourceName := prevState.Value
						cmd.UiState.Resource = cmd.Resources[resourceName]

						// If this was the default command, skip back to resources list
						if cmd.UiState.Resource.DefaultCommand != constants.EmptyString &&
							cmd.UiState.Command.Name == cmd.UiState.Resource.DefaultCommand {
							popNavigation() // Remove resource too
							profileName := cmd.UiState.Breadcrumbs[1]
							cmd.UiState.Breadcrumbs = []string{constants.Profiles, profileName}
							cmd.UiState.NavigationStack = []cmd.NavigationState{
								{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
								{Type: cmd.BreadcrumbProfile, Value: profileName},
							}
							Body = createResources(cmd.GetAvailableResourceNames())
						} else {
							// Show command list
							Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
						}
					}

				case cmd.BreadcrumbProcessedJson:
					// Go back from processed JSON to original JSON viewer
					popNavigation()
					cmd.UiState.ProcessedJsonData = nil

					if cmd.UiState.JsonViewerCallback != nil {
						cmd.UiState.JsonViewerCallback()
						return nil
					}

				case cmd.BreadcrumbJsonView:
					// Go back from JSON viewer to command result
					popNavigation()
					cmd.UiState.ProcessedJsonData = nil
					cmd.UiState.JsonViewerCallback = nil

					// Always use cached body when going back from JSON viewer
					// Don't re-run the command as it may hang (e.g., SQS receive-message with wait)
					currentCmdState := peekNavigation()
					if currentCmdState != nil && currentCmdState.CachedBody != nil {
						// Use cached result
						Body = currentCmdState.CachedBody
						logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Using cached result for command: %s", cmd.UiState.Command.Name))
					} else {
						// Fallback: Re-run current command to restore table view
						_, body := executeCommand(cmd.UiState.Command)
						Body = body
					}

				case cmd.BreadcrumbDependentCmd:
					// Go back from dependent command
					popNavigation() // Remove dependent command

					// Check if the previous state was a dependent commands selection list
					prevState := peekNavigation()
					if prevState != nil && prevState.Type == cmd.BreadcrumbDependentCmds {
						// We came from a selection list, go back 2 more levels (list + selected item)
						popNavigation() // Remove dependent commands list
						popNavigation() // Remove selected item
					} else {
						// Direct dependent command (only one option), go back 1 level
						popNavigation() // Remove selected item
					}

					// Get parent command from remaining stack
					parentState := peekNavigation()
					if parentState != nil && parentState.Type == cmd.BreadcrumbCommand {
						parentCommandName := parentState.Value
						cmd.UiState.Command = cmd.UiState.Resource.GetCommand(parentCommandName)

						// Check if we have cached result
						if parentState.CachedBody != nil && !cmd.UiState.Command.RerunOnBack {
							// Use cached result
							Body = parentState.CachedBody
							logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Using cached result for parent command: %s", parentCommandName))
						} else {
							// Re-run parent command
							_, body := executeCommand(cmd.UiState.Command)
							Body = body
						}
					}

				case cmd.BreadcrumbDependentCmds:
					// Go back from dependent commands selection to selected item
					popNavigation() // Remove dependent commands list
					popNavigation() // Remove selected item

					// Get current command state
					currentCmdState := peekNavigation()
					if currentCmdState != nil && currentCmdState.CachedBody != nil && !cmd.UiState.Command.RerunOnBack {
						// Use cached result
						Body = currentCmdState.CachedBody
						logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Using cached result for command: %s", cmd.UiState.Command.Name))
					} else {
						// Re-run current command
						_, body := executeCommand(cmd.UiState.Command)
						Body = body
					}

				case cmd.BreadcrumbSelectedItem:
					// This shouldn't happen as selectedItem should always be followed by dependentCmd or dependentCmds
					// But just in case, go back to parent command
					popNavigation()
					prevState := peekNavigation()
					if prevState != nil && prevState.Type == cmd.BreadcrumbCommand {
						parentCommandName := prevState.Value
						cmd.UiState.Command = cmd.UiState.Resource.GetCommand(parentCommandName)

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
			Rune:        'n',
			Description: "Next Page",
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				// Check if current command has pagination enabled
				if cmd.UiState.Command.Pagination != nil && cmd.UiState.Command.Pagination.Enabled {
					currentNav := peekNavigation()
					if currentNav != nil && currentNav.PaginationToken != "" {
						// Save current token to history
						if cmd.UiState.PageHistory == nil {
							cmd.UiState.PageHistory = []string{}
						}
						cmd.UiState.PageHistory = append(cmd.UiState.PageHistory, cmd.UiState.CurrentPageToken)

						// Set next page token
						cmd.UiState.CurrentPageToken = currentNav.PaginationToken

						// Re-execute command with new token
						_, body := executeCommand(cmd.UiState.Command)
						Body = body
						updateRootView(nil)
					}
				}
				return nil
			},
		},
		{
			Rune:        'p',
			Description: "Previous Page",
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				// Check if we have previous pages
				if len(cmd.UiState.PageHistory) > 0 {
					// Pop previous token
					lastIndex := len(cmd.UiState.PageHistory) - 1
					cmd.UiState.CurrentPageToken = cmd.UiState.PageHistory[lastIndex]
					cmd.UiState.PageHistory = cmd.UiState.PageHistory[:lastIndex]

					// Re-execute command with previous token
					_, body := executeCommand(cmd.UiState.Command)
					Body = body
					updateRootView(nil)
				}
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
