package ui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/helpers"
	"github.com/cmd-tools/aws-commander/logger"
	tcell "github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Column struct {
	Name  string
	Width int
}

type CustomTableViewProperties struct {
	Title          string
	Columns        []Column
	Rows           [][]string
	RowData        []interface{}
	Handler        func(selectedProfileName string)
	ShowJsonViewer bool
	App            *tview.Application
	RestoreRoot    func()
	CreateHeader   func() *tview.Flex
	CreateFooter   func([]string) *tview.Table
	LogView        *tview.TextView
	IsLogEnabled   bool
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
		// Only call handler if showJsonViewer is not enabled
		// (if showJsonViewer is enabled, Enter key is handled by InputCapture)
		if !properties.ShowJsonViewer {
			properties.Handler(table.GetCell(row, 0).Text)
		}
	})

	// Set up input capture for both JSON viewer and normal handler
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle clipboard copy with 'y' (yank) or Ctrl+C
		if event.Rune() == 'y' || event.Key() == tcell.KeyCtrlC {
			row, _ := table.GetSelection()
			if row > 0 && row <= len(properties.Rows) {
				// Copy the entire row as tab-separated values
				rowData := properties.Rows[row-1]
				rowText := strings.Join(rowData, "\t")
				
				err := clipboard.WriteAll(rowText)
				if err != nil {
					logger.Logger.Error().Err(err).Msg("Failed to copy to clipboard")
				} else {
					logger.Logger.Debug().Str("data", rowText).Msg("Copied row to clipboard")
				}
			}
			return nil
		}

		if event.Key() == tcell.KeyEnter {
			row, _ := table.GetSelection()

			// Handle JSON viewer if enabled and we have data
			if properties.ShowJsonViewer && len(properties.RowData) > 0 && properties.App != nil && row > 0 && row-1 < len(properties.RowData) {
				// Add JSON viewer entry to breadcrumbs and navigation stack
				jsonViewLabel := fmt.Sprintf("JSON View (Row %d)", row)
				cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, jsonViewLabel)
				cmd.UiState.NavigationStack = append(cmd.UiState.NavigationStack, cmd.NavigationState{
					Type:  cmd.BreadcrumbJsonView,
					Value: jsonViewLabel,
				})

				// Create a callback that rebuilds the JSON viewer with the current or processed data
				var onBack func()
				onBack = func() {
					breadcrumbsLen := len(cmd.UiState.Breadcrumbs)
					if breadcrumbsLen == 0 {
						return
					}

					lastBreadcrumb := cmd.UiState.Breadcrumbs[breadcrumbsLen-1]
					var jsonViewer *tview.TreeView
					var dataToShow interface{}
					var title string

					// Determine what to show based on the last breadcrumb
					if lastBreadcrumb == "Parsed JSON" || strings.HasPrefix(lastBreadcrumb, "Decompressed") {
						// Get processed data from the current navigation state
						if len(cmd.UiState.NavigationStack) > 0 {
							currentNav := cmd.UiState.NavigationStack[len(cmd.UiState.NavigationStack)-1]
							if currentNav.ProcessedData != nil {
								dataToShow = currentNav.ProcessedData
							} else {
								// Fallback to original row data
								dataToShow = properties.RowData[row-1]
							}
						} else {
							dataToShow = properties.RowData[row-1]
						}
						title = lastBreadcrumb
					} else {
						// Show the original row data
						dataToShow = properties.RowData[row-1]
						title = fmt.Sprintf("Row %d Data", row)
					}

					jsonViewer = CreateJsonTreeViewer(JsonViewerProperties{
						Title:  title,
						Data:   dataToShow,
						App:    properties.App,
						OnBack: onBack,
					})

					// Restore focus to the previously selected node if available
					if cmd.UiState.SelectedNodeText != "" {
						restoreFocusToNode(jsonViewer, cmd.UiState.SelectedNodeText)
						cmd.UiState.SelectedNodeText = "" // Clear after restoring
					}

					// Create view with header and footer
					if properties.CreateHeader != nil && properties.CreateFooter != nil {
						view := tview.NewFlex().SetDirection(tview.FlexRow).
							AddItem(properties.CreateHeader(), 7, 2, false).
							AddItem(jsonViewer, 0, 1, true).
							AddItem(properties.CreateFooter(cmd.UiState.Breadcrumbs), 2, 2, false)

						if properties.IsLogEnabled && properties.LogView != nil {
							view.AddItem(properties.LogView, 8, 3, false)
						}

						properties.App.SetRoot(view, true)
						properties.App.SetFocus(jsonViewer)
					} else {
						properties.App.SetRoot(jsonViewer, true)
						properties.App.SetFocus(jsonViewer)
					}
				}

				// Store the callback in UIState so ESC handler can use it
				cmd.UiState.JsonViewerCallback = onBack

				// Show initial JSON viewer
				jsonViewer := CreateJsonTreeViewer(JsonViewerProperties{
					Title:  fmt.Sprintf("Row %d Data", row),
					Data:   properties.RowData[row-1],
					App:    properties.App,
					OnBack: onBack,
				})

				// Create view with header and footer
				if properties.CreateHeader != nil && properties.CreateFooter != nil {
					view := tview.NewFlex().SetDirection(tview.FlexRow).
						AddItem(properties.CreateHeader(), 7, 2, false).
						AddItem(jsonViewer, 0, 1, true).
						AddItem(properties.CreateFooter(cmd.UiState.Breadcrumbs), 2, 2, false)

					if properties.IsLogEnabled && properties.LogView != nil {
						view.AddItem(properties.LogView, 8, 3, false)
					}

					properties.App.SetRoot(view, true)
					properties.App.SetFocus(jsonViewer)
				} else {
					properties.App.SetRoot(jsonViewer, true)
					properties.App.SetFocus(jsonViewer)
				}
				return nil
			} else {
				// Normal handler for tables without JSON viewer (like list-tables)
				if row > 0 {
					properties.Handler(table.GetCell(row, 0).Text)
				}
				return nil
			}
		}
		return event
	})

	return table
}

// restoreFocusToNode recursively searches the tree and sets focus to the node with matching text
func restoreFocusToNode(tree *tview.TreeView, targetText string) bool {
	root := tree.GetRoot()
	return findAndFocusNode(tree, root, targetText)
}

// findAndFocusNode recursively searches for a node with matching text
func findAndFocusNode(tree *tview.TreeView, node *tview.TreeNode, targetText string) bool {
	if node == nil {
		return false
	}

	// Check if this node matches
	if node.GetText() == targetText {
		tree.SetCurrentNode(node)
		return true
	}

	// Search children
	for _, child := range node.GetChildren() {
		if findAndFocusNode(tree, child, targetText) {
			return true
		}
	}

	return false
}

func validateRowColumnComposition(properties CustomTableViewProperties) {
	columnCount := len(properties.Columns)
	for _, row := range properties.Rows {
		rowCount := len(row)
		if rowCount != columnCount {
			_ = fmt.Errorf("column count does not match row count: %d != %d", columnCount, rowCount)
			// Could log this error if needed
		}
	}
}
