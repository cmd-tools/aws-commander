package cmd_test

import (
	"os"
	"os/exec"
	"testing"

	cmdpkg "github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/parser"
)

func TestConfigurationsRunAgainstRealAWSCLI(t *testing.T) {
	if os.Getenv("AWS_COMMANDER_SMOKE") == "" {
		t.Skip("set AWS_COMMANDER_SMOKE=1 to run real AWS CLI smoke tests")
	}

	if _, err := exec.LookPath("aws"); err != nil {
		t.Fatalf("aws CLI not found: %v", err)
	}

	profile := os.Getenv("AWS_COMMANDER_SMOKE_PROFILE")
	if profile == "" {
		profile = "default"
	}

	region := os.Getenv("AWS_COMMANDER_SMOKE_REGION")
	if region == "" {
		region = "us-east-1"
	}

	t.Setenv("AWS_REGION", region)
	t.Setenv("AWS_DEFAULT_REGION", region)

	loadConfigurations(t)

	tests := []struct {
		name      string
		resource  string
		command   string
		minRows   int
		mustParse bool
	}{
		{
			name:      "sts caller identity",
			resource:  "sts",
			command:   "get-caller-identity",
			minRows:   1,
			mustParse: true,
		},
		{
			name:      "ec2 availability zones",
			resource:  "ec2",
			command:   "describe-availability-zones",
			minRows:   1,
			mustParse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := cmdpkg.Resources[tt.resource]
			command := resource.GetCommand(tt.command)

			output := command.Run(resource.Name, profile)
			if output == "" {
				t.Fatal("expected command output")
			}

			parsed := parser.ParseCommand(command, output)
			if tt.mustParse && len(parsed.Values) < tt.minRows {
				t.Fatalf("expected at least %d parsed rows, got %d; raw output: %s", tt.minRows, len(parsed.Values), output)
			}
			if len(parsed.Header) == 1 && (parsed.Header[0] == "Info" || parsed.Header[0] == "Error") {
				t.Fatalf("expected parsed table data, got %v; raw output: %s", parsed.Values, output)
			}
		})
	}
}
