// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliServerInfo represents the API response from get_server_info endpoint
type TautulliServerInfo struct {
	Response TautulliServerInfoResponse `json:"response"`
}

type TautulliServerInfoResponse struct {
	Result  string                 `json:"result"`
	Message *string                `json:"message,omitempty"`
	Data    TautulliServerInfoData `json:"data"`
}

type TautulliServerInfoData struct {
	PMSIdentifier        string `json:"machine_identifier"`
	PMSName              string `json:"plex_server_name"`
	PMSVersion           string `json:"plex_server_version"`
	PMSPlatform          string `json:"plex_server_platform"`
	Platform             string `json:"platform"`
	PMSPlatformVersion   string `json:"platform_version"`
	PMSIP                string `json:"pms_ip"`
	PMSPort              int    `json:"pms_port"`
	PMSURL               string `json:"pms_url"`
	PMSWebURL            string `json:"pms_web_url"`
	PMSSSLEnabled        int    `json:"pms_ssl"`
	PMSIsRemote          int    `json:"pms_is_remote"`
	PMSIsCloud           int    `json:"pms_is_cloud"`
	PMSPlexpass          int    `json:"pms_plexpass"`
	PlexServerUpToDate   int    `json:"plex_server_up_to_date"`
	PMSUpdateAvailable   int    `json:"update_available"`
	PMSUpdateVersion     string `json:"update_version,omitempty"`
	PMSUpdateReleaseDate string `json:"update_release_date,omitempty"`
}

// TautulliServerFriendlyName represents the API response from get_server_friendly_name endpoint
type TautulliServerFriendlyName struct {
	Response TautulliServerFriendlyNameResponse `json:"response"`
}

type TautulliServerFriendlyNameResponse struct {
	Result  string                         `json:"result"`
	Message *string                        `json:"message,omitempty"`
	Data    TautulliServerFriendlyNameData `json:"data"`
}

type TautulliServerFriendlyNameData struct {
	FriendlyName string `json:"friendly_name"` // Server display name
}

// TautulliServerID represents the API response from get_server_id endpoint
type TautulliServerID struct {
	Response TautulliServerIDResponse `json:"response"`
}

type TautulliServerIDResponse struct {
	Result  string               `json:"result"`
	Message *string              `json:"message,omitempty"`
	Data    TautulliServerIDData `json:"data"`
}

type TautulliServerIDData struct {
	ServerID string `json:"server_id"` // Unique server identifier (machine ID)
}

// TautulliServerIdentity represents the API response from get_server_identity endpoint
type TautulliServerIdentity struct {
	Response TautulliServerIdentityResponse `json:"response"`
}

type TautulliServerIdentityResponse struct {
	Result  string                     `json:"result"`
	Message *string                    `json:"message,omitempty"`
	Data    TautulliServerIdentityData `json:"data"`
}

type TautulliServerIdentityData struct {
	MachineIdentifier string `json:"machineIdentifier"` // Unique machine ID
	Version           string `json:"version"`           // Plex Media Server version
	Platform          string `json:"platform"`          // OS platform (Linux, Windows, etc.)
	PlatformVersion   string `json:"platformVersion"`   // OS version
	Device            string `json:"device,omitempty"`  // Device type
	CreatedAt         int64  `json:"createdAt,omitempty"`
}

// TautulliTautulliInfo represents the API response from get_tautulli_info endpoint
type TautulliTautulliInfo struct {
	Response TautulliTautulliInfoResponse `json:"response"`
}

type TautulliTautulliInfoResponse struct {
	Result  string                   `json:"result"`
	Message *string                  `json:"message,omitempty"`
	Data    TautulliTautulliInfoData `json:"data"`
}

type TautulliTautulliInfoData struct {
	TautulliVersion     string `json:"tautulli_version"`                // Tautulli version
	TautulliInstallType string `json:"tautulli_install_type"`           // Installation type (git, docker, etc.)
	TautulliBranch      string `json:"tautulli_branch"`                 // Git branch
	TautulliCommit      string `json:"tautulli_commit"`                 // Git commit hash
	Platform            string `json:"platform"`                        // OS platform
	PlatformRelease     string `json:"platform_release"`                // OS release version
	PlatformVersion     string `json:"platform_version"`                // OS version
	PlatformLinuxDistro string `json:"platform_linux_distro,omitempty"` // Linux distribution
	PlatformDeviceName  string `json:"platform_device_name"`            // Device name
	PythonVersion       string `json:"python_version"`                  // Python version
}

// TautulliServerPref represents the API response from get_server_pref endpoint
type TautulliServerPref struct {
	Response TautulliServerPrefResponse `json:"response"`
}

type TautulliServerPrefResponse struct {
	Result  string                 `json:"result"`
	Message *string                `json:"message,omitempty"`
	Data    TautulliServerPrefData `json:"data"`
}

type TautulliServerPrefData struct {
	Pref  string `json:"pref"`  // Preference key
	Value string `json:"value"` // Preference value
}

// TautulliServerList represents the API response from get_server_list endpoint
type TautulliServerList struct {
	Response TautulliServerListResponse `json:"response"`
}

type TautulliServerListResponse struct {
	Result  string                   `json:"result"`
	Message *string                  `json:"message,omitempty"`
	Data    []TautulliServerListItem `json:"data"`
}

type TautulliServerListItem struct {
	MachineIdentifier string `json:"machineIdentifier"` // Unique machine ID
	Name              string `json:"name"`              // Server name
	Version           string `json:"version"`           // PMS version
	Platform          string `json:"platform"`          // OS platform
	PlatformVersion   string `json:"platformVersion"`   // OS version
	CreatedAt         int64  `json:"createdAt"`         // Server creation timestamp
	UpdatedAt         int64  `json:"updatedAt"`         // Last update timestamp
	Owned             int    `json:"owned"`             // Ownership status (0 or 1)
}

// TautulliServersInfo represents the API response from get_servers_info endpoint
type TautulliServersInfo struct {
	Response TautulliServersInfoResponse `json:"response"`
}

type TautulliServersInfoResponse struct {
	Result  string                   `json:"result"`
	Message *string                  `json:"message,omitempty"`
	Data    []TautulliServerListItem `json:"data"` // Reuses TautulliServerListItem struct
}

// TautulliPMSUpdate represents the API response from get_pms_update endpoint
type TautulliPMSUpdate struct {
	Response TautulliPMSUpdateResponse `json:"response"`
}

type TautulliPMSUpdateResponse struct {
	Result  string                `json:"result"`
	Message *string               `json:"message,omitempty"`
	Data    TautulliPMSUpdateData `json:"data"`
}

type TautulliPMSUpdateData struct {
	UpdateAvailable      int    `json:"update_available"`                 // Update availability (0 or 1)
	PlatformVersion      string `json:"platform_version"`                 // Current PMS version
	PlatformVersionLabel string `json:"platform_version_label,omitempty"` // Version label
	Version              string `json:"version,omitempty"`                // New version available
	ReleaseDate          string `json:"release_date,omitempty"`           // Release date
	Changelog            string `json:"changelog,omitempty"`              // Changelog/release notes
	Label                string `json:"label,omitempty"`                  // Release label
	DownloadURL          string `json:"download_url,omitempty"`           // Download URL
	Requirements         string `json:"requirements,omitempty"`           // System requirements
}
