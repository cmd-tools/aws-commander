package main

import (
	"bytes"
	"embed"
	"flag"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/cmd/profile"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/rivo/tview"
)

//go:embed configurations/*.yaml
var configurationsFS embed.FS

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

	cmd.Init(configurationsFS)
	cmd.Favourites.Load()
	cmd.Settings.Load()

	App = tview.NewApplication()
	Search = createSearchBar()

	if IsLogViewEnabled {
		go startLogViewListener()
	}

	// Show the full profile page layout immediately with an animation dialog
	// overlaid on an empty Profiles box while profiles load in the background
	done := make(chan struct{})
	cmd.UiState.Breadcrumbs = []string{constants.Profiles}
	cmd.UiState.NavigationStack = []cmd.NavigationState{
		{Type: cmd.BreadcrumbProfiles, Value: constants.Profiles},
	}

	// Create an empty bordered box matching the profile table style
	emptyProfileBox := tview.NewTable()
	emptyProfileBox.SetTitle(" Profiles ").
		SetBorder(true).
		SetBorderColor(tview.Styles.BorderColor).
		SetTitleAlign(tview.AlignCenter).
		SetBorderPadding(0, 1, 2, 2)

	Body = ui.ShowLoadingAnimation(App, emptyProfileBox, done, func() {
		Body = createBody()
		updateRootView(nil)
		App.SetFocus(Body)
	})
	mainFlexPanel := updateRootView(nil)

	go func() {
		ProfileList = profile.GetList()
		close(done)
	}()

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
