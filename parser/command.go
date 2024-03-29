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
}

func ParseCommand(command cmd.Command, commandOutput string) ParseCommandResult {
	jsonResult := orderedmap.New()
	json.Unmarshal([]byte(commandOutput), &jsonResult)

	var parseCommandResult = ParseCommandResult{Command: command.Name}
	baseAttribute, _ := jsonResult.Get(command.Parse.AttributeName)
	switch baseAttribute.(type) {
	case []interface{}:
		logger.Logger.Debug().Msg("Parse command list")
		for i, s := range baseAttribute.([]interface{}) {
			var values []string
			if command.Parse.Type == "object" {
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

func ParseToObject(viewType string, parsedResult ParseCommandResult, commandHandler func(selectedProfileName string)) tview.Primitive {
	switch viewType {
	case "tableView":
		logger.Logger.Debug().Msg("Parse to TableView")
		return parseToTableView(parsedResult, commandHandler)
	default:
		logger.Logger.Debug().Msg("View type not found")
		return nil
	}
}

func mapCommandHeaderToColumn(headers []string) []ui.Column {
	var uiColumn = []ui.Column{}
	for _, header := range headers {
		uiColumn = append(uiColumn, ui.Column{Name: header, Width: 0})
	}
	return uiColumn
}

func parseToTableView(parsedResult ParseCommandResult, commandHandler func(selectedProfileName string)) tview.Primitive {
	return ui.CreateCustomTableView(ui.CustomTableViewProperties{
		Title:   parsedResult.Command,
		Columns: mapCommandHeaderToColumn(parsedResult.Header),
		Rows:    parsedResult.Values,
		Handler: commandHandler,
	})
}
