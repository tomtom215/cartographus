// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package models

// Plex REST API Models
// These structures represent responses from Plex Media Server REST API endpoints
// Documentation: https://plexapi.dev and https://www.plexopedia.com/plex-media-server/api/

// ============================================================================
// Bandwidth Statistics Models - GET /statistics/bandwidth
// ============================================================================

// PlexBandwidthResponse represents the response from GET /statistics/bandwidth
// This endpoint provides historical bandwidth usage data for analytics
type PlexBandwidthResponse struct {
	MediaContainer PlexBandwidthContainer `json:"MediaContainer"`
}

// PlexBandwidthContainer wraps bandwidth statistics with related device and account info
type PlexBandwidthContainer struct {
	Size                int                       `json:"size"`                // Total count of bandwidth records
	Device              []PlexBandwidthDevice     `json:"Device,omitempty"`    // Devices that have used bandwidth
	Account             []PlexBandwidthAccount    `json:"Account,omitempty"`   // User accounts with bandwidth usage
	StatisticsBandwidth []PlexStatisticsBandwidth `json:"StatisticsBandwidth"` // Actual bandwidth records
}

// PlexBandwidthDevice represents a device that has consumed bandwidth
type PlexBandwidthDevice struct {
	ID               int    `json:"id"`               // Unique device ID
	Name             string `json:"name"`             // Device name (e.g., "Roku Express", "iPhone")
	Platform         string `json:"platform"`         // Platform (e.g., "Roku", "iOS", "Android")
	ClientIdentifier string `json:"clientIdentifier"` // Unique client identifier
	CreatedAt        int64  `json:"createdAt"`        // Unix timestamp when device was first seen
}

// PlexBandwidthAccount represents a user account with bandwidth usage
type PlexBandwidthAccount struct {
	ID                      int    `json:"id"`                                // Unique account ID
	Key                     string `json:"key"`                               // API path (e.g., "/accounts/12345")
	Name                    string `json:"name"`                              // Account display name
	DefaultAudioLanguage    string `json:"defaultAudioLanguage,omitempty"`    // Preferred audio language
	AutoSelectAudio         bool   `json:"autoSelectAudio,omitempty"`         // Auto-select audio track
	DefaultSubtitleLanguage string `json:"defaultSubtitleLanguage,omitempty"` // Preferred subtitle language
	SubtitleMode            int    `json:"subtitleMode,omitempty"`            // Subtitle display mode
	Thumb                   string `json:"thumb,omitempty"`                   // Avatar URL
}

// PlexStatisticsBandwidth represents a single bandwidth usage record
// Records are typically aggregated by time period (timespan parameter)
type PlexStatisticsBandwidth struct {
	AccountID int   `json:"accountID"` // Reference to PlexBandwidthAccount.ID
	DeviceID  int   `json:"deviceID"`  // Reference to PlexBandwidthDevice.ID
	Timespan  int   `json:"timespan"`  // Time aggregation period (seconds)
	At        int64 `json:"at"`        // Unix timestamp for this record
	LAN       bool  `json:"lan"`       // True if local network, false if remote
	Bytes     int64 `json:"bytes"`     // Bandwidth consumed in bytes
}

// ============================================================================
// Library Sections Models - GET /library/sections
// ============================================================================

// PlexLibrarySectionsResponse represents the response from GET /library/sections
// This endpoint lists all library sections (Movies, TV Shows, Music, etc.)
type PlexLibrarySectionsResponse struct {
	MediaContainer PlexLibrarySectionsContainer `json:"MediaContainer"`
}

// PlexLibrarySectionsContainer wraps the list of library sections
type PlexLibrarySectionsContainer struct {
	Size             int                  `json:"size"`                       // Number of sections
	AllowSync        bool                 `json:"allowSync,omitempty"`        // Whether sync is allowed
	Title1           string               `json:"title1,omitempty"`           // Container title
	Directory        []PlexLibrarySection `json:"Directory,omitempty"`        // Library sections
	LibrarySectionID int                  `json:"librarySectionID,omitempty"` // Current section ID (if in section context)
}

