package parser

import (
	"testing"

	"github.com/cmd-tools/aws-commander/cmd"
)

var awsCommandResult = `{
	"Buckets": [
	  {
		"Name": "sample-bucket",
		"CreationDate": "2024-03-26T06:28:38+00:00"
	  },
	  {
		"Name": "sample-bucket1",
		"CreationDate": "2024-03-26T06:28:38+00:01"
	  }
	],
	"Owner": {
	  "DisplayName": "webfile",
	  "ID": "75aa57f09aa0c8caeab4f8c24e99d10f8e7faeebf76c078efc7c6caea54ba06a"
	}
}
`

var awsCommandResult2 = `{
	"TableNames": [
	  "global01",
	  "global02"
	]
}
`

var awsCommandResult3 = `{
	"Items": [
	  {
		"id": {
		  "S": "foo4"
		}
	  },
	  {
		"id": {
		  "S": "foo1"
		}
	  }
	]
}
`

func TestParseCommandObjectCollection(t *testing.T) {
	command := cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Buckets",
		},
	}

	result := ParseCommand(command, awsCommandResult)

	if len(result.Header) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(result.Header))
	}
	if result.Header[0] != "Name" || result.Header[1] != "CreationDate" {
		t.Fatalf("unexpected headers: %v", result.Header)
	}
	if len(result.Values) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.Values))
	}
	if result.Values[0][0] != "sample-bucket" {
		t.Fatalf("expected first bucket name, got %q", result.Values[0][0])
	}
	if len(result.RawData) != 2 {
		t.Fatalf("expected raw data for 2 rows, got %d", len(result.RawData))
	}
}

func TestParseCommandObjectSingleObject(t *testing.T) {
	command := cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Owner",
		},
	}

	result := ParseCommand(command, awsCommandResult)

	if len(result.Values) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Values))
	}
	if len(result.Header) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(result.Header))
	}
	if result.Values[0][0] != "webfile" {
		t.Fatalf("expected owner display name, got %q", result.Values[0][0])
	}
}

func TestParseCommandListStrings(t *testing.T) {
	command := cmd.Command{
		Parse: cmd.Parse{
			Type:          "list",
			AttributeName: "TableNames",
		},
	}

	result := ParseCommand(command, awsCommandResult2)

	if len(result.Header) != 1 || result.Header[0] != "Item" {
		t.Fatalf("unexpected headers: %v", result.Header)
	}
	if len(result.Values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(result.Values))
	}
	if result.Values[1][0] != "global02" {
		t.Fatalf("expected second table name, got %q", result.Values[1][0])
	}
}

func TestParseCommandObjectDynamoDBItems(t *testing.T) {
	command := cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Items",
		},
	}

	result := ParseCommand(command, awsCommandResult3)

	if len(result.Header) != 1 || result.Header[0] != "id" {
		t.Fatalf("unexpected headers: %v", result.Header)
	}
	if len(result.Values) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(result.Values))
	}
	if result.Values[0][0] == "" {
		t.Fatal("expected first row id value to be present")
	}
}

func TestParseCommandObjectHeterogeneousKeys(t *testing.T) {
	heterogeneousInput := `{
		"Items": [
			{"id": {"S": "1"}, "name": {"S": "Alice"}},
			{"id": {"S": "2"}, "age": {"N": "30"}},
			{"id": {"S": "3"}, "name": {"S": "Charlie"}, "email": {"S": "c@example.com"}}
		]
	}`

	command := cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Items",
		},
	}

	result := ParseCommand(command, heterogeneousInput)

	expectedHeader := []string{"id", "name", "age", "email"}
	if len(result.Header) != len(expectedHeader) {
		t.Fatalf("header length: got %d, want %d; header=%v", len(result.Header), len(expectedHeader), result.Header)
	}
	for index, header := range expectedHeader {
		if result.Header[index] != header {
			t.Errorf("header[%d]: got %q, want %q", index, result.Header[index], header)
		}
	}
	if len(result.Values) != 3 {
		t.Fatalf("row count: got %d, want 3", len(result.Values))
	}
	for index, row := range result.Values {
		if len(row) != len(expectedHeader) {
			t.Errorf("row %d column count: got %d, want %d", index, len(row), len(expectedHeader))
		}
	}
	if result.Values[0][2] != "" || result.Values[0][3] != "" {
		t.Fatal("expected missing fields in first row to stay empty")
	}
	if result.Values[1][1] != "" || result.Values[1][3] != "" {
		t.Fatal("expected missing fields in second row to stay empty")
	}
	if result.Values[2][2] != "" {
		t.Fatal("expected missing age in third row to stay empty")
	}
}

func TestParseCommandMissingAttributeReturnsFallbackMessage(t *testing.T) {
	command := cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "DoesNotExist",
		},
	}

	result := ParseCommand(command, awsCommandResult)

	if len(result.Header) != 1 || result.Header[0] != "Info" {
		t.Fatalf("expected fallback info header, got %v", result.Header)
	}
	if len(result.Values) != 1 || len(result.Values[0]) != 1 {
		t.Fatalf("expected one fallback message row, got %v", result.Values)
	}
	if result.Values[0][0] != "No DoesNotExist found" {
		t.Fatalf("unexpected fallback message: %q", result.Values[0][0])
	}
}
