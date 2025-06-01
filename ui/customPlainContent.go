package ui

import (
	"github.com/rivo/tview"
)

type CustomerPlainContentProperties struct {
	Title   string
	Content string
	Handler func(selectedProfileName string)
}

func CreateCustomerPlainContent(properties CustomerPlainContentProperties) *tview.TextView {

	textView := tview.NewTextView()

	textView.SetBorder(true).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	textView.SetBorderPadding(0, 1, 2, 2)

	textView.
		SetText(properties.Content).
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetRegions(true).
		SetScrollable(true).
		SetTitle(properties.Title)

	return textView
}
