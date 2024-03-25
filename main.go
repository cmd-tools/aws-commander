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
	"github.com/rivo/tview"
)

var App *tview.Application

var Header tview.Box
var Search *tview.InputField
var Body tview.Primitive
var Footer tview.Box

func main() {

	logger.InitLog()

	logger.Logger.Info().Msg("Starting aws-commander")

	logger.Logger.Debug().Msg("Loading configurations")

	cmd.Init()

	App = tview.NewApplication()

	Search = tview.NewInputField()
	Search.SetLabel("️🔍 > ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetBorder(true).
		SetBackgroundColor(tcell.ColorDefault)

	profiles := profile.GetList()

	Body = ui.CreateCustomTableView(ui.CustomTableViewProperties{
		Title: fmt.Sprintf(" Profiles [%d] ", len(profiles)),
		Columns: []ui.Column{
			{Name: "NAME", Width: 0},
			{Name: "REGION", Width: 0},
			{Name: "SSO_REGION", Width: 0},
			{Name: "SSO_ROLE_NAME", Width: 0},
			{Name: "SSO_ACCOUNT_ID", Width: 0},
			{Name: "SSO_START_URL", Width: 0},
		},
		Rows: profiles.AsMatrix(),
		Handler: func(selectedProfileName string) {
			helpers.SetSelectedProfile(selectedProfileName)

			resources := cmd.GetAvailableResourceNames()

			Body = ui.CreateCustomListView(ui.ListViewBoxProperties{
				Title:   fmt.Sprintf(" Resources [%d] ", len(resources)),
				Options: resources,
				Handler: func(selectedResourceName string) {
					helpers.SetSelectedResource(selectedResourceName)

					commands := cmd.Resources[selectedResourceName]

					l := commands.GetCommandNames()

					Body = ui.CreateCustomListView(ui.ListViewBoxProperties{
						Title:   fmt.Sprintf(" Commands [%d] ", len(l)),
						Options: l,
						Handler: func(selectedCommandName string) {
							helpers.SetSelectedCommand(selectedCommandName)
							command := commands.GetCommand(selectedCommandName)
							Body = tview.NewTextView().
								SetText(command.Run(selectedResourceName, selectedProfileName))

							view := tview.
								NewFlex().
								SetDirection(tview.FlexRow).
								AddItem(CreateHeader(), 4, 2, false).
								AddItem(Search, 3, 2, false).
								AddItem(Body, 0, 1, true).
								AddItem(CreateFooter([]string{constants.Profiles, selectedProfileName, selectedResourceName, "output"}), 2, 2, false)

							App.SetRoot(view, true)

						},
					})

					view := tview.
						NewFlex().
						SetDirection(tview.FlexRow).
						AddItem(CreateHeader(), 4, 2, false).
						AddItem(Search, 3, 2, false).
						AddItem(Body, 0, 1, true).
						AddItem(CreateFooter([]string{constants.Profiles, selectedProfileName, selectedResourceName}), 2, 2, false)

					App.SetRoot(view, true)

				},
			})

			view := tview.
				NewFlex().
				SetDirection(tview.FlexRow).
				AddItem(CreateHeader(), 4, 2, false).
				AddItem(Search, 3, 2, false).
				AddItem(Body, 0, 1, true).
				AddItem(CreateFooter([]string{constants.Profiles, selectedProfileName}), 2, 2, false)

			App.SetRoot(view, true)
		},
	})

	mainFlexPanel := tview.NewFlex()
	mainFlexPanel.
		SetDirection(tview.FlexRow).
		AddItem(CreateHeader(), 4, 2, false).
		AddItem(Search, 3, 2, false).
		AddItem(Body, 0, 1, true).
		AddItem(CreateFooter([]string{constants.Profiles}), 2, 2, false)

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
			modal := ui.CreateModal(ui.ModalProperties{
				Title: fmt.Sprintf("I thought I could search `%s` too, but it's not implemented 😔", Search.GetText()),
				LeftChoice: ui.ModalChoice{
					Name: "Ok",
					Handler: func(previousFlex *tview.Flex) {
						App.Stop()
					},
				},
				RightChoice: ui.ModalChoice{
					Name: "Cancel",
					Handler: func(previousFlex *tview.Flex) {
						// TODO: go back to mainFlex
						App.Stop()
					},
				},
			}, mainFlex)
			Search.SetText(constants.EmptyString)
			App.SetRoot(modal, false)

			return nil
		}
		return event
	})
}
