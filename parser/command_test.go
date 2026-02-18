package parser

import (
	"fmt"
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
	  },
	  {
		"Name": "sample-bucket2",
		"CreationDate": "2024-03-26T06:28:38+00:02"
	  },
	  {
		"Name": "sample-bucket3",
		"CreationDate": "2024-03-26T06:28:38+00:03"
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
	  "global01"
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

func Test_ParseCommand_Object(t *testing.T) {
	var commandTest = cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Buckets",
		},
	}

	var jsonResult1 = ParseCommand(commandTest, awsCommandResult)
	fmt.Println(jsonResult1)

	commandTest = cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Owner",
		},
	}
	jsonResult1 = ParseCommand(commandTest, awsCommandResult)
	fmt.Println(jsonResult1)

	commandTest = cmd.Command{
		Parse: cmd.Parse{
			Type:          "object",
			AttributeName: "Items",
		},
	}
	jsonResult1 = ParseCommand(commandTest, awsCommandResult3)
	fmt.Println(jsonResult1)
}

// Test_ParseCommand_Object_HeterogeneousKeys verifies that items with different
// sets of attributes (as is common in DynamoDB's schemaless model) produce a
// header that is the union of all keys and rows that are correctly aligned to
// that header, with empty strings for missing attributes.
func Test_ParseCommand_Object_HeterogeneousKeys(t *testing.T) {
	// Item 1 has "id" and "name"; Item 2 has "id" and "age"; Item 3 has "id", "name", and "email".
	// Expected header (union, first-appearance order): id, name, age, email
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

	// Verify header is the union of all keys
	expectedHeader := []string{"id", "name", "age", "email"}
	if len(result.Header) != len(expectedHeader) {
		t.Fatalf("header length: got %d, want %d\nheader: %v", len(result.Header), len(expectedHeader), result.Header)
	}
	for i, h := range expectedHeader {
		if result.Header[i] != h {
			t.Errorf("header[%d]: got %q, want %q", i, result.Header[i], h)
		}
	}

	// Verify every row has exactly len(header) columns
	if len(result.Values) != 3 {
		t.Fatalf("row count: got %d, want 3", len(result.Values))
	}
	for i, row := range result.Values {
		if len(row) != len(expectedHeader) {
			t.Errorf("row %d column count: got %d, want %d\nrow: %v", i, len(row), len(expectedHeader), row)
		}
	}

	// Verify row alignment: missing attributes should be empty strings
	// Row 0: id={"S":"1"}, name={"S":"Alice"}, age="", email=""
	if result.Values[0][2] != "" {
		t.Errorf("row 0 col 2 (age): got %q, want empty string", result.Values[0][2])
	}
	if result.Values[0][3] != "" {
		t.Errorf("row 0 col 3 (email): got %q, want empty string", result.Values[0][3])
	}

	// Row 1: id={"S":"2"}, name="", age={"N":"30"}, email=""
	if result.Values[1][1] != "" {
		t.Errorf("row 1 col 1 (name): got %q, want empty string", result.Values[1][1])
	}
	if result.Values[1][3] != "" {
		t.Errorf("row 1 col 3 (email): got %q, want empty string", result.Values[1][3])
	}

	// Row 2: id={"S":"3"}, name={"S":"Charlie"}, age="", email={"S":"c@example.com"}
	if result.Values[2][2] != "" {
		t.Errorf("row 2 col 2 (age): got %q, want empty string", result.Values[2][2])
	}
}

func Test_ParseCommand_Array(t *testing.T) {
	var commandTest = cmd.Command{
		Parse: cmd.Parse{
			Type:          "list",
			AttributeName: "TableNames",
		},
	}

	var jsonResult1 = ParseCommand(commandTest, awsCommandResult2)
	fmt.Println(jsonResult1)
}
