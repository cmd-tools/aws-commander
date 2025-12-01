package ui

import (
	"github.com/atotto/clipboard"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sort"
)

type ListViewBoxProperties struct {
	Title   string
	Options []string
	Handler func(selectedOption string)
}

func CreateCustomListView(properties ListViewBoxProperties) *tview.List {
	list := tview.NewList()

	list.SetTitle(properties.Title).
		SetBorder(true).
		SetTitleAlign(tview.AlignCenter).
		SetBorderPadding(1, 1, 2, 2).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	// Remove secondary text
	list.ShowSecondaryText(false)

	sort.Strings(properties.Options)

	for _, option := range properties.Options {
		list.AddItem(option, constants.EmptyString, constants.EmptyRune, nil)
	}

	list.SetSelectedFunc(func(i int, s string, s2 string, r rune) {
		properties.Handler(s)
	})

	// Add input capture for clipboard copy
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle clipboard copy with 'y' (yank) or Ctrl+C
		if event.Rune() == 'y' || event.Key() == tcell.KeyCtrlC {
			currentIndex := list.GetCurrentItem()
			if currentIndex >= 0 && currentIndex < len(properties.Options) {
				itemText, _ := list.GetItemText(currentIndex)
				
				err := clipboard.WriteAll(itemText)
				if err != nil {
					logger.Logger.Error().Err(err).Msg("Failed to copy to clipboard")
				} else {
					logger.Logger.Debug().Str("data", itemText).Msg("Copied to clipboard")
				}
			}
			return nil
		}
		return event
	})

	return list
}
