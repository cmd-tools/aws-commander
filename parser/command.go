package parser

import (
	"encoding/json"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/rivo/tview"
)

type ParseCommandResult struct {
	Command string
	Header  []string
	Values  [][]string
}

func ParseCommand(command cmd.Command, commandOutput string) ParseCommandResult {
	var jsonResult map[string]any
	json.Unmarshal([]byte(commandOutput), &jsonResult)

	var parseCommandResult = ParseCommandResult{Command: command.Name}
	switch jsonResult[command.Parse.AttributeName].(type) {
	case []interface{}:
		logger.Logger.Debug().Msg("Parse command list")
		for i, s := range jsonResult[command.Parse.AttributeName].([]interface{}) {
			var values []string
			if command.Parse.Type == "object" {
				for key, value := range s.(map[string]any) {
					if i == 0 {
						parseCommandResult.Header = append(parseCommandResult.Header, key)
					}
					values = append(values, value.(string))
				}
				parseCommandResult.Values = append(parseCommandResult.Values, values)
			} else if command.Parse.Type == "list" {
				if i == 0 {
					parseCommandResult.Header = append(parseCommandResult.Header, "Item")
				}
				parseCommandResult.Values = append(parseCommandResult.Values, append(values, s.(string)))
			} else {
				logger.Logger.Debug().Msg("Wrong type. Accepted types [Object, List]")
			}
		}
	case map[string]interface{}:
		logger.Logger.Debug().Msg("Parse command objet")
		var values []string
		for key, value := range jsonResult[command.Parse.AttributeName].(map[string]any) {
			parseCommandResult.Header = append(parseCommandResult.Header, key)
			values = append(values, value.(string))
		}
		parseCommandResult.Values = append(parseCommandResult.Values, values)
	default:
		logger.Logger.Debug().Msg("Fail to Parse. Command result is not an Map or List")
	}
	return parseCommandResult
}

func ParseToObject(viewType string, parsedResult ParseCommandResult) tview.Primitive {
	switch viewType {
	case "tableView":
		logger.Logger.Debug().Msg("Parse to TableView")
		return parseToTableView(parsedResult)
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

func parseToTableView(parsedResult ParseCommandResult) tview.Primitive {
	return ui.CreateCustomTableView(ui.CustomTableViewProperties{
		Title:   parsedResult.Command,
		Columns: mapCommandHeaderToColumn(parsedResult.Header),
		Rows:    parsedResult.Values,
	})
}
