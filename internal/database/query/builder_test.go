// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package query

import (
	"testing"
	"time"
)

func TestWhereBuilder_Empty(t *testing.T) {
	wb := NewWhereBuilder()

	if !wb.IsEmpty() {
		t.Error("Expected new builder to be empty")
	}

	if wb.Count() != 0 {
		t.Errorf("Expected count 0, got %d", wb.Count())
	}

	whereClause, args := wb.Build()
	if whereClause != "1=1" {
		t.Errorf("Expected '1=1' for empty builder, got %q", whereClause)
	}
	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestWhereBuilder_AddDateRange(t *testing.T) {
	wb := NewWhereBuilder()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

	wb.AddDateRange(&start, &end)

	whereClause, args := wb.Build()
	expected := "started_at >= ? AND started_at <= ?"
	if whereClause != expected {
		t.Errorf("Expected %q, got %q", expected, whereClause)
	}
	if len(args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(args))
	}
}

func TestWhereBuilder_AddUsers(t *testing.T) {
	wb := NewWhereBuilder()
	users := []string{"user1", "user2", "user3"}

	wb.AddUsers(users)

	whereClause, args := wb.Build()
	expected := "username IN (?, ?, ?)"
	if whereClause != expected {
		t.Errorf("Expected %q, got %q", expected, whereClause)
	}
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
	for i, user := range users {
		if args[i] != user {
			t.Errorf("Expected arg[%d] = %q, got %q", i, user, args[i])
		}
	}
}

func TestWhereBuilder_Combined(t *testing.T) {
	wb := NewWhereBuilder()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	users := []string{"alice", "bob"}
	mediaTypes := []string{"movie", "episode"}

	wb.AddDateRange(&start, nil)
	wb.AddUsers(users)
	wb.AddMediaTypes(mediaTypes)

	whereClause, args := wb.Build()
	expected := "started_at >= ? AND username IN (?, ?) AND media_type IN (?, ?)"
	if whereClause != expected {
		t.Errorf("Expected %q, got %q", expected, whereClause)
	}
	if len(args) != 5 {
		t.Errorf("Expected 5 args, got %d", len(args))
	}
}

func TestWhereBuilder_BuildWithPrefix(t *testing.T) {
	wb := NewWhereBuilder()
	wb.AddClause("id = ?", 123)

	whereClause, args := wb.BuildWithPrefix()
	expected := "WHERE id = ?"
	if whereClause != expected {
		t.Errorf("Expected %q, got %q", expected, whereClause)
	}
	if len(args) != 1 || args[0] != 123 {
		t.Errorf("Expected args [123], got %v", args)
	}
}

func TestWhereBuilder_SkipEmpty(t *testing.T) {
	wb := NewWhereBuilder()
	wb.AddUsers([]string{})      // Should be skipped
	wb.AddMediaTypes([]string{}) // Should be skipped
	wb.AddClause("active = ?", true)

	whereClause, args := wb.Build()
	expected := "active = ?"
	if whereClause != expected {
		t.Errorf("Expected %q, got %q", expected, whereClause)
	}
	if len(args) != 1 {
		t.Errorf("Expected 1 arg, got %d", len(args))
	}
}

// TestWhereBuilder_AddPlatforms tests the AddPlatforms method
func TestWhereBuilder_AddPlatforms(t *testing.T) {

	tests := []struct {
		name           string
		platforms      []string
		expectedClause string
		expectedArgs   int
	}{
		{
			name:           "empty platforms skipped",
			platforms:      []string{},
			expectedClause: "1=1",
			expectedArgs:   0,
		},
		{
			name:           "single platform",
			platforms:      []string{"Roku"},
			expectedClause: "platform IN (?)",
			expectedArgs:   1,
		},
		{
			name:           "multiple platforms",
			platforms:      []string{"Roku", "Apple TV", "Android"},
			expectedClause: "platform IN (?, ?, ?)",
			expectedArgs:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wb := NewWhereBuilder()
			wb.AddPlatforms(tt.platforms)

			whereClause, args := wb.Build()
			if whereClause != tt.expectedClause {
				t.Errorf("Expected %q, got %q", tt.expectedClause, whereClause)
			}
			if len(args) != tt.expectedArgs {
				t.Errorf("Expected %d args, got %d", tt.expectedArgs, len(args))
			}
		})
	}
}

