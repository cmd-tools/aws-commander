package ui

import (
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// contentSize calculates the width and height needed to display the given text.
// Width is the length of the longest line, height is the number of lines.
func contentSize(text string) (width, height int) {
	lines := strings.Split(text, "\n")
	height = len(lines)
	for _, line := range lines {
		if len(line) > width {
			width = len(line)
		}
	}
	return
}

// maxContentSize returns the maximum width and height across all frames of an animation,
// including the message header. This ensures the box stays a fixed size across frames.
func maxContentSize(message string, animation loadingAnimation) (width, height int) {
	for _, frame := range animation {
		full := message + "\n\n" + frame
		w, h := contentSize(full)
		if w > width {
			width = w
		}
		if h > height {
			height = h
		}
	}
	return
}

// CreateLoadingModal creates a centered modal with an animated ASCII art loading indicator.
// The box is sized to fit the content tightly. It starts a goroutine that cycles through
// animation frames. The caller must close the returned stop channel when loading is complete.
func CreateLoadingModal(app *tview.Application, background tview.Primitive, message string) (*tview.Pages, chan struct{}) {
	animation := randomAnimation()
	text := message + "\n\n" + animation[0]

	textView := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText(text)
	textView.SetBackgroundColor(tcell.ColorDefault)

	box := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, false)
	box.SetBorder(true).
		SetBackgroundColor(tcell.ColorDefault)

	// Calculate fixed size from the largest frame so the box doesn't jump around
	maxW, maxH := maxContentSize(message, animation)
	// +4 for border (2) + padding (2)
	boxWidth := maxW + 4
	// +2 for border
	boxHeight := maxH + 2

	// Center the box on screen using a grid
	grid := tview.NewGrid().
		SetColumns(0, boxWidth, 0).
		SetRows(0, boxHeight, 0).
		AddItem(box, 1, 1, 1, 1, 0, 0, false)
	grid.SetBackgroundColor(tcell.ColorDefault)

	pages := tview.NewPages().
		AddPage("background", background, true, true).
		AddPage("loading", grid, true, true)

	stop := make(chan struct{})
	go func() {
		frame := 1
		ticker := time.NewTicker(400 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				art := animation[frame%len(animation)]
				frame++
				app.QueueUpdateDraw(func() {
					textView.SetText(message + "\n\n" + art)
				})
			}
		}
	}()

	return pages, stop
}

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
