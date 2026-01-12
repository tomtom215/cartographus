// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package newsletter provides newsletter generation and delivery functionality.
//
// builtin_templates.go - Built-in Newsletter Templates
//
// This file contains built-in templates for standard newsletter types.
// These templates provide sensible defaults and can be customized by users.
package newsletter

import (
	"github.com/tomtom215/cartographus/internal/models"
)

// GetBuiltinTemplates returns all built-in newsletter templates.
func GetBuiltinTemplates() []models.NewsletterTemplate {
	return []models.NewsletterTemplate{
		getRecentlyAddedTemplate(),
		getWeeklyDigestTemplate(),
		getMonthlyStatsTemplate(),
		getUserActivityTemplate(),
		getServerHealthTemplate(),
	}
}

// getRecentlyAddedTemplate returns the recently added content template.
func getRecentlyAddedTemplate() models.NewsletterTemplate {
	return models.NewsletterTemplate{
		ID:          "builtin_recently_added",
		Name:        "Recently Added",
		Description: "Shows recently added movies, TV shows, and music",
		Type:        models.NewsletterTypeRecentlyAdded,
		Subject:     "{{.ServerName}} - New Content Added ({{.DateRangeDisplay}})",
		BodyHTML:    recentlyAddedHTMLTemplate,
		BodyText:    recentlyAddedTextTemplate,
		IsBuiltIn:   true,
		IsActive:    true,
		Version:     1,
		DefaultConfig: &models.TemplateConfig{
			TimeFrame:           7,
			TimeFrameUnit:       models.TimeFrameUnitDays,
			IncludeMovies:       true,
			IncludeShows:        true,
			IncludeMusic:        false,
			MaxItems:            10,
			IncludePosterImages: true,
			ImageHosting:        "self_hosted",
		},
	}
}

// getWeeklyDigestTemplate returns the weekly digest template.
func getWeeklyDigestTemplate() models.NewsletterTemplate {
	return models.NewsletterTemplate{
		ID:          "builtin_weekly_digest",
		Name:        "Weekly Digest",
		Description: "Weekly summary of viewing activity and new content",
		Type:        models.NewsletterTypeWeeklyDigest,
		Subject:     "{{.ServerName}} - Weekly Digest ({{formatDateShort .DateRangeStart}} - {{formatDateShort .DateRangeEnd}})",
		BodyHTML:    weeklyDigestHTMLTemplate,
		BodyText:    weeklyDigestTextTemplate,
		IsBuiltIn:   true,
		IsActive:    true,
		Version:     1,
		DefaultConfig: &models.TemplateConfig{
			TimeFrame:           7,
			TimeFrameUnit:       models.TimeFrameUnitDays,
			IncludeMovies:       true,
			IncludeShows:        true,
			IncludeStats:        true,
			IncludeTopContent:   true,
			MaxItems:            5,
			IncludePosterImages: true,
		},
	}
}

// getMonthlyStatsTemplate returns the monthly statistics template.
func getMonthlyStatsTemplate() models.NewsletterTemplate {
	return models.NewsletterTemplate{
		ID:          "builtin_monthly_stats",
		Name:        "Monthly Statistics",
		Description: "Monthly viewing statistics and top content",
		Type:        models.NewsletterTypeMonthlyStats,
		Subject:     "{{.ServerName}} - Monthly Stats for {{formatDate .DateRangeStart \"January 2006\"}}",
		BodyHTML:    monthlyStatsHTMLTemplate,
		BodyText:    monthlyStatsTextTemplate,
		IsBuiltIn:   true,
		IsActive:    true,
		Version:     1,
		DefaultConfig: &models.TemplateConfig{
			TimeFrame:           1,
			TimeFrameUnit:       models.TimeFrameUnitMonths,
			IncludeStats:        true,
			IncludeTopContent:   true,
			MaxItems:            10,
			IncludePosterImages: true,
		},
	}
}