// PlexLibrarySection represents a single library section (Movies, TV Shows, etc.)
type PlexLibrarySection struct {
	// Identification
	Key      string `json:"key"`      // Section key/ID (used in URLs like /library/sections/{key})
	UUID     string `json:"uuid"`     // Unique section UUID
	Title    string `json:"title"`    // Section name (e.g., "Movies", "TV Shows")
	Type     string `json:"type"`     // Section type: "movie", "show", "artist", "photo"
	Agent    string `json:"agent"`    // Metadata agent (e.g., "tv.plex.agents.movie")
	Scanner  string `json:"scanner"`  // Scanner type (e.g., "Plex Movie")
	Language string `json:"language"` // Primary language (e.g., "en-US")

	// Display settings
	Thumb     string `json:"thumb,omitempty"`     // Section thumbnail
	Art       string `json:"art,omitempty"`       // Section artwork
	Composite string `json:"composite,omitempty"` // Composite image URL

	// Configuration
	Filters            bool `json:"filters,omitempty"`            // Has custom filters
	Refreshing         bool `json:"refreshing,omitempty"`         // Currently scanning
	Hidden             int  `json:"hidden,omitempty"`             // Hidden from UI (0 or 1)
	EnableAutoPhotoTag bool `json:"enableAutoPhotoTag,omitempty"` // Auto photo tagging enabled

	// Timestamps
	CreatedAt        int64 `json:"createdAt,omitempty"`        // Unix timestamp when created
	UpdatedAt        int64 `json:"updatedAt,omitempty"`        // Unix timestamp when last updated
	ScannedAt        int64 `json:"scannedAt,omitempty"`        // Unix timestamp of last scan
	ContentChangedAt int64 `json:"contentChangedAt,omitempty"` // Unix timestamp of last content change

	// Locations (library paths)
	Location []PlexLibraryLocation `json:"Location,omitempty"` // Storage locations for this section
}

// PlexLibraryLocation represents a storage location for a library section
type PlexLibraryLocation struct {
	ID   int    `json:"id"`   // Location ID
	Path string `json:"path"` // File system path
}

// ============================================================================
// Library Section Content Models - GET /library/sections/{id}/all
// ============================================================================

// PlexLibrarySectionContentResponse represents the response from GET /library/sections/{id}/all
// or GET /library/sections/{id}/recentlyAdded
type PlexLibrarySectionContentResponse struct {
	MediaContainer PlexLibrarySectionContentContainer `json:"MediaContainer"`
}

// PlexLibrarySectionContentContainer wraps library content items
type PlexLibrarySectionContentContainer struct {
	Size                int                   `json:"size"`                          // Number of items returned
	TotalSize           int                   `json:"totalSize,omitempty"`           // Total items in section
	Offset              int                   `json:"offset,omitempty"`              // Pagination offset
	AllowSync           bool                  `json:"allowSync,omitempty"`           // Sync allowed
	Art                 string                `json:"art,omitempty"`                 // Section art
	Identifier          string                `json:"identifier,omitempty"`          // Section identifier
	LibrarySectionID    int                   `json:"librarySectionID,omitempty"`    // Section ID
	LibrarySectionTitle string                `json:"librarySectionTitle,omitempty"` // Section title
	LibrarySectionUUID  string                `json:"librarySectionUUID,omitempty"`  // Section UUID
	MediaTagPrefix      string                `json:"mediaTagPrefix,omitempty"`      // Media tag URL prefix
	MediaTagVersion     int                   `json:"mediaTagVersion,omitempty"`     // Media tag version
	Thumb               string                `json:"thumb,omitempty"`               // Section thumbnail
	Title1              string                `json:"title1,omitempty"`              // Container title
	Title2              string                `json:"title2,omitempty"`              // Container subtitle
	ViewGroup           string                `json:"viewGroup,omitempty"`           // View grouping
	ViewMode            int                   `json:"viewMode,omitempty"`            // View mode
	Metadata            []PlexLibraryMetadata `json:"Metadata,omitempty"`            // Content items
}

