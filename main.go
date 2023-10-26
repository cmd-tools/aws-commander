package main

import (
	"fmt"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/rivo/tview"
)

var SelectedCommand string
var TableOutput *tview.TextView

func ChooseOption(option string, optionIndex int) {
	if optionIndex != 0 {
		switch option {
		case "List":
			{
				ListOption()
			}
		}
	}
}

func ListOption() {
	command := "aws"
	args := []string{"dynamodb", "list-tables", "--no-paginate", "--output", "json", "--profile", "profileName"}
	out := cmd.ExecCommand(command, args)
	fmt.Fprintf(TableOutput, "%s ", out)
}

func main() {
	app := tview.NewApplication()

	TableOutput = tview.NewTextView().
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		SetChangedFunc(func() {
			app.Draw()
		})
	TableOutput.SetTitle("Command Result").SetBorder(true)

	form := tview.NewForm().
		AddDropDown("Command", []string{"Choose", "List", "Query"}, 0, ChooseOption)
	form.SetBorder(true).SetTitle("Choose your option").SetTitleAlign(tview.AlignLeft)
	form.SetFocus(0)

	flex := tview.NewFlex().
		AddItem(form, 0, 1, true).
		AddItem(TableOutput, 0, 3, false)

	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
