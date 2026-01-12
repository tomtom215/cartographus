// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

import (
	"encoding/json"
	"testing"
)

func TestTautulliServerInfo_JSONUnmarshal(t *testing.T) {
	t.Run("complete server info", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"message": null,
				"data": {
					"machine_identifier": "abc123def456",
					"plex_server_name": "My Plex Server",
					"plex_server_version": "1.32.0.6865",
					"plex_server_platform": "Linux",
					"platform": "Linux",
					"platform_version": "5.15.0-91-generic",
					"pms_ip": "192.168.1.100",
					"pms_port": 32400,
					"pms_url": "http://192.168.1.100:32400",
					"pms_web_url": "https://app.plex.tv/desktop",
					"pms_ssl": 1,
					"pms_is_remote": 0,
					"pms_is_cloud": 0,
					"pms_plexpass": 1,
					"plex_server_up_to_date": 1,
					"update_available": 0
				}
			}
		}`

		var info TautulliServerInfo
		if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if info.Response.Result != "success" {
			t.Errorf("Expected result 'success', got %q", info.Response.Result)
		}

		data := info.Response.Data
		if data.PMSIdentifier != "abc123def456" {
			t.Errorf("Expected machine_identifier 'abc123def456', got %q", data.PMSIdentifier)
		}
		if data.PMSName != "My Plex Server" {
			t.Errorf("Expected plex_server_name 'My Plex Server', got %q", data.PMSName)
		}
		if data.PMSVersion != "1.32.0.6865" {
			t.Errorf("Expected plex_server_version '1.32.0.6865', got %q", data.PMSVersion)
		}
		if data.PMSPlatform != "Linux" {
			t.Errorf("Expected plex_server_platform 'Linux', got %q", data.PMSPlatform)
		}
		if data.Platform != "Linux" {
			t.Errorf("Expected platform 'Linux', got %q", data.Platform)
		}
		if data.PMSIP != "192.168.1.100" {
			t.Errorf("Expected pms_ip '192.168.1.100', got %q", data.PMSIP)
		}
		if data.PMSPort != 32400 {
			t.Errorf("Expected pms_port 32400, got %d", data.PMSPort)
		}
		if data.PMSSSLEnabled != 1 {
			t.Errorf("Expected pms_ssl 1, got %d", data.PMSSSLEnabled)
		}
		if data.PMSIsRemote != 0 {
			t.Errorf("Expected pms_is_remote 0, got %d", data.PMSIsRemote)
		}
		if data.PMSPlexpass != 1 {
			t.Errorf("Expected pms_plexpass 1, got %d", data.PMSPlexpass)
		}
		if data.PlexServerUpToDate != 1 {
			t.Errorf("Expected plex_server_up_to_date 1, got %d", data.PlexServerUpToDate)
		}
		if data.PMSUpdateAvailable != 0 {
			t.Errorf("Expected update_available 0, got %d", data.PMSUpdateAvailable)
		}
	})

	t.Run("with update available", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"machine_identifier": "xyz789",
					"plex_server_name": "Test Server",
					"plex_server_version": "1.31.0.0000",
					"plex_server_platform": "Windows",
					"platform": "Windows",
					"platform_version": "10.0.19045",
					"pms_ip": "10.0.0.50",
					"pms_port": 32400,
					"pms_url": "http://10.0.0.50:32400",
					"pms_web_url": "https://app.plex.tv/desktop",
					"pms_ssl": 0,
					"pms_is_remote": 1,
					"pms_is_cloud": 0,
					"pms_plexpass": 0,
					"plex_server_up_to_date": 0,
					"update_available": 1,
					"update_version": "1.32.0.6865",
					"update_release_date": "2024-01-15"
				}
			}
		}`

		var info TautulliServerInfo
		if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := info.Response.Data
		if data.PMSUpdateAvailable != 1 {
			t.Errorf("Expected update_available 1, got %d", data.PMSUpdateAvailable)
		}
		if data.PMSUpdateVersion != "1.32.0.6865" {
			t.Errorf("Expected update_version '1.32.0.6865', got %q", data.PMSUpdateVersion)
		}
		if data.PMSUpdateReleaseDate != "2024-01-15" {
			t.Errorf("Expected update_release_date '2024-01-15', got %q", data.PMSUpdateReleaseDate)
		}
	})

	t.Run("error response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "error",
				"message": "Server not connected",
				"data": {}
			}
		}`

		var info TautulliServerInfo
		if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if info.Response.Result != "error" {
			t.Errorf("Expected result 'error', got %q", info.Response.Result)
		}
		if info.Response.Message == nil {
			t.Error("Expected non-nil message")
		} else if *info.Response.Message != "Server not connected" {
			t.Errorf("Expected message 'Server not connected', got %q", *info.Response.Message)
		}
	})
}

func TestTautulliServerFriendlyName_JSONUnmarshal(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"friendly_name": "Home Theater Server"
				}
			}
		}`

		var name TautulliServerFriendlyName
		if err := json.Unmarshal([]byte(jsonData), &name); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if name.Response.Data.FriendlyName != "Home Theater Server" {
			t.Errorf("Expected friendly_name 'Home Theater Server', got %q", name.Response.Data.FriendlyName)
		}
	})
}

