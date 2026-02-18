package main

import (
	"strings"
	"testing"
)

func TestBuildQueryExpression(t *testing.T) {
	tests := []struct {
		name                   string
		indexKeys              []KeyInfo
		values                 map[string]string
		expectedKeyCondition   string
		expectedAttrValues     string
		expectAttrNames        bool
		expectedAttrNameSubstr string
	}{
		{
			name: "PK only no SK",
			indexKeys: []KeyInfo{
				{Name: "userId", Type: "PK", AttrType: "S"},
			},
			values: map[string]string{
				"userId": "user123",
			},
			expectedKeyCondition: "userId = :val0",
			expectedAttrValues:   `":val0": {"S": "user123"}`,
			expectAttrNames:      false,
		},
		{
			name: "PK and SK with Equals operator",
			indexKeys: []KeyInfo{
				{Name: "userId", Type: "PK", AttrType: "S"},
				{Name: "createdAt", Type: "SK", AttrType: "S"},
			},
			values: map[string]string{
				"userId":           "user123",
				"createdAt":        "2024-01-01",
				sortKeyOperatorKey: string(SortKeyEquals),
			},
			expectedKeyCondition: "userId = :val0 AND createdAt = :val1",
			expectedAttrValues:   `":val1": {"S": "2024-01-01"}`,
			expectAttrNames:      false,
		},
		{
			name: "PK and SK with Begins with operator",
			indexKeys: []KeyInfo{
				{Name: "userId", Type: "PK", AttrType: "S"},
				{Name: "createdAt", Type: "SK", AttrType: "S"},
			},
			values: map[string]string{
				"userId":           "user123",
				"createdAt":        "2024",
				sortKeyOperatorKey: string(SortKeyBeginsWith),
			},
			expectedKeyCondition: "userId = :val0 AND begins_with(createdAt, :val1)",
			expectedAttrValues:   `":val1": {"S": "2024"}`,
			expectAttrNames:      false,
		},
		{
			name: "reserved word SK with Begins with",
			indexKeys: []KeyInfo{
				{Name: "userId", Type: "PK", AttrType: "S"},
				{Name: "status", Type: "SK", AttrType: "S"},
			},
			values: map[string]string{
				"userId":           "user123",
				"status":           "ACT",
				sortKeyOperatorKey: string(SortKeyBeginsWith),
			},
			expectedKeyCondition:   "userId = :val0 AND begins_with(#key0, :val1)",
			expectedAttrValues:     `":val1": {"S": "ACT"}`,
			expectAttrNames:        true,
			expectedAttrNameSubstr: `"#key0": "status"`,
		},
		{
			name: "reserved word SK with Equals",
			indexKeys: []KeyInfo{
				{Name: "userId", Type: "PK", AttrType: "S"},
				{Name: "name", Type: "SK", AttrType: "S"},
			},
			values: map[string]string{
				"userId":           "user123",
				"name":             "John",
				sortKeyOperatorKey: string(SortKeyEquals),
			},
			expectedKeyCondition:   "userId = :val0 AND #key0 = :val1",
			expectedAttrValues:     `":val1": {"S": "John"}`,
			expectAttrNames:        true,
			expectedAttrNameSubstr: `"#key0": "name"`,
		},
		{
			name: "PK and SK with Number type and Equals",
			indexKeys: []KeyInfo{
				{Name: "userId", Type: "PK", AttrType: "S"},
				{Name: "score", Type: "SK", AttrType: "N"},
			},
			values: map[string]string{
				"userId":           "user123",
				"score":            "100",
				sortKeyOperatorKey: string(SortKeyEquals),
			},
			expectedKeyCondition: "userId = :val0 AND score = :val1",
			expectedAttrValues:   `":val1": {"N": "100"}`,
			expectAttrNames:      false,
		},
		{
			name: "SK empty value is skipped",
			indexKeys: []KeyInfo{
				{Name: "userId", Type: "PK", AttrType: "S"},
				{Name: "createdAt", Type: "SK", AttrType: "S"},
			},
			values: map[string]string{
				"userId":           "user123",
				"createdAt":        "",
				sortKeyOperatorKey: string(SortKeyBeginsWith),
			},
			expectedKeyCondition: "userId = :val0",
			expectedAttrValues:   `":val0": {"S": "user123"}`,
			expectAttrNames:      false,
		},
		{
			name: "no operator defaults to Equals",
			indexKeys: []KeyInfo{
				{Name: "userId", Type: "PK", AttrType: "S"},
				{Name: "createdAt", Type: "SK", AttrType: "S"},
			},
			values: map[string]string{
				"userId":    "user123",
				"createdAt": "2024-01-01",
			},
			expectedKeyCondition: "userId = :val0 AND createdAt = :val1",
			expectedAttrValues:   `":val1": {"S": "2024-01-01"}`,
			expectAttrNames:      false,
		},
		{
			name: "reserved word PK always uses Equals regardless of operator",
			indexKeys: []KeyInfo{
				{Name: "data", Type: "PK", AttrType: "S"},
				{Name: "sortField", Type: "SK", AttrType: "S"},
			},
			values: map[string]string{
				"data":             "mydata",
				"sortField":        "2024",
				sortKeyOperatorKey: string(SortKeyBeginsWith),
			},
			expectedKeyCondition:   "#key0 = :val0 AND begins_with(sortField, :val1)",
			expectedAttrValues:     `":val0": {"S": "mydata"}`,
			expectAttrNames:        true,
			expectedAttrNameSubstr: `"#key0": "data"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyCondition, attrValues, attrNames := buildQueryExpression(tt.indexKeys, tt.values)

			if keyCondition != tt.expectedKeyCondition {
				t.Errorf("keyCondition = %q, want %q", keyCondition, tt.expectedKeyCondition)
			}

			if !strings.Contains(attrValues, tt.expectedAttrValues) {
				t.Errorf("attrValues = %q, want to contain %q", attrValues, tt.expectedAttrValues)
			}

			if tt.expectAttrNames && attrNames == "" {
				t.Error("expected expressionAttrNames to be non-empty")
			}
			if !tt.expectAttrNames && attrNames != "" {
				t.Errorf("expected expressionAttrNames to be empty, got %q", attrNames)
			}
			if tt.expectedAttrNameSubstr != "" && !strings.Contains(attrNames, tt.expectedAttrNameSubstr) {
				t.Errorf("attrNames = %q, want to contain %q", attrNames, tt.expectedAttrNameSubstr)
			}
		})
	}
}

func TestSortKeyOperatorOptions(t *testing.T) {
	options := sortKeyOperatorOptions()
	if len(options) != 2 {
		t.Fatalf("expected 2 options, got %d", len(options))
	}
	if options[0] != string(SortKeyEquals) {
		t.Errorf("first option = %q, want %q", options[0], string(SortKeyEquals))
	}
	if options[1] != string(SortKeyBeginsWith) {
		t.Errorf("second option = %q, want %q", options[1], string(SortKeyBeginsWith))
	}
}
