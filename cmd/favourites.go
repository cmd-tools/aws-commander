package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/cmd-tools/aws-commander/logger"
	"gopkg.in/yaml.v2"
)

const favouritesFileName = "favourites.yaml"
const favouritePrefix = "★ "

// FavouritesStore holds the favourites data keyed by "resource:command" -> list of item names.
type FavouritesStore struct {
	Items map[string][]string `yaml:"favourites"`
	mu    sync.Mutex
	path  string
}

// Favourites is the global singleton for managing favourited items.
var Favourites = &FavouritesStore{}

// configDir returns the aws-commander config directory (~/.config/aws-commander).
func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to get home directory")
		return ""
	}
	return filepath.Join(home, ".config", "aws-commander")
}

// Load reads the favourites file from disk. If it doesn't exist, starts empty.
func (f *FavouritesStore) Load() {
	f.mu.Lock()
	defer f.mu.Unlock()

	dir := configDir()
	if dir == "" {
		f.Items = make(map[string][]string)
		return
	}

	f.path = filepath.Join(dir, favouritesFileName)

	data, err := os.ReadFile(f.path)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Logger.Error().Err(err).Msg("Failed to read favourites file")
		}
		f.Items = make(map[string][]string)
		return
	}

	if err := yaml.Unmarshal(data, f); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to parse favourites file")
		f.Items = make(map[string][]string)
		return
	}

	if f.Items == nil {
		f.Items = make(map[string][]string)
	}
}

// Save writes the favourites to disk.
func (f *FavouritesStore) Save() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.path == "" {
		dir := configDir()
		if dir == "" {
			return
		}
		f.path = filepath.Join(dir, favouritesFileName)
	}

	// Ensure config directory exists
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to create config directory")
		return
	}

	data, err := yaml.Marshal(f)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to marshal favourites")
		return
	}

	if err := os.WriteFile(f.path, data, 0644); err != nil {
		logger.Logger.Error().Err(err).Msg("Failed to write favourites file")
	}
}

// key builds the lookup key for a resource+command pair.
func key(resource string, command string) string {
	return fmt.Sprintf("%s:%s", resource, command)
}

// Toggle adds or removes an item from the favourites for the current resource:command.
// Returns true if the item is now a favourite, false if it was removed.
func (f *FavouritesStore) Toggle(resource string, command string, item string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	k := key(resource, command)
	items := f.Items[k]

	for i, v := range items {
		if v == item {
			// Remove
			f.Items[k] = append(items[:i], items[i+1:]...)
			return false
		}
	}

	// Add
	f.Items[k] = append(items, item)
	return true
}

// IsFavourite checks if an item is a favourite for the given resource:command.
func (f *FavouritesStore) IsFavourite(resource string, command string, item string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	k := key(resource, command)
	for _, v := range f.Items[k] {
		if v == item {
			return true
		}
	}
	return false
}

// ApplyFavourites reorders rows so favourited items appear at the top with a star prefix
// on their first column. rowData (for JSON viewer) is reordered in sync.
// Returns the reordered rows and rowData.
func (f *FavouritesStore) ApplyFavourites(resource string, command string, rows [][]string, rowData []interface{}) ([][]string, []interface{}) {
	f.mu.Lock()
	k := key(resource, command)
	favSet := make(map[string]bool, len(f.Items[k]))
	for _, v := range f.Items[k] {
		favSet[v] = true
	}
	f.mu.Unlock()

	if len(favSet) == 0 {
		return rows, rowData
	}

	type indexedRow struct {
		row     []string
		rawData interface{}
		isFav   bool
		name    string
	}

	indexed := make([]indexedRow, len(rows))
	for i, row := range rows {
		name := ""
		if len(row) > 0 {
			name = row[0]
		}
		var rd interface{}
		if i < len(rowData) {
			rd = rowData[i]
		}
		indexed[i] = indexedRow{row: row, rawData: rd, isFav: favSet[name], name: name}
	}

	// Stable sort: favourites first, preserve original order within each group
	sort.SliceStable(indexed, func(i, j int) bool {
		if indexed[i].isFav != indexed[j].isFav {
			return indexed[i].isFav
		}
		return false
	})

	newRows := make([][]string, len(indexed))
	var newRowData []interface{}
	if len(rowData) > 0 {
		newRowData = make([]interface{}, len(indexed))
	}

	for i, ir := range indexed {
		// Copy the row so we don't mutate the original
		rowCopy := make([]string, len(ir.row))
		copy(rowCopy, ir.row)
		if ir.isFav && len(rowCopy) > 0 {
			rowCopy[0] = favouritePrefix + rowCopy[0]
		}
		newRows[i] = rowCopy
		if newRowData != nil {
			newRowData[i] = ir.rawData
		}
	}

	return newRows, newRowData
}

// StripFavouritePrefix removes the favourite star prefix from an item name.
func StripFavouritePrefix(name string) string {
	return strings.TrimPrefix(name, favouritePrefix)
}
