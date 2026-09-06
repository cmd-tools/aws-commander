package cmd_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
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

	region := os.Getenv("AWS_COMMANDER_SMOKE_REGION")
	if region == "" {
		region = "eu-west-1"
	}

	profile := resolveSmokeProfile(t, region)

	t.Setenv("AWS_REGION", region)
	t.Setenv("AWS_DEFAULT_REGION", region)

	loadConfigurations(t)
	cmdpkg.UiState.SelectedItems = map[string]string{}

	tests := []struct {
		name           string
		resource       string
		command        string
		allowEmptyList bool
		setup          func(*testing.T, string)
	}{
		{name: "ec2 describe instances", resource: "ec2", command: "describe-instances", allowEmptyList: true},
		{name: "s3api list buckets", resource: "s3api", command: "list-buckets", allowEmptyList: true},
		{name: "lambda list functions", resource: "lambda", command: "list-functions", allowEmptyList: true},
		{name: "rds describe db instances", resource: "rds", command: "describe-db-instances", allowEmptyList: true},
		{name: "dynamodb list tables", resource: "dynamodb", command: "list-tables", allowEmptyList: true},
		{name: "iam list roles", resource: "iam", command: "list-roles", allowEmptyList: true},
		{name: "cloudwatch list metrics", resource: "cloudwatch", command: "list-metrics", allowEmptyList: true},
		{name: "logs describe log groups", resource: "logs", command: "describe-log-groups", allowEmptyList: true},
		{name: "sqs list queues", resource: "sqs", command: "list-queues", allowEmptyList: true},
		{name: "sqs get queue attributes", resource: "sqs", command: "get-queue-attributes", allowEmptyList: false, setup: setAnyQueueURL},
		{name: "sns list topics", resource: "sns", command: "list-topics", allowEmptyList: true},
		{name: "ecs list clusters", resource: "ecs", command: "list-clusters", allowEmptyList: true},
		{name: "eks list clusters", resource: "eks", command: "list-clusters", allowEmptyList: true},
		{name: "ecr describe repositories", resource: "ecr", command: "describe-repositories", allowEmptyList: true},
		{name: "cloudformation list stacks", resource: "cloudformation", command: "list-stacks", allowEmptyList: true},
		{name: "ssm list documents", resource: "ssm", command: "list-documents", allowEmptyList: true},
		{name: "sts get caller identity", resource: "sts", command: "get-caller-identity", allowEmptyList: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t, profile)
			}

			resource, exists := cmdpkg.Resources[tt.resource]
			if !exists {
				t.Fatalf("resource %q not found in loaded configurations", tt.resource)
			}
			if !slices.Contains(resource.GetCommandNames(), tt.command) {
				t.Fatalf("command %q not found in resource %q", tt.command, tt.resource)
			}

			command := resource.GetCommand(tt.command)

			output := command.Run(resource.Name, profile)
			if output == "" {
				t.Fatal("expected command output")
			}

			assertOutputIsJSON(t, output)
			assertNoAWSCliErrorText(t, output)

			parsed := parser.ParseCommand(command, output)
			assertParseResultIsUsable(t, command, output, parsed, tt.allowEmptyList)
		})
	}
}

func resolveSmokeProfile(t *testing.T, region string) string {
	t.Helper()

	if profile := os.Getenv("AWS_COMMANDER_SMOKE_PROFILE"); profile != "" {
		return profile
	}
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		return profile
	}

	profilesRaw, err := exec.Command("aws", "configure", "list-profiles").Output()
	if err != nil {
		t.Fatal("could not resolve a working AWS profile; set AWS_COMMANDER_SMOKE_PROFILE")
	}

	profiles := strings.Fields(string(profilesRaw))
	for _, profile := range profiles {
		cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile, "--region", region, "--output", "json")
		if err := cmd.Run(); err == nil {
			return profile
		}
	}

	if envProfile := createEnvBackedProfile(t); envProfile != "" {
		return envProfile
	}

	t.Fatal("no working AWS profile found; set AWS_COMMANDER_SMOKE_PROFILE to a valid profile")
	return ""
}

func createEnvBackedProfile(t *testing.T) string {
	t.Helper()

	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sessionToken := os.Getenv("AWS_SESSION_TOKEN")
	if accessKey == "" || secretKey == "" {
		return ""
	}

	dir := t.TempDir()
	credentialsPath := dir + "/credentials"

	content := fmt.Sprintf("[smoke-env]\naws_access_key_id = %s\naws_secret_access_key = %s\n", accessKey, secretKey)
	if sessionToken != "" {
		content += fmt.Sprintf("aws_session_token = %s\n", sessionToken)
	}

	if err := os.WriteFile(credentialsPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to create temporary credentials file: %v", err)
	}

	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialsPath)
	return "smoke-env"
}

func setAnyQueueURL(t *testing.T, profile string) {
	t.Helper()

	resource := cmdpkg.Resources["sqs"]
	listQueues := resource.GetCommand("list-queues")
	output := listQueues.Run(resource.Name, profile)

	var result struct {
		QueueURLs []string `json:"QueueUrls"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse sqs list-queues output: %v", err)
	}
	if len(result.QueueURLs) == 0 {
		t.Fatal("cannot run get-queue-attributes smoke test: no SQS queues available")
	}

	cmdpkg.UiState.SelectedItems["$QUEUENAME"] = result.QueueURLs[0]
}

func assertOutputIsJSON(t *testing.T, output string) {
	t.Helper()
	var obj interface{}
	if err := json.Unmarshal([]byte(output), &obj); err != nil {
		t.Fatalf("expected JSON output, got parse error: %v; output: %s", err, output)
	}
}

func assertNoAWSCliErrorText(t *testing.T, output string) {
	t.Helper()
	lower := strings.ToLower(output)
	if strings.Contains(lower, "an error occurred (") || strings.Contains(lower, "unable to locate credentials") {
		t.Fatalf("aws cli returned an error response: %s", output)
	}
}

func assertParseResultIsUsable(t *testing.T, command cmdpkg.Command, output string, parsed parser.ParseCommandResult, allowEmptyList bool) {
	t.Helper()

	if len(parsed.Header) == 1 && parsed.Header[0] == "Error" {
		t.Fatalf("parse failed with Error header; output: %s", output)
	}

	if len(parsed.Header) == 1 && parsed.Header[0] == "Info" {
		if len(parsed.Values) == 1 && len(parsed.Values[0]) == 1 {
			message := parsed.Values[0][0]
			if strings.HasPrefix(message, "No ") {
				t.Fatalf("parse attribute missing (%q). command parse config likely wrong; output: %s", message, output)
			}
			if allowEmptyList && strings.Contains(message, "Empty - no items available") {
				return
			}
		}
		t.Fatalf("unexpected info-only parse result for %s: %+v", command.Name, parsed.Values)
	}

	if len(parsed.Values) == 0 {
		t.Fatalf("expected parsed rows for %s; output: %s", command.Name, output)
	}
}