// getUserActivityTemplate returns the user activity template.
func getUserActivityTemplate() models.NewsletterTemplate {
	return models.NewsletterTemplate{
		ID:          "builtin_user_activity",
		Name:        "User Activity Summary",
		Description: "Personalized activity summary for individual users",
		Type:        models.NewsletterTypeUserActivity,
		Subject:     "{{.ServerName}} - Your Activity Summary ({{.DateRangeDisplay}})",
		BodyHTML:    userActivityHTMLTemplate,
		BodyText:    userActivityTextTemplate,
		IsBuiltIn:   true,
		IsActive:    true,
		Version:     1,
		DefaultConfig: &models.TemplateConfig{
			TimeFrame:          7,
			TimeFrameUnit:      models.TimeFrameUnitDays,
			PersonalizeForUser: true,
			IncludeStats:       true,
		},
	}
}

// getServerHealthTemplate returns the server health template.
func getServerHealthTemplate() models.NewsletterTemplate {
	return models.NewsletterTemplate{
		ID:          "builtin_server_health",
		Name:        "Server Health Report",
		Description: "Server status and health metrics for administrators",
		Type:        models.NewsletterTypeServerHealth,
		Subject:     "{{.ServerName}} - Health Report ({{formatDateDefault .GeneratedAt}})",
		BodyHTML:    serverHealthHTMLTemplate,
		BodyText:    serverHealthTextTemplate,
		IsBuiltIn:   true,
		IsActive:    true,
		Version:     1,
	}
}

// HTML Templates

const recentlyAddedHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.ServerName}} - Recently Added</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #eee; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: #16213e; border-radius: 10px; overflow: hidden; }
    .header { background: linear-gradient(135deg, #e94560 0%, #0f3460 100%); padding: 30px; text-align: center; }
    .header h1 { margin: 0; color: #fff; font-size: 24px; }
    .header p { margin: 10px 0 0; color: rgba(255,255,255,0.8); font-size: 14px; }
    .content { padding: 20px; }
    .section { margin-bottom: 30px; }
    .section h2 { color: #e94560; font-size: 18px; margin: 0 0 15px; border-bottom: 2px solid #e94560; padding-bottom: 10px; }
    .media-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 15px; }
    .media-card { background: #0f3460; border-radius: 8px; overflow: hidden; }
    .media-card img { width: 100%; aspect-ratio: 2/3; object-fit: cover; }
    .media-card .info { padding: 10px; }
    .media-card .title { font-weight: 600; font-size: 14px; color: #fff; margin: 0; }
    .media-card .meta { font-size: 12px; color: #aaa; margin-top: 5px; }
    .footer { background: #0f3460; padding: 20px; text-align: center; font-size: 12px; color: #888; }
    .footer a { color: #e94560; text-decoration: none; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>{{.ServerName}}</h1>
      <p>New Content Added - {{.DateRangeDisplay}}</p>
    </div>
    <div class="content">
      {{if .NewMovies}}
      <div class="section">
        <h2>New Movies ({{len .NewMovies}})</h2>
        <div class="media-grid">
          {{range .NewMovies}}
          <div class="media-card">
            {{if .PosterURL}}<img src="{{.PosterURL}}" alt="{{.Title}}">{{end}}
            <div class="info">
              <p class="title">{{truncate .Title 30}}</p>
              <p class="meta">{{.Year}}{{if .Duration}} - {{formatDuration .Duration}}{{end}}</p>
            </div>
          </div>
          {{end}}
        </div>
      </div>
      {{end}}

      {{if .NewShows}}
      <div class="section">
        <h2>New TV Shows ({{len .NewShows}})</h2>
        <div class="media-grid">
          {{range .NewShows}}
          <div class="media-card">
            {{if .PosterURL}}<img src="{{.PosterURL}}" alt="{{.Title}}">{{end}}
            <div class="info">
              <p class="title">{{truncate .Title 30}}</p>
              <p class="meta">{{.NewEpisodesCount}} new episode{{if gt .NewEpisodesCount 1}}s{{end}}</p>
            </div>
          </div>
          {{end}}
        </div>
      </div>
      {{end}}

      {{if .NewMusic}}
      <div class="section">
        <h2>New Music ({{len .NewMusic}})</h2>
        <div class="media-grid">
          {{range .NewMusic}}
          <div class="media-card">
            {{if .PosterURL}}<img src="{{.PosterURL}}" alt="{{.Title}}">{{end}}
            <div class="info">
              <p class="title">{{truncate .Title 30}}</p>
            </div>
          </div>
          {{end}}
        </div>
      </div>
      {{end}}
    </div>
    <div class="footer">
      <p>Generated on {{formatDateTime .GeneratedAt}}</p>
      {{if .UnsubscribeURL}}<p><a href="{{.UnsubscribeURL}}">Unsubscribe</a></p>{{end}}
    </div>
  </div>
</body>
</html>`

const recentlyAddedTextTemplate = `{{.ServerName}} - Recently Added
{{.DateRangeDisplay}}
========================================

{{if .NewMovies}}
NEW MOVIES ({{len .NewMovies}})
{{range .NewMovies}}
- {{.Title}} ({{.Year}}){{if .Duration}} - {{formatDuration .Duration}}{{end}}
{{end}}
{{end}}

{{if .NewShows}}
NEW TV SHOWS ({{len .NewShows}})
{{range .NewShows}}
- {{.Title}} - {{.NewEpisodesCount}} new episode{{if gt .NewEpisodesCount 1}}s{{end}}
{{end}}
{{end}}

{{if .NewMusic}}
NEW MUSIC ({{len .NewMusic}})
{{range .NewMusic}}
- {{.Title}}
{{end}}
{{end}}

---
Generated: {{formatDateTime .GeneratedAt}}
{{if .UnsubscribeURL}}Unsubscribe: {{.UnsubscribeURL}}{{end}}`

const weeklyDigestHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.ServerName}} - Weekly Digest</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #eee; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: #16213e; border-radius: 10px; overflow: hidden; }
    .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 30px; text-align: center; }
    .header h1 { margin: 0; color: #fff; font-size: 24px; }
    .header p { margin: 10px 0 0; color: rgba(255,255,255,0.8); font-size: 14px; }
    .stats-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 15px; padding: 20px; }
    .stat-card { background: #0f3460; border-radius: 8px; padding: 20px; text-align: center; }
    .stat-value { font-size: 32px; font-weight: 700; color: #667eea; }
    .stat-label { font-size: 12px; color: #aaa; text-transform: uppercase; margin-top: 5px; }
    .content { padding: 20px; }
    .section { margin-bottom: 30px; }
    .section h2 { color: #667eea; font-size: 18px; margin: 0 0 15px; }
    .list-item { display: flex; align-items: center; gap: 15px; padding: 10px; background: #0f3460; border-radius: 8px; margin-bottom: 10px; }
    .list-item img { width: 60px; height: 90px; object-fit: cover; border-radius: 4px; }
    .list-item .info { flex: 1; }
    .list-item .title { font-weight: 600; color: #fff; margin: 0; }
    .list-item .meta { font-size: 12px; color: #aaa; margin-top: 5px; }
    .list-item .rank { font-size: 24px; font-weight: 700; color: #667eea; min-width: 30px; }
    .footer { background: #0f3460; padding: 20px; text-align: center; font-size: 12px; color: #888; }
    .footer a { color: #667eea; text-decoration: none; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>{{.ServerName}} Weekly Digest</h1>
      <p>{{formatDateShort .DateRangeStart}} - {{formatDateShort .DateRangeEnd}}</p>
    </div>

    {{if .Stats}}
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-value">{{formatNumber .Stats.TotalPlaybacks}}</div>
        <div class="stat-label">Total Plays</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{formatHours .Stats.TotalWatchTimeHours}}</div>
        <div class="stat-label">Watch Time</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{.Stats.UniqueUsers}}</div>
        <div class="stat-label">Active Users</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{.Stats.UniqueContent}}</div>
        <div class="stat-label">Unique Titles</div>
      </div>
    </div>
    {{end}}

    <div class="content">
      {{if .TopMovies}}
      <div class="section">
        <h2>Top Movies This Week</h2>
        {{range $i, $m := .TopMovies}}
        <div class="list-item">
          <span class="rank">{{add $i 1}}</span>
          {{if $m.PosterURL}}<img src="{{$m.PosterURL}}" alt="{{$m.Title}}">{{end}}
          <div class="info">
            <p class="title">{{$m.Title}}</p>
            <p class="meta">{{$m.WatchCount}} plays - {{formatHours $m.WatchTime}}</p>
          </div>
        </div>
        {{end}}
      </div>
      {{end}}

      {{if .TopShows}}
      <div class="section">
        <h2>Top Shows This Week</h2>
        {{range $i, $s := .TopShows}}
        <div class="list-item">
          <span class="rank">{{add $i 1}}</span>
          {{if $s.PosterURL}}<img src="{{$s.PosterURL}}" alt="{{$s.Title}}">{{end}}
          <div class="info">
            <p class="title">{{$s.Title}}</p>
            <p class="meta">{{$s.WatchCount}} plays - {{formatHours $s.WatchTime}}</p>
          </div>
        </div>
        {{end}}
      </div>
      {{end}}

      {{if .NewMovies}}
      <div class="section">
        <h2>Newly Added</h2>
        <p style="color: #aaa; font-size: 14px;">{{len .NewMovies}} movies{{if .NewShows}}, {{len .NewShows}} shows{{end}} added this week</p>
      </div>
      {{end}}
    </div>

    <div class="footer">
      <p>Generated on {{formatDateTime .GeneratedAt}}</p>
      {{if .UnsubscribeURL}}<p><a href="{{.UnsubscribeURL}}">Unsubscribe</a></p>{{end}}
    </div>
  </div>
</body>
</html>`

const weeklyDigestTextTemplate = `{{.ServerName}} - Weekly Digest
{{formatDateShort .DateRangeStart}} - {{formatDateShort .DateRangeEnd}}
========================================

{{if .Stats}}
THIS WEEK'S STATS
- Total Plays: {{formatNumber .Stats.TotalPlaybacks}}
- Watch Time: {{formatHours .Stats.TotalWatchTimeHours}}
- Active Users: {{.Stats.UniqueUsers}}
- Unique Titles: {{.Stats.UniqueContent}}
{{end}}

{{if .TopMovies}}
TOP MOVIES
{{range $i, $m := .TopMovies}}
{{add $i 1}}. {{$m.Title}} - {{$m.WatchCount}} plays ({{formatHours $m.WatchTime}})
{{end}}
{{end}}

{{if .TopShows}}
TOP SHOWS
{{range $i, $s := .TopShows}}
{{add $i 1}}. {{$s.Title}} - {{$s.WatchCount}} plays ({{formatHours $s.WatchTime}})
{{end}}
{{end}}

---
Generated: {{formatDateTime .GeneratedAt}}`

const monthlyStatsHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.ServerName}} - Monthly Statistics</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0a0a0a; color: #eee; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: #1a1a1a; border-radius: 10px; overflow: hidden; }
    .header { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); padding: 40px 30px; text-align: center; }
    .header h1 { margin: 0; color: #fff; font-size: 28px; }
    .header p { margin: 10px 0 0; color: rgba(255,255,255,0.9); font-size: 16px; }
    .stats-hero { padding: 30px; text-align: center; background: #111; }
    .stats-hero .big-number { font-size: 64px; font-weight: 700; background: linear-gradient(135deg, #f093fb, #f5576c); -webkit-background-clip: text; -webkit-text-fill-color: transparent; }
    .stats-hero .label { font-size: 14px; color: #888; text-transform: uppercase; }
    .stats-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 1px; background: #333; }
    .stat-item { background: #1a1a1a; padding: 20px; text-align: center; }
    .stat-item .value { font-size: 24px; font-weight: 600; color: #f5576c; }
    .stat-item .label { font-size: 11px; color: #888; text-transform: uppercase; margin-top: 5px; }
    .content { padding: 20px; }
    .section { margin-bottom: 30px; }
    .section h2 { color: #f5576c; font-size: 16px; margin: 0 0 15px; text-transform: uppercase; letter-spacing: 1px; }
    .top-list { counter-reset: rank; }
    .top-item { display: flex; align-items: center; gap: 15px; padding: 15px; background: #111; border-radius: 8px; margin-bottom: 8px; }
    .top-item::before { counter-increment: rank; content: counter(rank); font-size: 20px; font-weight: 700; color: #f5576c; min-width: 30px; }
    .top-item img { width: 50px; height: 75px; object-fit: cover; border-radius: 4px; }
    .top-item .info { flex: 1; }
    .top-item .title { font-weight: 600; color: #fff; margin: 0; }
    .top-item .meta { font-size: 12px; color: #666; margin-top: 3px; }
    .footer { background: #111; padding: 20px; text-align: center; font-size: 12px; color: #666; }
    .footer a { color: #f5576c; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>Monthly Statistics</h1>
      <p>{{formatDate .DateRangeStart "January 2006"}}</p>
    </div>

    {{if .Stats}}
    <div class="stats-hero">
      <div class="big-number">{{formatHours .Stats.TotalWatchTimeHours}}</div>
      <div class="label">Total Watch Time</div>
    </div>

    <div class="stats-grid">
      <div class="stat-item">
        <div class="value">{{formatNumber .Stats.TotalPlaybacks}}</div>
        <div class="label">Plays</div>
      </div>
      <div class="stat-item">
        <div class="value">{{.Stats.UniqueUsers}}</div>
        <div class="label">Users</div>
      </div>
      <div class="stat-item">
        <div class="value">{{.Stats.UniqueContent}}</div>
        <div class="label">Titles</div>
      </div>
    </div>
    {{end}}

    <div class="content">
      {{if .TopMovies}}
      <div class="section">
        <h2>Top 10 Movies</h2>
        <div class="top-list">
          {{range .TopMovies}}
          <div class="top-item">
            {{if .PosterURL}}<img src="{{.PosterURL}}" alt="{{.Title}}">{{end}}
            <div class="info">
              <p class="title">{{.Title}}</p>
              <p class="meta">{{.WatchCount}} plays / {{formatHours .WatchTime}}</p>
            </div>
          </div>
          {{end}}
        </div>
      </div>
      {{end}}

      {{if .TopShows}}
      <div class="section">
        <h2>Top 10 Shows</h2>
        <div class="top-list">
          {{range .TopShows}}
          <div class="top-item">
            {{if .PosterURL}}<img src="{{.PosterURL}}" alt="{{.Title}}">{{end}}
            <div class="info">
              <p class="title">{{.Title}}</p>
              <p class="meta">{{.WatchCount}} plays / {{formatHours .WatchTime}}</p>
            </div>
          </div>
          {{end}}
        </div>
      </div>
      {{end}}
    </div>

    <div class="footer">
      <p>{{.ServerName}} - Generated {{formatDateTime .GeneratedAt}}</p>
      {{if .UnsubscribeURL}}<p><a href="{{.UnsubscribeURL}}">Unsubscribe</a></p>{{end}}
    </div>
  </div>
</body>
</html>`

const monthlyStatsTextTemplate = `{{.ServerName}} - Monthly Statistics
{{formatDate .DateRangeStart "January 2006"}}
========================================

{{if .Stats}}
OVERVIEW
- Total Watch Time: {{formatHours .Stats.TotalWatchTimeHours}}
- Total Plays: {{formatNumber .Stats.TotalPlaybacks}}
- Active Users: {{.Stats.UniqueUsers}}
- Unique Titles: {{.Stats.UniqueContent}}
{{end}}

{{if .TopMovies}}
TOP 10 MOVIES
{{range $i, $m := .TopMovies}}
{{add $i 1}}. {{$m.Title}}
   {{$m.WatchCount}} plays / {{formatHours $m.WatchTime}}
{{end}}
{{end}}

{{if .TopShows}}
TOP 10 SHOWS
{{range $i, $s := .TopShows}}
{{add $i 1}}. {{$s.Title}}
   {{$s.WatchCount}} plays / {{formatHours $s.WatchTime}}
{{end}}
{{end}}

---
Generated: {{formatDateTime .GeneratedAt}}`

const userActivityHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.ServerName}} - Your Activity</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #0d1117; color: #c9d1d9; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: #161b22; border-radius: 10px; border: 1px solid #30363d; overflow: hidden; }
    .header { background: linear-gradient(135deg, #238636 0%, #2ea043 100%); padding: 30px; text-align: center; }
    .header h1 { margin: 0; color: #fff; font-size: 24px; }
    .header .username { margin: 10px 0 0; color: rgba(255,255,255,0.9); font-size: 18px; font-weight: 600; }
    .header .period { color: rgba(255,255,255,0.7); font-size: 14px; margin-top: 5px; }
    .user-stats { padding: 30px; background: #0d1117; text-align: center; }
    .user-stats .big-stat { font-size: 48px; font-weight: 700; color: #58a6ff; }
    .user-stats .label { color: #8b949e; font-size: 14px; }
    .content { padding: 20px; }
    .highlight-box { background: #21262d; border-radius: 8px; padding: 20px; margin-bottom: 20px; }
    .highlight-box h3 { color: #58a6ff; margin: 0 0 10px; font-size: 14px; text-transform: uppercase; }
    .highlight-box .value { font-size: 20px; font-weight: 600; color: #c9d1d9; }
    .footer { background: #0d1117; padding: 20px; text-align: center; font-size: 12px; color: #8b949e; border-top: 1px solid #30363d; }
    .footer a { color: #58a6ff; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>Your Activity Summary</h1>
      {{if .User}}<div class="username">{{.User.Username}}</div>{{end}}
      <div class="period">{{.DateRangeDisplay}}</div>
    </div>

    {{if .User}}
    <div class="user-stats">
      <div class="big-stat">{{formatHours .User.WatchTimeHours}}</div>
      <div class="label">Watch Time This Week</div>
    </div>

    <div class="content">
      <div class="highlight-box">
        <h3>Playback Count</h3>
        <div class="value">{{.User.PlaybackCount}} plays</div>
      </div>

      {{if .User.TopGenre}}
      <div class="highlight-box">
        <h3>Top Genre</h3>
        <div class="value">{{.User.TopGenre}}</div>
      </div>
      {{end}}

      {{if .User.TopShow}}
      <div class="highlight-box">
        <h3>Most Watched Show</h3>
        <div class="value">{{.User.TopShow}}</div>
      </div>
      {{end}}

      {{if .User.TopMovie}}
      <div class="highlight-box">
        <h3>Top Movie</h3>
        <div class="value">{{.User.TopMovie}}</div>
      </div>
      {{end}}
    </div>
    {{end}}

    <div class="footer">
      <p>{{.ServerName}} - Generated {{formatDateTime .GeneratedAt}}</p>
      {{if .UnsubscribeURL}}<p><a href="{{.UnsubscribeURL}}">Manage Preferences</a></p>{{end}}
    </div>
  </div>
</body>
</html>`

const userActivityTextTemplate = `{{.ServerName}} - Your Activity Summary
{{if .User}}{{.User.Username}}{{end}}
{{.DateRangeDisplay}}
========================================

{{if .User}}
YOUR STATS
- Watch Time: {{formatHours .User.WatchTimeHours}}
- Playback Count: {{.User.PlaybackCount}} plays
{{if .User.TopGenre}}- Top Genre: {{.User.TopGenre}}{{end}}
{{if .User.TopShow}}- Most Watched Show: {{.User.TopShow}}{{end}}
{{if .User.TopMovie}}- Top Movie: {{.User.TopMovie}}{{end}}
{{end}}

---
Generated: {{formatDateTime .GeneratedAt}}`

const serverHealthHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{.ServerName}} - Health Report</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #1a1a2e; color: #eee; margin: 0; padding: 20px; }
    .container { max-width: 600px; margin: 0 auto; background: #16213e; border-radius: 10px; overflow: hidden; }
    .header { padding: 30px; text-align: center; }
    .header.healthy { background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); }
    .header.degraded { background: linear-gradient(135deg, #f2994a 0%, #f2c94c 100%); }
    .header.unhealthy { background: linear-gradient(135deg, #eb3349 0%, #f45c43 100%); }
    .header h1 { margin: 0; color: #fff; font-size: 24px; }
    .header .status { font-size: 36px; font-weight: 700; color: #fff; margin-top: 10px; text-transform: uppercase; }
    .content { padding: 20px; }
    .metric { display: flex; justify-content: space-between; padding: 15px; background: #0f3460; border-radius: 8px; margin-bottom: 10px; }
    .metric .label { color: #aaa; }
    .metric .value { font-weight: 600; color: #fff; }
    .warnings { background: #2d1f1f; border: 1px solid #eb3349; border-radius: 8px; padding: 15px; margin-top: 20px; }
    .warnings h3 { color: #eb3349; margin: 0 0 10px; }
    .warnings ul { margin: 0; padding-left: 20px; color: #f45c43; }
    .footer { background: #0f3460; padding: 20px; text-align: center; font-size: 12px; color: #888; }
  </style>
</head>
<body>
  <div class="container">
    {{if .Health}}
    <div class="header {{.Health.ServerStatus}}">
      <h1>{{.ServerName}} Health Report</h1>
      <div class="status">{{.Health.ServerStatus}}</div>
    </div>

    <div class="content">
      <div class="metric">
        <span class="label">Uptime</span>
        <span class="value">{{formatPercent .Health.UptimePercent}}</span>
      </div>
      <div class="metric">
        <span class="label">Active Streams</span>
        <span class="value">{{.Health.ActiveStreams}}</span>
      </div>
      <div class="metric">
        <span class="label">Total Libraries</span>
        <span class="value">{{.Health.TotalLibraries}}</span>
      </div>
      <div class="metric">
        <span class="label">Total Content</span>
        <span class="value">{{formatNumber .Health.TotalContent}}</span>
      </div>
      <div class="metric">
        <span class="label">Database Size</span>
        <span class="value">{{.Health.DatabaseSize}}</span>
      </div>
      {{if .Health.LastSyncAt}}
      <div class="metric">
        <span class="label">Last Sync</span>
        <span class="value">{{formatDateTime (timePtr .Health.LastSyncAt)}}</span>
      </div>
      {{end}}

      {{if .Health.Warnings}}
      <div class="warnings">
        <h3>Warnings</h3>
        <ul>
          {{range .Health.Warnings}}
          <li>{{.}}</li>
          {{end}}
        </ul>
      </div>
      {{end}}
    </div>
    {{else}}
    <div class="header unhealthy">
      <h1>{{.ServerName}} Health Report</h1>
      <div class="status">Unable to retrieve health data</div>
    </div>
    {{end}}

    <div class="footer">
      <p>Generated {{formatDateTime .GeneratedAt}}</p>
    </div>
  </div>
</body>
</html>`

const serverHealthTextTemplate = `{{.ServerName}} - Health Report
{{formatDateDefault .GeneratedAt}}
========================================

{{if .Health}}
STATUS: {{.Health.ServerStatus | toUpperCase}}

METRICS
- Uptime: {{formatPercent .Health.UptimePercent}}
- Active Streams: {{.Health.ActiveStreams}}
- Total Libraries: {{.Health.TotalLibraries}}
- Total Content: {{formatNumber .Health.TotalContent}}
- Database Size: {{.Health.DatabaseSize}}
{{if .Health.LastSyncAt}}- Last Sync: {{formatDateTime (timePtr .Health.LastSyncAt)}}{{end}}

{{if .Health.Warnings}}
WARNINGS
{{range .Health.Warnings}}
- {{.}}
{{end}}
{{end}}
{{else}}
STATUS: UNKNOWN
Unable to retrieve health data.
{{end}}

---
Generated: {{formatDateTime .GeneratedAt}}`
