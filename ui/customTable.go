package ui

import (
	"github.com/cmd-tools/aws-commander/helpers"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Column struct {
	Name  string
	Width int
}

type CustomTableViewProperties struct {
	Title   string
	Columns []Column
	Rows    [][]string
	Handler func(selectedProfileName string)
}

func CreateCustomTableView(properties CustomTableViewProperties) *tview.Table {

	validateRowColumnComposition(properties)

	table := tview.NewTable()

	if !helpers.IsStringEmpty(properties.Title) {
		table.SetTitle(properties.Title).
			SetBorder(true).
			SetBorderColor(tview.Styles.BorderColor).
			SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	}

	table.
		SetTitleAlign(tview.AlignCenter).
		SetBorderPadding(0, 1, 2, 2)

	// TODO: maybe configure style via env or yaml?
	table.SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorGold))

	for colIndex, columnName := range properties.Columns {
		table.SetCell(0, colIndex, tview.NewTableCell(columnName.Name).
			SetAlign(tview.AlignLeft).
			SetMaxWidth(0).
			SetSelectable(false))
	}

	for rowIndex, rowData := range properties.Rows {
		for colIndex, cellData := range rowData {
			table.SetCell(rowIndex+1, colIndex, tview.
				NewTableCell(cellData).
				SetExpansion(1).
				SetAlign(tview.AlignLeft).
				SetSelectable(true))
		}
	}

	table.SetSelectedFunc(func(row, column int) {
		properties.Handler(table.GetCell(row, 0).Text)
	})

	return table
}

func validateRowColumnComposition(properties CustomTableViewProperties) {
	columnCount := len(properties.Columns)
	for _, row := range properties.Rows {
		rowCount := len(row)
		if rowCount != columnCount {
			//TODO: understand if we need to panic or not
			// panic(fmt.Errorf("column count does not match row count: %d != %d", columnCount, rowCount))
		}
	}
}
