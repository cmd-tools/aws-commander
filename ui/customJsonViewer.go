package ui

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/gdamore/tcell/v2"
	"github.com/iancoleman/orderedmap"
	"github.com/rivo/tview"
)

type JsonViewerProperties struct {
	Title  string
	Data   interface{}
	App    *tview.Application
	OnBack func()
}

// tryParseStringifiedJSON attempts to parse a string as JSON
func tryParseStringifiedJSON(s string) (interface{}, bool) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil, false
	}

	// Check if it looks like JSON
	if (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]")) {
		var result interface{}
		if err := json.Unmarshal([]byte(s), &result); err == nil {
			return result, true
		}
	}
	return nil, false
}

// tryDecompressBase64Gzip attempts to decompress a Base64 gzipped string
func tryDecompressBase64Gzip(s string) (string, bool) {
	s = strings.TrimSpace(s)

	// Check for gzip magic number in Base64
	if !strings.HasPrefix(s, "H4sI") {
		return "", false
	}

	// Decode Base64
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", false
	}

	// Decompress gzip
	reader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return "", false
	}
	defer reader.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return "", false
	}

	return buf.String(), true
}

// extractValueFromNode extracts the actual value from a tree node text
func extractValueFromNode(nodeText string) string {
	// Node text format: "key: value" or just "value"
	if idx := strings.Index(nodeText, ": "); idx != -1 {
		value := nodeText[idx+2:]
		// Remove color tags and quotes
		value = strings.TrimPrefix(value, "[green]\"")
		value = strings.TrimSuffix(value, "\"")
		value = strings.TrimPrefix(value, "[white]")
		value = strings.TrimPrefix(value, "[red]")
		value = strings.TrimPrefix(value, "[gray]")
		return value
	}
	return nodeText
}

// CreateJsonTreeViewer creates an interactive tree view for JSON data
func CreateJsonTreeViewer(properties JsonViewerProperties) *tview.TreeView {
	root := tview.NewTreeNode(properties.Title).
		SetColor(tcell.ColorGold).
		SetExpanded(true)

	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	// Build tree from JSON data
	buildJsonTree(properties.Data, root)

	// Auto-expand all nodes
	expandAllNodes(root)

	tree.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s (JSON Viewer) ", properties.Title)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	// Add input handler for Enter key to process stringified JSON or Base64 gzip
	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle clipboard copy with 'y' (yank) or Ctrl+C
		if event.Rune() == 'y' || event.Key() == tcell.KeyCtrlC {
			currentNode := tree.GetCurrentNode()
			if currentNode != nil {
				nodeText := currentNode.GetText()
				value := extractValueFromNode(nodeText)
				
				// If it's a leaf node, copy just the value, otherwise copy the full node text
				var textToCopy string
				if len(currentNode.GetChildren()) == 0 {
					textToCopy = value
				} else {
					textToCopy = nodeText
				}
				
				err := clipboard.WriteAll(textToCopy)
				if err != nil {
					logger.Logger.Error().Err(err).Msg("Failed to copy to clipboard")
				} else {
					logger.Logger.Debug().Str("data", textToCopy).Msg("Copied to clipboard")
				}
			}
			return nil
		}

		if event.Key() == tcell.KeyEnter && properties.App != nil {
			currentNode := tree.GetCurrentNode()
			if currentNode != nil && len(currentNode.GetChildren()) == 0 {
				// This is a leaf node (has a value)
				nodeText := currentNode.GetText()
				value := extractValueFromNode(nodeText)

				var newData interface{}
				var newTitle string
				processed := false

				// Try to decompress Base64 gzip first
				if decompressed, ok := tryDecompressBase64Gzip(value); ok {
					newTitle = "Decompressed Data"
					// Try to parse decompressed data as JSON
					if parsed, ok := tryParseStringifiedJSON(decompressed); ok {
						newData = parsed
						newTitle = "Decompressed JSON"
					} else {
						// Show as text if not JSON
						newData = map[string]interface{}{"content": decompressed}
					}
					processed = true
				} else if parsed, ok := tryParseStringifiedJSON(value); ok {
					// Try to parse as stringified JSON
					newData = parsed
					newTitle = "Parsed JSON"
					processed = true
				}

				if processed {
					// Store the processed data in navigation stack (not global)
					navState := cmd.NavigationState{
						Type:          cmd.BreadcrumbProcessedJson,
						Value:         newTitle,
						ProcessedData: newData,
					}

					// Store the current node text for focus restoration
					cmd.UiState.SelectedNodeText = nodeText

					// Add breadcrumb and navigation state for the expanded view
					cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, newTitle)
					cmd.UiState.NavigationStack = append(cmd.UiState.NavigationStack, navState)

					// Create a new JSON viewer with the processed data
					newViewer := CreateJsonTreeViewer(JsonViewerProperties{
						Title:  newTitle,
						Data:   newData,
						App:    properties.App,
						OnBack: properties.OnBack,
					})

					// Rebuild view with header/footer if callback is provided
					if properties.OnBack != nil {
						properties.OnBack()
					} else {
						properties.App.SetRoot(newViewer, true)
						properties.App.SetFocus(newViewer)
					}

					return nil
				}
			}
		}
		return event
	})

	return tree
}

