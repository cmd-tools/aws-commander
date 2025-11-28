package main

import (
	"bytes"
	"flag"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/cmd/profile"
	"github.com/cmd-tools/aws-commander/logger"
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

	App = tview.NewApplication()
	Search = createSearchBar()
	ProfileList = profile.GetList()
	Body = createBody()

	mainFlexPanel := updateRootView(nil)

	if IsLogViewEnabled {
		go startLogViewListener()
	}

	if err := App.SetRoot(mainFlexPanel, true).EnableMouse(true).Run(); err != nil {
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