// TestWhereBuilder_AddLibraries tests the AddLibraries method
func TestWhereBuilder_AddLibraries(t *testing.T) {

	tests := []struct {
		name           string
		libraries      []string
		expectedClause string
		expectedArgs   int
	}{
		{
			name:           "empty libraries skipped",
			libraries:      []string{},
			expectedClause: "1=1",
			expectedArgs:   0,
		},
		{
			name:           "single library",
			libraries:      []string{"Movies"},
			expectedClause: "library_name IN (?)",
			expectedArgs:   1,
		},
		{
			name:           "multiple libraries",
			libraries:      []string{"Movies", "TV Shows", "Music"},
			expectedClause: "library_name IN (?, ?, ?)",
			expectedArgs:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wb := NewWhereBuilder()
			wb.AddLibraries(tt.libraries)

			whereClause, args := wb.Build()
			if whereClause != tt.expectedClause {
				t.Errorf("Expected %q, got %q", tt.expectedClause, whereClause)
			}
			if len(args) != tt.expectedArgs {
				t.Errorf("Expected %d args, got %d", tt.expectedArgs, len(args))
			}
		})
	}
}

// TestWhereBuilder_AddMediaTypes tests the AddMediaTypes method with various scenarios
func TestWhereBuilder_AddMediaTypes(t *testing.T) {

	tests := []struct {
		name           string
		mediaTypes     []string
		expectedClause string
		expectedArgs   int
	}{
		{
			name:           "empty media types skipped",
			mediaTypes:     []string{},
			expectedClause: "1=1",
			expectedArgs:   0,
		},
		{
			name:           "single media type",
			mediaTypes:     []string{"movie"},
			expectedClause: "media_type IN (?)",
			expectedArgs:   1,
		},
		{
			name:           "all media types",
			mediaTypes:     []string{"movie", "episode", "track", "clip"},
			expectedClause: "media_type IN (?, ?, ?, ?)",
			expectedArgs:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wb := NewWhereBuilder()
			wb.AddMediaTypes(tt.mediaTypes)

			whereClause, args := wb.Build()
			if whereClause != tt.expectedClause {
				t.Errorf("Expected %q, got %q", tt.expectedClause, whereClause)
			}
			if len(args) != tt.expectedArgs {
				t.Errorf("Expected %d args, got %d", tt.expectedArgs, len(args))
			}
		})
	}
}

// TestWhereBuilder_AddDateRange_EdgeCases tests date range edge cases
func TestWhereBuilder_AddDateRange_EdgeCases(t *testing.T) {

	tests := []struct {
		name           string
		startDate      *time.Time
		endDate        *time.Time
		expectedClause string
		expectedArgs   int
	}{
		{
			name:           "both nil dates",
			startDate:      nil,
			endDate:        nil,
			expectedClause: "1=1",
			expectedArgs:   0,
		},
		{
			name:           "only start date",
			startDate:      timePtr(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			endDate:        nil,
			expectedClause: "started_at >= ?",
			expectedArgs:   1,
		},
		{
			name:           "only end date",
			startDate:      nil,
			endDate:        timePtr(time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)),
			expectedClause: "started_at <= ?",
			expectedArgs:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wb := NewWhereBuilder()
			wb.AddDateRange(tt.startDate, tt.endDate)

			whereClause, args := wb.Build()
			if whereClause != tt.expectedClause {
				t.Errorf("Expected %q, got %q", tt.expectedClause, whereClause)
			}
			if len(args) != tt.expectedArgs {
				t.Errorf("Expected %d args, got %d", tt.expectedArgs, len(args))
			}
		})
	}
}

// TestWhereBuilder_AddClause_MultipleArgs tests AddClause with multiple arguments
func TestWhereBuilder_AddClause_MultipleArgs(t *testing.T) {

	wb := NewWhereBuilder()
	wb.AddClause("status IN (?, ?, ?)", "active", "pending", "completed")

	whereClause, args := wb.Build()
	expected := "status IN (?, ?, ?)"
	if whereClause != expected {
		t.Errorf("Expected %q, got %q", expected, whereClause)
	}
	if len(args) != 3 {
		t.Errorf("Expected 3 args, got %d", len(args))
	}
	if args[0] != "active" || args[1] != "pending" || args[2] != "completed" {
		t.Errorf("Unexpected args: %v", args)
	}
}

