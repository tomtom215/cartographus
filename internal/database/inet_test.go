// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"testing"
)

// TestInetExtensionAvailable verifies the INET extension is loaded
func TestInetExtensionAvailable(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Check if INET extension is available
	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	// Verify we can use INET functions
	var result string
	err := db.conn.QueryRow("SELECT host('192.168.1.1'::INET)").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute INET host() function: %v", err)
	}

	if result != "192.168.1.1" {
		t.Errorf("Expected host() to return '192.168.1.1', got '%s'", result)
	}
}

// TestInetCasting tests casting TEXT IP addresses to INET type
func TestInetCasting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	tests := []struct {
		name    string
		ip      string
		wantErr bool
	}{
		{"valid IPv4", "192.168.1.1", false},
		{"valid IPv4 with zeros", "10.0.0.1", false},
		{"valid IPv6", "2001:db8::1", false},
		{"valid IPv6 full", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"valid CIDR", "192.168.0.0/24", false},
		{"localhost IPv4", "127.0.0.1", false},
		{"localhost IPv6", "::1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.conn.QueryRow("SELECT host($1::INET)", tt.ip).Scan(&result)
			if (err != nil) != tt.wantErr {
				t.Errorf("Casting %s to INET: error = %v, wantErr %v", tt.ip, err, tt.wantErr)
			}
		})
	}
}

// TestInetComparison tests that INET comparison is numeric, not lexicographic
func TestInetComparison(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	// With TEXT: '192.168.1.10' < '192.168.1.2' (lexicographic - WRONG!)
	// With INET: '192.168.1.10' > '192.168.1.2' (numeric - CORRECT!)
	var result bool
	err := db.conn.QueryRow("SELECT '192.168.1.10'::INET > '192.168.1.2'::INET").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to compare INET values: %v", err)
	}

	if !result {
		t.Error("INET comparison failed: expected 192.168.1.10 > 192.168.1.2 to be true")
	}
}

// TestInetNetworkFunctions tests network-related INET functions
func TestInetNetworkFunctions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "host() extracts IP from CIDR",
			query:    "SELECT host('192.168.1.100/24'::INET)",
			expected: "192.168.1.100",
		},
		{
			name:     "host() on plain IP",
			query:    "SELECT host('127.0.0.1'::INET)",
			expected: "127.0.0.1",
		},
		{
			// netmask() returns INET type, cast to VARCHAR for scanning
			name:     "netmask() returns network mask with CIDR",
			query:    "SELECT netmask('192.168.1.5/24'::INET)::VARCHAR",
			expected: "255.255.255.0/24",
		},
		{
			// network() returns INET type, cast to VARCHAR for scanning
			name:     "network() returns network address",
			query:    "SELECT network('192.168.1.100/24'::INET)::VARCHAR",
			expected: "192.168.1.0/24",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.conn.QueryRow(tt.query).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestInetBroadcastFunction tests the broadcast() function separately
// Note: broadcast() behavior may vary by DuckDB version
func TestInetBroadcastFunction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	// Just verify broadcast() function works without error
	// broadcast() returns INET type, cast to VARCHAR for scanning
	var result string
	err := db.conn.QueryRow("SELECT broadcast('192.168.1.5/24'::INET)::VARCHAR").Scan(&result)
	if err != nil {
		t.Fatalf("broadcast() function failed: %v", err)
	}
	t.Logf("broadcast('192.168.1.5/24') = %s", result)
}

