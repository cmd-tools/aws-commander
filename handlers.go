package main

import (
	"fmt"
	"strings"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/logger"
	commandParser "github.com/cmd-tools/aws-commander/parser"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// executeCommand runs a command and optionally caches the result
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

// executeDependentCommand handles execution of dependent commands
func executeDependentCommand(selectedCommandName string) {
	cmd.UiState.Command = cmd.UiState.Resource.GetCommand(selectedCommandName)
	AutoCompletionWordList = append(cmd.UiState.Resource.GetCommandNames(), constants.Profiles)

	pushNavigation(cmd.BreadcrumbDependentCmd, cmd.UiState.Command.Name)

	// Check if command requires key input (e.g., DynamoDB query)
	if cmd.UiState.Command.RequiresKeyInput {
		showKeyInputForm()
		return
	}

	_, body := executeCommand(cmd.UiState.Command)
	Body = body

	updateRootView(nil)
}

// createExecuteCommandView handles command selection and execution
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

// itemHandler handles item selection from command results
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

		// Check if command requires key input (e.g., DynamoDB query)
		if cmd.UiState.Command.RequiresKeyInput {
			showKeyInputForm()
			return
		}

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

// defaultKeyCombinations defines the default keyboard shortcuts
func defaultKeyCombinations() []ui.CustomShortCut {
	return []ui.CustomShortCut{
		{
			Name:        "esc",
			Key:         tcell.KeyEsc,
			Description: "Back",
			Rune:        -1,
			Handle:      handleEscKey,
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
			Handle:      handleNextPage,
		},
		{
			Rune:        'p',
			Description: "Previous Page",
			Handle:      handlePreviousPage,
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

// handleEscKey processes ESC key navigation
func handleEscKey(event *tcell.EventKey) *tcell.EventKey {
	if Search.HasFocus() {
		return event
	}

	currentState := peekNavigation()
	if currentState == nil {
		return nil
	}

	logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Current state: %s = %s, Stack length: %d", currentState.Type, currentState.Value, len(cmd.UiState.NavigationStack)))

	switch currentState.Type {
	case cmd.BreadcrumbProfiles:
		return nil

	case cmd.BreadcrumbProfile:
		popNavigation()
		Body = createBody()

	case cmd.BreadcrumbResource:
		popNavigation()
		Body = createResources(cmd.GetAvailableResourceNames())

	case cmd.BreadcrumbCommand:
		handleCommandBack()

	case cmd.BreadcrumbProcessedJson:
		if handleProcessedJsonBack() {
			return nil
		}

	case cmd.BreadcrumbJsonView:
		handleJsonViewBack()

	case cmd.BreadcrumbDependentCmd:
		handleDependentCommandBack()

	case cmd.BreadcrumbDependentCmds:
		handleDependentCommandsBack()

	case cmd.BreadcrumbSelectedItem:
		handleSelectedItemBack()
	}

	updateRootView(nil)
	App.SetFocus(Body)
	return nil
}

// handleCommandBack navigates back from a command result
func handleCommandBack() {
	popNavigation()
	prevState := peekNavigation()

	if prevState != nil && prevState.Type == cmd.BreadcrumbResource {
		resourceName := prevState.Value
		cmd.UiState.Resource = cmd.Resources[resourceName]

		// If this was the default command, skip back to resources list
		if cmd.UiState.Resource.DefaultCommand != constants.EmptyString &&
			cmd.UiState.Command.Name == cmd.UiState.Resource.DefaultCommand {
			popNavigation()
			profileName := cmd.UiState.Breadcrumbs[1]
			cmd.UiState.Breadcrumbs = []string{constants.Profiles, profileName}
			cmd.UiState.NavigationStack = []cmd.NavigationState{
				{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
				{Type: cmd.BreadcrumbProfile, Value: profileName},
			}
			Body = createResources(cmd.GetAvailableResourceNames())
		} else {
			Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
		}
	}
}

// handleProcessedJsonBack navigates back from processed JSON view
func handleProcessedJsonBack() bool {
	popNavigation()
	cmd.UiState.ProcessedJsonData = nil

	if cmd.UiState.JsonViewerCallback != nil {
		cmd.UiState.JsonViewerCallback()
		return true // Signal to return early from ESC handler
	}
	return false
}

// handleJsonViewBack navigates back from JSON viewer
func handleJsonViewBack() {
	popNavigation()
	cmd.UiState.ProcessedJsonData = nil
	cmd.UiState.JsonViewerCallback = nil

	currentCmdState := peekNavigation()
	if currentCmdState != nil && currentCmdState.CachedBody != nil {
		Body = currentCmdState.CachedBody
		logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Using cached result for command: %s", cmd.UiState.Command.Name))
	} else {
		_, body := executeCommand(cmd.UiState.Command)
		Body = body
	}
}

// handleDependentCommandBack navigates back from a dependent command
func handleDependentCommandBack() {
	popNavigation()

	prevState := peekNavigation()
	if prevState != nil && prevState.Type == cmd.BreadcrumbDependentCmds {
		popNavigation()
		popNavigation()
	} else {
		popNavigation()
	}

	parentState := peekNavigation()
	if parentState != nil && (parentState.Type == cmd.BreadcrumbCommand || parentState.Type == cmd.BreadcrumbDependentCmd) {
		parentCommandName := parentState.Value
		cmd.UiState.Command = cmd.UiState.Resource.GetCommand(parentCommandName)

		if parentState.CachedBody != nil && !cmd.UiState.Command.RerunOnBack {
			Body = parentState.CachedBody
			logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Using cached result for parent command: %s", parentCommandName))
		} else {
			_, body := executeCommand(cmd.UiState.Command)
			Body = body
		}
	}
}

// handleDependentCommandsBack navigates back from dependent commands selection
func handleDependentCommandsBack() {
	popNavigation()
	popNavigation()

	currentCmdState := peekNavigation()
	if currentCmdState != nil && currentCmdState.CachedBody != nil && !cmd.UiState.Command.RerunOnBack {
		Body = currentCmdState.CachedBody
		logger.Logger.Debug().Msg(fmt.Sprintf("[ESC] Using cached result for command: %s", cmd.UiState.Command.Name))
	} else {
		_, body := executeCommand(cmd.UiState.Command)
		Body = body
	}
}

// handleSelectedItemBack navigates back from a selected item
func handleSelectedItemBack() {
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

// handleNextPage handles pagination to next page
func handleNextPage(event *tcell.EventKey) *tcell.EventKey {
	// Don't handle if an input form has focus
	if App.GetFocus() != Body {
		return event
	}

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
}

// handlePreviousPage handles pagination to previous page
func handlePreviousPage(event *tcell.EventKey) *tcell.EventKey {
	// Don't handle if an input form has focus
	if App.GetFocus() != Body {
		return event
	}

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
}
