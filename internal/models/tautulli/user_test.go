// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliUser_JSONUnmarshal(t *testing.T) {
	t.Run("complete user data", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"user_id": 12345,
					"username": "testuser",
					"friendly_name": "Test User",
					"user_thumb": "https://plex.tv/users/abc/avatar",
					"email": "test@example.com",
					"is_home_user": 1,
					"is_allow_sync": 1,
					"is_restricted": 0,
					"do_notify": 1,
					"keep_history": 1,
					"deleted_user": 0,
					"allow_guest": 0,
					"server_token": "server-token-xyz",
					"shared_libraries": ["1", "2", "3"],
					"filter_all": "",
					"filter_movies": "contentRating=PG-13",
					"filter_tv": "",
					"filter_music": "",
					"filter_photos": ""
				}
			}
		}`

		var user TautulliUser
		if err := json.Unmarshal([]byte(jsonData), &user); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if user.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", user.Response.Result)
		}

		data := user.Response.Data
		if data.UserID != 12345 {
			t.Errorf("Expected user_id 12345, got %d", data.UserID)
		}
		if data.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got %q", data.Username)
		}
		if data.FriendlyName != "Test User" {
			t.Errorf("Expected friendly_name 'Test User', got %q", data.FriendlyName)
		}
		if data.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %q", data.Email)
		}
		if data.IsHomeUser != 1 {
			t.Errorf("Expected is_home_user 1, got %d", data.IsHomeUser)
		}
		if data.IsAllowSync != 1 {
			t.Errorf("Expected is_allow_sync 1, got %d", data.IsAllowSync)
		}
		if data.IsRestricted != 0 {
			t.Errorf("Expected is_restricted 0, got %d", data.IsRestricted)
		}
		if data.DoNotify != 1 {
			t.Errorf("Expected do_notify 1, got %d", data.DoNotify)
		}
		if data.KeepHistory != 1 {
			t.Errorf("Expected keep_history 1, got %d", data.KeepHistory)
		}
		if data.DeletedUser != 0 {
			t.Errorf("Expected deleted_user 0, got %d", data.DeletedUser)
		}
		if data.ServerToken != "server-token-xyz" {
			t.Errorf("Expected server_token 'server-token-xyz', got %q", data.ServerToken)
		}
		if len(data.SharedLibraries) != 3 {
			t.Errorf("Expected 3 shared libraries, got %d", len(data.SharedLibraries))
		}
		if data.FilterMovies != "contentRating=PG-13" {
			t.Errorf("Expected filter_movies 'contentRating=PG-13', got %q", data.FilterMovies)
		}
	})

	t.Run("restricted user", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"user_id": 99,
					"username": "kiduser",
					"friendly_name": "Kid Account",
					"user_thumb": "",
					"email": "",
					"is_home_user": 0,
					"is_allow_sync": 0,
					"is_restricted": 1,
					"do_notify": 0,
					"keep_history": 1,
					"deleted_user": 0,
					"allow_guest": 0,
					"server_token": "",
					"shared_libraries": ["1"],
					"filter_all": "contentRating!=NC-17|R",
					"filter_movies": "contentRating!=NC-17|R",
					"filter_tv": "contentRating!=TV-MA",
					"filter_music": "",
					"filter_photos": ""
				}
			}
		}`

		var user TautulliUser
		if err := json.Unmarshal([]byte(jsonData), &user); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := user.Response.Data
		if data.IsRestricted != 1 {
			t.Errorf("Expected is_restricted 1, got %d", data.IsRestricted)
		}
		if data.IsAllowSync != 0 {
			t.Errorf("Expected is_allow_sync 0, got %d", data.IsAllowSync)
		}
		if data.FilterAll != "contentRating!=NC-17|R" {
			t.Errorf("Expected filter_all with restrictions, got %q", data.FilterAll)
		}
	})

	t.Run("error response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "User not found",
				"data": {}
			}
		}`

		var user TautulliUser
		if err := json.Unmarshal([]byte(jsonData), &user); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if user.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", user.Response.Result)
		}
		if user.Response.Message == nil || *user.Response.Message != "User not found" {
			t.Error("Expected error message 'User not found'")
		}
	})
}

func TestTautulliUsersTable_JSONUnmarshal(t *testing.T) {
	t.Run("multiple users", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsTotal": 100,
					"recordsFiltered": 50,
					"draw": 1,
					"data": [
						{
							"user_id": 1,
							"user": "admin",
							"friendly_name": "Admin User",
							"user_thumb": "/thumb/admin.jpg",
							"plays": 500,
							"duration": 1800000,
							"last_seen": 1640995200,
							"last_played": "Breaking Bad - S01E01",
							"ip_address": "192.168.1.100",
							"platform": "Roku",
							"player": "Roku Ultra",
							"media_type": "episode",
							"title": "Pilot",
							"thumb": "/thumb/episode.jpg",
							"rating_key": "12345"
						},
						{
							"user_id": 2,
							"user": "guest",
							"friendly_name": "Guest User",
							"user_thumb": "",
							"plays": 10,
							"duration": 36000,
							"last_seen": 1640908800,
							"last_played": "Inception"
						}
					]
				}
			}
		}`

		var table TautulliUsersTable
		if err := json.Unmarshal([]byte(jsonData), &table); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if table.Response.Data.RecordsTotal != 100 {
			t.Errorf("Expected recordsTotal 100, got %d", table.Response.Data.RecordsTotal)
		}
		if table.Response.Data.RecordsFiltered != 50 {
			t.Errorf("Expected recordsFiltered 50, got %d", table.Response.Data.RecordsFiltered)
		}
		if len(table.Response.Data.Data) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(table.Response.Data.Data))
		}

		user1 := table.Response.Data.Data[0]
		if user1.UserID != 1 {
			t.Errorf("Expected user_id 1, got %d", user1.UserID)
		}
		if user1.Username != "admin" {
			t.Errorf("Expected user 'admin', got %q", user1.Username)
		}
		if user1.Plays != 500 {
			t.Errorf("Expected plays 500, got %d", user1.Plays)
		}
		if user1.Duration != 1800000 {
			t.Errorf("Expected duration 1800000, got %d", user1.Duration)
		}
		if user1.IPAddress != "192.168.1.100" {
			t.Errorf("Expected ip_address '192.168.1.100', got %q", user1.IPAddress)
		}

		user2 := table.Response.Data.Data[1]
		if user2.Plays != 10 {
			t.Errorf("Expected plays 10, got %d", user2.Plays)
		}
		if user2.IPAddress != "" {
			t.Errorf("Expected empty ip_address, got %q", user2.IPAddress)
		}
	})

	t.Run("empty users", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsTotal": 0,
					"recordsFiltered": 0,
					"draw": 1,
					"data": []
				}
			}
		}`

		var table TautulliUsersTable
		if err := json.Unmarshal([]byte(jsonData), &table); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(table.Response.Data.Data) != 0 {
			t.Errorf("Expected empty users, got %d", len(table.Response.Data.Data))
		}
	})
}

func TestTautulliUserIPs_JSONUnmarshal(t *testing.T) {
	t.Run("multiple IPs", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"friendly_name": "Test User",
						"ip_address": "192.168.1.100",
						"last_seen": 1640995200,
						"last_played": "Breaking Bad - S01E01",
						"play_count": 50,
						"platform_name": "Roku",
						"player_name": "Roku Ultra",
						"user_id": 1
					},
					{
						"friendly_name": "Test User",
						"ip_address": "10.0.0.50",
						"last_seen": 1640908800,
						"last_played": "Inception",
						"play_count": 25,
						"platform_name": "iOS",
						"player_name": "iPhone",
						"user_id": 1
					}
				]
			}
		}`

		var ips TautulliUserIPs
		if err := json.Unmarshal([]byte(jsonData), &ips); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(ips.Response.Data) != 2 {
			t.Fatalf("Expected 2 IPs, got %d", len(ips.Response.Data))
		}

		ip1 := ips.Response.Data[0]
		if ip1.IPAddress != "192.168.1.100" {
			t.Errorf("Expected ip_address '192.168.1.100', got %q", ip1.IPAddress)
		}
		if ip1.PlayCount != 50 {
			t.Errorf("Expected play_count 50, got %d", ip1.PlayCount)
		}
		if ip1.PlatformName != "Roku" {
			t.Errorf("Expected platform_name 'Roku', got %q", ip1.PlatformName)
		}
	})
}

