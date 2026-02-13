package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/cmd/profile"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/logger"
	commandParser "github.com/cmd-tools/aws-commander/parser"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// executeCommand runs a command and optionally caches the result
func executeCommand(command cmd.Command) (string, tview.Primitive) {
	// Handle contentView commands (e.g., get-object) that download content to a file
	if command.View == "contentView" {
		return executeContentViewCommand(command)
	}

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

// executeContentViewCommand handles commands that download content to a file (e.g., s3api get-object).
// It creates a temp file, runs the command with the file as output, then renders the content
// as raw text by default. The user can press 'v' to toggle to the pretty/formatted view.
func executeContentViewCommand(command cmd.Command) (string, tview.Primitive) {
	// Determine file extension from the selected object key
	objectKey := cmd.UiState.SelectedItems["$OBJECT"]
	ext := filepath.Ext(objectKey)

	// Create temp file with the correct extension so content type detection works
	tmpFile, err := os.CreateTemp("", "aws-commander-*"+ext)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to create temp file for content download")
		return "", commandParser.CreateErrorView(command.Name, "Failed to create temp file")
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Run the command with the temp file as output destination
	metadataOutput := command.RunToFile(cmd.UiState.Resource.Name, cmd.UiState.Profile, tmpPath)
	logger.Logger.Debug().Str("metadata", metadataOutput).Str("file", tmpPath).Msg("Content downloaded")

	// Read the downloaded file content
	fileContent, err := os.ReadFile(tmpPath)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to read downloaded content")
		os.Remove(tmpPath)
		return "", commandParser.CreateErrorView(command.Name, "Failed to read downloaded content")
	}

	// Clean up temp file
	os.Remove(tmpPath)

	// Store content for 'v' toggle
	cmd.UiState.InContentView = true
	cmd.UiState.ContentViewPretty = false
	cmd.UiState.ContentViewObjectKey = objectKey
	cmd.UiState.ContentViewData = fileContent

	// Default: show raw text view
	body := commandParser.CreateContentView(command.Name, objectKey, fileContent)

	if !command.RerunOnBack {
		updateNavigationCache(metadataOutput, body)
	}

	return metadataOutput, body
}

// executeDependentCommand handles execution of dependent commands
func executeDependentCommand(selectedCommandName string) {
	cmd.UiState.Command = cmd.UiState.Resource.GetCommand(selectedCommandName)
	AutoCompletionWordList = append(cmd.UiState.Resource.GetCommandNames(), constants.Profiles)

	pushNavigation(cmd.BreadcrumbDependentCmd, cmd.UiState.Command.Name)

	// Check if command requires key input (e.g., DynamoDB query)
	if cmd.UiState.Command.RequiresKeyInput {
		cmd.UiState.CommandBarVisible = false
		Search.SetText("")
		cmd.UiState.OriginalTableData = nil
		showKeyInputForm()
		return
	}

	cmd.UiState.CommandBarVisible = false
	Search.SetText("")
	cmd.UiState.OriginalTableData = nil
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

	cmd.UiState.CommandBarVisible = false
	Search.SetText("")
	cmd.UiState.OriginalTableData = nil
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
		cmd.UiState.CommandBarVisible = false
		Search.SetText("")
		cmd.UiState.OriginalTableData = nil
		Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
	} else if len(dependentCommands) == 1 {
		// Only one dependent command, execute it directly
		pushNavigation(cmd.BreadcrumbSelectedItem, selectedItemName)

		cmd.UiState.Command = dependentCommands[0]
		pushNavigation(cmd.BreadcrumbDependentCmd, cmd.UiState.Command.Name)

		// Check if command requires key input (e.g., DynamoDB query)
		if cmd.UiState.Command.RequiresKeyInput {
			cmd.UiState.CommandBarVisible = false
			Search.SetText("")
			cmd.UiState.OriginalTableData = nil
			showKeyInputForm()
			return
		}

		cmd.UiState.CommandBarVisible = false
		Search.SetText("")
		cmd.UiState.OriginalTableData = nil
		_, body := executeCommand(cmd.UiState.Command)
		Body = body
	} else {
		// Multiple dependent commands, show selection list
		pushNavigation(cmd.BreadcrumbSelectedItem, selectedItemName)

		var commandNames []string
		for _, c := range dependentCommands {
			commandNames = append(commandNames, c.Name)
		}

		cmd.UiState.CommandBarVisible = false
		Search.SetText("")
		cmd.UiState.OriginalTableData = nil
		pushNavigation(cmd.BreadcrumbDependentCmds, "Select Command")
		Body = createDependentCommandView(commandNames)
	}

	updateRootView(nil)
}

