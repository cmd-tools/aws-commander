package main

import (
	"fmt"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/cmd/profile"
	"github.com/cmd-tools/aws-commander/helpers"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/common-nighthawk/go-figure"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var SelectedCommand string
var TableOutput *tview.TextView
var ProfileList *tview.List
var Form *tview.Form
var CommandList []string
var CommandName = ""
var ProfileName = ""
var Resource = "dynamodb"
var App *tview.Application

func ChooseCommand(option string, optionIndex int) {
	if len(CommandList) != 0 {
		CommandName = option
		resource := cmd.Resources[Resource]
		command := resource.GetCommand(CommandName)
		TableOutput.Clear()
		fmt.Fprintf(TableOutput, "%s", command.Run(resource.Name, ProfileName))
	}
}

func ChooseProfile(option string, optionIndex int) {
	ProfileName = option
	logger.Logger.Debug().Msg(fmt.Sprintf("Selected profile: %s", option))
}

func ChooseResource(option string, optionIndex int) {
	Resource = option
	resource := cmd.Resources[Resource]
	CommandList = resource.GetCommandNames()
	if Form != nil {
		dropDown := Form.GetFormItemByLabel("Command").(*tview.DropDown)
		dropDown.SetOptions(CommandList, nil)
		dropDown.SetSelectedFunc(ChooseCommand)
	}
	logger.Logger.Debug().Msg(fmt.Sprintf("Selected resource: %s", option))
}

func main() {

	logger.InitLog()

	logger.Logger.Info().Msg("Starting aws-commander")

	logger.Logger.Debug().Msg("Loading configurations")

	cmd.Init()

	figure := figure.NewFigure("AWS Commander", "puffy", false)

	App = tview.NewApplication()

	asciiTitle := figure.String()
	title := tview.NewTextView().SetText(asciiTitle[:len(asciiTitle)-1] + " v0.0.1").SetDynamicColors(true).SetTextAlign(tview.AlignLeft).SetTextColor(tcell.Color100)

	TableOutput = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			App.Draw()
		})

	TableOutput.SetTitle("Command Result").SetBorder(true)

	Form = tview.NewForm().
		AddDropDown("Profile", profile.GetList(), 0, ChooseProfile).
		AddDropDown("Resource", cmd.GetAvailableResourceNames(), 0, ChooseResource).
		AddDropDown("Command", CommandList, 0, ChooseCommand)
	Form.SetBorder(true).SetTitle(fmt.Sprintf("Choose your option (aws cli: %s)", helpers.GetAWSClientVersion())).SetTitleAlign(tview.AlignLeft)
	Form.SetFocus(0)

	mainFlexPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	columns := tview.NewFlex().SetDirection(tview.FlexColumn)

	mainFlexPanel.AddItem(title, 0, 1, false)

	columns.
		AddItem(Form, 0, 1, true).
		AddItem(TableOutput, 0, 1, false)

	mainFlexPanel.AddItem(columns, 0, 3, false)

	if err := App.SetRoot(mainFlexPanel, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