func TestTautulliServerID_JSONUnmarshal(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"server_id": "machine-id-123456789"
				}
			}
		}`

		var id TautulliServerID
		if err := json.Unmarshal([]byte(jsonData), &id); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if id.Response.Data.ServerID != "machine-id-123456789" {
			t.Errorf("Expected server_id 'machine-id-123456789', got %q", id.Response.Data.ServerID)
		}
	})
}

func TestTautulliServerIdentity_JSONUnmarshal(t *testing.T) {
	t.Run("complete identity", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"machineIdentifier": "unique-machine-id",
					"version": "1.32.0.6865",
					"platform": "Linux",
					"platformVersion": "Ubuntu 22.04",
					"device": "Server",
					"createdAt": 1609459200
				}
			}
		}`

		var identity TautulliServerIdentity
		if err := json.Unmarshal([]byte(jsonData), &identity); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := identity.Response.Data
		if data.MachineIdentifier != "unique-machine-id" {
			t.Errorf("Expected machineIdentifier 'unique-machine-id', got %q", data.MachineIdentifier)
		}
		if data.Version != "1.32.0.6865" {
			t.Errorf("Expected version '1.32.0.6865', got %q", data.Version)
		}
		if data.Platform != "Linux" {
			t.Errorf("Expected platform 'Linux', got %q", data.Platform)
		}
		if data.PlatformVersion != "Ubuntu 22.04" {
			t.Errorf("Expected platformVersion 'Ubuntu 22.04', got %q", data.PlatformVersion)
		}
		if data.Device != "Server" {
			t.Errorf("Expected device 'Server', got %q", data.Device)
		}
		if data.CreatedAt != 1609459200 {
			t.Errorf("Expected createdAt 1609459200, got %d", data.CreatedAt)
		}
	})
}

