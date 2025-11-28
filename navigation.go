package main

import (
	"github.com/cmd-tools/aws-commander/cmd"
	"github.com/rivo/tview"
)

// pushNavigation adds a new navigation state to the stack
func pushNavigation(navType cmd.BreadcrumbType, value string) {
	pushNavigationWithCache(navType, value, "", nil)
}

// pushNavigationWithCache adds a new navigation state with cached result
func pushNavigationWithCache(navType cmd.BreadcrumbType, value string, cachedResult string, cachedBody tview.Primitive) {
	cmd.UiState.NavigationStack = append(cmd.UiState.NavigationStack, cmd.NavigationState{
		Type:         navType,
		Value:        value,
		CachedResult: cachedResult,
		CachedBody:   cachedBody,
	})
	cmd.UiState.Breadcrumbs = append(cmd.UiState.Breadcrumbs, value)
}

// updateNavigationCache updates the cached result and body for the current navigation state
func updateNavigationCache(cachedResult string, cachedBody tview.Primitive) {
	if len(cmd.UiState.NavigationStack) > 0 {
		lastIndex := len(cmd.UiState.NavigationStack) - 1
		cmd.UiState.NavigationStack[lastIndex].CachedResult = cachedResult
		cmd.UiState.NavigationStack[lastIndex].CachedBody = cachedBody
	}
}

// popNavigation removes the last navigation state from the stack
func popNavigation() *cmd.NavigationState {
	if len(cmd.UiState.NavigationStack) == 0 {
		return nil
	}

	lastIndex := len(cmd.UiState.NavigationStack) - 1
	state := cmd.UiState.NavigationStack[lastIndex]
	cmd.UiState.NavigationStack = cmd.UiState.NavigationStack[:lastIndex]

	if len(cmd.UiState.Breadcrumbs) > 0 {
		cmd.UiState.Breadcrumbs = cmd.UiState.Breadcrumbs[:len(cmd.UiState.Breadcrumbs)-1]
	}

	return &state
}

// peekNavigation returns the current navigation state without removing it
func peekNavigation() *cmd.NavigationState {
	if len(cmd.UiState.NavigationStack) == 0 {
		return nil
	}
	return &cmd.UiState.NavigationStack[len(cmd.UiState.NavigationStack)-1]
}
