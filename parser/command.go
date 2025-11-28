package parser

import (
	"encoding/json"
	"fmt"

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
	jsonResult := orderedmap.New()
	err := json.Unmarshal([]byte(commandOutput), &jsonResult)
	if err != nil {
		panic(fmt.Sprintf("Unable to unmarshal json for command: %s", command.Name))
	}

	var parseCommandResult = ParseCommandResult{Command: command.Name}
	baseAttribute, _ := jsonResult.Get(command.Parse.AttributeName)
	switch baseAttribute.(type) {
	case []interface{}:
		logger.Logger.Debug().Msg("Parse command list")
		for i, s := range baseAttribute.([]interface{}) {
			var values []string
			if command.Parse.Type == "object" {
				// Store raw data for JSON viewer
				parseCommandResult.RawData = append(parseCommandResult.RawData, s)

				item := s.(orderedmap.OrderedMap)
				for _, key := range item.Keys() {
					if i == 0 {
						parseCommandResult.Header = append(parseCommandResult.Header, key)
					}
					value, exists := item.Get(key)
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
		item := baseAttribute.(orderedmap.OrderedMap)
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
