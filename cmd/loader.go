package cmd

import (
	"fmt"
	"github.com/cmd-tools/aws-commander/constants"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
	Name           string      `yaml:"name"`
	ResourceName   string      `yaml:"resourceName"`
	DefaultCommand string      `yaml:"defaultCommand"`
	Arguments      []string    `yaml:"arguments"`
	View           string      `yaml:"view"`
	Parse          []Parse     `yaml:"parse"`
	Overrides      []Overrides `yaml:"overrides"`
}

type Overrides struct {
	When []When `yaml:"when"`
}

type When struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
	Then  string `yaml:"then"`
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

func (command *Command) GetCommandForSelectedItem(selectedItem string) string {
	for _, overrides := range command.Overrides {
		for _, when := range overrides.When {
			switch when.Type {
			case "regex":
				re := regexp.MustCompile(when.Value)
				if re.MatchString(selectedItem) {
					return when.Then
				}
				break
			default:
				panic(fmt.Sprintf("Requested type %s does not exists", when.Type))
			}
		}
	}

	return command.DefaultCommand
}

func (resource *Resource) GetCommand(name string) Command {
	for _, command := range resource.Commands {
		if command.Name == name {
			return command
		}
	}
	panic(fmt.Sprintf("Requested command %s does not exists in %s resource", name, resource.Name))
}

func (command *Command) Run(resource string, profile string) string {
	binaryName := "aws"
	var argumentsCopy = make([]string, len(command.Arguments))
	copy(argumentsCopy, command.Arguments)
	args := []string{resource, command.Name, "--profile", profile, "--no-cli-pager"}
	args = append(args, replaceVariablesOnCommandArguments(command.Arguments)...)
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
			} else {
				arguments[index] = constants.EmptyString
			}
		}
	}
	return arguments
}