// defaultKeyCombinations defines the default keyboard shortcuts
func defaultKeyCombinations() []ui.CustomShortCut {
	shortcuts := []ui.CustomShortCut{
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
			Rune:        'y',
			Description: "Copy (Yank)",
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				// Copy is handled in individual UI components
				return event
			},
		},
		{
			Rune:        '?',
			Description: "Help",
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				return nil
			},
		},
		{
			Rune:        'r',
			Description: "Refresh",
			Handle:      handleRerunCommand,
		},
	}

	// Add 'v' shortcut only when viewing DynamoDB items in JSON viewer
	if cmd.UiState.InDynamoDBJsonViewer {
		shortcuts = append(shortcuts, ui.CustomShortCut{
			Rune:        'v',
			Description: "Toggle JSON Format",
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				// Handled in JSON viewer component
				return event
			},
		})
	} else if cmd.UiState.InContentView && commandParser.ContentViewHasPrettyFormat(cmd.UiState.ContentViewObjectKey, cmd.UiState.ContentViewData) {
		description := "Pretty View"
		if cmd.UiState.ContentViewPretty {
			description = "Raw View"
		}
		shortcuts = append(shortcuts, ui.CustomShortCut{
			Rune:        'v',
			Description: description,
			Handle:      handleToggleContentView,
		})
	}

	// Add 'u' shortcut to update SSO role when viewing the profiles table
	currentNav := peekNavigation()
	if currentNav != nil && currentNav.Type == cmd.BreadcrumbProfiles {
		shortcuts = append(shortcuts, ui.CustomShortCut{
			Rune:        'u',
			Description: "Update SSO Role",
			Handle:      handleUpdateSSORole,
		})
	}

	// Add action-based shortcuts from the current command's configuration
	for _, action := range cmd.UiState.Command.Actions {
		actionCopy := action // capture loop variable
		if len(actionCopy.Rune) != 1 {
			continue
		}
		r := rune(actionCopy.Rune[0])

		switch actionCopy.Type {
		case "download":
			shortcuts = append(shortcuts, ui.CustomShortCut{
				Rune:        r,
				Description: actionCopy.Description,
				Handle:      makeDownloadHandler(actionCopy),
			})
		}
	}

	return shortcuts
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
		cmd.UiState.CommandBarVisible = false
		Search.SetText("")
		cmd.UiState.OriginalTableData = nil
		Body = createBody()

	case cmd.BreadcrumbResource:
		popNavigation()
		cmd.UiState.CommandBarVisible = false
		Search.SetText("")
		cmd.UiState.OriginalTableData = nil
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
			cmd.UiState.CommandBarVisible = false
			Search.SetText("")
			cmd.UiState.OriginalTableData = nil
			Body = createResources(cmd.GetAvailableResourceNames())
		} else {
			cmd.UiState.CommandBarVisible = false
			Search.SetText("")
			cmd.UiState.OriginalTableData = nil
			Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
		}
	}
}

