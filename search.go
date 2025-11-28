package main

import (
	"fmt"
	"strings"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/constants"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// createSearchBar creates and configures the search input field with autocomplete
func createSearchBar() *tview.InputField {
	searchBar := tview.NewInputField()
	searchBar.SetLabel("️🔍 > ").
		SetFieldWidth(0).
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetBorder(true).
		SetBackgroundColor(tcell.ColorDefault)

	searchBar.SetChangedFunc(func(text string) {
		// Filter table rows if Body is a table
		if table, ok := Body.(*tview.Table); ok {
			if text != "" {
				filterTableRows(table, text)
			} else {
				restoreTableRows(table)
			}
		}
	})

	searchBar.SetAutocompleteFunc(func(currentText string) (entries []string) {
		if len(currentText) == 0 {
			return
		}
		for _, word := range AutoCompletionWordList {
			if strings.HasPrefix(strings.ToLower(word), strings.ToLower(currentText)) {
				entries = append(entries, word)
			}
		}
		return
	})

	searchBar.SetAutocompletedFunc(func(text string, index, source int) bool {
		if source != tview.AutocompletedNavigate {
			Search.SetText(text)
		}
		return source == tview.AutocompletedEnter || source == tview.AutocompletedClick
	})

	searchBar.SetInputCapture(handleSearchInput)

	return searchBar
}

// handleSearchInput processes keyboard input for the search bar
func handleSearchInput(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEsc && Search.HasFocus() {
		logger.Logger.Debug().Msg("[Search section] Got ESC")
		cmd.UiState.CommandBarVisible = false
		Search.SetText(constants.EmptyString)
		cmd.UiState.OriginalTableData = nil
		updateRootView(nil)
		return nil
	}

	// Move focus to table/list on Enter or arrow keys
	if Search.HasFocus() && (event.Key() == tcell.KeyEnter ||
		event.Key() == tcell.KeyDown || event.Key() == tcell.KeyUp ||
		event.Key() == tcell.KeyLeft || event.Key() == tcell.KeyRight) {

		logger.Logger.Debug().Msg("[Search section] Moving focus to body")
		App.SetFocus(Body)
		return nil
	}

	if event.Key() == tcell.KeyEnter && Search.HasFocus() {
		logger.Logger.Debug().Msg("[Search section] Got ENTER")
		text := Search.GetText()
		logger.Logger.Debug().Msg(fmt.Sprintf("[Search section] Got text: %s", Search.GetText()))
		switch text {
		case constants.Profiles:
			Body = createBody()
		case constants.Resources:
			Body = createResources(cmd.GetAvailableResourceNames())
		}
		cmd.UiState.CommandBarVisible = false
		updateRootView(nil)
		Search.SetText(constants.EmptyString)
		return nil
	}
	return event
}

// filterTableRows filters table rows based on search text
func filterTableRows(table *tview.Table, filter string) {
	// Save original data if not already saved
	if cmd.UiState.OriginalTableData == nil {
		saveOriginalTableData(table)
	}

	filter = strings.ToLower(filter)
	visibleRow := 1 // Start after header

	// Get total rows from original data
	if cmd.UiState.OriginalTableData == nil {
		return
	}

	totalRows := len(cmd.UiState.OriginalTableData.Rows)

	// Iterate through original data and show matching rows
	for originalRow := 0; originalRow < totalRows; originalRow++ {
		rowData := cmd.UiState.OriginalTableData.Rows[originalRow]
		if rowData == nil {
			continue
		}

		// Check if any cell in the row matches the filter
		matches := false
		for _, cellText := range rowData {
			if strings.Contains(strings.ToLower(cellText), filter) {
				matches = true
				break
			}
		}

		if matches {
			// Copy row to visible position
			colCount := table.GetColumnCount()
			for col := 0; col < colCount; col++ {
				cellText := ""
				if col < len(rowData) {
					cellText = rowData[col]
				}
				table.SetCell(visibleRow, col,
					tview.NewTableCell(cellText).
						SetTextColor(tcell.ColorWhite).
						SetAlign(tview.AlignLeft))
			}
			visibleRow++
		}
	}

	// Clear remaining rows
	rowCount := table.GetRowCount()
	for row := visibleRow; row < rowCount; row++ {
		for col := 0; col < table.GetColumnCount(); col++ {
			table.SetCell(row, col, tview.NewTableCell(""))
		}
	}

	// Update table title with filter info
	if title := table.GetTitle(); title != "" {
		baseTitle := strings.Split(title, " [filtered:")[0]
		table.SetTitle(fmt.Sprintf("%s [filtered: %d rows]", baseTitle, visibleRow-1))
	}
}

// saveOriginalTableData caches the original table data before filtering
func saveOriginalTableData(table *tview.Table) {
	rowCount := table.GetRowCount()
	colCount := table.GetColumnCount()

	rows := make([][]string, rowCount)

	for row := 0; row < rowCount; row++ {
		rows[row] = make([]string, colCount)
		for col := 0; col < colCount; col++ {
			cell := table.GetCell(row, col)
			if cell != nil {
				rows[row][col] = cell.Text
			}
		}
	}

	cmd.UiState.OriginalTableData = &cmd.TableData{
		Rows: rows,
	}
}

// restoreTableRows restores the original table data after filtering
func restoreTableRows(table *tview.Table) {
	if cmd.UiState.OriginalTableData == nil {
		return
	}

	rowCount := len(cmd.UiState.OriginalTableData.Rows)
	colCount := table.GetColumnCount()

	for row := 0; row < rowCount; row++ {
		rowData := cmd.UiState.OriginalTableData.Rows[row]
		if rowData == nil {
			continue
		}

		for col := 0; col < colCount && col < len(rowData); col++ {
			cellText := rowData[col]
			table.SetCell(row, col,
				tview.NewTableCell(cellText).
					SetTextColor(tcell.ColorWhite).
					SetAlign(tview.AlignLeft))
		}
	}

	// Update table title to remove filter info
	if title := table.GetTitle(); title != "" {
		baseTitle := strings.Split(title, " [filtered:")[0]
		table.SetTitle(baseTitle)
	}

	// Clear the cached original data
	cmd.UiState.OriginalTableData = nil
}