// PlexLibraryMetadata represents a media item in a library section
type PlexLibraryMetadata struct {
	// Identification
	RatingKey            string `json:"ratingKey"`                      // Unique item identifier
	Key                  string `json:"key"`                            // API path to item
	ParentRatingKey      string `json:"parentRatingKey,omitempty"`      // Parent item key (season for episode)
	GrandparentRatingKey string `json:"grandparentRatingKey,omitempty"` // Grandparent key (show for episode)
	GUID                 string `json:"guid,omitempty"`                 // External identifiers
	Studio               string `json:"studio,omitempty"`               // Production studio
	Type                 string `json:"type"`                           // Item type: movie, show, season, episode

	// Titles
	Title            string `json:"title"`                      // Item title
	TitleSort        string `json:"titleSort,omitempty"`        // Sort title
	OriginalTitle    string `json:"originalTitle,omitempty"`    // Original language title
	ParentTitle      string `json:"parentTitle,omitempty"`      // Parent title (season name)
	GrandparentTitle string `json:"grandparentTitle,omitempty"` // Grandparent title (show name)
	ContentRating    string `json:"contentRating,omitempty"`    // Age rating (PG-13, TV-MA)
	Summary          string `json:"summary,omitempty"`          // Item description
	Tagline          string `json:"tagline,omitempty"`          // Movie tagline

	// Ratings
	Rating         float64 `json:"rating,omitempty"`         // Plex rating
	AudienceRating float64 `json:"audienceRating,omitempty"` // Audience rating
	UserRating     float64 `json:"userRating,omitempty"`     // User's personal rating
	RatingImage    string  `json:"ratingImage,omitempty"`    // Rating badge image

	// Display
	Thumb            string `json:"thumb,omitempty"`            // Item thumbnail
	Art              string `json:"art,omitempty"`              // Item artwork
	Banner           string `json:"banner,omitempty"`           // Banner image
	ParentThumb      string `json:"parentThumb,omitempty"`      // Parent thumbnail
	GrandparentThumb string `json:"grandparentThumb,omitempty"` // Grandparent thumbnail
	GrandparentArt   string `json:"grandparentArt,omitempty"`   // Grandparent artwork

	// Episode/Season info
	Index           int `json:"index,omitempty"`           // Episode number
	ParentIndex     int `json:"parentIndex,omitempty"`     // Season number
	LeafCount       int `json:"leafCount,omitempty"`       // Number of episodes (for seasons/shows)
	ViewedLeafCount int `json:"viewedLeafCount,omitempty"` // Number of watched episodes
	ChildCount      int `json:"childCount,omitempty"`      // Number of children

	// Timestamps
	Year                  int    `json:"year,omitempty"`                  // Release year
	OriginallyAvailableAt string `json:"originallyAvailableAt,omitempty"` // Release date (YYYY-MM-DD)
	AddedAt               int64  `json:"addedAt,omitempty"`               // Unix timestamp when added
	UpdatedAt             int64  `json:"updatedAt,omitempty"`             // Unix timestamp when updated
	LastViewedAt          int64  `json:"lastViewedAt,omitempty"`          // Unix timestamp of last view

	// Media info
	Duration   int64 `json:"duration,omitempty"`   // Duration in milliseconds
	ViewOffset int64 `json:"viewOffset,omitempty"` // Current view position
	ViewCount  int   `json:"viewCount,omitempty"`  // Number of views

	// Detailed media (optional, may need to request separately)
	Media []PlexLibraryMedia `json:"Media,omitempty"` // Media versions
}

// PlexLibraryMedia represents a media version of a library item
type PlexLibraryMedia struct {
	ID              int     `json:"id"`                        // Media ID
	Duration        int64   `json:"duration,omitempty"`        // Duration in milliseconds
	Bitrate         int     `json:"bitrate,omitempty"`         // Total bitrate (kbps)
	Width           int     `json:"width,omitempty"`           // Video width
	Height          int     `json:"height,omitempty"`          // Video height
	AspectRatio     float64 `json:"aspectRatio,omitempty"`     // Aspect ratio
	AudioChannels   int     `json:"audioChannels,omitempty"`   // Audio channel count
	AudioCodec      string  `json:"audioCodec,omitempty"`      // Audio codec
	VideoCodec      string  `json:"videoCodec,omitempty"`      // Video codec
	VideoResolution string  `json:"videoResolution,omitempty"` // Resolution (720, 1080, 4k)
	Container       string  `json:"container,omitempty"`       // Container format
	VideoFrameRate  string  `json:"videoFrameRate,omitempty"`  // Frame rate
	VideoProfile    string  `json:"videoProfile,omitempty"`    // Video profile

	Part []PlexLibraryMediaPart `json:"Part,omitempty"` // Media file parts
}

