package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type CustomShortCut struct {
	KeyComb string
	Name    string
}

type CustomShortCutProperties struct {
	Keys []CustomShortCut
}

func CreateCustomShortCutsView(properties CustomShortCutProperties) *tview.Table {
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

	itemListCount := len(properties.Keys)

	for j := 0; j < maxRows; j = j + 2 {
		for i := 0; i < maxColumn; i++ {
			key := properties.Keys[k]

			cellForKeyComb := tview.NewTableCell(fmt.Sprintf("<%s>", key.KeyComb)).
				SetAlign(tview.AlignLeft).
				SetMaxWidth(0).
				SetTextColor(tcell.ColorGold).
				SetSelectable(false)

			cellForName := tview.NewTableCell(key.Name).
				SetAlign(tview.AlignLeft).
				SetMaxWidth(0).
				SetTextColor(tcell.ColorGray).
				SetSelectable(false)

			table.SetCell(i, j, cellForKeyComb)
			table.SetCell(i, j+1, cellForName).SetBorderPadding(0, 1, 1, 1)

			k++
			if k >= itemListCount {
				return table
			}
		}
	}

	return table
}
