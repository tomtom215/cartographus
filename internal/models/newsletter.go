// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package models provides data structures for the Cartographus application.
//
// newsletter.go - Newsletter and Digest Generator Models
//
// This file contains models for the Newsletter Generator system, which provides
// Tautulli-parity (and beyond) functionality for creating and distributing
// periodic media digests.
//
// Feature Set:
//   - Multiple newsletter types: Recently Added, Weekly Digest, User Activity, etc.
//   - Multiple delivery channels: Email, Discord, Slack, Telegram, Webhooks
//   - Template-based customization with variable substitution
//   - Cron-based scheduling with timezone support
//   - RBAC integration for template and schedule management
//   - Delivery analytics and history tracking
//   - User preferences for opt-in/opt-out
//   - A/B testing for template optimization
//
// Security:
//   - Templates are sanitized to prevent XSS
//   - SMTP credentials are encrypted at rest
//   - Webhook URLs are validated
//   - RBAC enforces who can create/edit templates and schedules
package models

import (
	"time"
)

// ============================================================================
// Newsletter Types and Constants
// ============================================================================

// NewsletterType defines the type of newsletter content.
type NewsletterType string

const (
	// NewsletterTypeRecentlyAdded includes newly added content (Tautulli parity).
	NewsletterTypeRecentlyAdded NewsletterType = "recently_added"

	// NewsletterTypeWeeklyDigest includes weekly viewing statistics and highlights.
	NewsletterTypeWeeklyDigest NewsletterType = "weekly_digest"

	// NewsletterTypeMonthlyStats includes monthly analytics summary.
	NewsletterTypeMonthlyStats NewsletterType = "monthly_stats"

	// NewsletterTypeUserActivity includes personalized user activity summary.
	NewsletterTypeUserActivity NewsletterType = "user_activity"

	// NewsletterTypeRecommendations includes personalized content recommendations.
	NewsletterTypeRecommendations NewsletterType = "recommendations"

	// NewsletterTypeServerHealth includes server status and health metrics.
	NewsletterTypeServerHealth NewsletterType = "server_health"

	// NewsletterTypeCustom allows fully custom template-based newsletters.
	NewsletterTypeCustom NewsletterType = "custom"
)

// ValidNewsletterTypes contains all valid newsletter types.
var ValidNewsletterTypes = []NewsletterType{
	NewsletterTypeRecentlyAdded,
	NewsletterTypeWeeklyDigest,
	NewsletterTypeMonthlyStats,
	NewsletterTypeUserActivity,
	NewsletterTypeRecommendations,
	NewsletterTypeServerHealth,
	NewsletterTypeCustom,
}

