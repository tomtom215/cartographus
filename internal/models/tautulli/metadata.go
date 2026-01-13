// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliMetadata represents the API response from Tautulli's get_metadata endpoint
type TautulliMetadata struct {
	Response TautulliMetadataResponse `json:"response"`
}

type TautulliMetadataResponse struct {
	Result  string               `json:"result"`
	Message *string              `json:"message,omitempty"`
	Data    TautulliMetadataData `json:"data"`
}

type TautulliMetadataData struct {
	RatingKey             string              `json:"rating_key"`
	ParentRatingKey       string              `json:"parent_rating_key"`
	GrandparentRatingKey  string              `json:"grandparent_rating_key"`
	Title                 string              `json:"title"`
	ParentTitle           string              `json:"parent_title"`
	GrandparentTitle      string              `json:"grandparent_title"`
	OriginalTitle         string              `json:"original_title"`
	SortTitle             string              `json:"sort_title"`
	MediaIndex            int                 `json:"media_index"`
	ParentMediaIndex      int                 `json:"parent_media_index"`
	Studio                string              `json:"studio"`
	ContentRating         string              `json:"content_rating"`
	Summary               string              `json:"summary"`
	Tagline               string              `json:"tagline"`
	Rating                float64             `json:"rating"`
	RatingImage           string              `json:"rating_image"`
	AudienceRating        float64             `json:"audience_rating"`
	AudienceRatingImage   string              `json:"audience_rating_image"`
	UserRating            float64             `json:"user_rating"`
	Duration              int                 `json:"duration"`
	Year                  int                 `json:"year"`
	Thumb                 string              `json:"thumb"`
	ParentThumb           string              `json:"parent_thumb"`
	GrandparentThumb      string              `json:"grandparent_thumb"`
	Art                   string              `json:"art"`
	Banner                string              `json:"banner"`
	OriginallyAvailableAt string              `json:"originally_available_at"`
	AddedAt               int64               `json:"added_at"`
	UpdatedAt             int64               `json:"updated_at"`
	LastViewedAt          int64               `json:"last_viewed_at"`
	GUID                  string              `json:"guid"`
	Guids                 []string            `json:"guids"`
	Directors             []string            `json:"directors"`
	Writers               []string            `json:"writers"`
	Actors                []string            `json:"actors"`
	Genres                []string            `json:"genres"`
	Labels                []string            `json:"labels"`
	Collections           []string            `json:"collections"`
	MediaInfo             []TautulliMediaInfo `json:"media_info"`
}

type TautulliMediaInfo struct {
	ID                 int     `json:"id"`
	Container          string  `json:"container"`
	Bitrate            int     `json:"bitrate"`
	Height             int     `json:"height"`
	Width              int     `json:"width"`
	AspectRatio        float64 `json:"aspect_ratio"`
	VideoCodec         string  `json:"video_codec"`
	VideoResolution    string  `json:"video_resolution"`
	VideoFramerate     string  `json:"video_framerate"`
	VideoBitDepth      int     `json:"video_bit_depth"`
	VideoProfile       string  `json:"video_profile"`
	AudioCodec         string  `json:"audio_codec"`
	AudioChannels      int     `json:"audio_channels"`
	AudioChannelLayout string  `json:"audio_channel_layout"`
	AudioBitrate       int     `json:"audio_bitrate"`
	OptimizedVersion   int     `json:"optimized_version"`
}

// TautulliChildrenMetadata represents the API response from Tautulli's get_children_metadata endpoint
type TautulliChildrenMetadata struct {
	Response TautulliChildrenMetadataResponse `json:"response"`
}

type TautulliChildrenMetadataResponse struct {
	Result  string                       `json:"result"`
	Message *string                      `json:"message,omitempty"`
	Data    TautulliChildrenMetadataData `json:"data"`
}

type TautulliChildrenMetadataData struct {
	ChildrenCount int                             `json:"children_count"`
	ChildrenList  []TautulliChildrenMetadataChild `json:"children_list"`
}

type TautulliChildrenMetadataChild struct {
	MediaType            string `json:"media_type"`
	SectionID            int    `json:"section_id"`
	LibraryName          string `json:"library_name,omitempty"`
	RatingKey            string `json:"rating_key"`
	ParentRatingKey      string `json:"parent_rating_key,omitempty"`
	GrandparentRatingKey string `json:"grandparent_rating_key,omitempty"`
	Title                string `json:"title"`
	ParentTitle          string `json:"parent_title,omitempty"`
	GrandparentTitle     string `json:"grandparent_title,omitempty"`
	OriginalTitle        string `json:"original_title,omitempty"`
	SortTitle            string `json:"sort_title,omitempty"`
	MediaIndex           int    `json:"media_index,omitempty"`
	ParentMediaIndex     int    `json:"parent_media_index,omitempty"`
	Year                 int    `json:"year,omitempty"`
	Thumb                string `json:"thumb,omitempty"`
	ParentThumb          string `json:"parent_thumb,omitempty"`
	GrandparentThumb     string `json:"grandparent_thumb,omitempty"`
	AddedAt              int64  `json:"added_at,omitempty"`
	UpdatedAt            int64  `json:"updated_at,omitempty"`
	LastViewedAt         int64  `json:"last_viewed_at,omitempty"`
	GUID                 string `json:"guid,omitempty"`
	ParentGUID           string `json:"parent_guid,omitempty"`
	GrandparentGUID      string `json:"grandparent_guid,omitempty"`
}