func TestTautulliTautulliInfo_JSONUnmarshal(t *testing.T) {
	t.Run("complete tautulli info", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"tautulli_version": "2.13.4",
					"tautulli_install_type": "docker",
					"tautulli_branch": "master",
					"tautulli_commit": "abc123def",
					"platform": "Linux",
					"platform_release": "5.15.0-91-generic",
					"platform_version": "#101-Ubuntu SMP",
					"platform_linux_distro": "Ubuntu 22.04.3 LTS",
					"platform_device_name": "media-server",
					"python_version": "3.11.6"
				}
			}
		}`

		var info TautulliTautulliInfo
		if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := info.Response.Data
		if data.TautulliVersion != "2.13.4" {
			t.Errorf("Expected tautulli_version '2.13.4', got %q", data.TautulliVersion)
		}
		if data.TautulliInstallType != "docker" {
			t.Errorf("Expected tautulli_install_type 'docker', got %q", data.TautulliInstallType)
		}
		if data.TautulliBranch != "master" {
			t.Errorf("Expected tautulli_branch 'master', got %q", data.TautulliBranch)
		}
		if data.TautulliCommit != "abc123def" {
			t.Errorf("Expected tautulli_commit 'abc123def', got %q", data.TautulliCommit)
		}
		if data.PlatformLinuxDistro != "Ubuntu 22.04.3 LTS" {
			t.Errorf("Expected platform_linux_distro 'Ubuntu 22.04.3 LTS', got %q", data.PlatformLinuxDistro)
		}
		if data.PythonVersion != "3.11.6" {
			t.Errorf("Expected python_version '3.11.6', got %q", data.PythonVersion)
		}
	})
}

func TestTautulliServerPref_JSONUnmarshal(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"pref": "LogVerbose",
					"value": "0"
				}
			}
		}`

		var pref TautulliServerPref
		if err := json.Unmarshal([]byte(jsonData), &pref); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if pref.Response.Data.Pref != "LogVerbose" {
			t.Errorf("Expected pref 'LogVerbose', got %q", pref.Response.Data.Pref)
		}
		if pref.Response.Data.Value != "0" {
			t.Errorf("Expected value '0', got %q", pref.Response.Data.Value)
		}
	})
}

func TestTautulliServerList_JSONUnmarshal(t *testing.T) {
	t.Run("multiple servers", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": [
					{
						"machineIdentifier": "server1-id",
						"name": "Primary Server",
						"version": "1.32.0.6865",
						"platform": "Linux",
						"platformVersion": "Ubuntu 22.04",
						"createdAt": 1609459200,
						"updatedAt": 1640995200,
						"owned": 1
					},
					{
						"machineIdentifier": "server2-id",
						"name": "Backup Server",
						"version": "1.31.0.5000",
						"platform": "Windows",
						"platformVersion": "10.0.19045",
						"createdAt": 1620000000,
						"updatedAt": 1640990000,
						"owned": 0
					}
				]
			}
		}`

		var list TautulliServerList
		if err := json.Unmarshal([]byte(jsonData), &list); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(list.Response.Data) != 2 {
			t.Fatalf("Expected 2 servers, got %d", len(list.Response.Data))
		}

		server1 := list.Response.Data[0]
		if server1.MachineIdentifier != "server1-id" {
			t.Errorf("Expected machineIdentifier 'server1-id', got %q", server1.MachineIdentifier)
		}
		if server1.Name != "Primary Server" {
			t.Errorf("Expected name 'Primary Server', got %q", server1.Name)
		}
		if server1.Owned != 1 {
			t.Errorf("Expected owned 1, got %d", server1.Owned)
		}

		server2 := list.Response.Data[1]
		if server2.Platform != "Windows" {
			t.Errorf("Expected platform 'Windows', got %q", server2.Platform)
		}
		if server2.Owned != 0 {
			t.Errorf("Expected owned 0, got %d", server2.Owned)
		}
	})

	t.Run("empty server list", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": []
			}
		}`

		var list TautulliServerList
		if err := json.Unmarshal([]byte(jsonData), &list); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if len(list.Response.Data) != 0 {
			t.Errorf("Expected empty list, got %d items", len(list.Response.Data))
		}
	})
}