// IsValidNewsletterType checks if a newsletter type is valid.
func IsValidNewsletterType(t NewsletterType) bool {
	for _, valid := range ValidNewsletterTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// DeliveryChannel defines the delivery method for newsletters.
type DeliveryChannel string

const (
	// DeliveryChannelEmail sends newsletters via SMTP email.
	DeliveryChannelEmail DeliveryChannel = "email"

	// DeliveryChannelDiscord sends newsletters via Discord webhook.
	DeliveryChannelDiscord DeliveryChannel = "discord"

	// DeliveryChannelSlack sends newsletters via Slack webhook.
	DeliveryChannelSlack DeliveryChannel = "slack"

	// DeliveryChannelTelegram sends newsletters via Telegram bot.
	DeliveryChannelTelegram DeliveryChannel = "telegram"

	// DeliveryChannelWebhook sends newsletters via generic HTTP webhook.
	DeliveryChannelWebhook DeliveryChannel = "webhook"

	// DeliveryChannelInApp sends newsletters as in-app notifications.
	DeliveryChannelInApp DeliveryChannel = "in_app"
)

// ValidDeliveryChannels contains all valid delivery channels.
var ValidDeliveryChannels = []DeliveryChannel{
	DeliveryChannelEmail,
	DeliveryChannelDiscord,
	DeliveryChannelSlack,
	DeliveryChannelTelegram,
	DeliveryChannelWebhook,
	DeliveryChannelInApp,
}

// IsValidDeliveryChannel checks if a delivery channel is valid.
func IsValidDeliveryChannel(c DeliveryChannel) bool {
	for _, valid := range ValidDeliveryChannels {
		if c == valid {
			return true
		}
	}
	return false
}

// DeliveryStatus represents the delivery status of a newsletter.
type DeliveryStatus string

const (
	// DeliveryStatusPending indicates the delivery is scheduled but not yet sent.
	DeliveryStatusPending DeliveryStatus = "pending"

	// DeliveryStatusSending indicates the delivery is currently in progress.
	DeliveryStatusSending DeliveryStatus = "sending"

	// DeliveryStatusDelivered indicates successful delivery.
	DeliveryStatusDelivered DeliveryStatus = "delivered"

	// DeliveryStatusFailed indicates delivery failure.
	DeliveryStatusFailed DeliveryStatus = "failed"

	// DeliveryStatusPartial indicates partial delivery (some recipients failed).
	DeliveryStatusPartial DeliveryStatus = "partial"

	// DeliveryStatusCanceled indicates the delivery was canceled.
	DeliveryStatusCanceled DeliveryStatus = "canceled"
)

// TimeFrameUnit defines the unit for time-based content selection.
type TimeFrameUnit string

const (
	TimeFrameUnitHours  TimeFrameUnit = "hours"
	TimeFrameUnitDays   TimeFrameUnit = "days"
	TimeFrameUnitWeeks  TimeFrameUnit = "weeks"
	TimeFrameUnitMonths TimeFrameUnit = "months"
)

// ============================================================================
// Newsletter Template
// ============================================================================

// NewsletterTemplate represents a reusable template for newsletter content.
// Templates support HTML/plaintext with variable substitution.
//
// Key Features:
//   - HTML and plaintext body support
//   - Variable substitution (e.g., {{.ServerName}}, {{.NewMovies}})
//   - Version control for template changes
//   - RBAC: Only editors/admins can create/modify templates
type NewsletterTemplate struct {
	// ID is the unique template identifier.
	ID string `json:"id"`

	// Name is the human-readable template name.
	Name string `json:"name" validate:"required,min=1,max=100"`

	// Description provides details about the template purpose.
	Description string `json:"description,omitempty" validate:"max=500"`

	// Type is the newsletter type this template supports.
	Type NewsletterType `json:"type" validate:"required"`

	// Subject is the email/notification subject line with variable support.
	Subject string `json:"subject" validate:"required,min=1,max=200"`

	// BodyHTML is the HTML content with variable support.
	BodyHTML string `json:"body_html" validate:"required"`

	// BodyText is the plaintext fallback content.
	BodyText string `json:"body_text,omitempty"`

	// Variables lists the available template variables with descriptions.
	Variables []TemplateVariable `json:"variables,omitempty"`

	// DefaultConfig provides default values for template-specific settings.
	DefaultConfig *TemplateConfig `json:"default_config,omitempty"`

	// Version is incremented on each update for audit tracking.
	Version int `json:"version"`

	// IsBuiltIn indicates if this is a system-provided template.
	IsBuiltIn bool `json:"is_built_in"`

	// IsActive indicates if the template is available for use.
	IsActive bool `json:"is_active"`

	// CreatedBy is the user ID who created the template.
	CreatedBy string `json:"created_by"`

	// CreatedAt is when the template was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedBy is the user ID who last updated the template.
	UpdatedBy string `json:"updated_by,omitempty"`

	// UpdatedAt is when the template was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// TemplateVariable describes a variable available in a template.
type TemplateVariable struct {
	// Name is the variable name (e.g., "ServerName").
	Name string `json:"name"`

	// Description explains what the variable contains.
	Description string `json:"description"`

	// Type is the data type (string, number, array, object).
	Type string `json:"type"`

	// Example shows a sample value.
	Example string `json:"example,omitempty"`

	// Required indicates if the variable must be available.
	Required bool `json:"required"`
}

// TemplateConfig provides configurable options for newsletter content generation.
type TemplateConfig struct {
	// TimeFrame is the lookback period for content (e.g., 7 for "last 7 days").
	TimeFrame int `json:"time_frame" validate:"min=1,max=365"`

	// TimeFrameUnit is the unit for TimeFrame (hours, days, weeks, months).
	TimeFrameUnit TimeFrameUnit `json:"time_frame_unit"`

	// IncludeMovies determines if movies are included.
	IncludeMovies bool `json:"include_movies"`

	// IncludeShows determines if TV shows are included.
	IncludeShows bool `json:"include_shows"`

	// IncludeMusic determines if music is included.
	IncludeMusic bool `json:"include_music"`

	// LibraryFilter limits content to specific library IDs (empty = all).
	LibraryFilter []string `json:"library_filter,omitempty"`

	// MaxItems limits the number of items per category.
	MaxItems int `json:"max_items" validate:"min=1,max=100"`

	// IncludePosterImages determines if poster images are embedded/linked.
	IncludePosterImages bool `json:"include_poster_images"`

	// ImageHosting determines how images are served (self_hosted, imgur, omit).
	ImageHosting string `json:"image_hosting"`

	// IncludeStats determines if statistics are included.
	IncludeStats bool `json:"include_stats"`

	// IncludeTopContent determines if top content rankings are included.
	IncludeTopContent bool `json:"include_top_content"`

	// PersonalizeForUser determines if content is personalized per-recipient.
	PersonalizeForUser bool `json:"personalize_for_user"`
}

// ============================================================================
// Newsletter Schedule
// ============================================================================

// NewsletterSchedule represents a scheduled newsletter delivery.
// Schedules link templates to recipients with cron-based timing.
//
// Key Features:
//   - Cron expression for flexible scheduling
//   - Multiple recipient types (users, emails, webhooks)
//   - Timezone-aware scheduling
//   - RBAC: Only editors/admins can create/modify schedules
type NewsletterSchedule struct {
	// ID is the unique schedule identifier.
	ID string `json:"id"`

	// Name is the human-readable schedule name.
	Name string `json:"name" validate:"required,min=1,max=100"`

	// Description provides details about the schedule.
	Description string `json:"description,omitempty" validate:"max=500"`

	// TemplateID references the template to use.
	TemplateID string `json:"template_id" validate:"required"`

	// TemplateName is the template name (populated on read).
	TemplateName string `json:"template_name,omitempty"`

	// Recipients is the list of recipients for this schedule.
	Recipients []NewsletterRecipient `json:"recipients" validate:"required,min=1"`

	// CronExpression defines when the newsletter is sent (e.g., "0 9 * * 1").
	CronExpression string `json:"cron_expression" validate:"required"`

	// Timezone is the timezone for cron evaluation (e.g., "America/New_York").
	Timezone string `json:"timezone" validate:"required"`

	// Config overrides the template's default configuration.
	Config *TemplateConfig `json:"config,omitempty"`

	// Channels is the list of delivery channels to use.
	Channels []DeliveryChannel `json:"channels" validate:"required,min=1"`

	// ChannelConfigs provides channel-specific configuration.
	ChannelConfigs map[DeliveryChannel]*ChannelConfig `json:"channel_configs,omitempty"`

	// IsEnabled indicates if the schedule is active.
	IsEnabled bool `json:"is_enabled"`

	// LastRunAt is when the schedule was last executed.
	LastRunAt *time.Time `json:"last_run_at,omitempty"`

	// NextRunAt is when the schedule will next execute.
	NextRunAt *time.Time `json:"next_run_at,omitempty"`

	// LastRunStatus is the status of the last execution.
	LastRunStatus DeliveryStatus `json:"last_run_status,omitempty"`

	// RunCount is the total number of executions.
	RunCount int `json:"run_count"`

	// SuccessCount is the number of successful deliveries.
	SuccessCount int `json:"success_count"`

	// FailureCount is the number of failed deliveries.
	FailureCount int `json:"failure_count"`

	// CreatedBy is the user ID who created the schedule.
	CreatedBy string `json:"created_by"`

	// CreatedAt is when the schedule was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedBy is the user ID who last updated the schedule.
	UpdatedBy string `json:"updated_by,omitempty"`

	// UpdatedAt is when the schedule was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// NewsletterRecipient represents a recipient for newsletters.
type NewsletterRecipient struct {
	// Type is the recipient type (user, email, webhook).
	Type string `json:"type" validate:"required,oneof=user email webhook"`

	// Target is the recipient identifier (user_id, email address, or webhook URL).
	Target string `json:"target" validate:"required"`

	// Name is an optional display name for the recipient.
	Name string `json:"name,omitempty"`

	// Preferences contains recipient-specific preferences.
	Preferences *RecipientPreferences `json:"preferences,omitempty"`
}

// RecipientPreferences defines per-recipient newsletter preferences.
type RecipientPreferences struct {
	// OptedOut indicates the recipient has opted out of this newsletter.
	OptedOut bool `json:"opted_out"`

	// OptedOutAt is when the recipient opted out.
	OptedOutAt *time.Time `json:"opted_out_at,omitempty"`

	// Frequency limits how often the recipient receives newsletters.
	// Values: "all", "weekly", "monthly", "never"
	Frequency string `json:"frequency,omitempty"`

	// PreferredChannel overrides the schedule's delivery channel.
	PreferredChannel *DeliveryChannel `json:"preferred_channel,omitempty"`

	// IncludeSpoilers determines if spoiler content is included.
	IncludeSpoilers bool `json:"include_spoilers"`

	// Language overrides the newsletter language.
	Language string `json:"language,omitempty"`
}

// ChannelConfig provides channel-specific delivery configuration.
type ChannelConfig struct {
	// Email configuration
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUser     string `json:"smtp_user,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"` // Encrypted at rest
	SMTPFrom     string `json:"smtp_from,omitempty"`
	SMTPFromName string `json:"smtp_from_name,omitempty"`
	UseTLS       bool   `json:"use_tls,omitempty"`

	// Discord configuration
	DiscordWebhookURL string `json:"discord_webhook_url,omitempty"`
	DiscordUsername   string `json:"discord_username,omitempty"`
	DiscordAvatarURL  string `json:"discord_avatar_url,omitempty"`

	// Slack configuration
	SlackWebhookURL string `json:"slack_webhook_url,omitempty"`
	SlackChannel    string `json:"slack_channel,omitempty"`
	SlackUsername   string `json:"slack_username,omitempty"`
	SlackIconEmoji  string `json:"slack_icon_emoji,omitempty"`

	// Telegram configuration
	TelegramBotToken  string `json:"telegram_bot_token,omitempty"`
	TelegramChatID    string `json:"telegram_chat_id,omitempty"`
	TelegramParseMode string `json:"telegram_parse_mode,omitempty"` // HTML, Markdown, MarkdownV2

	// Generic webhook configuration
	WebhookURL     string            `json:"webhook_url,omitempty"`
	WebhookMethod  string            `json:"webhook_method,omitempty"` // POST, PUT
	WebhookHeaders map[string]string `json:"webhook_headers,omitempty"`
	WebhookAuth    string            `json:"webhook_auth,omitempty"` // Basic auth header value
}

// ============================================================================
// Newsletter Delivery History
// ============================================================================

// NewsletterDelivery represents a single newsletter delivery attempt.
// This provides audit trail and delivery analytics.
type NewsletterDelivery struct {
	// ID is the unique delivery identifier.
	ID string `json:"id"`

	// ScheduleID references the schedule that triggered this delivery.
	ScheduleID string `json:"schedule_id"`

	// ScheduleName is the schedule name (populated on read).
	ScheduleName string `json:"schedule_name,omitempty"`

	// TemplateID references the template used.
	TemplateID string `json:"template_id"`

	// TemplateName is the template name (populated on read).
	TemplateName string `json:"template_name,omitempty"`

	// TemplateVersion is the template version at time of delivery.
	TemplateVersion int `json:"template_version"`

	// Channel is the delivery channel used.
	Channel DeliveryChannel `json:"channel"`

	// Status is the overall delivery status.
	Status DeliveryStatus `json:"status"`

	// RecipientsTotal is the total number of recipients.
	RecipientsTotal int `json:"recipients_total"`

	// RecipientsDelivered is the number of successful deliveries.
	RecipientsDelivered int `json:"recipients_delivered"`

	// RecipientsFailed is the number of failed deliveries.
	RecipientsFailed int `json:"recipients_failed"`

	// RecipientDetails contains per-recipient delivery results.
	RecipientDetails []RecipientDeliveryResult `json:"recipient_details,omitempty"`

	// ContentSummary provides a brief description of content included.
	ContentSummary string `json:"content_summary,omitempty"`

	// ContentStats contains content statistics for this delivery.
	ContentStats *DeliveryContentStats `json:"content_stats,omitempty"`

	// RenderedSubject is the final subject after variable substitution.
	RenderedSubject string `json:"rendered_subject,omitempty"`

	// RenderedBodySize is the size of the rendered content in bytes.
	RenderedBodySize int `json:"rendered_body_size,omitempty"`

	// StartedAt is when the delivery started.
	StartedAt time.Time `json:"started_at"`

	// CompletedAt is when the delivery completed (or failed).
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// DurationMS is the total delivery duration in milliseconds.
	DurationMS int64 `json:"duration_ms,omitempty"`

	// ErrorMessage contains any error details.
	ErrorMessage string `json:"error_message,omitempty"`

	// ErrorDetails contains additional error context.
	ErrorDetails map[string]interface{} `json:"error_details,omitempty"`

	// TriggeredBy indicates what triggered the delivery (schedule, manual, api).
	TriggeredBy string `json:"triggered_by"`

	// TriggeredByUserID is the user who triggered manual deliveries.
	TriggeredByUserID string `json:"triggered_by_user_id,omitempty"`
}

// RecipientDeliveryResult contains the delivery result for a single recipient.
type RecipientDeliveryResult struct {
	// Recipient is the recipient identifier.
	Recipient string `json:"recipient"`

	// RecipientType is the type (user, email, webhook).
	RecipientType string `json:"recipient_type"`

	// Status is the delivery status for this recipient.
	Status DeliveryStatus `json:"status"`

	// DeliveredAt is when delivery succeeded.
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`

	// ErrorMessage contains any error for this recipient.
	ErrorMessage string `json:"error_message,omitempty"`

	// RetryCount is the number of retry attempts.
	RetryCount int `json:"retry_count"`
}

// DeliveryContentStats contains statistics about delivered content.
type DeliveryContentStats struct {
	// NewMoviesCount is the number of new movies included.
	NewMoviesCount int `json:"new_movies_count"`

	// NewShowsCount is the number of new TV shows included.
	NewShowsCount int `json:"new_shows_count"`

	// NewEpisodesCount is the number of new episodes included.
	NewEpisodesCount int `json:"new_episodes_count"`

	// NewMusicCount is the number of new music items included.
	NewMusicCount int `json:"new_music_count"`

	// TopContentCount is the number of top content items included.
	TopContentCount int `json:"top_content_count"`

	// DateRangeStart is the start of the content date range.
	DateRangeStart time.Time `json:"date_range_start"`

	// DateRangeEnd is the end of the content date range.
	DateRangeEnd time.Time `json:"date_range_end"`
}

// ============================================================================
// Newsletter Content Data
// ============================================================================

// NewsletterContentData represents the data available for template rendering.
// This is populated by the content resolver and passed to the template engine.
type NewsletterContentData struct {
	// Server information
	ServerName     string `json:"server_name"`
	ServerURL      string `json:"server_url"`
	NewsletterURL  string `json:"newsletter_url"`
	UnsubscribeURL string `json:"unsubscribe_url"`

	// Date information
	GeneratedAt      time.Time `json:"generated_at"`
	DateRangeStart   time.Time `json:"date_range_start"`
	DateRangeEnd     time.Time `json:"date_range_end"`
	DateRangeDisplay string    `json:"date_range_display"`

	// Recently added content
	NewMovies []NewsletterMediaItem `json:"new_movies,omitempty"`
	NewShows  []NewsletterShowItem  `json:"new_shows,omitempty"`
	NewMusic  []NewsletterMediaItem `json:"new_music,omitempty"`

	// Statistics
	Stats *NewsletterStats `json:"stats,omitempty"`

	// Top content
	TopMovies []NewsletterMediaItem `json:"top_movies,omitempty"`
	TopShows  []NewsletterShowItem  `json:"top_shows,omitempty"`

	// User-specific data (when personalized)
	User            *NewsletterUserData   `json:"user,omitempty"`
	Recommendations []NewsletterMediaItem `json:"recommendations,omitempty"`

	// Server health (for server_health type)
	Health *NewsletterHealthData `json:"health,omitempty"`
}

// NewsletterMediaItem represents a media item for newsletter display.
type NewsletterMediaItem struct {
	// Identification
	RatingKey string `json:"rating_key"`
	Title     string `json:"title"`
	Year      int    `json:"year,omitempty"`

	// Media details
	MediaType     string   `json:"media_type"` // movie, episode, track
	Summary       string   `json:"summary,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Duration      int      `json:"duration_minutes,omitempty"`
	ContentRating string   `json:"content_rating,omitempty"`

	// Images
	PosterURL string `json:"poster_url,omitempty"`
	ThumbURL  string `json:"thumb_url,omitempty"`
	ArtURL    string `json:"art_url,omitempty"`

	// Dates
	AddedAt    *time.Time `json:"added_at,omitempty"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`

	// Statistics (for top content)
	WatchCount int     `json:"watch_count,omitempty"`
	WatchTime  float64 `json:"watch_time_hours,omitempty"`

	// Links
	PlexURL string `json:"plex_url,omitempty"`
	IMDBURL string `json:"imdb_url,omitempty"`
	TMDBURL string `json:"tmdb_url,omitempty"`
}

// NewsletterShowItem represents a TV show with episodes for newsletter display.
type NewsletterShowItem struct {
	// Show identification
	RatingKey string `json:"rating_key"`
	Title     string `json:"title"`
	Year      int    `json:"year,omitempty"`

	// Show details
	Summary       string   `json:"summary,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	ContentRating string   `json:"content_rating,omitempty"`

	// Images
	PosterURL string `json:"poster_url,omitempty"`
	ThumbURL  string `json:"thumb_url,omitempty"`

	// Seasons with new episodes
	Seasons []NewsletterSeasonItem `json:"seasons,omitempty"`

	// Episode counts
	NewEpisodesCount   int `json:"new_episodes_count"`
	TotalEpisodesCount int `json:"total_episodes_count,omitempty"`

	// Statistics (for top content)
	WatchCount int     `json:"watch_count,omitempty"`
	WatchTime  float64 `json:"watch_time_hours,omitempty"`

	// Links
	PlexURL string `json:"plex_url,omitempty"`
}

// NewsletterSeasonItem represents a season within a show.
type NewsletterSeasonItem struct {
	SeasonNumber int                     `json:"season_number"`
	Episodes     []NewsletterEpisodeItem `json:"episodes,omitempty"`
}

// NewsletterEpisodeItem represents an episode for newsletter display.
type NewsletterEpisodeItem struct {
	RatingKey     string     `json:"rating_key"`
	Title         string     `json:"title"`
	EpisodeNumber int        `json:"episode_number"`
	Summary       string     `json:"summary,omitempty"`
	Duration      int        `json:"duration_minutes,omitempty"`
	AddedAt       *time.Time `json:"added_at,omitempty"`
	ThumbURL      string     `json:"thumb_url,omitempty"`
}

// NewsletterStats contains statistics for newsletter display.
type NewsletterStats struct {
	// Period stats
	TotalPlaybacks      int     `json:"total_playbacks"`
	TotalWatchTimeHours float64 `json:"total_watch_time_hours"`
	UniqueUsers         int     `json:"unique_users"`
	UniqueContent       int     `json:"unique_content"`

	// Content breakdown
	MoviesWatched   int `json:"movies_watched"`
	EpisodesWatched int `json:"episodes_watched"`
	TracksPlayed    int `json:"tracks_played"`

	// Platform breakdown
	TopPlatforms []PlatformStat `json:"top_platforms,omitempty"`

	// User breakdown
	TopUsers []UserStat `json:"top_users,omitempty"`
}

// PlatformStat represents platform usage statistics.
type PlatformStat struct {
	Platform   string  `json:"platform"`
	WatchCount int     `json:"watch_count"`
	WatchTime  float64 `json:"watch_time_hours"`
	Percentage float64 `json:"percentage"`
}

// UserStat represents user activity statistics.
type UserStat struct {
	UserID     string  `json:"user_id"`
	Username   string  `json:"username"`
	WatchCount int     `json:"watch_count"`
	WatchTime  float64 `json:"watch_time_hours"`
}

// NewsletterUserData contains user-specific data for personalized newsletters.
type NewsletterUserData struct {
	UserID         string  `json:"user_id"`
	Username       string  `json:"username"`
	WatchTimeHours float64 `json:"watch_time_hours"`
	PlaybackCount  int     `json:"playback_count"`
	TopGenre       string  `json:"top_genre,omitempty"`
	TopShow        string  `json:"top_show,omitempty"`
	TopMovie       string  `json:"top_movie,omitempty"`
}

// NewsletterHealthData contains server health data for health newsletters.
type NewsletterHealthData struct {
	ServerStatus   string     `json:"server_status"` // healthy, degraded, unhealthy
	UptimePercent  float64    `json:"uptime_percent"`
	ActiveStreams  int        `json:"active_streams"`
	TotalLibraries int        `json:"total_libraries"`
	TotalContent   int        `json:"total_content"`
	DatabaseSize   string     `json:"database_size"`
	LastSyncAt     *time.Time `json:"last_sync_at,omitempty"`
	Warnings       []string   `json:"warnings,omitempty"`
}

// ============================================================================
// API Request/Response Types
// ============================================================================

// CreateTemplateRequest is the request body for creating a newsletter template.
type CreateTemplateRequest struct {
	Name          string          `json:"name" validate:"required,min=1,max=100"`
	Description   string          `json:"description,omitempty" validate:"max=500"`
	Type          NewsletterType  `json:"type" validate:"required"`
	Subject       string          `json:"subject" validate:"required,min=1,max=200"`
	BodyHTML      string          `json:"body_html" validate:"required"`
	BodyText      string          `json:"body_text,omitempty"`
	DefaultConfig *TemplateConfig `json:"default_config,omitempty"`
}

// UpdateTemplateRequest is the request body for updating a newsletter template.
type UpdateTemplateRequest struct {
	Name          *string         `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description   *string         `json:"description,omitempty" validate:"omitempty,max=500"`
	Subject       *string         `json:"subject,omitempty" validate:"omitempty,min=1,max=200"`
	BodyHTML      *string         `json:"body_html,omitempty"`
	BodyText      *string         `json:"body_text,omitempty"`
	DefaultConfig *TemplateConfig `json:"default_config,omitempty"`
	IsActive      *bool           `json:"is_active,omitempty"`
}

// CreateScheduleRequest is the request body for creating a newsletter schedule.
type CreateScheduleRequest struct {
	Name           string                             `json:"name" validate:"required,min=1,max=100"`
	Description    string                             `json:"description,omitempty" validate:"max=500"`
	TemplateID     string                             `json:"template_id" validate:"required"`
	Recipients     []NewsletterRecipient              `json:"recipients" validate:"required,min=1"`
	CronExpression string                             `json:"cron_expression" validate:"required"`
	Timezone       string                             `json:"timezone" validate:"required"`
	Config         *TemplateConfig                    `json:"config,omitempty"`
	Channels       []DeliveryChannel                  `json:"channels" validate:"required,min=1"`
	ChannelConfigs map[DeliveryChannel]*ChannelConfig `json:"channel_configs,omitempty"`
	IsEnabled      bool                               `json:"is_enabled"`
}

// UpdateScheduleRequest is the request body for updating a newsletter schedule.
type UpdateScheduleRequest struct {
	Name           *string                            `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description    *string                            `json:"description,omitempty" validate:"omitempty,max=500"`
	TemplateID     *string                            `json:"template_id,omitempty"`
	Recipients     []NewsletterRecipient              `json:"recipients,omitempty"`
	CronExpression *string                            `json:"cron_expression,omitempty"`
	Timezone       *string                            `json:"timezone,omitempty"`
	Config         *TemplateConfig                    `json:"config,omitempty"`
	Channels       []DeliveryChannel                  `json:"channels,omitempty"`
	ChannelConfigs map[DeliveryChannel]*ChannelConfig `json:"channel_configs,omitempty"`
	IsEnabled      *bool                              `json:"is_enabled,omitempty"`
}

// SendNewsletterRequest is the request body for sending a newsletter immediately.
type SendNewsletterRequest struct {
	TemplateID     string                             `json:"template_id" validate:"required"`
	Recipients     []NewsletterRecipient              `json:"recipients" validate:"required,min=1"`
	Config         *TemplateConfig                    `json:"config,omitempty"`
	Channels       []DeliveryChannel                  `json:"channels" validate:"required,min=1"`
	ChannelConfigs map[DeliveryChannel]*ChannelConfig `json:"channel_configs,omitempty"`
}

// PreviewNewsletterRequest is the request body for previewing a newsletter.
type PreviewNewsletterRequest struct {
	TemplateID string          `json:"template_id" validate:"required"`
	Config     *TemplateConfig `json:"config,omitempty"`
	ForUserID  *string         `json:"for_user_id,omitempty"` // For personalized preview
}

// PreviewNewsletterResponse is the response body for newsletter preview.
type PreviewNewsletterResponse struct {
	Subject  string                 `json:"subject"`
	BodyHTML string                 `json:"body_html"`
	BodyText string                 `json:"body_text"`
	Data     *NewsletterContentData `json:"data"`
}

// ListTemplatesResponse is the response body for listing templates.
type ListTemplatesResponse struct {
	Templates  []NewsletterTemplate `json:"templates"`
	TotalCount int                  `json:"total_count"`
}

// ListSchedulesResponse is the response body for listing schedules.
type ListSchedulesResponse struct {
	Schedules  []NewsletterSchedule `json:"schedules"`
	TotalCount int                  `json:"total_count"`
}

// ListDeliveriesResponse is the response body for listing delivery history.
type ListDeliveriesResponse struct {
	Deliveries []NewsletterDelivery `json:"deliveries"`
	Pagination PaginationInfo       `json:"pagination"`
}

// NewsletterStatsResponse contains aggregated newsletter statistics.
type NewsletterStatsResponse struct {
	TotalTemplates       int            `json:"total_templates"`
	ActiveTemplates      int            `json:"active_templates"`
	TotalSchedules       int            `json:"total_schedules"`
	EnabledSchedules     int            `json:"enabled_schedules"`
	TotalDeliveries      int            `json:"total_deliveries"`
	SuccessfulDeliveries int            `json:"successful_deliveries"`
	FailedDeliveries     int            `json:"failed_deliveries"`
	DeliveriesByChannel  map[string]int `json:"deliveries_by_channel"`
	DeliveriesByType     map[string]int `json:"deliveries_by_type"`
	Last7DaysDeliveries  int            `json:"last_7_days_deliveries"`
	Last30DaysDeliveries int            `json:"last_30_days_deliveries"`
}

// ============================================================================
// User Preferences
// ============================================================================

// NewsletterUserPreferences represents a user's newsletter preferences.
type NewsletterUserPreferences struct {
	// UserID is the user identifier.
	UserID string `json:"user_id"`

	// Username is the display name.
	Username string `json:"username"`

	// GlobalOptOut indicates the user has opted out of all newsletters.
	GlobalOptOut bool `json:"global_opt_out"`

	// GlobalOptOutAt is when the user opted out.
	GlobalOptOutAt *time.Time `json:"global_opt_out_at,omitempty"`

	// SchedulePreferences contains per-schedule preferences.
	SchedulePreferences map[string]*RecipientPreferences `json:"schedule_preferences,omitempty"`

	// PreferredChannel is the user's preferred delivery channel.
	PreferredChannel *DeliveryChannel `json:"preferred_channel,omitempty"`

	// PreferredEmail is the user's preferred email address.
	PreferredEmail string `json:"preferred_email,omitempty"`

	// Language is the user's preferred language.
	Language string `json:"language,omitempty"`

	// UpdatedAt is when preferences were last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// ============================================================================
// Audit Logging
// ============================================================================

// NewsletterAuditEntry represents an audit log entry for newsletter operations.
type NewsletterAuditEntry struct {
	// ID is the unique audit entry identifier.
	ID string `json:"id"`

	// Timestamp is when the action occurred.
	Timestamp time.Time `json:"timestamp"`

	// ActorID is the user who performed the action.
	ActorID string `json:"actor_id"`

	// ActorUsername is the display name of the actor.
	ActorUsername string `json:"actor_username,omitempty"`

	// Action is the type of action (create, update, delete, send, etc.).
	Action string `json:"action"`

	// ResourceType is the type of resource affected (template, schedule, delivery).
	ResourceType string `json:"resource_type"`

	// ResourceID is the ID of the affected resource.
	ResourceID string `json:"resource_id"`

	// ResourceName is the name of the affected resource.
	ResourceName string `json:"resource_name,omitempty"`

	// Details contains additional context about the action.
	Details map[string]interface{} `json:"details,omitempty"`

	// IPAddress is the client IP address.
	IPAddress string `json:"ip_address,omitempty"`

	// UserAgent is the client user agent.
	UserAgent string `json:"user_agent,omitempty"`
}

// Newsletter audit action constants.
const (
	NewsletterAuditActionCreate  = "create"
	NewsletterAuditActionUpdate  = "update"
	NewsletterAuditActionDelete  = "delete"
	NewsletterAuditActionEnable  = "enable"
	NewsletterAuditActionDisable = "disable"
	NewsletterAuditActionSend    = "send"
	NewsletterAuditActionPreview = "preview"
	NewsletterAuditActionOptOut  = "opt_out"
	NewsletterAuditActionOptIn   = "opt_in"
)

// Newsletter audit resource type constants.
const (
	NewsletterResourceTemplate    = "template"
	NewsletterResourceSchedule    = "schedule"
	NewsletterResourceDelivery    = "delivery"
	NewsletterResourcePreferences = "preferences"
)
