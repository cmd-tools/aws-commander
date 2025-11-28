package view

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var MainFlexPanel *tview.Flex
var TableOutput *tview.TextView
var ProfileList *tview.List
var Form *tview.Form
var App *tview.Application

func CreateMainFlexPanel() *tview.Flex {
	MainFlexPanel = tview.NewFlex().SetDirection(tview.FlexRow)
	return MainFlexPanel
}

func CreateTextView(text string, align int, color tcell.Color) *tview.TextView {
	title := tview.NewTextView().SetText(text).SetDynamicColors(true).SetTextAlign(align).SetTextColor(color)
	return title
}

func AddTextView(textView *tview.TextView, fixedSize int, proportion int, focus bool) {
	MainFlexPanel.AddItem(textView, 0, 1, false)
}

func AddForm(title string, align int) *tview.Form {
	Form = tview.NewForm()
	Form.SetBorder(true).SetTitle(title).SetTitleAlign(align)
	Form.SetFocus(0)
	return Form
}

func AddCmd(label string, commands []string) {
	inputField := tview.NewInputField().
		SetLabel(label).
		SetFieldWidth(30).
		SetDoneFunc(func(key tcell.Key) {
			// app.Stop()
		})
	inputField.SetAutocompleteFunc(func(currentText string) (entries []string) {
		if len(currentText) == 0 {
			return
		}
		for _, word := range commands {
			if strings.HasPrefix(strings.ToLower(word), strings.ToLower(currentText)) {
				entries = append(entries, word)
			}
		}
		if len(entries) <= 1 {
			entries = nil
		}
		return entries
	})
	inputField.SetAutocompletedFunc(func(text string, index, source int) bool {
		if source != tview.AutocompletedNavigate {
			inputField.SetText(text)
		}
		return source == tview.AutocompletedEnter || source == tview.AutocompletedClick
	})
	MainFlexPanel.AddItem(inputField, 0, 1, false)
}