func TestTautulliPMSUpdate_JSONUnmarshal(t *testing.T) {
	t.Run("update available", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"update_available": 1,
					"platform_version": "1.31.0.5000",
					"platform_version_label": "Stable",
					"version": "1.32.0.6865",
					"release_date": "2024-01-15",
					"changelog": "Bug fixes and improvements",
					"label": "Recommended",
					"download_url": "https://plex.tv/downloads/latest",
					"requirements": "Linux x86_64"
				}
			}
		}`

		var update TautulliPMSUpdate
		if err := json.Unmarshal([]byte(jsonData), &update); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := update.Response.Data
		if data.UpdateAvailable != 1 {
			t.Errorf("Expected update_available 1, got %d", data.UpdateAvailable)
		}
		if data.PlatformVersion != "1.31.0.5000" {
			t.Errorf("Expected platform_version '1.31.0.5000', got %q", data.PlatformVersion)
		}
		if data.Version != "1.32.0.6865" {
			t.Errorf("Expected version '1.32.0.6865', got %q", data.Version)
		}
		if data.ReleaseDate != "2024-01-15" {
			t.Errorf("Expected release_date '2024-01-15', got %q", data.ReleaseDate)
		}
		if data.Changelog != "Bug fixes and improvements" {
			t.Errorf("Expected changelog content, got %q", data.Changelog)
		}
		if data.DownloadURL != "https://plex.tv/downloads/latest" {
			t.Errorf("Expected download_url, got %q", data.DownloadURL)
		}
	})

	t.Run("no update available", func(t *testing.T) {
		jsonData := `{
			"response": {
				"result": "success",
				"data": {
					"update_available": 0,
					"platform_version": "1.32.0.6865"
				}
			}
		}`

		var update TautulliPMSUpdate
		if err := json.Unmarshal([]byte(jsonData), &update); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		data := update.Response.Data
		if data.UpdateAvailable != 0 {
			t.Errorf("Expected update_available 0, got %d", data.UpdateAvailable)
		}
		if data.Version != "" {
			t.Errorf("Expected empty version, got %q", data.Version)
		}
	})
}

func TestTautulliServerInfo_RoundTrip(t *testing.T) {
	msg := "test"
	original := TautulliServerInfo{
		Response: TautulliServerInfoResponse{
			Result:  "success",
			Message: &msg,
			Data: TautulliServerInfoData{
				PMSIdentifier:      "roundtrip-id",
				PMSName:            "RoundTrip Server",
				PMSVersion:         "1.32.0.6865",
				PMSPlatform:        "Linux",
				Platform:           "Linux",
				PMSPlatformVersion: "5.15.0",
				PMSIP:              "192.168.1.100",
				PMSPort:            32400,
				PMSURL:             "http://192.168.1.100:32400",
				PMSSSLEnabled:      1,
				PMSIsRemote:        0,
				PMSPlexpass:        1,
				PlexServerUpToDate: 1,
				PMSUpdateAvailable: 0,
			},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var result TautulliServerInfo
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if result.Response.Data.PMSName != original.Response.Data.PMSName {
		t.Error("PMSName not preserved in round-trip")
	}
	if result.Response.Data.PMSPort != original.Response.Data.PMSPort {
		t.Error("PMSPort not preserved in round-trip")
	}
	if result.Response.Data.PMSSSLEnabled != original.Response.Data.PMSSSLEnabled {
		t.Error("PMSSSLEnabled not preserved in round-trip")
	}
}

func TestTautulliServerInfo_SpecialCharacters(t *testing.T) {
	jsonData := `{
		"response": {
			"result": "success",
			"data": {
				"machine_identifier": "id-with-special-chars!@#$",
				"plex_server_name": "Server's \"Name\" & <More>",
				"plex_server_version": "1.32.0.6865",
				"plex_server_platform": "Linux",
				"platform": "Linux",
				"platform_version": "Ubuntu 22.04 (Jammy)",
				"pms_ip": "192.168.1.100",
				"pms_port": 32400,
				"pms_url": "http://server's-url.local:32400",
				"pms_web_url": "https://app.plex.tv/desktop#!/server/abc",
				"pms_ssl": 1,
				"pms_is_remote": 0,
				"pms_is_cloud": 0,
				"pms_plexpass": 1,
				"plex_server_up_to_date": 1,
				"update_available": 0
			}
		}
	}`

	var info TautulliServerInfo
	if err := json.Unmarshal([]byte(jsonData), &info); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if info.Response.Data.PMSName != "Server's \"Name\" & <More>" {
		t.Errorf("Server name with special chars not preserved: %q", info.Response.Data.PMSName)
	}
	if info.Response.Data.PMSPlatformVersion != "Ubuntu 22.04 (Jammy)" {
		t.Errorf("Platform version with parens not preserved: %q", info.Response.Data.PMSPlatformVersion)
	}
}