// TestWhereBuilder_ChainedCalls tests method chaining
func TestWhereBuilder_ChainedCalls(t *testing.T) {

	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	wb := NewWhereBuilder().
		AddDateRange(&start, &end).
		AddUsers([]string{"alice", "bob"}).
		AddMediaTypes([]string{"movie"}).
		AddPlatforms([]string{"Roku"}).
		AddLibraries([]string{"Movies"}).
		AddClause("active = ?", true)

	whereClause, args := wb.Build()

	// Check clause count: AddDateRange adds 2 clauses (start and end), so:
	// 2 (dates) + 1 (users) + 1 (media) + 1 (platform) + 1 (library) + 1 (custom) = 7
	if wb.Count() != 7 {
		t.Errorf("Expected 7 clauses, got %d", wb.Count())
	}

	// Check total args: 2 dates + 2 users + 1 media + 1 platform + 1 library + 1 custom = 8
	if len(args) != 8 {
		t.Errorf("Expected 8 args, got %d", len(args))
	}

	// Check that the clause contains expected parts
	expectedParts := []string{
		"started_at >= ?",
		"started_at <= ?",
		"username IN",
		"media_type IN",
		"platform IN",
		"library_name IN",
		"active = ?",
	}

	for _, part := range expectedParts {
		if !containsString(whereClause, part) {
			t.Errorf("Expected clause to contain %q, got %q", part, whereClause)
		}
	}
}

// TestWhereBuilder_IsEmpty tests the IsEmpty method
func TestWhereBuilder_IsEmpty(t *testing.T) {

	wb := NewWhereBuilder()
	if !wb.IsEmpty() {
		t.Error("New builder should be empty")
	}

	wb.AddClause("test = ?", 1)
	if wb.IsEmpty() {
		t.Error("Builder should not be empty after adding clause")
	}
}

// TestWhereBuilder_Count tests the Count method
func TestWhereBuilder_Count(t *testing.T) {

	wb := NewWhereBuilder()
	if wb.Count() != 0 {
		t.Errorf("Expected count 0, got %d", wb.Count())
	}

	wb.AddClause("a = ?", 1)
	if wb.Count() != 1 {
		t.Errorf("Expected count 1, got %d", wb.Count())
	}

	wb.AddClause("b = ?", 2)
	if wb.Count() != 2 {
		t.Errorf("Expected count 2, got %d", wb.Count())
	}
}

// TestWhereBuilder_BuildWithPrefix_Empty tests BuildWithPrefix with empty builder
func TestWhereBuilder_BuildWithPrefix_Empty(t *testing.T) {

	wb := NewWhereBuilder()
	whereClause, args := wb.BuildWithPrefix()

	expected := "WHERE 1=1"
	if whereClause != expected {
		t.Errorf("Expected %q, got %q", expected, whereClause)
	}
	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

// TestWhereBuilder_ArgumentOrder tests that arguments are in correct order
func TestWhereBuilder_ArgumentOrder(t *testing.T) {

	start := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	wb := NewWhereBuilder().
		AddDateRange(&start, nil).
		AddUsers([]string{"user1"}).
		AddClause("custom = ?", "value")

	_, args := wb.Build()

	// Verify argument order: date, user, custom
	if len(args) != 3 {
		t.Fatalf("Expected 3 args, got %d", len(args))
	}

	// First arg should be the date
	if _, ok := args[0].(time.Time); !ok {
		t.Errorf("Expected first arg to be time.Time, got %T", args[0])
	}

	// Second arg should be user string
	if args[1] != "user1" {
		t.Errorf("Expected second arg to be 'user1', got %v", args[1])
	}

	// Third arg should be custom value
	if args[2] != "value" {
		t.Errorf("Expected third arg to be 'value', got %v", args[2])
	}
}

// BenchmarkWhereBuilder_Build benchmarks the Build method
func BenchmarkWhereBuilder_Build(b *testing.B) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb := NewWhereBuilder().
			AddDateRange(&start, &end).
			AddUsers([]string{"alice", "bob", "charlie"}).
			AddMediaTypes([]string{"movie", "episode"}).
			AddPlatforms([]string{"Roku", "Android"}).
			AddLibraries([]string{"Movies", "TV Shows"})
		_, _ = wb.Build()
	}
}

// BenchmarkWhereBuilder_Large benchmarks with many values
func BenchmarkWhereBuilder_Large(b *testing.B) {
	users := make([]string, 100)
	for i := range users {
		users[i] = "user" + string(rune('0'+i%10))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wb := NewWhereBuilder()
		wb.AddUsers(users)
		_, _ = wb.Build()
	}
}

// Helper functions
func timePtr(t time.Time) *time.Time {
	return &t
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
