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
	Command   string
	Header    []string
	Values    [][]string
	Count     uint64
	NextToken string
}

func ParseCommand(command cmd.Command, commandOutput string) ParseCommandResult {
	jsonResult := orderedmap.New()
	err := json.Unmarshal([]byte(commandOutput), &jsonResult)
	if err != nil {
		panic(fmt.Sprintf("Unable to unmarshal json for command: %s", command))
	}

	var parseCommandResult = ParseCommandResult{Command: command.Name}
	baseAttribute, _ := jsonResult.Get(command.Parse.AttributeName)

	if count, ok := jsonResult.Get("Count"); ok && count != nil {
		parseCommandResult.Count = uint64(int(count.(float64)))
	}

	if nextToken, isPaginating := jsonResult.Get("NextToken"); isPaginating && nextToken != nil {
		parseCommandResult.NextToken = nextToken.(string)
	}

	switch baseAttribute.(type) {
	case []interface{}:
		logger.Logger.Debug().Msg("Parse command list")
		for i, s := range baseAttribute.([]interface{}) {
			var values []string
			switch command.Parse.Type {
			case "object":
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
							break
						default:
							bytes, _ := json.Marshal(value)
							values = append(values, fmt.Sprintf("%v", string(bytes)))
						}
					}
				}
				parseCommandResult.Values = append(parseCommandResult.Values, values)
				break
			case "list":
				if i == 0 {
					parseCommandResult.Header = append(parseCommandResult.Header, "Item")
				}
				if s != nil {
					parseCommandResult.Values = append(parseCommandResult.Values, append(values, s.(string)))
				}
				break
			default:
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
		logger.Logger.Debug().Msg(fmt.Sprintf("Parse to %s", viewType))
		return parseToTableView(parsedResult, commandHandler)
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

func parseToTableView(parsedResult ParseCommandResult, commandHandler func(selectedProfileName string)) tview.Primitive {
	return ui.CreateCustomTableView(ui.CustomTableViewProperties{
		Title:   fmt.Sprintf(" %s [%d] ", parsedResult.Command, len(parsedResult.Values)),
		Columns: mapCommandHeaderToColumn(parsedResult.Header),
		Rows:    parsedResult.Values,
		Handler: commandHandler,
	})
}
