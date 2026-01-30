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
		// Mark sort key as optional for primary key queries
		if key.Type == "SK" && indexType == "Primary Index" {
			displayLabel = fmt.Sprintf("%s (%s, optional)", key.Name, key.Type)
		}
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
		// Validate that partition key (PK) has a value - it's always required
		// Sort key (SK) is optional for primary key queries
		for _, key := range indexKeys {
			if key.Type == "PK" && values[key.Name] == "" {
				logger.Logger.Warn().Str("key", key.Name).Msg("Partition key value cannot be empty")
				return
			}
			// For non-primary indexes or when SK is provided, validate it's not empty
			// But for primary index, SK can be empty
			if key.Type == "SK" && indexType != "Primary Index" && values[key.Name] == "" {
				logger.Logger.Warn().Str("key", key.Name).Msg("Sort key value cannot be empty for this index type")
				return
			}
		}

		// Build the key-condition-expression
		keyConditionExpr, expressionAttrValues, expressionAttrNames := buildQueryExpression(indexKeys, values)

		logger.Logger.Debug().
			Str("keyConditionExpr", keyConditionExpr).
			Str("expressionAttrValues", expressionAttrValues).
			Str("expressionAttrNames", expressionAttrNames).
			Msg("Built query expression")

		// Add the query parameters to the command arguments
		cmd.UiState.Command.Arguments = append(cmd.UiState.Command.Arguments,
			"--key-condition-expression", keyConditionExpr,
			"--expression-attribute-values", expressionAttrValues,
		)

		if expressionAttrNames != "" {
			cmd.UiState.Command.Arguments = append(cmd.UiState.Command.Arguments,
				"--expression-attribute-names", expressionAttrNames,
			)
		}

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