// CreateJsonTextViewer creates a formatted text view for JSON data
func CreateJsonTextViewer(properties JsonViewerProperties) *tview.TextView {
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(false)

	// Pretty print JSON
	jsonBytes, err := json.MarshalIndent(properties.Data, "", "  ")
	if err != nil {
		textView.SetText(fmt.Sprintf("Error formatting JSON: %v", err))
	} else {
		textView.SetText(string(jsonBytes))
	}

	textView.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s (JSON Viewer) ", properties.Title)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	return textView
}

func buildJsonTree(data interface{}, parent *tview.TreeNode) {
	switch v := data.(type) {
	case orderedmap.OrderedMap:
		// Handle orderedmap.OrderedMap type
		for _, key := range v.Keys() {
			val, _ := v.Get(key)
			node := tview.NewTreeNode(fmt.Sprintf("[yellow]%s", key)).
				SetColor(tcell.ColorYellow).
				SetSelectable(true).
				SetExpanded(true)
			parent.AddChild(node)
			buildJsonTree(val, node)
		}
	case map[string]interface{}:
		for key, val := range v {
			node := tview.NewTreeNode(fmt.Sprintf("[yellow]%s", key)).
				SetColor(tcell.ColorYellow).
				SetSelectable(true).
				SetExpanded(true)
			parent.AddChild(node)
			buildJsonTree(val, node)
		}
	case []interface{}:
		for i, val := range v {
			node := tview.NewTreeNode(fmt.Sprintf("[white][%d]", i)).
				SetColor(tcell.ColorWhite).
				SetSelectable(true).
				SetExpanded(true)
			parent.AddChild(node)
			buildJsonTree(val, node)
		}
	case string:
		parent.SetText(fmt.Sprintf("%s: [green]\"%v\"", parent.GetText(), v))
		parent.SetColor(tcell.ColorWhite)
	case float64, int, int64:
		parent.SetText(fmt.Sprintf("%s: [white]%v", parent.GetText(), v))
		parent.SetColor(tcell.ColorWhite)
	case bool:
		parent.SetText(fmt.Sprintf("%s: [red]%v", parent.GetText(), v))
		parent.SetColor(tcell.ColorWhite)
	case nil:
		parent.SetText(fmt.Sprintf("%s: [gray]null", parent.GetText()))
		parent.SetColor(tcell.ColorGray)
	default:
		parent.SetText(fmt.Sprintf("%s: %v", parent.GetText(), v))
	}
}

func expandAllNodes(node *tview.TreeNode) {
	node.SetExpanded(true)
	for _, child := range node.GetChildren() {
		expandAllNodes(child)
	}
}
