package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestConfigurationsAreInternallyConsistent(t *testing.T) {
	Resources = map[string]Resource{}
	Init(os.DirFS(".."))

	if len(Resources) == 0 {
		t.Fatal("expected embedded configurations to load")
	}

	for resourceName, resource := range Resources {
		seenCommands := map[string]bool{}

		if resource.Name != resourceName {
			t.Fatalf("resource map key %q does not match resource.Name %q", resourceName, resource.Name)
		}

		if resource.DefaultCommand != "" {
			if !commandExists(resource, resource.DefaultCommand) {
				t.Fatalf("resource %q has missing default command %q", resourceName, resource.DefaultCommand)
			}
		}

		for _, command := range resource.Commands {
			if command.Name == "" {
				t.Fatalf("resource %q contains a command with an empty name", resourceName)
			}
			if seenCommands[command.Name] {
				t.Fatalf("resource %q contains duplicate command %q", resourceName, command.Name)
			}
			seenCommands[command.Name] = true

			if strings.Contains(command.Name, "_") {
				t.Fatalf("resource %q command %q uses underscores instead of CLI hyphens", resourceName, command.Name)
			}

			if command.DependsOn != "" && !commandExists(resource, command.DependsOn) {
				t.Fatalf("resource %q command %q depends on missing command %q", resourceName, command.Name, command.DependsOn)
			}

			for _, action := range command.Actions {
				if action.TargetCommand != "" && !commandExists(resource, action.TargetCommand) {
					t.Fatalf("resource %q command %q action targets missing command %q", resourceName, command.Name, action.TargetCommand)
				}
			}

			if command.Pagination != nil && command.Pagination.Enabled {
				if command.Pagination.NextTokenParam == "" && command.Pagination.NextTokenJsonPath != "" {
					t.Fatalf("resource %q command %q has pagination enabled without nextTokenParam", resourceName, command.Name)
				}
				if command.Pagination.NextTokenParam != "" && !strings.HasPrefix(command.Pagination.NextTokenParam, "--") {
					t.Fatalf("resource %q command %q has invalid pagination flag %q", resourceName, command.Name, command.Pagination.NextTokenParam)
				}
			}
		}
	}
}

func commandExists(resource Resource, commandName string) bool {
	for _, command := range resource.Commands {
		if command.Name == commandName {
			return true
		}
	}
	return false
}