func TestTautulliUserLogins_JSONUnmarshal(t *testing.T) {
	t.Run("successful and failed logins", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"recordsTotal": 100,
					"recordsFiltered": 10,
					"draw": 1,
					"data": [
						{
							"timestamp": 1640995200,
							"time": "2022-01-01 00:00:00",
							"user_id": 1,
							"user": "admin",
							"friendly_name": "Admin",
							"ip_address": "192.168.1.100",
							"host": "client.local",
							"user_agent": "Mozilla/5.0",
							"os": "Windows 10",
							"browser": "Chrome 96",
							"success": 1
						},
						{
							"timestamp": 1640908800,
							"time": "2021-12-31 00:00:00",
							"user_id": 2,
							"user": "hacker",
							"friendly_name": "Unknown",
							"ip_address": "203.0.113.50",
							"host": "attacker.bad",
							"user_agent": "curl/7.68.0",
							"os": "Linux",
							"browser": "Other",
							"success": 0
						}
					]
				}
			}
		}`

		var logins TautulliUserLogins
		if err := json.Unmarshal([]byte(jsonData), &logins); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(logins.Response.Data.Data) != 2 {
			t.Fatalf("Expected 2 logins, got %d", len(logins.Response.Data.Data))
		}

		login1 := logins.Response.Data.Data[0]
		if login1.Success != 1 {
			t.Errorf("Expected success 1, got %d", login1.Success)
		}
		if login1.Browser != "Chrome 96" {
			t.Errorf("Expected browser 'Chrome 96', got %q", login1.Browser)
		}
		if login1.OS != "Windows 10" {
			t.Errorf("Expected os 'Windows 10', got %q", login1.OS)
		}

		login2 := logins.Response.Data.Data[1]
		if login2.Success != 0 {
			t.Errorf("Expected success 0, got %d", login2.Success)
		}
		if login2.IPAddress != "203.0.113.50" {
			t.Errorf("Expected ip_address '203.0.113.50', got %q", login2.IPAddress)
		}
	})
}

func TestTautulliUserPlayerStats_JSONUnmarshal(t *testing.T) {
	t.Run("multiple players", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"player_name": "Roku Ultra",
						"platform_name": "Roku",
						"platform_type": "streaming",
						"result_id": 1,
						"row_id": 100,
						"total_plays": 250,
						"last_play": 1640995200,
						"last_played": "Breaking Bad - S01E01",
						"media_type": "episode",
						"rating_key": "12345",
						"thumb": "/thumb/episode.jpg",
						"title": "Pilot",
						"user_id": 1
					},
					{
						"player_name": "iPhone",
						"platform_name": "iOS",
						"platform_type": "mobile",
						"result_id": 2,
						"row_id": 101,
						"total_plays": 100,
						"last_play": 1640908800,
						"last_played": "Inception"
					}
				]
			}
		}`

		var stats TautulliUserPlayerStats
		if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(stats.Response.Data) != 2 {
			t.Fatalf("Expected 2 player stats, got %d", len(stats.Response.Data))
		}

		stat1 := stats.Response.Data[0]
		if stat1.PlayerName != "Roku Ultra" {
			t.Errorf("Expected player_name 'Roku Ultra', got %q", stat1.PlayerName)
		}
		if stat1.PlatformType != "streaming" {
			t.Errorf("Expected platform_type 'streaming', got %q", stat1.PlatformType)
		}
		if stat1.TotalPlays != 250 {
			t.Errorf("Expected total_plays 250, got %d", stat1.TotalPlays)
		}

		stat2 := stats.Response.Data[1]
		if stat2.PlatformType != "mobile" {
			t.Errorf("Expected platform_type 'mobile', got %q", stat2.PlatformType)
		}
	})
}

