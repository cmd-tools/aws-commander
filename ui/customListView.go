package ui

import (
	"github.com/cmd-tools/aws-commander/constants"
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

	return list
}