// handleProcessedJsonBack navigates back from processed JSON view
func handleProcessedJsonBack() bool {
	popNavigation()
	cmd.UiState.ProcessedJsonData = nil

	if cmd.UiState.JsonViewerCallback != nil {
		cmd.UiState.CommandBarVisible = false
		Search.SetText("")
		cmd.UiState.OriginalTableData = nil
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
	cmd.UiState.InDynamoDBJsonViewer = false

	cmd.UiState.CommandBarVisible = false
	Search.SetText("")
	cmd.UiState.OriginalTableData = nil

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

	// Clear content view state when leaving a content view
	cmd.UiState.InContentView = false
	cmd.UiState.ContentViewPretty = false
	cmd.UiState.ContentViewObjectKey = ""
	cmd.UiState.ContentViewData = nil

	prevState := peekNavigation()
	if prevState != nil && prevState.Type == cmd.BreadcrumbDependentCmds {
		popNavigation()
		popNavigation()
	} else {
		popNavigation()
	}

	cmd.UiState.CommandBarVisible = false
	Search.SetText("")
	cmd.UiState.OriginalTableData = nil

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

	cmd.UiState.CommandBarVisible = false
	Search.SetText("")
	cmd.UiState.OriginalTableData = nil

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
	cmd.UiState.CommandBarVisible = false
	Search.SetText("")
	cmd.UiState.OriginalTableData = nil

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
		if currentNav != nil {
			// Check if we have a token-based pagination (like list-queues)
			if currentNav.PaginationToken != "" {
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
			} else if cmd.UiState.Command.Pagination.NextTokenJsonPath == "" {
				// Token-less pagination (like receive-message): just re-execute to get next batch
				// Save current state to history (use empty string as marker)
				if cmd.UiState.PageHistory == nil {
					cmd.UiState.PageHistory = []string{}
				}
				cmd.UiState.PageHistory = append(cmd.UiState.PageHistory, cmd.UiState.CurrentPageToken)
				// Reset token to empty for next execution
				cmd.UiState.CurrentPageToken = ""

				// Re-execute command to fetch next batch
				_, body := executeCommand(cmd.UiState.Command)
				Body = body
				updateRootView(nil)
			}
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

// handleRerunCommand re-executes the current command
func handleRerunCommand(event *tcell.EventKey) *tcell.EventKey {
	if cmd.UiState.Command.Name == "" {
		return nil
	}

	logger.Logger.Debug().Msg(fmt.Sprintf("Re-running command: %s", cmd.UiState.Command.Name))

	cmd.UiState.CommandBarVisible = false
	Search.SetText("")
	cmd.UiState.OriginalTableData = nil

	_, body := executeCommand(cmd.UiState.Command)
	Body = body
	updateRootView(nil)
	App.SetFocus(Body)

	if boxed, ok := Body.(ui.Boxed); ok {
		ui.ShowToast(App, boxed, ui.ToastRefreshMessage)
	}

	return nil
}

// handleToggleContentView toggles between raw and pretty/formatted views
// for S3 object content. It rebuilds the view from the stored content bytes.
func handleToggleContentView(event *tcell.EventKey) *tcell.EventKey {
	cmd.UiState.ContentViewPretty = !cmd.UiState.ContentViewPretty

	var newView tview.Primitive
	if cmd.UiState.ContentViewPretty {
		newView = commandParser.CreatePrettyContentView(cmd.UiState.Command.Name, cmd.UiState.ContentViewObjectKey, cmd.UiState.ContentViewData)
	} else {
		newView = commandParser.CreateContentView(cmd.UiState.Command.Name, cmd.UiState.ContentViewObjectKey, cmd.UiState.ContentViewData)
	}

	Body = newView
	updateRootView(nil)
	App.SetFocus(Body)
	return nil
}

// handleUpdateSSORole handles the 'u' shortcut to update the SSO role for the selected profile.
// It fetches available roles from the SSO session and shows a list for the user to pick from.
func handleUpdateSSORole(event *tcell.EventKey) *tcell.EventKey {
	table, ok := Body.(*tview.Table)
	if !ok {
		return nil
	}

	row, _ := table.GetSelection()
	if row < 1 {
		return nil
	}

	profileName := table.GetCell(row, 0).Text
	if profileName == "" {
		return nil
	}

	p := ProfileList.FindProfile(profileName)
	if p == nil {
		logger.Logger.Error().Str("profile", profileName).Msg("Profile not found")
		return nil
	}

	if p.SSO.AccountId == "" || p.SSO.AccountId == "n/a" {
		logger.Logger.Debug().Str("profile", profileName).Msg("Profile is not an SSO profile, skipping")
		if boxed, ok := Body.(ui.Boxed); ok {
			ui.ShowToast(App, boxed, " Not an SSO profile ")
		}
		return nil
	}

	// Save the current body so we can restore it on cancel
	cachedBody := Body

	roles, err := profile.ListAccountRoles(p)
	if err != nil {
		logger.Logger.Error().Err(err).Str("profile", profileName).Msg("Failed to list SSO roles")
		if boxed, ok := Body.(ui.Boxed); ok {
			ui.ShowToast(App, boxed, " Failed to list roles ")
		}
		return nil
	}

	if len(roles) == 0 {
		if boxed, ok := Body.(ui.Boxed); ok {
			ui.ShowToast(App, boxed, " No roles found ")
		}
		return nil
	}

	// Show role selection list
	roleList := ui.CreateCustomListView(ui.ListViewBoxProperties{
		Title:   fmt.Sprintf(" SSO Roles for %s [%d] ", profileName, len(roles)),
		Options: roles,
		Handler: func(selectedRole string) {
			profile.UpdateSSORole(profileName, selectedRole)
			logger.Logger.Debug().
				Str("profile", profileName).
				Str("role", selectedRole).
				Msg("Updated SSO role")

			// Refresh the profile list and rebuild the profiles view
			ProfileList = profile.GetList()
			Body = createBody()
			updateRootView(nil)
			App.SetFocus(Body)

			if boxed, ok := Body.(ui.Boxed); ok {
				ui.ShowToast(App, boxed, " Role updated! ")
			}
		},
		App: App,
	})

	// Override the list's input capture to handle ESC back to profiles
	roleList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			Body = cachedBody
			updateRootView(nil)
			App.SetFocus(Body)
			return nil
		}
		return event
	})

	Body = roleList
	updateRootView(nil)
	App.SetFocus(Body)
	return nil
}

