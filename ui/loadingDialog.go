package ui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const loadingFrameDelay = 400 * time.Millisecond

// loadingDots maps a cycle index (0-3) to the corresponding dots suffix.
var loadingDots = [4]string{"", " .", " . .", " . . ."}

// createLoadingDialog builds a small centered dialog box containing only a loading text
// view with animated dots. The dialog is centered on screen using a Flex layout and floats
// over whatever background is behind it.
func createLoadingDialog() (loadingView *tview.TextView, centered tview.Primitive) {
	loadingView = tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(false)
	loadingView.SetBackgroundColor(tcell.ColorDefault)

	// Inner dialog box with border
	dialog := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(loadingView, 1, 0, false).
		AddItem(nil, 0, 1, false)
	dialog.SetBackgroundColor(tcell.ColorDefault)
	dialog.SetBorder(true).
		SetBorderColor(tview.Styles.BorderColor).
		SetBorderPadding(1, 1, 2, 2)

	// Center the dialog on screen with fixed dimensions
	dialogWidth := 30
	dialogHeight := 5
	centered = tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(dialog, dialogHeight, 0, false).
			AddItem(nil, 0, 1, false), dialogWidth, 0, false).
		AddItem(nil, 0, 1, false)

	return loadingView, centered
}

// ShowLoadingAnimation displays a centered dialog overlay on top of the provided background
// with an animated "Loading" text that cycles dots (Loading, Loading ., Loading . .,
// Loading . . .) until the done channel is closed. Once done is signalled, onDone is called
// via QueueUpdateDraw to swap the view. Returns a tview.Pages primitive to use as body.
func ShowLoadingAnimation(app *tview.Application, background tview.Primitive, done <-chan struct{}, onDone func()) *tview.Pages {
	loadingView, dialog := createLoadingDialog()

	pages := tview.NewPages().
		AddPage("background", background, true, true).
		AddPage("loading", dialog, true, true)

	go func() {
		i := 0
		for {
			select {
			case <-done:
				app.QueueUpdateDraw(func() {
					onDone()
				})
				return
			default:
				dots := loadingDots[i%len(loadingDots)]
				app.QueueUpdateDraw(func() {
					loadingView.SetText("Loading" + dots)
				})
				time.Sleep(loadingFrameDelay)
				i++
			}
		}
	}()

	return pages
}
