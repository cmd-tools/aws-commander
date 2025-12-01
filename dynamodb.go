package main

import (
	"fmt"
	"strings"

	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/logger"
	"github.com/cmd-tools/aws-commander/ui"
	"github.com/rivo/tview"
)

// KeyInfo represents a DynamoDB key with its metadata
type KeyInfo struct {
	Name     string // Key name
	Type     string // PK (Partition Key) or SK (Sort Key)
	AttrType string // S (String), N (Number), B (Binary)
}

// showKeyInputForm displays an input form for DynamoDB query parameters
func showKeyInputForm() {
	// Hide search bar when showing input form
	cmd.UiState.CommandBarVisible = false
	Search.SetText("")
	cmd.UiState.OriginalTableData = nil

	selectedIndexName := getSelectedIndexName()
	indexKeys, indexType := extractIndexDetails(selectedIndexName)

	logger.Logger.Debug().
		Str("selectedIndexName", selectedIndexName).
		Str("indexType", indexType).
		Interface("indexKeys", indexKeys).
		Msg("Extracted index details for query")

	inputForm := createQueryInputForm(selectedIndexName, indexKeys, indexType)

	Body = inputForm
	updateRootView(nil)
	App.SetFocus(inputForm)
}

// getSelectedIndexName retrieves the selected index name from navigation or state
func getSelectedIndexName() string {
	// Look through the navigation stack to find the selected item from describe-table
	for i := len(cmd.UiState.NavigationStack) - 1; i >= 0; i-- {
		state := cmd.UiState.NavigationStack[i]
		if state.Type == cmd.BreadcrumbSelectedItem {
			return state.Value
		}
	}

	// Fallback: try the describe-table's resourceName
	for _, c := range cmd.UiState.Resource.Commands {
		if c.Name == "describe-table" && c.ResourceName != "" {
			resourceKey := cmd.VariablePlaceHolderPrefix + strings.ToUpper(c.ResourceName)
			if selectedName, ok := cmd.UiState.SelectedItems[resourceKey]; ok {
				return selectedName
			}
		}
	}

	return ""
}

// extractIndexDetails parses index information from the cached describe-table result
func extractIndexDetails(selectedIndexName string) ([]KeyInfo, string) {
	var indexKeys []KeyInfo
	var indexType string

	// Find the describe-table navigation state with cached result
	for i := len(cmd.UiState.NavigationStack) - 1; i >= 0; i-- {
		state := cmd.UiState.NavigationStack[i]
		if state.Type == cmd.BreadcrumbDependentCmd && state.Value == "describe-table" && state.CachedBody != nil {
			if table, ok := state.CachedBody.(*tview.Table); ok {
				indexKeys, indexType = parseIndexFromTable(table, selectedIndexName)
				break
			}
		}
	}

	return indexKeys, indexType
}

// parseIndexFromTable extracts key details from the describe-table result table
func parseIndexFromTable(table *tview.Table, selectedIndexName string) ([]KeyInfo, string) {
	var indexKeys []KeyInfo
	var indexType string

	rowCount := table.GetRowCount()
	for row := 1; row < rowCount; row++ { // Skip header row
		indexNameCell := table.GetCell(row, 0)
		if indexNameCell == nil || indexNameCell.Text != selectedIndexName {
			continue
		}

		// Found the selected index row
		indexTypeCell := table.GetCell(row, 1)
		if indexTypeCell != nil {
			indexType = indexTypeCell.Text
		}

		keyDetailsCell := table.GetCell(row, 3)
		if keyDetailsCell != nil {
			indexKeys = parseKeyDetails(keyDetailsCell.Text)
		}
		break
	}

	return indexKeys, indexType
}

// parseKeyDetails parses key information from formatted string
// Format: "KeyName (PK:S), KeyName (SK:S)"
func parseKeyDetails(keyDetailsText string) []KeyInfo {
	var indexKeys []KeyInfo

	keyDetailsParts := strings.Split(keyDetailsText, ", ")
	for _, keyDetail := range keyDetailsParts {
		if idx := strings.Index(keyDetail, " ("); idx > 0 {
			keyName := keyDetail[:idx]
			typesPart := keyDetail[idx+2 : len(keyDetail)-1] // Remove " (" and ")"
			types := strings.Split(typesPart, ":")
			if len(types) == 2 {
				indexKeys = append(indexKeys, KeyInfo{
					Name:     keyName,
					Type:     types[0], // PK or SK
					AttrType: types[1], // S, N, B
				})
			}
		}
	}

	return indexKeys
}

