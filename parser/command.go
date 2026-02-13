package parser

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/iancoleman/orderedmap"
	"github.com/rivo/tview"
	"gopkg.in/yaml.v2"
)

type ParseCommandResult struct {
	Command   string
	Header    []string
	Values    [][]string
	RawData   []interface{}
	RawOutput string
}

func ParseCommand(command cmd.Command, commandOutput string) ParseCommandResult {
	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			logger.Logger.Error().
				Interface("panic", r).
				Str("command", command.Name).
				Str("output", commandOutput).
				Msg("Panic occurred while parsing command")
		}
	}()

	// Handle empty output
	if commandOutput == "" || len(strings.TrimSpace(commandOutput)) == 0 {
		logger.Logger.Debug().Msg("Command returned empty output")
		return ParseCommandResult{
			Command: command.Name,
			Header:  []string{"Info"},
			Values:  [][]string{{"No output returned from command"}},
		}
	}

	jsonResult := orderedmap.New()
	err := json.Unmarshal([]byte(commandOutput), &jsonResult)
	if err != nil {
		logger.Logger.Error().Err(err).Str("output", commandOutput).Msg(fmt.Sprintf("Unable to unmarshal json for command: %s", command.Name))
		return ParseCommandResult{
			Command:   command.Name,
			RawOutput: strings.TrimSpace(commandOutput),
		}
	}

	var parseCommandResult = ParseCommandResult{Command: command.Name}
	baseAttribute, exists := jsonResult.Get(command.Parse.AttributeName)

	logger.Logger.Debug().
		Str("attribute", command.Parse.AttributeName).
		Bool("exists", exists).
		Interface("value", baseAttribute).
		Msg("Parsing command attribute")

	// Handle missing attribute (e.g., empty SQS queue returns {} without Messages key)
	if !exists {
		logger.Logger.Debug().Msg(fmt.Sprintf("Attribute '%s' not found in command output", command.Parse.AttributeName))
		return ParseCommandResult{
			Command: command.Name,
			Header:  []string{"Info"},
			Values:  [][]string{{fmt.Sprintf("No %s found", command.Parse.AttributeName)}},
		}
	}

	// Handle null attribute value
	if baseAttribute == nil {
		logger.Logger.Debug().Msg(fmt.Sprintf("Attribute '%s' is null", command.Parse.AttributeName))
		return ParseCommandResult{
			Command: command.Name,
			Header:  []string{"Info"},
			Values:  [][]string{{fmt.Sprintf("No %s available", command.Parse.AttributeName)}},
		}
	}

	switch baseAttribute.(type) {
	case []interface{}:
		logger.Logger.Debug().Msg("Parse command list")
		items := baseAttribute.([]interface{})

		// Handle empty array (e.g., empty SQS queue)
		if len(items) == 0 {
			logger.Logger.Debug().Msg(fmt.Sprintf("Attribute '%s' is an empty array", command.Parse.AttributeName))
			return ParseCommandResult{
				Command: command.Name,
				Header:  []string{"Info"},
				Values:  [][]string{{"Empty - no items available"}},
			}
		}

		for i, s := range items {
			var values []string
			if command.Parse.Type == "object" {
				// Store raw data for JSON viewer
				parseCommandResult.RawData = append(parseCommandResult.RawData, s)

				// Try orderedmap first, then regular map
				var itemMap map[string]interface{}
				var keys []string

				if orderedItem, ok := s.(orderedmap.OrderedMap); ok {
					// It's an orderedmap
					keys = orderedItem.Keys()
					if i == 0 {
						parseCommandResult.Header = keys
					}
					for _, key := range keys {
						value, exists := orderedItem.Get(key)
						if exists {
							switch value.(type) {
							case string:
								values = append(values, fmt.Sprintf("%v", value))
							default:
								bytes, _ := json.Marshal(value)
								values = append(values, fmt.Sprintf("%v", string(bytes)))
							}
						}
					}
				} else if regularMap, ok := s.(map[string]interface{}); ok {
					// It's a regular map
					itemMap = regularMap
					for key := range itemMap {
						keys = append(keys, key)
					}
					if i == 0 {
						parseCommandResult.Header = keys
					}
					for _, key := range keys {
						value := itemMap[key]
						switch value.(type) {
						case string:
							values = append(values, fmt.Sprintf("%v", value))
						default:
							bytes, _ := json.Marshal(value)
							values = append(values, fmt.Sprintf("%v", string(bytes)))
						}
					}
				} else {
					logger.Logger.Error().
						Interface("item", s).
						Str("type", fmt.Sprintf("%T", s)).
						Msg("Item is neither orderedmap nor regular map")
					continue
				}

				parseCommandResult.Values = append(parseCommandResult.Values, values)
			} else if command.Parse.Type == "list" {
				if i == 0 {
					parseCommandResult.Header = append(parseCommandResult.Header, "Item")
				}
				if s != nil {
					parseCommandResult.Values = append(parseCommandResult.Values, append(values, s.(string)))
				}
			} else {
				logger.Logger.Debug().Msg("Wrong type. Accepted types [Object, List]")
			}
		}
	case interface{}:
		logger.Logger.Debug().Msg("Parse command object")

		// Special handling for "keys" type - extract partition and sort keys from DynamoDB table description
		if command.Parse.Type == "keys" {
			parseCommandResult = parseTableKeys(baseAttribute)
			return parseCommandResult
		}

		var values []string

		// Type assertion with error handling
		item, ok := baseAttribute.(orderedmap.OrderedMap)
		if !ok {
			logger.Logger.Error().
				Interface("attribute", baseAttribute).
				Str("type", fmt.Sprintf("%T", baseAttribute)).
				Msg("Failed to assert attribute as orderedmap.OrderedMap")
			return ParseCommandResult{
				Command: command.Name,
				Header:  []string{"Error"},
				Values:  [][]string{{"Unexpected data format"}},
			}
		}

		for _, key := range item.Keys() {
			parseCommandResult.Header = append(parseCommandResult.Header, key)
			value, exists := item.Get(key)
			if exists {
				values = append(values, fmt.Sprintf("%v", value))
			}
		}
		parseCommandResult.Values = append(parseCommandResult.Values, values)
	default:
		logger.Logger.Debug().Msg("Fail to Parse. Command result is not an Map or List")
	}
	return parseCommandResult
}

