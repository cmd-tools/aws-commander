package main

import (
	"fmt"
	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/cmd/profile"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/helpers"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/gdamore/tcell/v2"
	_ "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"strings"
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

	mainFlexPanel := UpdateRootView([]string{constants.Profiles})

	SetSearchBarListener(mainFlexPanel)

	if err := App.SetRoot(mainFlexPanel, true).EnableMouse(false).Run(); err != nil {
		panic(err)
	}
}

func CreateHeader() *tview.Table {
	header := tview.NewTable().
		SetBorders(false).
		SetSelectable(false, false)

	header.SetCell(0, 0, tview.NewTableCell("Aws CLI Rev:").SetTextColor(tcell.ColorGold))
	header.SetCell(0, 1, tview.NewTableCell(helpers.GetAWSClientVersion()).SetTextColor(tcell.ColorWhite))
	header.SetCell(1, 0, tview.NewTableCell("Aws Commander Rev:").SetTextColor(tcell.ColorGold))
	// TODO: get AWS commander version from somewhere else?
	header.SetCell(1, 1, tview.NewTableCell("v0.0.1").SetTextColor(tcell.ColorWhite))

	header.SetBorderPadding(1, 1, 1, 1)
	return header
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
		header.SetCell(0, i*2, tview.NewTableCell(fmt.Sprintf(" <%s>", section)).
			SetBackgroundColor(sectionBackgroundColorTags[i]).
			SetTextColor(sectionForegroundColorTags[i]).
			SetAlign(tview.AlignCenter))
	}

	header.SetBorderPadding(0, 1, 1, 1)
	return header
}

func CreateBody() *tview.Table {
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
			helpers.SetSelectedProfile(selectedProfileName)

			resources := cmd.GetAvailableResourceNames()

			AutoCompletionWordList = []string{constants.Profiles}

			Body = ui.CreateCustomListView(ui.ListViewBoxProperties{
				Title:   fmt.Sprintf(" Resources [%d] ", len(resources)),
				Options: resources,
				Handler: func(selectedResourceName string) {
					helpers.SetSelectedResource(selectedResourceName)

					resource := cmd.Resources[selectedResourceName]

					commandNames := resource.GetCommandNames()

					AutoCompletionWordList = append(resources, constants.Profiles)

					Body = ui.CreateCustomListView(ui.ListViewBoxProperties{
						Title:   fmt.Sprintf(" Commands [%d] ", len(commandNames)),
						Options: commandNames,
						Handler: func(selectedCommandName string) {
							helpers.SetSelectedCommand(selectedCommandName)
							command := resource.GetCommand(selectedCommandName)

							AutoCompletionWordList = append(commandNames, constants.Profiles)

							Body = tview.NewTextView().
								SetText(command.Run(selectedResourceName, selectedProfileName))

							UpdateRootView([]string{constants.Profiles, selectedProfileName, selectedResourceName, constants.OutPut})
						},
					})
					UpdateRootView([]string{constants.Profiles, selectedProfileName, selectedResourceName})
				},
			})
			UpdateRootView([]string{constants.Profiles, selectedProfileName})
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

	return searchBar
}

func UpdateRootView(navigationStrings []string) *tview.Flex {
	view := tview.
		NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(CreateHeader(), 4, 2, false).
		AddItem(Search, 3, 2, false).
		AddItem(Body, 0, 1, true).
		AddItem(CreateFooter(navigationStrings), 2, 2, false)
	App.SetRoot(view, true)
	return view
}

func SetSearchBarListener(mainFlex *tview.Flex) {

	App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == ':' {
			logger.Logger.Debug().Msg(fmt.Sprintf("Entered text: %s", Search.GetText()))
			// TODO: search bar should appear if user press ':'
			App.SetFocus(Search)
			return nil
		}

		if event.Key() == tcell.KeyESC {
			Search.SetText(constants.EmptyString)
			logger.Logger.Debug().Msg("ESC")
			// TODO: enable once found a way to add the search bar dynamically
			//mainFlex.RemoveItem(Search)
			App.SetFocus(Body)
			return nil
		}

		if event.Key() == tcell.KeyEnter && Search.HasFocus() {
			logger.Logger.Debug().Msg("ENTER")

			if Search.GetText() == constants.Profiles {
				Body = CreateBody()
				UpdateRootView([]string{constants.Profiles})
			}

			Search.SetText(constants.EmptyString)

			return nil
		}
		return event
	})
}