// createQueryInputForm creates the input form for query parameters
func createQueryInputForm(selectedIndexName string, indexKeys []KeyInfo, indexType string) *tview.Form {
	// Create input fields for all keys in the index
	var inputFields []ui.InputField
	for _, key := range indexKeys {
		displayLabel := fmt.Sprintf("%s (%s)", key.Name, key.Type)
		inputFields = append(inputFields, ui.InputField{
			Label:        displayLabel,
			Key:          key.Name,
			DefaultValue: "",
		})
	}

	// Build title showing index name
	formTitle := fmt.Sprintf(" Enter values for: %s ", selectedIndexName)
	if len(indexKeys) == 1 {
		formTitle = fmt.Sprintf(" Enter value for: %s ", selectedIndexName)
	}

	return ui.CreateInputForm(ui.InputFormProperties{
		Title:    formTitle,
		Fields:   inputFields,
		OnSubmit: createQuerySubmitHandler(indexKeys, indexType, selectedIndexName),
		OnCancel: createQueryCancelHandler(),
		App:      App,
	})
}

// createQuerySubmitHandler returns the submit handler for the query form
func createQuerySubmitHandler(indexKeys []KeyInfo, indexType, selectedIndexName string) func(map[string]string) {
	return func(values map[string]string) {
		// Validate that all required fields have values
		for _, key := range indexKeys {
			if values[key.Name] == "" {
				logger.Logger.Warn().Str("key", key.Name).Msg("Key value cannot be empty")
				return
			}
		}

		// Build the key-condition-expression
		keyConditionExpr, expressionAttrValues := buildQueryExpression(indexKeys, values)

		logger.Logger.Debug().
			Str("keyConditionExpr", keyConditionExpr).
			Str("expressionAttrValues", expressionAttrValues).
			Msg("Built query expression")

		// Add the query parameters to the command arguments
		cmd.UiState.Command.Arguments = append(cmd.UiState.Command.Arguments,
			"--key-condition-expression", keyConditionExpr,
			"--expression-attribute-values", expressionAttrValues,
		)

		// If querying a GSI or LSI, add the --index-name parameter
		if indexType == "Global Secondary Index" || indexType == "Local Secondary Index" {
			cmd.UiState.Command.Arguments = append(cmd.UiState.Command.Arguments,
				"--index-name", selectedIndexName,
			)
		}

		// Reset pagination state for new query
		cmd.UiState.CurrentPageToken = ""
		cmd.UiState.PageHistory = []string{}

		// Execute the query command
		_, body := executeCommand(cmd.UiState.Command)
		Body = body

		updateRootView(nil)
	}
}

// buildQueryExpression builds the DynamoDB query expression and attribute values
func buildQueryExpression(indexKeys []KeyInfo, values map[string]string) (string, string) {
	var keyConditionParts []string
	expressionAttrValuesMap := make(map[string]map[string]string)

	for i, key := range indexKeys {
		placeholder := fmt.Sprintf(":val%d", i)

		// Add condition for key
		keyConditionParts = append(keyConditionParts, fmt.Sprintf("%s = %s", key.Name, placeholder))

		// Map attribute type (S, N, B) to value
		expressionAttrValuesMap[placeholder] = map[string]string{
			key.AttrType: values[key.Name],
		}
	}

	keyConditionExpr := strings.Join(keyConditionParts, " AND ")

	// Build expression-attribute-values JSON
	var attrValuePairs []string
	for placeholder, attrValue := range expressionAttrValuesMap {
		for attrType, value := range attrValue {
			attrValuePairs = append(attrValuePairs, fmt.Sprintf(`"%s": {"%s": "%s"}`, placeholder, attrType, value))
		}
	}
	expressionAttrValues := fmt.Sprintf(`{%s}`, strings.Join(attrValuePairs, ", "))

	return keyConditionExpr, expressionAttrValues
}

// createQueryCancelHandler returns the cancel handler for the query form
func createQueryCancelHandler() func() {
	return func() {
		// Go back from the input form to the describe-table results
		popNavigation()

		// Navigate back to the describe-table (index list)
		parentState := peekNavigation()
		if parentState != nil && (parentState.Type == cmd.BreadcrumbCommand || parentState.Type == cmd.BreadcrumbDependentCmd) {
			parentCommandName := parentState.Value
			cmd.UiState.Command = cmd.UiState.Resource.GetCommand(parentCommandName)

			if parentState.CachedBody != nil && !cmd.UiState.Command.RerunOnBack {
				Body = parentState.CachedBody
			} else {
				_, body := executeCommand(cmd.UiState.Command)
				Body = body
			}
		}
		updateRootView(nil)
	}
}