// parseTableKeys extracts partition keys, sort keys, and GSI/LSI from DynamoDB describe-table output
// Returns indexes as selectable items instead of individual keys
func parseTableKeys(tableAttribute interface{}) ParseCommandResult {
	result := ParseCommandResult{
		Command: "describe-table",
		Header:  []string{"Index Name", "Index Type", "Keys", "Key Details"},
		Values:  [][]string{},
	}

	tableMap, ok := tableAttribute.(orderedmap.OrderedMap)
	if !ok {
		logger.Logger.Error().Msg("Failed to parse table description as orderedmap")
		return ParseCommandResult{
			Command: "describe-table",
			Header:  []string{"Error"},
			Values:  [][]string{{"Failed to parse table description"}},
		}
	}

	// Get KeySchema for primary key
	keySchemaAttr, exists := tableMap.Get("KeySchema")
	if !exists {
		logger.Logger.Error().Msg("KeySchema not found in table description")
		return result
	}

	// Get AttributeDefinitions to map attribute types
	attributeDefsAttr, _ := tableMap.Get("AttributeDefinitions")
	attributeTypes := make(map[string]string)
	if attributeDefs, ok := attributeDefsAttr.([]interface{}); ok {
		for _, attr := range attributeDefs {
			if attrMap, ok := attr.(orderedmap.OrderedMap); ok {
				name, _ := attrMap.Get("AttributeName")
				attrType, _ := attrMap.Get("AttributeType")
				if name != nil && attrType != nil {
					attributeTypes[name.(string)] = attrType.(string)
				}
			}
		}
	}

	// Parse primary key (HASH and RANGE)
	var primaryKeys []string
	if keySchema, ok := keySchemaAttr.([]interface{}); ok {
		for _, key := range keySchema {
			if keyMap, ok := key.(orderedmap.OrderedMap); ok {
				keyName, _ := keyMap.Get("AttributeName")
				keyType, _ := keyMap.Get("KeyType")
				if keyName != nil && keyType != nil {
					attrType := attributeTypes[keyName.(string)]
					displayKeyType := "PK"
					if keyType.(string) == "RANGE" {
						displayKeyType = "SK"
					}
					primaryKeys = append(primaryKeys, fmt.Sprintf("%s (%s:%s)", keyName.(string), displayKeyType, attrType))
				}
			}
		}
	}

	if len(primaryKeys) > 0 {
		result.Values = append(result.Values, []string{
			"Primary",
			"Primary Index",
			fmt.Sprintf("%d", len(primaryKeys)),
			strings.Join(primaryKeys, ", "),
		})
	}

	// Parse Global Secondary Indexes (GSI)
	if gsiAttr, exists := tableMap.Get("GlobalSecondaryIndexes"); exists {
		if gsiList, ok := gsiAttr.([]interface{}); ok {
			for _, gsi := range gsiList {
				if gsiMap, ok := gsi.(orderedmap.OrderedMap); ok {
					indexName, _ := gsiMap.Get("IndexName")
					keySchema, _ := gsiMap.Get("KeySchema")

					var gsiKeys []string
					if keySchemaList, ok := keySchema.([]interface{}); ok {
						for _, key := range keySchemaList {
							if keyMap, ok := key.(orderedmap.OrderedMap); ok {
								keyName, _ := keyMap.Get("AttributeName")
								keyType, _ := keyMap.Get("KeyType")
								if keyName != nil && keyType != nil {
									attrType := attributeTypes[keyName.(string)]
									displayKeyType := "PK"
									if keyType.(string) == "RANGE" {
										displayKeyType = "SK"
									}
									gsiKeys = append(gsiKeys, fmt.Sprintf("%s (%s:%s)", keyName.(string), displayKeyType, attrType))
								}
							}
						}
					}

					if len(gsiKeys) > 0 {
						result.Values = append(result.Values, []string{
							indexName.(string),
							"Global Secondary Index",
							fmt.Sprintf("%d", len(gsiKeys)),
							strings.Join(gsiKeys, ", "),
						})
					}
				}
			}
		}
	}

	// Parse Local Secondary Indexes (LSI)
	if lsiAttr, exists := tableMap.Get("LocalSecondaryIndexes"); exists {
		if lsiList, ok := lsiAttr.([]interface{}); ok {
			for _, lsi := range lsiList {
				if lsiMap, ok := lsi.(orderedmap.OrderedMap); ok {
					indexName, _ := lsiMap.Get("IndexName")
					keySchema, _ := lsiMap.Get("KeySchema")

					var lsiKeys []string
					if keySchemaList, ok := keySchema.([]interface{}); ok {
						for _, key := range keySchemaList {
							if keyMap, ok := key.(orderedmap.OrderedMap); ok {
								keyName, _ := keyMap.Get("AttributeName")
								keyType, _ := keyMap.Get("KeyType")
								if keyName != nil && keyType != nil {
									attrType := attributeTypes[keyName.(string)]
									displayKeyType := "PK"
									if keyType.(string) == "RANGE" {
										displayKeyType = "SK"
									}
									lsiKeys = append(lsiKeys, fmt.Sprintf("%s (%s:%s)", keyName.(string), displayKeyType, attrType))
								}
							}
						}
					}

					if len(lsiKeys) > 0 {
						result.Values = append(result.Values, []string{
							indexName.(string),
							"Local Secondary Index",
							fmt.Sprintf("%d", len(lsiKeys)),
							strings.Join(lsiKeys, ", "),
						})
					}
				}
			}
		}
	}

	if len(result.Values) == 0 {
		result.Values = [][]string{{"No indexes found", "", "", ""}}
	}

	return result
}

