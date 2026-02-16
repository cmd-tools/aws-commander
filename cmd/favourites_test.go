package cmd

import (
	"testing"
)

func TestFavourites_Toggle(t *testing.T) {
	store := &FavouritesStore{Items: make(map[string][]string)}

	// Adding returns true
	if added := store.Toggle("s3api", "list-buckets", "my-bucket"); !added {
		t.Error("expected Toggle to return true when adding")
	}
	if !store.IsFavourite("s3api", "list-buckets", "my-bucket") {
		t.Error("expected my-bucket to be a favourite")
	}

	// Removing returns false
	if added := store.Toggle("s3api", "list-buckets", "my-bucket"); added {
		t.Error("expected Toggle to return false when removing")
	}
	if store.IsFavourite("s3api", "list-buckets", "my-bucket") {
		t.Error("expected my-bucket to no longer be a favourite")
	}
}

func TestFavourites_IsFavourite(t *testing.T) {
	store := &FavouritesStore{Items: map[string][]string{
		"dynamodb:list-tables": {"users", "orders"},
	}}

	tests := []struct {
		name     string
		resource string
		command  string
		item     string
		expected bool
	}{
		{name: "existing favourite", resource: "dynamodb", command: "list-tables", item: "users", expected: true},
		{name: "another favourite", resource: "dynamodb", command: "list-tables", item: "orders", expected: true},
		{name: "not a favourite", resource: "dynamodb", command: "list-tables", item: "products", expected: false},
		{name: "different command", resource: "dynamodb", command: "scan", item: "users", expected: false},
		{name: "different resource", resource: "s3api", command: "list-tables", item: "users", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := store.IsFavourite(tt.resource, tt.command, tt.item)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFavourites_ApplyFavourites(t *testing.T) {
	store := &FavouritesStore{Items: map[string][]string{
		"s3api:list-buckets": {"bucket-b"},
	}}

	rows := [][]string{
		{"bucket-a", "us-east-1"},
		{"bucket-b", "eu-west-1"},
		{"bucket-c", "ap-south-1"},
	}
	rowData := []interface{}{"dataA", "dataB", "dataC"}

	newRows, newRowData := store.ApplyFavourites("s3api", "list-buckets", rows, rowData)

	// bucket-b should be first with star prefix
	if len(newRows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(newRows))
	}
	if newRows[0][0] != "★ bucket-b" {
		t.Errorf("expected first row to be '★ bucket-b', got '%s'", newRows[0][0])
	}
	if newRows[0][1] != "eu-west-1" {
		t.Errorf("expected first row second column to be 'eu-west-1', got '%s'", newRows[0][1])
	}
	// rowData should be reordered too
	if newRowData[0] != "dataB" {
		t.Errorf("expected first rowData to be 'dataB', got '%v'", newRowData[0])
	}

	// Non-favourites should not have star prefix
	if newRows[1][0] != "bucket-a" {
		t.Errorf("expected second row to be 'bucket-a', got '%s'", newRows[1][0])
	}
	if newRows[2][0] != "bucket-c" {
		t.Errorf("expected third row to be 'bucket-c', got '%s'", newRows[2][0])
	}
}

func TestFavourites_ApplyFavourites_NoFavourites(t *testing.T) {
	store := &FavouritesStore{Items: make(map[string][]string)}

	rows := [][]string{
		{"bucket-a", "us-east-1"},
		{"bucket-b", "eu-west-1"},
	}

	newRows, _ := store.ApplyFavourites("s3api", "list-buckets", rows, nil)

	// Should return rows unmodified
	if newRows[0][0] != "bucket-a" {
		t.Errorf("expected 'bucket-a', got '%s'", newRows[0][0])
	}
	if newRows[1][0] != "bucket-b" {
		t.Errorf("expected 'bucket-b', got '%s'", newRows[1][0])
	}
}

func TestFavourites_ApplyFavourites_DoesNotMutateOriginal(t *testing.T) {
	store := &FavouritesStore{Items: map[string][]string{
		"s3api:list-buckets": {"bucket-a"},
	}}

	rows := [][]string{
		{"bucket-a", "us-east-1"},
		{"bucket-b", "eu-west-1"},
	}

	store.ApplyFavourites("s3api", "list-buckets", rows, nil)

	// Original rows should not be mutated
	if rows[0][0] != "bucket-a" {
		t.Errorf("original row was mutated: got '%s', want 'bucket-a'", rows[0][0])
	}
}

func TestStripFavouritePrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "with prefix", input: "★ my-bucket", expected: "my-bucket"},
		{name: "without prefix", input: "my-bucket", expected: "my-bucket"},
		{name: "empty string", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripFavouritePrefix(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}
