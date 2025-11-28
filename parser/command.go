package parser

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/iancoleman/orderedmap"
	"github.com/rivo/tview"
)

type ParseCommandResult struct {
	Command string
	Header  []string
	Values  [][]string
	RawData []interface{}
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
			Command: command.Name,
			Header:  []string{"Error"},
			Values:  [][]string{{"Failed to parse JSON output"}},
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
		logger.Logger.Debug().Msg("Parse command objet")
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

func ParseToObject(viewType string, parsedResult ParseCommandResult, command cmd.Command, commandHandler func(selectedProfileName string), app *tview.Application, restoreRootView func(), createHeader func() *tview.Flex, createFooter func([]string) *tview.Table, logView *tview.TextView, isLogEnabled bool) tview.Primitive {
	switch viewType {
	case "tableView":
		logger.Logger.Debug().Msg(fmt.Sprintf("Parse to %s", viewType))
		return parseToTableView(parsedResult, command, commandHandler, app, restoreRootView, createHeader, createFooter, logView, isLogEnabled)
	default:
		logger.Logger.Debug().Msg(fmt.Sprintf("View type '%s' not found", viewType))
		return nil
	}
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
