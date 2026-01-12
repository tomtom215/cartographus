// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package scheduler provides cron-based scheduling for newsletter delivery.
package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CronExpression represents a parsed cron expression.
// Standard 5-field format: minute hour day-of-month month day-of-week
type CronExpression struct {
	Minutes     []int // 0-59
	Hours       []int // 0-23
	DaysOfMonth []int // 1-31
	Months      []int // 1-12
	DaysOfWeek  []int // 0-6 (0 = Sunday)
}

// ParseCron parses a standard 5-field cron expression.
// Format: minute hour day-of-month month day-of-week
//
// Supported syntax:
//   - * (any value)
//   - n (specific value)
//   - n-m (range)
//   - n,m,o (list)
//   - */n (step from start)
//   - n-m/s (step in range)
//
// Examples:
//   - "0 9 * * *" - Daily at 9:00 AM
//   - "0 9 * * 1" - Every Monday at 9:00 AM
//   - "*/15 * * * *" - Every 15 minutes
//   - "0 0 1 * *" - First day of every month at midnight
func ParseCron(expr string) (*CronExpression, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}

	minutes, err := parseField(fields[0], 0, 59)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}

	hours, err := parseField(fields[1], 0, 23)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}

	daysOfMonth, err := parseField(fields[2], 1, 31)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-month field: %w", err)
	}

	months, err := parseField(fields[3], 1, 12)
	if err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}

	daysOfWeek, err := parseField(fields[4], 0, 7)
	if err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	// Normalize day 7 (Sunday) to day 0
	normalizedDOW := make([]int, 0, len(daysOfWeek))
	for _, d := range daysOfWeek {
		if d == 7 {
			d = 0
		}
		normalizedDOW = append(normalizedDOW, d)
	}
	// Remove duplicates
	daysOfWeek = uniqueInts(normalizedDOW)

	return &CronExpression{
		Minutes:     minutes,
		Hours:       hours,
		DaysOfMonth: daysOfMonth,
		Months:      months,
		DaysOfWeek:  daysOfWeek,
	}, nil
}

// NextRun calculates the next run time after the given time.
// If loc is nil, UTC is used.
func (c *CronExpression) NextRun(after time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	t := after.In(loc)

	// Start from the next minute
	t = t.Add(time.Minute)
	t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), 0, 0, loc)

	// Limit search to prevent infinite loops (max 4 years)
	maxIterations := 365 * 24 * 60 * 4 // 4 years in minutes
	for i := 0; i < maxIterations; i++ {
		if c.matches(t) {
			return t
		}
		t = t.Add(time.Minute)
	}

	// Should never reach here with valid expressions
	return time.Time{}
}

// matches checks if the given time matches the cron expression.
func (c *CronExpression) matches(t time.Time) bool {
	// Check minute
	if !containsInt(c.Minutes, t.Minute()) {
		return false
	}

	// Check hour
	if !containsInt(c.Hours, t.Hour()) {
		return false
	}

	// Check month
	if !containsInt(c.Months, int(t.Month())) {
		return false
	}

	// Day-of-month and day-of-week are OR'd together (standard cron behavior)
	// If both are specified (not *), either matching is sufficient
	domMatch := containsInt(c.DaysOfMonth, t.Day())
	dowMatch := containsInt(c.DaysOfWeek, int(t.Weekday()))

	// If both are wildcards (full range), accept
	domWildcard := len(c.DaysOfMonth) == 31
	dowWildcard := len(c.DaysOfWeek) == 7

	if domWildcard && dowWildcard {
		return true
	}

	// If only one is specified, use that
	if domWildcard {
		return dowMatch
	}
	if dowWildcard {
		return domMatch
	}

	// Both are specified - either must match
	return domMatch || dowMatch
}

// parseField parses a single cron field.
func parseField(field string, minVal, maxVal int) ([]int, error) {
	// Handle wildcard
	if field == "*" {
		return rangeInts(minVal, maxVal), nil
	}

	// Handle list (e.g., "1,3,5")
	if strings.Contains(field, ",") {
		var result []int
		for _, part := range strings.Split(field, ",") {
			values, err := parseFieldPart(part, minVal, maxVal)
			if err != nil {
				return nil, err
			}
			result = append(result, values...)
		}
		return uniqueInts(result), nil
	}

	return parseFieldPart(field, minVal, maxVal)
}

// parseFieldPart parses a single part of a cron field (non-list).
//
//nolint:gocyclo // Cron parsing requires handling multiple format cases
func parseFieldPart(part string, minVal, maxVal int) ([]int, error) {
	// Handle step (e.g., "*/5" or "0-30/5")
	if strings.Contains(part, "/") {
		parts := strings.SplitN(part, "/", 2)
		step, err := strconv.Atoi(parts[1])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step value: %s", parts[1])
		}

		var rangeStart, rangeEnd int
		if parts[0] == "*" {
			rangeStart = minVal
			rangeEnd = maxVal
		} else if strings.Contains(parts[0], "-") {
			rangeParts := strings.SplitN(parts[0], "-", 2)
			rangeStart, err = strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid range start: %s", rangeParts[0])
			}
			rangeEnd, err = strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid range end: %s", rangeParts[1])
			}
		} else {
			rangeStart, err = strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid value: %s", parts[0])
			}
			rangeEnd = maxVal
		}

		var result []int
		for i := rangeStart; i <= rangeEnd; i += step {
			if i >= minVal && i <= maxVal {
				result = append(result, i)
			}
		}
		return result, nil
	}

	// Handle range (e.g., "1-5")
	if strings.Contains(part, "-") {
		rangeParts := strings.SplitN(part, "-", 2)
		start, err := strconv.Atoi(rangeParts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid range start: %s", rangeParts[0])
		}
		end, err := strconv.Atoi(rangeParts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid range end: %s", rangeParts[1])
		}
		if start > end || start < minVal || end > maxVal {
			return nil, fmt.Errorf("invalid range: %d-%d (minVal=%d, maxVal=%d)", start, end, minVal, maxVal)
		}
		return rangeInts(start, end), nil
	}

	// Handle single value
	val, err := strconv.Atoi(part)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %s", part)
	}
	if val < minVal || val > maxVal {
		return nil, fmt.Errorf("value out of range: %d (minVal=%d, maxVal=%d)", val, minVal, maxVal)
	}
	return []int{val}, nil
}

// rangeInts returns a slice of integers from start to end (inclusive).
func rangeInts(start, end int) []int {
	result := make([]int, end-start+1)
	for i := range result {
		result[i] = start + i
	}
	return result
}

// containsInt checks if a slice contains a value.
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// uniqueInts removes duplicates and sorts the slice.
func uniqueInts(slice []int) []int {
	seen := make(map[int]bool)
	var result []int
	for _, v := range slice {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	// Sort for consistency
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i] > result[j] {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// CalculateNextRun is a convenience function to parse a cron expression and
// calculate the next run time.
func CalculateNextRun(cronExpr string, after time.Time, timezone string) (time.Time, error) {
	cron, err := ParseCron(cronExpr)
	if err != nil {
		return time.Time{}, err
	}

	var loc *time.Location
	if timezone != "" {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid timezone %q: %w", timezone, err)
		}
	}

	return cron.NextRun(after, loc), nil
}