func ParseToObject(viewType string, parsedResult ParseCommandResult, command cmd.Command, commandHandler func(selectedProfileName string), app *tview.Application, restoreRootView func(), createHeader func() *tview.Flex, createFooter func([]string) *tview.Table, logView *tview.TextView, isLogEnabled bool) tview.Primitive {
	// If the output was unparsable, show it as plain text
	if parsedResult.RawOutput != "" {
		logger.Logger.Debug().Msg("Showing raw output as plain text")
		return createRawOutputView(parsedResult)
	}

	switch viewType {
	case "tableView":
		logger.Logger.Debug().Msg(fmt.Sprintf("Parse to %s", viewType))
		return parseToTableView(parsedResult, command, commandHandler, app, restoreRootView, createHeader, createFooter, logView, isLogEnabled)
	default:
		logger.Logger.Debug().Msg(fmt.Sprintf("View type '%s' not found", viewType))
		return nil
	}
}

func createRawOutputView(parsedResult ParseCommandResult) tview.Primitive {
	textView := tview.NewTextView().
		SetText(parsedResult.RawOutput).
		SetDynamicColors(false).
		SetScrollable(true).
		SetWrap(true)

	textView.
		SetBorder(true).
		SetTitle(fmt.Sprintf(" %s - Raw Output ", parsedResult.Command)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderPadding(1, 1, 2, 2)

	return textView
}

func mapCommandHeaderToColumn(headers []string) []ui.Column {
	var uiColumn []ui.Column
	for _, header := range headers {
		uiColumn = append(uiColumn, ui.Column{Name: header, Width: 0})
	}
	return uiColumn
}

func parseToTableView(parsedResult ParseCommandResult, command cmd.Command, commandHandler func(selectedProfileName string), app *tview.Application, restoreRootView func(), createHeader func() *tview.Flex, createFooter func([]string) *tview.Table, logView *tview.TextView, isLogEnabled bool) tview.Primitive {
	return ui.CreateCustomTableView(ui.CustomTableViewProperties{
		Title:          fmt.Sprintf(" %s [%d] ", parsedResult.Command, len(parsedResult.Values)),
		Columns:        mapCommandHeaderToColumn(parsedResult.Header),
		Rows:           parsedResult.Values,
		RowData:        parsedResult.RawData,
		Handler:        commandHandler,
		ShowJsonViewer: command.ShowJsonViewer,
		App:            app,
		RestoreRoot:    restoreRootView,
		CreateHeader:   createHeader,
		CreateFooter:   createFooter,
		LogView:        logView,
		IsLogEnabled:   isLogEnabled,
	})
}

// CreateErrorView creates a simple error message view
func CreateErrorView(commandName string, message string) tview.Primitive {
	textView := tview.NewTextView().
		SetText(message).
		SetDynamicColors(false).
		SetScrollable(false).
		SetWrap(true)

	textView.
		SetBorder(true).
		SetTitle(fmt.Sprintf(" %s - Error ", commandName)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderPadding(1, 1, 2, 2)

	return textView
}

// CreateContentView creates the default view for downloaded file content.
// Images are rendered with tview.Image; everything else is shown as raw text.
// Use CreatePrettyContentView to get the formatted/structured view.
func CreateContentView(commandName string, objectKey string, content []byte) tview.Primitive {
	fileName := filepath.Base(objectKey)
	contentType := detectContentType(objectKey, content)

	logger.Logger.Debug().
		Str("objectKey", objectKey).
		Str("contentType", contentType).
		Int("contentLength", len(content)).
		Msg("Creating content view (raw)")

	// Images are always rendered as images -- there is no useful "raw" alternative
	if strings.HasPrefix(contentType, "image/") {
		return createImageContentView(commandName, fileName, content)
	}

	return createTextContentView(commandName, fileName, content)
}

// CreatePrettyContentView creates a structured/formatted view for downloaded file content.
// Routes to the appropriate viewer based on detected content type:
// - Images: rendered with tview.Image (same as raw)
// - JSON: interactive tree view
// - JSONL/NDJSON: table with one row per JSON line
// - YAML: parsed to map, then rendered as tree view
// - CSV/TSV: parsed and rendered as a table
// - Everything else: plain text (same as raw)
func CreatePrettyContentView(commandName string, objectKey string, content []byte) tview.Primitive {
	fileName := filepath.Base(objectKey)
	contentType := detectContentType(objectKey, content)

	logger.Logger.Debug().
		Str("objectKey", objectKey).
		Str("contentType", contentType).
		Int("contentLength", len(content)).
		Msg("Creating content view (pretty)")

	switch {
	case strings.HasPrefix(contentType, "image/"):
		return createImageContentView(commandName, fileName, content)
	case contentType == "application/json":
		return createJsonContentView(commandName, fileName, content)
	case contentType == "application/jsonl":
		return createJsonlContentView(commandName, fileName, content)
	case contentType == "text/yaml":
		return createYamlContentView(commandName, fileName, content)
	case contentType == "text/csv":
		return createCsvContentView(commandName, fileName, content, ',')
	case contentType == "text/tab-separated-values":
		return createCsvContentView(commandName, fileName, content, '\t')
	default:
		return createTextContentView(commandName, fileName, content)
	}
}

// ContentViewHasPrettyFormat returns true if the content type supports a pretty/structured view
// that is different from the raw text view. Used to decide whether to show the 'v' toggle shortcut.
func ContentViewHasPrettyFormat(objectKey string, content []byte) bool {
	contentType := detectContentType(objectKey, content)
	switch {
	case contentType == "application/json",
		contentType == "application/jsonl",
		contentType == "text/yaml",
		contentType == "text/csv",
		contentType == "text/tab-separated-values":
		return true
	default:
		return false
	}
}

// detectContentType determines the MIME type of the content using the file extension
// first, then falling back to http.DetectContentType for sniffing the bytes.
func detectContentType(objectKey string, content []byte) string {
	ext := strings.ToLower(filepath.Ext(objectKey))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".bmp":
		return "image/bmp"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".csv":
		return "text/csv"
	case ".tsv":
		return "text/tab-separated-values"
	case ".txt", ".log", ".md":
		return "text/plain"
	case ".json":
		return "application/json"
	case ".jsonl", ".ndjson":
		return "application/jsonl"
	case ".xml", ".html", ".htm":
		return "text/html"
	case ".yaml", ".yml":
		return "text/yaml"
	}

	// Fall back to content sniffing
	return http.DetectContentType(content)
}

// createImageContentView renders image content using tview.Image
func createImageContentView(commandName string, fileName string, content []byte) tview.Primitive {
	img, _, err := image.Decode(bytes.NewReader(content))
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to decode image, falling back to text view")
		return createTextContentView(commandName, fileName, content)
	}

	imageView := tview.NewImage()
	imageView.SetImage(img)
	imageView.SetColors(tview.TrueColor)
	imageView.SetDithering(tview.DitheringFloydSteinberg)

	imageView.
		SetBorder(true).
		SetTitle(fmt.Sprintf(" %s - %s ", commandName, fileName)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderPadding(0, 0, 1, 1)

	return imageView
}

// createTextContentView renders text/binary content in a scrollable TextView
func createTextContentView(commandName string, fileName string, content []byte) tview.Primitive {
	textView := tview.NewTextView().
		SetText(string(content)).
		SetDynamicColors(false).
		SetScrollable(true).
		SetWrap(true)

	textView.
		SetBorder(true).
		SetTitle(fmt.Sprintf(" %s - %s ", commandName, fileName)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderPadding(1, 1, 2, 2)

	return textView
}

// createJsonContentView parses JSON content and renders it as an interactive tree view.
// Falls back to plain text view if the content cannot be parsed as JSON.
func createJsonContentView(commandName string, fileName string, content []byte) tview.Primitive {
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to parse JSON content, falling back to text view")
		return createTextContentView(commandName, fileName, content)
	}

	root := tview.NewTreeNode(fileName).
		SetColor(tcell.ColorGold).
		SetExpanded(true)

	buildContentTree(data, root)
	expandAllContentNodes(root)

	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	tree.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s - %s (JSON) ", commandName, fileName)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	return tree
}

// createYamlContentView parses YAML content and renders it as an interactive tree view.
// Falls back to plain text view if the content cannot be parsed as YAML.
func createYamlContentView(commandName string, fileName string, content []byte) tview.Primitive {
	var data interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to parse YAML content, falling back to text view")
		return createTextContentView(commandName, fileName, content)
	}

	// yaml.Unmarshal produces map[interface{}]interface{}, convert to map[string]interface{}
	data = normalizeYamlValue(data)

	root := tview.NewTreeNode(fileName).
		SetColor(tcell.ColorGold).
		SetExpanded(true)

	buildContentTree(data, root)
	expandAllContentNodes(root)

	tree := tview.NewTreeView().
		SetRoot(root).
		SetCurrentNode(root)

	tree.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s - %s (YAML) ", commandName, fileName)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	return tree
}