// PlexLibraryMediaPart represents a file part of a media item
type PlexLibraryMediaPart struct {
	ID           int    `json:"id"`                     // Part ID
	Key          string `json:"key,omitempty"`          // Download key
	Duration     int64  `json:"duration,omitempty"`     // Duration in milliseconds
	File         string `json:"file,omitempty"`         // File path
	Size         int64  `json:"size,omitempty"`         // File size in bytes
	Container    string `json:"container,omitempty"`    // Container format
	VideoProfile string `json:"videoProfile,omitempty"` // Video profile
}

// ============================================================================
// Server Activities Models - GET /activities
// ============================================================================

// PlexActivitiesResponse represents the response from GET /activities
// This endpoint shows background tasks like library scans, optimizations, etc.
type PlexActivitiesResponse struct {
	MediaContainer PlexActivitiesContainer `json:"MediaContainer"`
}

// PlexActivitiesContainer wraps the list of server activities
type PlexActivitiesContainer struct {
	Size     int            `json:"size"`               // Number of activities
	Activity []PlexActivity `json:"Activity,omitempty"` // Active server tasks
}

// PlexActivity represents a background server task
// Examples: library scans, database optimization, media analysis
type PlexActivity struct {
	UUID        string                 `json:"uuid"`                  // Unique activity identifier
	Type        string                 `json:"type"`                  // Activity type (e.g., "library.refresh.items", "library.scan")
	Title       string                 `json:"title"`                 // User-friendly title
	Subtitle    string                 `json:"subtitle,omitempty"`    // Additional description
	Progress    int                    `json:"progress"`              // Progress percentage (0-100, -1 for indeterminate)
	UserID      int                    `json:"userID,omitempty"`      // User who initiated the activity
	Cancellable bool                   `json:"cancellable,omitempty"` // Can be canceled by user
	Context     map[string]interface{} `json:"Context,omitempty"`     // Additional context (flexible)
	Response    map[string]interface{} `json:"Response,omitempty"`    // Async operation result (flexible)
}

// ============================================================================
// Helper Methods
// ============================================================================

// IsLANBandwidth returns true if this record represents LAN traffic
func (b *PlexStatisticsBandwidth) IsLANBandwidth() bool {
	return b.LAN
}

// GetBandwidthMB returns bandwidth in megabytes
func (b *PlexStatisticsBandwidth) GetBandwidthMB() float64 {
	return float64(b.Bytes) / (1024 * 1024)
}

// GetBandwidthGB returns bandwidth in gigabytes
func (b *PlexStatisticsBandwidth) GetBandwidthGB() float64 {
	return float64(b.Bytes) / (1024 * 1024 * 1024)
}

// IsRefreshing returns true if the library section is currently being scanned
func (s *PlexLibrarySection) IsRefreshing() bool {
	return s.Refreshing
}

// IsHidden returns true if the library section is hidden from UI
func (s *PlexLibrarySection) IsHidden() bool {
	return s.Hidden != 0
}

// IsMovie returns true if this is a movie library section
func (s *PlexLibrarySection) IsMovie() bool {
	return s.Type == "movie"
}

// IsTV returns true if this is a TV show library section
func (s *PlexLibrarySection) IsTV() bool {
	return s.Type == "show"
}

// IsMusic returns true if this is a music library section
func (s *PlexLibrarySection) IsMusic() bool {
	return s.Type == "artist"
}

// IsPhoto returns true if this is a photo library section
func (s *PlexLibrarySection) IsPhoto() bool {
	return s.Type == "photo"
}

// IsInProgress returns true if the activity is still running
func (a *PlexActivity) IsInProgress() bool {
	return a.Progress >= 0 && a.Progress < 100
}

// IsIndeterminate returns true if the activity has no known progress
func (a *PlexActivity) IsIndeterminate() bool {
	return a.Progress == -1
}

// IsComplete returns true if the activity has completed
func (a *PlexActivity) IsComplete() bool {
	return a.Progress == 100
}

// GetLibrarySectionID extracts the library section ID from activity context if available
func (a *PlexActivity) GetLibrarySectionID() string {
	if a.Context == nil {
		return ""
	}
	if sectionID, ok := a.Context["librarySectionID"].(string); ok {
		return sectionID
	}
	return ""
}
