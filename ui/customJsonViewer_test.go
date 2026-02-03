package ui

import (
	"encoding/json"
	"testing"
)

func TestConvertDynamoDBToRegularJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "Simple string attribute",
			input: `{
				"PK": {"S": "user#123"},
				"SK": {"S": "profile"}
			}`,
			expected: `{
				"PK": "user#123",
				"SK": "profile"
			}`,
		},
		{
			name: "Mixed types",
			input: `{
				"id": {"S": "123"},
				"age": {"N": "25"},
				"active": {"BOOL": true}
			}`,
			expected: `{
				"id": "123",
				"age": "25",
				"active": true
			}`,
		},
		{
			name: "Nested map",
			input: `{
				"user": {"M": {
					"name": {"S": "John"},
					"age": {"N": "30"}
				}}
			}`,
			expected: `{
				"user": {
					"name": "John",
					"age": "30"
				}
			}`,
		},
		{
			name: "Deeply nested map",
			input: `{
				"data": {"M": {
					"level1": {"M": {
						"level2": {"M": {
							"value": {"S": "deep"}
						}}
					}}
				}}
			}`,
			expected: `{
				"data": {
					"level1": {
						"level2": {
							"value": "deep"
						}
					}
				}
			}`,
		},
		{
			name: "Map with mixed nested types",
			input: `{
				"config": {"M": {
					"enabled": {"BOOL": true},
					"count": {"N": "5"},
					"items": {"L": [
						{"S": "item1"},
						{"S": "item2"}
					]},
					"nested": {"M": {
						"key": {"S": "value"}
					}}
				}}
			}`,
			expected: `{
				"config": {
					"enabled": true,
					"count": "5",
					"items": ["item1", "item2"],
					"nested": {
						"key": "value"
					}
				}
			}`,
		},
		{
			name: "List type",
			input: `{
				"tags": {"L": [
					{"S": "tag1"},
					{"S": "tag2"}
				]}
			}`,
			expected: `{
				"tags": ["tag1", "tag2"]
			}`,
		},
		{
			name: "Null value",
			input: `{
				"optional": {"NULL": true}
			}`,
			expected: `{
				"optional": null
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var input interface{}
			if err := json.Unmarshal([]byte(tt.input), &input); err != nil {
				t.Fatalf("Failed to unmarshal input: %v", err)
			}

			var expected interface{}
			if err := json.Unmarshal([]byte(tt.expected), &expected); err != nil {
				t.Fatalf("Failed to unmarshal expected: %v", err)
			}

			result := convertDynamoDBToRegularJSON(input)

			// Marshal both to JSON for comparison
			resultJSON, _ := json.Marshal(result)
			expectedJSON, _ := json.Marshal(expected)

			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("Result doesn't match.\nGot: %s\nExpected: %s", string(resultJSON), string(expectedJSON))
			}
		})
	}
}