// TestInetArithmetic tests IP address arithmetic operations
func TestInetArithmetic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "add integer to IPv4",
			query:    "SELECT host('127.0.0.1'::INET + 10)",
			expected: "127.0.0.11",
		},
		{
			name:     "subtract integer from IPv4",
			query:    "SELECT host('127.0.0.11'::INET - 10)",
			expected: "127.0.0.1",
		},
		{
			name:     "add to IPv6",
			query:    "SELECT host('fe80::10'::INET - 9)",
			expected: "fe80::7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.conn.QueryRow(tt.query).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// TestInetContainment tests CIDR containment operators
func TestInetContainment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{
			name:     "IP within CIDR (contained by)",
			query:    "SELECT '192.168.1.100'::INET <<= '192.168.1.0/24'::INET",
			expected: true,
		},
		{
			name:     "IP outside CIDR",
			query:    "SELECT '192.168.2.100'::INET <<= '192.168.1.0/24'::INET",
			expected: false,
		},
		{
			name:     "CIDR contains IP (contains)",
			query:    "SELECT '192.168.0.0/16'::INET >>= '192.168.1.100'::INET",
			expected: true,
		},
		{
			name:     "Private network check",
			query:    "SELECT '10.0.0.1'::INET <<= '10.0.0.0/8'::INET",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result bool
			err := db.conn.QueryRow(tt.query).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestInetSortOrder tests that INET values sort correctly (numerically)
func TestInetSortOrder(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	// Create a temporary table with IP addresses and verify sort order
	_, err := db.conn.Exec(`
		CREATE TEMPORARY TABLE test_ips (ip TEXT);
		INSERT INTO test_ips VALUES
			('192.168.1.2'),
			('192.168.1.10'),
			('192.168.1.1'),
			('10.0.0.1'),
			('172.16.0.1');
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Query sorted by INET (numeric order)
	rows, err := db.conn.Query("SELECT ip FROM test_ips ORDER BY ip::INET")
	if err != nil {
		t.Fatalf("Failed to query with INET sort: %v", err)
	}
	defer rows.Close()

	expectedOrder := []string{
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"192.168.1.2",
		"192.168.1.10", // Would be wrong position with TEXT sort
	}

	var i int
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		if i >= len(expectedOrder) {
			t.Fatalf("Too many rows returned")
		}
		if ip != expectedOrder[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expectedOrder[i], ip)
		}
		i++
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}

	if i != len(expectedOrder) {
		t.Errorf("Expected %d rows, got %d", len(expectedOrder), i)
	}
}

// TestInetIPv4BeforeIPv6 tests that IPv4 addresses sort before IPv6
func TestInetIPv4BeforeIPv6(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	// Per DuckDB docs: "IPv4 will sort before IPv6"
	_, err := db.conn.Exec(`
		CREATE TEMPORARY TABLE test_mixed_ips (ip TEXT);
		INSERT INTO test_mixed_ips VALUES
			('2001:db8:3c4d:15::1a2f:1a2b'),
			('127.0.0.1'),
			('fe80::7'),
			('192.168.1.1');
	`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	rows, err := db.conn.Query("SELECT ip FROM test_mixed_ips ORDER BY ip::INET ASC")
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	// IPv4 should come before IPv6
	expectedOrder := []string{
		"127.0.0.1",
		"192.168.1.1",
		"2001:db8:3c4d:15::1a2f:1a2b",
		"fe80::7",
	}

	var i int
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			t.Fatalf("Failed to scan row: %v", err)
		}
		if i >= len(expectedOrder) {
			t.Fatalf("Too many rows")
		}
		if ip != expectedOrder[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expectedOrder[i], ip)
		}
		i++
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("Row iteration error: %v", err)
	}
}

// TestInetHtmlFunctions tests HTML escape/unescape functions included in inet extension
func TestInetHtmlFunctions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	if !db.IsInetAvailable() {
		t.Skip("INET extension not available")
		return
	}

	tests := []struct {
		name     string
		query    string
		expected string
	}{
		{
			name:     "html_escape ampersand",
			query:    "SELECT html_escape('&')",
			expected: "&amp;",
		},
		{
			name:     "html_escape less than",
			query:    "SELECT html_escape('<')",
			expected: "&lt;",
		},
		{
			name:     "html_escape greater than",
			query:    "SELECT html_escape('>')",
			expected: "&gt;",
		},
		{
			name:     "html_unescape ampersand",
			query:    "SELECT html_unescape('&amp;')",
			expected: "&",
		},
		{
			name:     "html_unescape less than",
			query:    "SELECT html_unescape('&lt;')",
			expected: "<",
		},
		{
			name:     "html_escape full string",
			query:    "SELECT html_escape('<script>alert(\"xss\")</script>')",
			expected: "&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			err := db.conn.QueryRow(tt.query).Scan(&result)
			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}