// makeDownloadHandler returns a handler for "download" type actions.
// It reads the target command from the action config, shows a save dialog,
// and downloads the object using RunToFile.
func makeDownloadHandler(action cmd.Action) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		table, ok := Body.(*tview.Table)
		if !ok {
			return nil
		}

		row, _ := table.GetSelection()
		if row < 1 {
			return nil
		}

		objectKey := table.GetCell(row, 0).Text
		if objectKey == "" {
			return nil
		}

		// Build default download path: <cwd>/<filename>
		cwd, err := os.Getwd()
		if err != nil {
			cwd = "."
		}
		fileName := filepath.Base(objectKey)
		defaultPath := filepath.Join(cwd, fileName)

		// Store the cached body so we can restore it after the form
		cachedBody := Body

		// Set the selected item variable for the resource being acted on
		resourceName := cmd.VariablePlaceHolderPrefix + strings.ToUpper(cmd.UiState.Command.ResourceName)
		cmd.UiState.SelectedItems[resourceName] = objectKey

		// Show download form
		downloadForm := ui.CreateInputForm(ui.InputFormProperties{
			Title: fmt.Sprintf(" Download: %s ", objectKey),
			Fields: []ui.InputField{
				{
					Label:        "Save to",
					Key:          "path",
					DefaultValue: defaultPath,
				},
			},
			OnSubmit: func(values map[string]string) {
				destPath := values["path"]
				if destPath == "" {
					destPath = defaultPath
				}

				// Ensure the destination directory exists
				destDir := filepath.Dir(destPath)
				if err := os.MkdirAll(destDir, 0755); err != nil {
					logger.Logger.Error().Err(err).Msg("Failed to create destination directory")
					Body = cachedBody
					updateRootView(nil)
					App.SetFocus(Body)
					return
				}

				// Resolve and run the target command
				targetCmd := cmd.UiState.Resource.GetCommand(action.TargetCommand)
				targetCmd.RunToFile(cmd.UiState.Resource.Name, cmd.UiState.Profile, destPath)

				logger.Logger.Debug().
					Str("key", objectKey).
					Str("path", destPath).
					Str("targetCommand", action.TargetCommand).
					Msg("Downloaded object via action")

				// Restore the previous view
				Body = cachedBody
				updateRootView(nil)
				App.SetFocus(Body)

				if boxed, ok := Body.(ui.Boxed); ok {
					ui.ShowToast(App, boxed, " Downloaded! ")
				}
			},
			OnCancel: func() {
				Body = cachedBody
				updateRootView(nil)
				App.SetFocus(Body)
			},
			App: App,
		})

		Body = downloadForm
		updateRootView(nil)
		App.SetFocus(downloadForm)

		return nil
	}
}