func TestTautulliUserWatchTimeStats_JSONUnmarshal(t *testing.T) {
	t.Run("watch time by media type", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"query_days": 30,
						"total_time": 360000,
						"total_plays": 100,
						"media_type": "movie"
					},
					{
						"query_days": 30,
						"total_time": 720000,
						"total_plays": 200,
						"media_type": "episode"
					},
					{
						"query_days": 30,
						"total_time": 18000,
						"total_plays": 50,
						"media_type": "track"
					}
				]
			}
		}`

		var stats TautulliUserWatchTimeStats
		if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(stats.Response.Data) != 3 {
			t.Fatalf("Expected 3 watch time stats, got %d", len(stats.Response.Data))
		}

		movieStat := stats.Response.Data[0]
		if movieStat.MediaType != "movie" {
			t.Errorf("Expected media_type 'movie', got %q", movieStat.MediaType)
		}
		if movieStat.TotalTime != 360000 {
			t.Errorf("Expected total_time 360000, got %d", movieStat.TotalTime)
		}
		if movieStat.TotalPlays != 100 {
			t.Errorf("Expected total_plays 100, got %d", movieStat.TotalPlays)
		}
		if movieStat.QueryDays != 30 {
			t.Errorf("Expected query_days 30, got %d", movieStat.QueryDays)
		}

		episodeStat := stats.Response.Data[1]
		if episodeStat.TotalTime != 720000 {
			t.Errorf("Expected total_time 720000, got %d", episodeStat.TotalTime)
		}
	})
}

func TestTautulliUsers_JSONUnmarshal(t *testing.T) {
	t.Run("multiple users list", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"user_id": 1,
						"username": "admin",
						"friendly_name": "Admin",
						"user_thumb": "/thumb/admin.jpg",
						"email": "admin@example.com",
						"is_home_user": 0,
						"is_allow_sync": 1,
						"is_restricted": 0,
						"do_notify": 1,
						"keep_history": 1,
						"deleted_user": 0,
						"allow_guest": 0,
						"server_token": "token1",
						"shared_libraries": ["1", "2", "3"],
						"filter_all": "",
						"filter_movies": "",
						"filter_tv": "",
						"filter_music": "",
						"filter_photos": ""
					},
					{
						"user_id": 2,
						"username": "guest",
						"friendly_name": "Guest",
						"user_thumb": "",
						"email": "",
						"is_home_user": 1,
						"is_allow_sync": 0,
						"is_restricted": 1,
						"do_notify": 0,
						"keep_history": 0,
						"deleted_user": 0,
						"allow_guest": 1,
						"server_token": "",
						"shared_libraries": ["1"],
						"filter_all": "contentRating!=R",
						"filter_movies": "contentRating!=R",
						"filter_tv": "",
						"filter_music": "",
						"filter_photos": ""
					}
				]
			}
		}`

		var users TautulliUsers
		if err := json.Unmarshal([]byte(jsonData), &users); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(users.Response.Data) != 2 {
			t.Fatalf("Expected 2 users, got %d", len(users.Response.Data))
		}

		user1 := users.Response.Data[0]
		if user1.UserID != 1 {
			t.Errorf("Expected user_id 1, got %d", user1.UserID)
		}
		if user1.IsHomeUser != 0 {
			t.Errorf("Expected is_home_user 0, got %d", user1.IsHomeUser)
		}
		if len(user1.SharedLibraries) != 3 {
			t.Errorf("Expected 3 shared libraries, got %d", len(user1.SharedLibraries))
		}

		user2 := users.Response.Data[1]
		if user2.IsRestricted != 1 {
			t.Errorf("Expected is_restricted 1, got %d", user2.IsRestricted)
		}
		if user2.FilterAll != "contentRating!=R" {
			t.Errorf("Expected filter_all 'contentRating!=R', got %q", user2.FilterAll)
		}
	})

	t.Run("empty users list", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": []
			}
		}`

		var users TautulliUsers
		if err := json.Unmarshal([]byte(jsonData), &users); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(users.Response.Data) != 0 {
			t.Errorf("Expected empty users, got %d", len(users.Response.Data))
		}
	})
}

func TestTautulliUserData_RoundTrip(t *testing.T) {
	original := TautulliUserData{
		UserID:          999,
		Username:        "roundtrip",
		FriendlyName:    "Round Trip User",
		UserThumb:       "/thumb/user.jpg",
		Email:           "roundtrip@example.com",
		IsHomeUser:      1,
		IsAllowSync:     1,
		IsRestricted:    0,
		DoNotify:        1,
		KeepHistory:     1,
		DeletedUser:     0,
		AllowGuest:      0,
		ServerToken:     "token-xyz",
		SharedLibraries: []string{"1", "2", "3"},
		FilterAll:       "",
		FilterMovies:    "contentRating=PG",
		FilterTV:        "",
		FilterMusic:     "",
		FilterPhotos:    "",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliUserData
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.UserID != original.UserID {
		t.Error("UserID not preserved")
	}
	if result.Username != original.Username {
		t.Error("Username not preserved")
	}
	if result.Email != original.Email {
		t.Error("Email not preserved")
	}
	if len(result.SharedLibraries) != 3 {
		t.Error("SharedLibraries not preserved")
	}
	if result.FilterMovies != original.FilterMovies {
		t.Error("FilterMovies not preserved")
	}
}

func TestTautulliUserData_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"user_id": 1,
				"username": "user<script>alert('xss')</script>",
				"friendly_name": "Test \"User\" & <Special>",
				"user_thumb": "/thumb/user's-avatar.jpg",
				"email": "user+tag@example.com",
				"is_home_user": 0,
				"is_allow_sync": 1,
				"is_restricted": 0,
				"do_notify": 1,
				"keep_history": 1,
				"deleted_user": 0,
				"allow_guest": 0,
				"server_token": "token-with-special!@#$",
				"shared_libraries": ["1"],
				"filter_all": "title~=O'Brien",
				"filter_movies": "",
				"filter_tv": "",
				"filter_music": "",
				"filter_photos": ""
			}
		}
	}`

	var user TautulliUser
	if err := json.Unmarshal([]byte(jsonData), &user); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	data := user.Response.Data
	if data.Username != "user<script>alert('xss')</script>" {
		t.Errorf("Username with HTML not preserved: %q", data.Username)
	}
	if data.FriendlyName != "Test \"User\" & <Special>" {
		t.Errorf("FriendlyName with special chars not preserved: %q", data.FriendlyName)
	}
	if data.Email != "user+tag@example.com" {
		t.Errorf("Email with plus not preserved: %q", data.Email)
	}
	if data.FilterAll != "title~=O'Brien" {
		t.Errorf("FilterAll with apostrophe not preserved: %q", data.FilterAll)
	}
}

