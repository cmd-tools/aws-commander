package ui

import (
	"fmt"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"sort"
)

type CustomShortCut struct {
	Name        string
	Description string
}

type CustomShortCutProperties struct {
	Keys []CustomShortCut
}

func CreateCustomShortCutsView(properties CustomShortCutProperties) *tview.Table {

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

	itemListCount := len(properties.Keys)

	for j := 0; j < maxRows; j = j + 2 {
		for i := 0; i < maxColumn; i++ {
			key := properties.Keys[k]

			cellForKeyComb := tview.NewTableCell(fmt.Sprintf("<%s>", key.Name)).
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
			table.SetCell(i, j+1, cellForName).SetBorderPadding(0, 1, 1, 1)

			k++
			if k >= itemListCount {
				return table
			}
		}
	}

	return table
}

func (c CustomShortCutProperties) Len() int { return len(c.Keys) }
func (c CustomShortCutProperties) Swap(i, j int) {
	c.Keys[i], c.Keys[j] = c.Keys[j], c.Keys[i]
}
func (c CustomShortCutProperties) Less(i, j int) bool {
	return c.Keys[i].Name < c.Keys[j].Name
}