var dynamoReservedWords = map[string]bool{
	"ABORT": true, "ABSOLUTE": true, "ACTION": true, "ADD": true, "AFTER": true, "AGENT": true, "AGGREGATE": true, "ALL": true, "ALLOCATE": true, "ALTER": true,
	"ANALYZE": true, "AND": true, "ANY": true, "ARCHIVE": true, "ARE": true, "ARRAY": true, "AS": true, "ASC": true, "ASCII": true, "ASENSITIVE": true,
	"ASSERTION": true, "ASYMMETRIC": true, "AT": true, "ATOMIC": true, "ATTACH": true, "ATTRIBUTE": true, "AUTH": true, "AUTHORIZATION": true, "AUTHORIZE": true,
	"AUTO": true, "AVG": true, "BACK": true, "BACKUP": true, "BASE": true, "BATCH": true, "BEFORE": true, "BEGIN": true, "BETWEEN": true, "BIGINT": true,
	"BINARY": true, "BIT": true, "BLOB": true, "BLOCK": true, "BOOLEAN": true, "BOTH": true, "BREADTH": true, "BUCKET": true, "BULK": true, "BY": true,
	"BYTE": true, "CALL": true, "CALLED": true, "CALLING": true, "CAPACITY": true, "CASCADE": true, "CASCADED": true, "CASE": true, "CAST": true, "CATALOG": true,
	"CHAR": true, "CHARACTER": true, "CHECK": true, "CLASS": true, "CLOB": true, "CLOSE": true, "CLUSTER": true, "CLUSTERED": true, "CLUSTERING": true, "CLUSTERS": true,
	"COALESCE": true, "COLLATE": true, "COLLATION": true, "COLLECTION": true, "COLUMN": true, "COLUMNS": true, "COMBINE": true, "COMMENT": true, "COMMIT": true,
	"COMPACT": true, "COMPILE": true, "COMPRESS": true, "CONDITION": true, "CONFLICT": true, "CONNECT": true, "CONNECTION": true, "CONSISTENCY": true, "CONSISTENT": true,
	"CONSTRAINT": true, "CONSTRAINTS": true, "CONSTRUCTOR": true, "CONSUMED": true, "CONTINUE": true, "CONVERT": true, "COPY": true, "CORRESPONDING": true, "COUNT": true,
	"COUNTER": true, "CREATE": true, "CROSS": true, "CUBE": true, "CURRENT": true, "CURSOR": true, "CYCLE": true, "DATA": true, "DATABASE": true, "DATE": true,
	"DATETIME": true, "DAY": true, "DEALLOCATE": true, "DEC": true, "DECIMAL": true, "DECLARE": true, "DEFAULT": true, "DEFERRABLE": true, "DEFERRED": true, "DEFINE": true,
	"DEFINED": true, "DEFINITION": true, "DELETE": true, "DELIMITED": true, "DEPTH": true, "DEREF": true, "DESC": true, "DESCRIBE": true, "DESCRIPTOR": true,
	"DETACH": true, "DETERMINISTIC": true, "DIAGNOSTICS": true, "DIRECTORIES": true, "DISABLE": true, "DISCONNECT": true, "DISTINCT": true, "DISTRIBUTE": true, "DO": true,
	"DOMAIN": true, "DOUBLE": true, "DROP": true, "DUMP": true, "DURATION": true, "DYNAMIC": true, "EACH": true, "ELEMENT": true, "ELSE": true, "ELSEIF": true,
	"EMPTY": true, "ENABLE": true, "END": true, "EQUAL": true, "EQUALS": true, "ERROR": true, "ESCAPE": true, "ESCAPED": true, "EVAL": true, "EVALUATE": true,
	"EXCEEDED": true, "EXCEPT": true, "EXCEPTION": true, "EXCEPTIONS": true, "EXCLUSIVE": true, "EXEC": true, "EXECUTE": true, "EXISTS": true, "EXIT": true, "EXPLAIN": true,
	"EXPLODE": true, "EXPORT": true, "EXPRESSION": true, "EXTENDED": true, "EXTERNAL": true, "EXTRACT": true, "FAIL": true, "FALSE": true, "FAMILY": true, "FETCH": true,
	"FIELDS": true, "FILE": true, "FILTER": true, "FILTERING": true, "FINAL": true, "FINISH": true, "FIRST": true, "FIXED": true, "FLATTERN": true, "FLOAT": true,
	"FOR": true, "FORCE": true, "FOREIGN": true, "FORMAT": true, "FORWARD": true, "FOUND": true, "FREE": true, "FROM": true, "FULL": true, "FUNCTION": true,
	"FUNCTIONS": true, "GENERAL": true, "GENERATE": true, "GET": true, "GLOB": true, "GLOBAL": true, "GO": true, "GOTO": true, "GRANT": true, "GREATER": true,
	"GROUP": true, "GROUPING": true, "HANDLER": true, "HASH": true, "HAVE": true, "HAVING": true, "HEAP": true, "HIDDEN": true, "HOLD": true, "HOUR": true,
	"IDENTIFIED": true, "IDENTITY": true, "IF": true, "IGNORE": true, "IMMEDIATE": true, "IMPORT": true, "IN": true, "INCLUDING": true, "INCLUSIVE": true, "INCREMENT": true,
	"INCREMENTAL": true, "INDEX": true, "INDEXED": true, "INDEXES": true, "INDICATOR": true, "INFINITE": true, "INITIALLY": true, "INLINE": true, "INNER": true, "INNTER": true,
	"INOUT": true, "INPUT": true, "INSENSITIVE": true, "INSERT": true, "INSTEAD": true, "INT": true, "INTEGER": true, "INTERSECT": true, "INTERVAL": true, "INTO": true,
	"INVALIDATE": true, "IS": true, "ISOLATION": true, "ITEM": true, "ITEMS": true, "ITERATE": true, "JOIN": true, "KEY": true, "KEYS": true, "LAG": true,
	"LANGUAGE": true, "LARGE": true, "LAST": true, "LATERAL": true, "LEAD": true, "LEADING": true, "LEAVE": true, "LEFT": true, "LENGTH": true, "LESS": true,
	"LEVEL": true, "LIKE": true, "LIMIT": true, "LIMITED": true, "LINES": true, "LIST": true, "LOAD": true, "LOCAL": true, "LOCALTIME": true, "LOCALTIMESTAMP": true,
	"LOCATION": true, "LOCATOR": true, "LOCK": true, "LOCKS": true, "LOG": true, "LOGED": true, "LONG": true, "LOOP": true, "LOWER": true, "MAP": true,
	"MATCH": true, "MATERIALIZED": true, "MAX": true, "MAXLEN": true, "MEMBER": true, "MERGE": true, "METHOD": true, "METRICS": true, "MIN": true, "MINUS": true,
	"MINUTE": true, "MISSING": true, "MOD": true, "MODE": true, "MODIFIES": true, "MODIFY": true, "MODULE": true, "MONTH": true, "MULTI": true, "MULTISET": true,
	"NAME": true, "NAMES": true, "NATIONAL": true, "NATURAL": true, "NCHAR": true, "NCLOB": true, "NEW": true, "NEXT": true, "NO": true, "NONE": true,
	"NOT": true, "NULL": true, "NULLIF": true, "NUMBER": true, "NUMERIC": true, "OBJECT": true, "OF": true, "OFFLINE": true, "OFFSET": true, "OLD": true,
	"ON": true, "ONLINE": true, "ONLY": true, "OPAQUE": true, "OPEN": true, "OPERATOR": true, "OPTION": true, "OR": true, "ORDER": true, "ORDINALITY": true,
	"OTHER": true, "OTHERS": true, "OUT": true, "OUTER": true, "OUTPUT": true, "OVER": true, "OVERLAPS": true, "OVERRIDE": true, "OWNER": true, "PAD": true,
	"PARALLEL": true, "PARAMETER": true, "PARAMETERS": true, "PARTIAL": true, "PARTITION": true, "PARTITIONED": true, "PARTITIONS": true, "PATH": true, "PERCENT": true,
	"PERCENTILE": true, "PERMISSION": true, "PERMISSIONS": true, "PIPE": true, "PIPELINED": true, "PLAN": true, "POOL": true, "POSITION": true, "PRECISION": true, "PREPARE": true,
	"PRESERVE": true, "PRIMARY": true, "PRIOR": true, "PRIVATE": true, "PRIVILEGES": true, "PROCEDURE": true, "PROCESSED": true, "PROJECT": true, "PROJECTION": true,
	"PROPERTY": true, "PROVISIONING": true, "PUBLIC": true, "PUT": true, "QUERY": true, "QUIT": true, "QUORUM": true, "RAISE": true, "RANDOM": true, "RANGE": true,
	"RANK": true, "RAW": true, "READ": true, "READS": true, "REAL": true, "REBUILD": true, "RECORD": true, "RECURSIVE": true, "REDUCE": true, "REF": true,
	"REFERENCE": true, "REFERENCES": true, "REFERENCING": true, "REGEXP": true, "REGION": true, "REINDEX": true, "RELATIVE": true, "RELEASE": true, "REMAINDER": true,
	"RENAME": true, "REPEAT": true, "REPLACE": true, "REQUEST": true, "RESET": true, "RESIGNAL": true, "RESOURCE": true, "RESPONSE": true, "RESTORE": true, "RESTRICT": true,
	"RESULT": true, "RETURN": true, "RETURNING": true, "RETURNS": true, "REVERSE": true, "REVOKE": true, "RIGHT": true, "ROLE": true, "ROLES": true, "ROLLBACK": true,
	"ROLLUP": true, "ROUTINE": true, "ROW": true, "ROWS": true, "RULE": true, "RULES": true, "SAMPLE": true, "SATISFIES": true, "SAVE": true, "SAVEPOINT": true,
	"SCAN": true, "SCHEMA": true, "SCOPE": true, "SCROLL": true, "SEARCH": true, "SECOND": true, "SECTION": true, "SEGMENT": true, "SEGMENTS": true, "SELECT": true,
	"SELF": true, "SEMI": true, "SENSITIVE": true, "SEPARATE": true, "SEQUENCE": true, "SERIALIZABLE": true, "SESSION": true, "SET": true, "SETS": true, "SHARD": true,
	"SHARE": true, "SHARED": true, "SHORT": true, "SHOW": true, "SIGNAL": true, "SIMILAR": true, "SIZE": true, "SKEWED": true, "SMALLINT": true, "SNAPSHOT": true,
	"SOME": true, "SOURCE": true, "SPACE": true, "SPACES": true, "SPARSE": true, "SPECIFIC": true, "SPECIFICTYPE": true, "SPLIT": true, "SQL": true, "SQLCODE": true,
	"SQLERROR": true, "SQLEXCEPTION": true, "SQLSTATE": true, "SQLWARNING": true, "START": true, "STATE": true, "STATIC": true, "STATUS": true, "STORAGE": true,
	"STORE": true, "STORED": true, "STREAM": true, "STRING": true, "STRUCT": true, "STYLE": true, "SUB": true, "SUBMULTISET": true, "SUBPARTITION": true, "SUBSTRING": true,
	"SUBTYPE": true, "SUM": true, "SUPER": true, "SYMMETRIC": true, "SYNONYM": true, "SYSTEM": true, "TABLE": true, "TABLESAMPLE": true, "TEMP": true, "TEMPORARY": true,
	"TERMINATED": true, "TEXT": true, "THAN": true, "THEN": true, "THROUGHPUT": true, "TIME": true, "TIMESTAMP": true, "TIMEZONE": true, "TINYINT": true, "TO": true,
	"TOKEN": true, "TOTAL": true, "TOUCH": true, "TRAILING": true, "TRANSACTION": true, "TRANSFORM": true, "TRANSLATE": true, "TRANSLATION": true, "TREAT": true, "TRIGGER": true,
	"TRIM": true, "TRUE": true, "TRUNCATE": true, "TTL": true, "TUPLE": true, "TYPE": true, "UNDER": true, "UNDO": true, "UNION": true, "UNIQUE": true,
	"UNIT": true, "UNKNOWN": true, "UNLOGGED": true, "UNNEST": true, "UNPROCESSED": true, "UNSIGNED": true, "UNTIL": true, "UPDATE": true, "UPPER": true, "URL": true,
	"USAGE": true, "USE": true, "USER": true, "USERS": true, "USING": true, "UUID": true, "VACUUM": true, "VALUE": true, "VALUED": true, "VALUES": true,
	"VARCHAR": true, "VARIABLE": true, "VARIANCE": true, "VARINT": true, "VARYING": true, "VIEW": true, "VIEWS": true, "VIRTUAL": true, "VOID": true, "WAIT": true,
	"WHEN": true, "WHENEVER": true, "WHERE": true, "WHILE": true, "WINDOW": true, "WITH": true, "WITHIN": true, "WITHOUT": true, "WORK": true, "WRAPPED": true,
	"WRITE": true, "YEAR": true, "ZONE": true,
}

