package ui

import (
	"time"

	"github.com/rivo/tview"
)

const toastDuration = 2 * time.Second
const toastMessage = " Copied to clipboard! "
const ToastRefreshMessage = " Refreshing... "

// ShowToastOnTable temporarily changes a table's title to show a toast message,
// then restores the original title after a short delay.
func ShowToastOnTable(app *tview.Application, table *tview.Table) {
	originalTitle := table.GetTitle()
	table.SetTitle(toastMessage)
	app.ForceDraw()

	go func() {
		time.Sleep(toastDuration)
		app.QueueUpdateDraw(func() {
			table.SetTitle(originalTitle)
		})
	}()
}

// ShowToastOnList temporarily changes a list's title to show a toast message,
// then restores the original title after a short delay.
func ShowToastOnList(app *tview.Application, list *tview.List) {
	originalTitle := list.GetTitle()
	list.SetTitle(toastMessage)
	app.ForceDraw()

	go func() {
		time.Sleep(toastDuration)
		app.QueueUpdateDraw(func() {
			list.SetTitle(originalTitle)
		})
	}()
}

// ShowToastOnTreeView temporarily changes a tree view's title to show a toast message,
// then restores the original title after a short delay.
func ShowToastOnTreeView(app *tview.Application, tree *tview.TreeView) {
	originalTitle := tree.GetTitle()
	tree.SetTitle(toastMessage)
	app.ForceDraw()

	go func() {
		time.Sleep(toastDuration)
		app.QueueUpdateDraw(func() {
			tree.SetTitle(originalTitle)
		})
	}()
}

// Boxed is an interface for tview primitives that have Get/SetTitle (i.e. embed tview.Box).
type Boxed interface {
	GetTitle() string
	SetTitle(title string) *tview.Box
}

// ShowToast temporarily changes a widget's title to show a custom toast message,
// then restores the original title after a short delay.
func ShowToast(app *tview.Application, widget Boxed, message string) {
	originalTitle := widget.GetTitle()
	widget.SetTitle(message)
	app.ForceDraw()

	go func() {
		time.Sleep(toastDuration)
		app.QueueUpdateDraw(func() {
			widget.SetTitle(originalTitle)
		})
	}()
}
