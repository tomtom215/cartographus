// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"strings"
)

// parseList is a helper function to parse DuckDB LIST output
func parseList(listStr string) []string {
	if listStr == "" || listStr == "[]" {
		return []string{}
	}
	// Remove brackets and split
	listStr = strings.Trim(listStr, "[]")
	parts := strings.Split(listStr, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" && trimmed != "NULL" {
			result = append(result, trimmed)
		}
	}
	return result
}
