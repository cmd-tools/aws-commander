package main

import (
	"bytes"
	"flag"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/cmd/profile"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/rivo/tview"
)

// Global application state
var (
	App                    *tview.Application
	Search                 *tview.InputField
	Body                   tview.Primitive
	AutoCompletionWordList []string
	ProfileList            profile.Profiles
	LogView                *tview.TextView
	LogViewTextBuffer      bytes.Buffer
	IsLogViewEnabled       bool
)

func main() {
	flag.BoolVar(&IsLogViewEnabled, "logview", false, "Enable log view while using the tool.")
	flag.Parse()

	logger.InitLog(IsLogViewEnabled)
	logger.Logger.Info().Msg("Starting aws-commander")
	logger.Logger.Debug().Msg("Loading configurations")

	cmd.Init()
	cmd.Favourites.Load()
	cmd.Settings.Load()

	App = tview.NewApplication()
	Search = createSearchBar()

	if IsLogViewEnabled {
		go startLogViewListener()
	}

	// Always show loading animation while profiles load in the background
	done := make(chan struct{})
	loadingView := ui.ShowLoadingAnimation(App, done, func() {
		Body = createBody()
		mainFlexPanel := updateRootView(nil)
		App.SetRoot(mainFlexPanel, true)
		App.SetFocus(Body)
	})

	go func() {
		ProfileList = profile.GetList()
		close(done)
	}()

	if err := App.SetRoot(loadingView, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// startLogViewListener monitors the log channel and updates the log view
func startLogViewListener() {
	for {
		select {
		case logMessage := <-logger.LogChannel:
			LogViewTextBuffer.WriteString(logMessage)
			if nil != LogView {
				App.QueueUpdateDraw(func() {
					if nil != LogView {
						LogView.SetText(LogViewTextBuffer.String())
					}
				})
			}
		}
	}
}
