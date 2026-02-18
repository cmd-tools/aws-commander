package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type InputFormProperties struct {
	Title          string
	Fields         []InputField
	DropdownFields []DropdownField
	OnSubmit       func(values map[string]string)
	OnCancel       func()
	App            *tview.Application
	PreviousView   tview.Primitive
}

type InputField struct {
	Label        string
	Key          string
	DefaultValue string
}

// DropdownField represents a dropdown selection field in a form
type DropdownField struct {
	Label        string
	Key          string
	Options      []string
	DefaultIndex int
}

func CreateInputForm(properties InputFormProperties) *tview.Form {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(properties.Title).SetTitleAlign(tview.AlignLeft)
	form.SetBackgroundColor(tcell.ColorDefault)

	values := make(map[string]string)

	// Add fields in order: process Fields and DropdownFields by insertion order
	// DropdownFields are inserted before the input field at their position
	dropdownsByPosition := make(map[int]DropdownField)
	for i, df := range properties.DropdownFields {
		dropdownsByPosition[i] = df
	}

	fieldIndex := 0
	for _, field := range properties.Fields {
		// Insert any dropdown that should appear before this input field
		if df, ok := dropdownsByPosition[fieldIndex]; ok && df.Key != "" {
			dfKey := df.Key
			dfOptions := df.Options
			defaultIdx := df.DefaultIndex
			if defaultIdx >= 0 && defaultIdx < len(dfOptions) {
				values[dfKey] = dfOptions[defaultIdx]
			}
			form.AddDropDown(df.Label, dfOptions, defaultIdx, func(option string, index int) {
				values[dfKey] = option
			})
		}

		fieldKey := field.Key
		form.AddInputField(field.Label, field.DefaultValue, 0, nil, func(text string) {
			values[fieldKey] = text
		})
		fieldIndex++
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