// createCsvContentView parses CSV (or TSV) content and renders it as a table.
// The first row is used as column headers.
// Falls back to plain text view if the content cannot be parsed.
func createCsvContentView(commandName string, fileName string, content []byte, delimiter rune) tview.Primitive {
	reader := csv.NewReader(bytes.NewReader(content))
	reader.Comma = delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	records, err := reader.ReadAll()
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to parse CSV content, falling back to text view")
		return createTextContentView(commandName, fileName, content)
	}

	if len(records) == 0 {
		return createTextContentView(commandName, fileName, content)
	}

	table := tview.NewTable()

	table.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s - %s [%d rows] ", commandName, fileName, len(records)-1)).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderPadding(0, 1, 2, 2)

	table.SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorGold))

	// First row as header
	for colIndex, header := range records[0] {
		table.SetCell(0, colIndex, tview.NewTableCell(header).
			SetAlign(tview.AlignLeft).
			SetMaxWidth(0).
			SetSelectable(false))
	}

	// Data rows
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		for colIndex, cellData := range records[rowIndex] {
			table.SetCell(rowIndex, colIndex, tview.NewTableCell(cellData).
				SetExpansion(1).
				SetAlign(tview.AlignLeft).
				SetSelectable(true))
		}
	}

	return table
}

// createJsonlContentView parses JSONL/NDJSON content (one JSON object per line) and renders
// it as a table. Column headers are derived from the union of all keys across all lines.
// Falls back to plain text view if no lines can be parsed.
func createJsonlContentView(commandName string, fileName string, content []byte) tview.Primitive {
	scanner := bufio.NewScanner(bytes.NewReader(content))

	// Collect all parsed objects and track column order
	var objects []map[string]interface{}
	var columnOrder []string
	columnSet := map[string]bool{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			// Skip non-JSON lines
			continue
		}
		// Track new keys in order of first appearance
		for key := range obj {
			if !columnSet[key] {
				columnSet[key] = true
				columnOrder = append(columnOrder, key)
			}
		}
		objects = append(objects, obj)
	}

	if len(objects) == 0 {
		return createTextContentView(commandName, fileName, content)
	}

	table := tview.NewTable()

	table.SetBorder(true).
		SetTitle(fmt.Sprintf(" %s - %s [%d rows] (JSONL) ", commandName, fileName, len(objects))).
		SetTitleAlign(tview.AlignCenter).
		SetBorderColor(tview.Styles.BorderColor).
		SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorderPadding(0, 1, 2, 2)

	table.SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorGold))

	// Header row
	for colIndex, header := range columnOrder {
		table.SetCell(0, colIndex, tview.NewTableCell(header).
			SetAlign(tview.AlignLeft).
			SetMaxWidth(0).
			SetSelectable(false))
	}

	// Data rows
	for rowIndex, obj := range objects {
		for colIndex, key := range columnOrder {
			val, exists := obj[key]
			cellText := ""
			if exists && val != nil {
				switch v := val.(type) {
				case string:
					cellText = v
				default:
					b, _ := json.Marshal(v)
					cellText = string(b)
				}
			}
			table.SetCell(rowIndex+1, colIndex, tview.NewTableCell(cellText).
				SetExpansion(1).
				SetAlign(tview.AlignLeft).
				SetSelectable(true))
		}
	}

	return table
}