// isReservedWord checks if a word is a DynamoDB reserved word (case-insensitive)
func isReservedWord(word string) bool {
	return dynamoReservedWords[strings.ToUpper(word)]
}

// buildQueryExpression builds the DynamoDB query expression and attribute values
func buildQueryExpression(indexKeys []KeyInfo, values map[string]string) (string, string, string) {
	var keyConditionParts []string
	expressionAttrValuesMap := make(map[string]map[string]string)
	expressionAttrNamesMap := make(map[string]string)
	placeholderIndex := 0

	for _, key := range indexKeys {
		// Skip keys with empty values (e.g., optional sort key)
		if values[key.Name] == "" {
			continue
		}

		placeholder := fmt.Sprintf(":val%d", placeholderIndex)
		placeholderIndex++

		// Handle reserved words by using expression attribute names
		keyRef := key.Name
		if isReservedWord(key.Name) {
			keyRef = fmt.Sprintf("#key%d", len(expressionAttrNamesMap))
			expressionAttrNamesMap[keyRef] = key.Name
		}

		// Add condition for key
		keyConditionParts = append(keyConditionParts, fmt.Sprintf("%s = %s", keyRef, placeholder))

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

	var expressionAttrNames string
	if len(expressionAttrNamesMap) > 0 {
		var attrNamePairs []string
		for placeholder, keyName := range expressionAttrNamesMap {
			attrNamePairs = append(attrNamePairs, fmt.Sprintf(`"%s": "%s"`, placeholder, keyName))
		}
		expressionAttrNames = fmt.Sprintf(`{%s}`, strings.Join(attrNamePairs, ", "))
	}

	return keyConditionExpr, expressionAttrValues, expressionAttrNames
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
