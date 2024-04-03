package main

import (
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
	_ "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var App *tview.Application

var Header tview.Box
var Search *tview.InputField
var Body tview.Primitive
var AutoCompletionWordList []string
var ProfileList profile.Profiles
var Footer tview.Box

func main() {

	logger.InitLog()

	logger.Logger.Info().Msg("Starting aws-commander")

	logger.Logger.Debug().Msg("Loading configurations")

	cmd.Init()

	App = tview.NewApplication()

	Search = CreateSearchBar()

	ProfileList = profile.GetList()

	Body = CreateBody()

	mainFlexPanel := UpdateRootView(nil)

	if err := App.SetRoot(mainFlexPanel, true).EnableMouse(false).Run(); err != nil {
		panic(err)
	}
}

func CreateHeader(keyCombs []ui.CustomShortCut) *tview.Flex {

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
		Shortcuts: append(keyCombs, DefaultKeyCombinations()...),
	})

	flex.AddItem(header, 0, 2, false).
		AddItem(shortcuts, 0, 4, false).
		AddItem(CreateLogo(), 0, 2, false)

	return flex
}

func CreateFooter(sections []string) *tview.Table {

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

func CreateBody() *tview.Table {
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
			UpdateRootView(nil)
		},
	})
}

func CreateSearchBar() *tview.InputField {
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
			logger.Logger.Debug().Msg("ESC")
			cmd.UiState.CommandBarVisible = false
			Search.SetText(constants.EmptyString)
			UpdateRootView(nil)
			return nil
		}

		if event.Key() == tcell.KeyEnter && Search.HasFocus() {
			logger.Logger.Debug().Msg("ENTER")

			switch Search.GetText() {
			case constants.Profiles:
				Body = CreateBody()
			case constants.Resources:
				Body = createResources(cmd.GetAvailableResourceNames())
			}

			UpdateRootView(nil)
			Search.SetText(constants.EmptyString)
			return nil
		}
		return event
	})

	return searchBar
}

func CreateLogo() *tview.TextView {
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

func UpdateRootView(shortcuts []ui.CustomShortCut) *tview.Flex {
	view := tview.
		NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(CreateHeader(shortcuts), 7, 2, false)

	if cmd.UiState.CommandBarVisible {
		view.AddItem(Search, 3, 2, false)
	}

	view.AddItem(Body, 0, 1, true).
		AddItem(CreateFooter(cmd.UiState.Breadcrumbs), 2, 2, false)

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
			if cmd.UiState.Resource.DefaultCommand == "" {
				Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
			} else {
				cmd.UiState.Command = cmd.UiState.Resource.GetCommand(cmd.UiState.Resource.DefaultCommand)
				cmd.UiState.Breadcrumbs = []string{constants.Profiles, cmd.UiState.Profile, cmd.UiState.Resource.Name, cmd.UiState.Command.Name}
				var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
				Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, ItemHandler)
			}

			UpdateRootView(nil)
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

	var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
	Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, ItemHandler)

	cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, cmd.UiState.Command.Name)
	UpdateRootView(nil)
}

func ItemHandler(selectedItemName string) {
	resourceName := "$" + strings.ToUpper(cmd.UiState.Command.ResourceName)
	cmd.UiState.SelectedItems[resourceName] = selectedItemName

	AutoCompletionWordList = append(cmd.UiState.Resource.GetCommandNames(), constants.Profiles)
	if cmd.UiState.Command.DefaultCommand == "" {
		Body = createCommandView(cmd.UiState.Resource.GetCommandNames())
	} else {
		cmd.UiState.Command = cmd.UiState.Resource.GetCommand(cmd.UiState.Command.DefaultCommand)
		cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, cmd.UiState.Command.Name)
		var commandParsed = commandParser.ParseCommand(cmd.UiState.Command, cmd.UiState.Command.Run(cmd.UiState.Resource.Name, cmd.UiState.Profile))
		Body = commandParser.ParseToObject(cmd.UiState.Command.View, commandParsed, ItemHandler)
	}

	UpdateRootView(nil)
}

func DefaultKeyCombinations() []ui.CustomShortCut {
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

				switch len(cmd.UiState.Breadcrumbs) {
				case 0, 1, 2:
					Body = CreateBody()
				case 3, 4:
					Body = createResources(cmd.GetAvailableResourceNames())
				default:
					createExecuteCommandView(cmd.UiState.Breadcrumbs[len(cmd.UiState.Breadcrumbs)-2])
					cmd.UiState.Breadcrumbs = cmd.UiState.Breadcrumbs[:len(cmd.UiState.Breadcrumbs)-2]
				}
				UpdateRootView(nil)
				App.SetFocus(Body)
				return nil
			},
		},
		{
			Rune:        ':',
			Description: "Search",
			Handle: func(event *tcell.EventKey) *tcell.EventKey {
				cmd.UiState.CommandBarVisible = true
				UpdateRootView(nil)
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
