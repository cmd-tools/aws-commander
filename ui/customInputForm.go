package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type InputFormProperties struct {
	Title        string
	Fields       []InputField
	OnSubmit     func(values map[string]string)
	OnCancel     func()
	App          *tview.Application
	PreviousView tview.Primitive
}

type InputField struct {
	Label        string
	Key          string
	DefaultValue string
}

func CreateInputForm(properties InputFormProperties) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(properties.Title).SetTitleAlign(tview.AlignLeft)
	form.SetBackgroundColor(tcell.ColorDefault)

	values := make(map[string]string)

	// Add input fields
	for _, field := range properties.Fields {
		fieldKey := field.Key
		form.AddInputField(field.Label, field.DefaultValue, 0, nil, func(text string) {
			values[fieldKey] = text
		})
	}

	// Add buttons
	form.AddButton("Submit", func() {
		if properties.OnSubmit != nil {
			properties.OnSubmit(values)
		}
	})

	form.AddButton("Cancel", func() {
		if properties.OnCancel != nil {
			properties.OnCancel()
		}
	})

	// Handle ESC key
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if properties.OnCancel != nil {
				properties.OnCancel()
			}
			return nil
		}
		return event
	})

	return form
}
