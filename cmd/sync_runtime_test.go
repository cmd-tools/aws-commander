package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cmdpkg "github.com/cmd-tools/aws-commander/cmd"
	"github.com/cmd-tools/aws-commander/parser"
)

func TestGeneratedConfigurationsRunAndParseAgainstFakeAWS(t *testing.T) {
	logFile := installFakeAWS(t)
	loadConfigurations(t)

	tests := []struct {
		name               string
		resource           string
		command            string
		expectedInvocation string
		expectedToken      string
		minRows            int
	}{
		{
			name:               "access analyzer list analyzers",
			resource:           "accessanalyzer",
			command:            "list-analyzers",
			expectedInvocation: "accessanalyzer list-analyzers",
			expectedToken:      "aa-next",
			minRows:            1,
		},
		{
			name:               "cloudformation generated templates",
			resource:           "cloudformation",
			command:            "list-generated-templates",
			expectedInvocation: "cloudformation list-generated-templates",
			expectedToken:      "cf-next",
			minRows:            1,
		},
		{
			name:               "dynamodb backups",
			resource:           "dynamodb",
			command:            "list-backups",
			expectedInvocation: "dynamodb list-backups",
			expectedToken:      "arn:aws:dynamodb:backup/next",
			minRows:            1,
		},
		{
			name:               "ec2 recycle bin images",
			resource:           "ec2",
			command:            "list-images-in-recycle-bin",
			expectedInvocation: "ec2 list-images-in-recycle-bin",
			expectedToken:      "ec2-next",
			minRows:            1,
		},
		{
			name:               "dynamodb list tables string list",
			resource:           "dynamodb",
			command:            "list-tables",
			expectedInvocation: "dynamodb list-tables",
			expectedToken:      "",
			minRows:            2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := cmdpkg.Resources[tt.resource]
			command := resource.GetCommand(tt.command)

			output := command.Run(resource.Name, "test-profile")
			parsed := parser.ParseCommand(command, output)
			token := cmdpkg.ExtractPaginationToken(output, command)

			if len(parsed.Values) < tt.minRows {
				t.Fatalf("expected at least %d parsed rows, got %d", tt.minRows, len(parsed.Values))
			}
			if len(parsed.Header) == 1 && parsed.Header[0] == "Info" {
				t.Fatalf("expected parsed data, got fallback info message: %v", parsed.Values)
			}
			if token != tt.expectedToken {
				t.Fatalf("expected pagination token %q, got %q", tt.expectedToken, token)
			}

			invocations := readInvocations(t, logFile)
			if !containsInvocation(invocations, tt.expectedInvocation) {
				t.Fatalf("expected invocation %q, got %v", tt.expectedInvocation, invocations)
			}
		})
	}
}

func installFakeAWS(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "aws-invocations.log")
	awsPath := filepath.Join(tempDir, "aws")

	script := `#!/usr/bin/env bash
set -euo pipefail
printf '%s %s
' "$1" "$2" >> "$FAKE_AWS_LOG"
case "$1:$2" in
  accessanalyzer:list-analyzers)
    cat <<'EOF'
{"analyzers":[{"id":"analyzer-1","name":"main"}],"nextToken":"aa-next"}
EOF
    ;;
  cloudformation:list-generated-templates)
    cat <<'EOF'
{"Summaries":[{"TemplateId":"tmpl-1","Status":"ACTIVE"}],"NextToken":"cf-next"}
EOF
    ;;
  dynamodb:list-backups)
    cat <<'EOF'
{"BackupSummaries":[{"TableName":"users","BackupArn":"arn:aws:dynamodb:backup/1"}],"LastEvaluatedBackupArn":"arn:aws:dynamodb:backup/next"}
EOF
    ;;
  ec2:list-images-in-recycle-bin)
    cat <<'EOF'
{"Images":[{"ImageId":"ami-123","Name":"sample-image"}],"NextToken":"ec2-next"}
EOF
    ;;
  dynamodb:list-tables)
    cat <<'EOF'
{"TableNames":["users","orders"],"LastEvaluatedTableName":"table-next"}
EOF
    ;;
  *)
    printf 'unexpected invocation: %s %s
' "$1" "$2"
    exit 1
    ;;
esac
`

	if err := os.WriteFile(awsPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake aws binary: %v", err)
	}

	t.Setenv("FAKE_AWS_LOG", logFile)
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	return logFile
}

func loadConfigurations(t *testing.T) {
	t.Helper()
	cmdpkg.Resources = map[string]cmdpkg.Resource{}
	cmdpkg.Init(os.DirFS(".."))
}

func readInvocations(t *testing.T, logFile string) []string {
	t.Helper()
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read invocation log: %v", err)
	}
	trimmed := strings.TrimSpace(string(content))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

func containsInvocation(invocations []string, expected string) bool {
	for _, invocation := range invocations {
		if invocation == expected {
			return true
		}
	}
	return false
}
