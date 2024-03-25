package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ModalChoice struct {
	Name    string
	Handler func(currentFlex *tview.Flex)
}

type ModalProperties struct {
	Title       string
	LeftChoice  ModalChoice
	RightChoice ModalChoice
}

func CreateModal(properties ModalProperties, currentFlex *tview.Flex) *tview.Modal {
	return tview.NewModal().
		SetText(properties.Title).
		SetBackgroundColor(tcell.ColorDefault).
		AddButtons([]string{properties.LeftChoice.Name, properties.RightChoice.Name}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case properties.LeftChoice.Name:
				properties.LeftChoice.Handler(currentFlex)
				break
			case properties.RightChoice.Name:
				properties.LeftChoice.Handler(currentFlex)
				break
			default:
				panic("Unexpected choice")
			}
		})
}
