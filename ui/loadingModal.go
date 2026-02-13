package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// CreateErrorModal creates a centered modal that displays an error message with an OK button.
// It returns a Pages primitive that overlays the modal on top of the provided background.
// When the user presses OK or ESC, the onDismiss callback is called to restore the previous view.
func CreateErrorModal(background tview.Primitive, message string, onDismiss func()) *tview.Pages {
	modal := tview.NewModal().
		SetText(message).
		SetBackgroundColor(tcell.ColorDefault).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			onDismiss()
		})
	modal.Box.SetBackgroundColor(tcell.ColorDefault)

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			onDismiss()
			return nil
		}
		return event
	})

	pages := tview.NewPages().
		AddPage("background", background, true, true).
		AddPage("error", modal, true, true)

	return pages
}
