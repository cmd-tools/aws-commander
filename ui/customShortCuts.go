package ui

import (
	"fmt"
	"github.com/cmd-tools/aws-commander/helpers"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sort"
)

type CustomShortCut struct {
	Name        string
	Description string
	Rune        rune
	Key         tcell.Key
	Handle      func(event *tcell.EventKey) *tcell.EventKey
}

type CustomShortCutProperties struct {
	Shortcuts []CustomShortCut
}

func CreateCustomShortCutsView(App *tview.Application, properties CustomShortCutProperties) *tview.Table {

	sort.Sort(properties)

	table := tview.NewTable()

	table.
		SetBorderPadding(0, 0, 0, 0).
		SetBorder(false)

	table.SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorGold))
	k := 0
	maxRows := 6
	maxColumn := 6

	itemListCount := len(properties.Shortcuts)

	j := 0
	for j < maxRows && k < itemListCount {
		i := 0
		for i < maxColumn && k < itemListCount {
			key := properties.Shortcuts[k]

			name := key.Name

			if helpers.IsStringEmpty(name) {
				name = string(key.Rune)
			}

			cellForKeyComb := tview.NewTableCell(fmt.Sprintf("<%s>", name)).
				SetAlign(tview.AlignLeft).
				SetMaxWidth(0).
				SetTextColor(tcell.ColorGold).
				SetSelectable(false)

			cellForName := tview.NewTableCell(key.Description).
				SetAlign(tview.AlignLeft).
				SetMaxWidth(0).
				SetTextColor(tcell.ColorGray).
				SetSelectable(false)

			table.SetCell(i, j, cellForKeyComb)
			table.SetCell(i, j+1, cellForName).
				SetBorderPadding(0, 1, 1, 1)

			k++
			i++
		}
		j += 2
	}

	App.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		for _, shortcut := range properties.Shortcuts {
			if event.Rune() == shortcut.Rune {
				logger.Logger.Debug().Msg(fmt.Sprintf("Got rune event: %s", shortcut.Description))
				return shortcut.Handle(event)
			}

			if event.Key() == shortcut.Key {
				logger.Logger.Debug().Msg(fmt.Sprintf("Got key event: %s", shortcut.Description))
				return shortcut.Handle(event)
			}
		}

		return event
	})

	return table
}

func (c CustomShortCutProperties) Len() int { return len(c.Shortcuts) }
func (c CustomShortCutProperties) Swap(i, j int) {
	c.Shortcuts[i], c.Shortcuts[j] = c.Shortcuts[j], c.Shortcuts[i]
}
func (c CustomShortCutProperties) Less(i, j int) bool {
	return c.Shortcuts[i].Name < c.Shortcuts[j].Name
}