func TestTautulliUsersTableRow_ZeroValues(t *testing.T) {
	row := TautulliUsersTableRow{
		UserID:       0,
		Username:     "zerouser",
		FriendlyName: "Zero User",
		Plays:        0,
		Duration:     0,
		LastSeen:     0,
	}

	data, err := json.Marshal(row)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliUsersTableRow
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.UserID != 0 {
		t.Errorf("Expected user_id 0, got %d", result.UserID)
	}
	if result.Plays != 0 {
		t.Errorf("Expected plays 0, got %d", result.Plays)
	}
	if result.Duration != 0 {
		t.Errorf("Expected duration 0, got %d", result.Duration)
	}
	if result.LastSeen != 0 {
		t.Errorf("Expected last_seen 0, got %d", result.LastSeen)
	}
}

func TestTautulliUserWatchTimeStatRow_LargeValues(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": [
				{
					"query_days": 365,
					"total_time": 31536000,
					"total_plays": 10000,
					"media_type": "movie"
				}
			]
		}
	}`

	var stats TautulliUserWatchTimeStats
	if err := json.Unmarshal([]byte(jsonData), &stats); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	stat := stats.Response.Data[0]
	// 31536000 seconds = 365 days
	if stat.TotalTime != 31536000 {
		t.Errorf("Expected total_time 31536000, got %d", stat.TotalTime)
	}
	if stat.TotalPlays != 10000 {
		t.Errorf("Expected total_plays 10000, got %d", stat.TotalPlays)
	}
	if stat.QueryDays != 365 {
		t.Errorf("Expected query_days 365, got %d", stat.QueryDays)
	}
}