// buildContentTree recursively builds tree nodes from parsed data (JSON or YAML).
func buildContentTree(data interface{}, parent *tview.TreeNode) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			node := tview.NewTreeNode(fmt.Sprintf("[yellow]%s", key)).
				SetColor(tcell.ColorYellow).
				SetSelectable(true).
				SetExpanded(true)
			parent.AddChild(node)
			buildContentTree(val, node)
		}
	case []interface{}:
		for i, val := range v {
			node := tview.NewTreeNode(fmt.Sprintf("[white][%d]", i)).
				SetColor(tcell.ColorWhite).
				SetSelectable(true).
				SetExpanded(true)
			parent.AddChild(node)
			buildContentTree(val, node)
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

// expandAllContentNodes recursively expands all tree nodes.
func expandAllContentNodes(node *tview.TreeNode) {
	node.SetExpanded(true)
	for _, child := range node.GetChildren() {
		expandAllContentNodes(child)
	}
}

// normalizeYamlValue converts yaml.Unmarshal output (which uses map[interface{}]interface{})
// into standard map[string]interface{} so it can be rendered by the tree builder.
func normalizeYamlValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]interface{})
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = normalizeYamlValue(v)
		}
		return result
	case []interface{}:
		for i, item := range val {
			val[i] = normalizeYamlValue(item)
		}
		return val
	default:
		return v
	}
}
