package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cmd-tools/aws-commander/executor"
	"github.com/cmd-tools/aws-commander/logger"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v2"
)

var ConfigurationsRelativeFilePath = "./configurations"
var ConfigurationsRelativeFileExtension = ".yaml"

const VariablePlaceHolderPrefix = "$"

var Resources = map[string]Resource{}

type Command struct {
	Name             string      `yaml:"name"`
	ResourceName     string      `yaml:"resourceName"`
	DefaultCommand   string      `yaml:"defaultCommand"`
	DependsOn        string      `yaml:"depends_on"`
	Arguments        []string    `yaml:"arguments"`
	View             string      `yaml:"view"`
	Parse            Parse       `yaml:"parse"`
	ShowJsonViewer   bool        `yaml:"showJsonViewer"`
	RerunOnBack      bool        `yaml:"rerunOnBack"`          // If true, rerun command when navigating back; if false, use cached result
	RequiresKeyInput bool        `yaml:"requiresKeyInput"`     // If true, prompt user for key value before executing
	Pagination       *Pagination `yaml:"pagination,omitempty"` // Pagination configuration
}

type Pagination struct {
	Enabled           bool   `yaml:"enabled"`
	NextTokenParam    string `yaml:"nextTokenParam"`    // Parameter name for next token (e.g., "--starting-token" or "--exclusive-start-key")
	NextTokenJsonPath string `yaml:"nextTokenJsonPath"` // JSON path to extract next token (e.g., "NextToken" or "LastEvaluatedKey")
}

type Parse struct {
	Type          string `yaml:"type"`
	AttributeName string `yaml:"attributeName"`
}

type Resource struct {
	Name           string    `yaml:"name"`
	DefaultCommand string    `yaml:"defaultCommand"`
	Commands       []Command `yaml:"commands"`
}

func Init() {

	entries, err := os.ReadDir(ConfigurationsRelativeFilePath)
	if err != nil {
		log.Fatal(err)
	}

	channel := make(chan Resource)

	for _, e := range entries {
		if filepath.Ext(e.Name()) == ConfigurationsRelativeFileExtension {
			go processConfigurationFile(channel, fmt.Sprintf("%s/%s", ConfigurationsRelativeFilePath, e.Name()))
			resource := <-channel
			Resources[resource.Name] = resource
		}
	}

	logger.Logger.Debug().Msg(fmt.Sprintf("Loaded %d configurations", len(Resources)))
}

func GetAvailableResourceNames() []string {
	if len(Resources) == 0 {
		logger.Logger.Warn().Msg("No resources found, try to load from configuration again")
		Init()
	}

	if len(Resources) == 0 {
		logger.Logger.Error().Msg("No resources found after trying a second loading")
		return []string{}
	}
	return maps.Keys(Resources)
}

func (resource *Resource) GetCommandNames() []string {
	commandNames := []string{}
	for _, command := range resource.Commands {
		commandNames = append(commandNames, command.Name)
	}
	return commandNames
}

func (resource *Resource) GetCommand(name string) Command {
	for _, command := range resource.Commands {
		if command.Name == name {
			return command
		}
	}
	panic(fmt.Sprintf("Requested command %s does not exists in %s resouce", name, resource.Name))
}

func (command *Command) Run(resource string, profile string) string {
	return command.RunWithPaginationToken(resource, profile, "")
}

func (command *Command) RunWithPaginationToken(resource string, profile string, paginationToken string) string {
	binaryName := "aws"
	var argumentsCopy = make([]string, len(command.Arguments))
	copy(argumentsCopy, command.Arguments)
	args := []string{resource, command.Name, "--profile", profile}
	args = append(args, replaceVariablesOnCommandArguments(command.Arguments)...)

	// Add pagination token if provided and pagination is enabled
	// Only add token parameter if both token and parameter name are non-empty
	if paginationToken != "" && command.Pagination != nil && command.Pagination.Enabled && command.Pagination.NextTokenParam != "" {
		args = append(args, command.Pagination.NextTokenParam, paginationToken)
	}

	logger.Logger.Debug().Msg(fmt.Sprintf("Running: %s %s", binaryName, strings.Join(args, " ")))
	start := time.Now()
	output := executor.ExecCommand(binaryName, args)
	// set again original args which contains placeholders
	copy(command.Arguments, argumentsCopy)
	logger.Logger.Debug().Msg(fmt.Sprintf("Execution time %s", time.Since(start)))

	return output
}

func processConfigurationFile(channel chan Resource, filename string) {
	logger.Logger.Debug().Msg(fmt.Sprintf("[Worker] Loading configurations from: %s", filename))
	resource := Resource{}
	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		logger.Logger.Error().Msg(fmt.Sprintf("[Worker] Error while reading: %s, description: #%v", filename, err))
	}

	err = yaml.Unmarshal(yamlFile, &resource)
	if err != nil {
		logger.Logger.Error().Msg(fmt.Sprintf("[Worker] Error while unmarshalling: %s, description: #%v", filename, err))
	}

	logger.Logger.Debug().Msg(fmt.Sprintf("[Worker] Loaded resource: %s, which contains %d commands", resource.Name, len(resource.Commands)))

	channel <- resource
}

func replaceVariablesOnCommandArguments(arguments []string) []string {
	for index, item := range arguments {
		if strings.HasPrefix(item, VariablePlaceHolderPrefix) {
			value, exists := UiState.SelectedItems[item]
			if exists {
				arguments[index] = value
			}
		}
	}
	return arguments
}

// ExtractPaginationToken extracts the next page token from JSON output
func ExtractPaginationToken(jsonOutput string, command Command) string {
	if command.Pagination == nil || !command.Pagination.Enabled {
		return ""
	}

	// If NextTokenJsonPath is empty, this is a token-less pagination (e.g., SQS receive-message)
	// Return empty string to indicate pagination is enabled but no token is available
	if command.Pagination.NextTokenJsonPath == "" {
		return ""
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutput), &result); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to parse JSON for pagination token extraction")
		return ""
	}

	// Get the token from the JSON path
	if token, exists := result[command.Pagination.NextTokenJsonPath]; exists && token != nil {
		// Handle different token types:
		// - String tokens (e.g., SQS NextToken) should be returned as-is
		// - Object tokens (e.g., DynamoDB LastEvaluatedKey) need JSON marshaling
		switch v := token.(type) {
		case string:
			// For string tokens, return directly without JSON encoding
			return v
		default:
			// For complex objects (maps, arrays), JSON marshal them
			tokenBytes, err := json.Marshal(token)
			if err != nil {
				logger.Logger.Error().Err(err).Msg("Failed to marshal pagination token")
				return ""
			}
			return string(tokenBytes)
		}
	}

	return ""
}
