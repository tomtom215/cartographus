// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/tomtom215/cartographus/internal/models"
)

// GetAudioAnalytics returns comprehensive audio quality analytics including channel distribution,
// codec distribution, downmix events, surround sound adoption, lossless adoption, and Atmos playback counts.
func (db *DB) GetAudioAnalytics(ctx context.Context, filter LocationStatsFilter) (*models.AudioAnalytics, error) {
	ctx, cancel := db.ensureContext(ctx)
	defer cancel()

	whereClauses, args := buildFilterConditions(filter, false, 1)
	whereClause := buildWhereClause(whereClauses)

	// Get total playbacks
	total, err := db.getTotalPlaybacks(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get total playbacks", err)
	}

	// Get channel distribution with surround adoption
	channelDist, surroundAdoption, err := db.getAudioChannelDistribution(ctx, whereClause, args, total)
	if err != nil {
		return nil, errorContext("get channel distribution", err)
	}

	// Get codec distribution with lossless/atmos statistics
	codecDist, losslessAdoption, avgBitrate, atmosCount, err := db.getAudioCodecDistribution(ctx, whereClause, args, total)
	if err != nil {
		return nil, errorContext("get codec distribution", err)
	}

	// Get downmix events
	downmixEvents, err := db.getAudioDownmixEvents(ctx, whereClause, args)
	if err != nil {
		return nil, errorContext("get downmix events", err)
	}

	return &models.AudioAnalytics{
		TotalPlaybacks:        total,
		ChannelDistribution:   channelDist,
		CodecDistribution:     codecDist,
		DownmixEvents:         downmixEvents,
		SurroundSoundAdoption: surroundAdoption,
		LosslessAdoption:      losslessAdoption,
		AvgBitrate:            avgBitrate,
		AtmosPlaybacks:        atmosCount,
	}, nil
}

// getAudioChannelDistribution retrieves audio channel distribution and calculates surround sound adoption
func (db *DB) getAudioChannelDistribution(ctx context.Context, whereClause string, args []interface{}, total int) ([]models.AudioChannelDistribution, float64, error) {
	query := fmt.Sprintf(`
		SELECT
			COALESCE(audio_channels, '2') as channels,
			COALESCE(audio_channel_layout, '') as layout,
			COUNT(*) as playback_count,
			(COUNT(*) * 100.0 / %d) as percentage,
			AVG(percent_complete) as avg_completion
		FROM playback_events
		%s
		GROUP BY audio_channels, audio_channel_layout
		ORDER BY playback_count DESC
	`, total, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query channel distribution: %w", err)
	}
	defer rows.Close()

	var channelDist []models.AudioChannelDistribution
	surroundCount := 0
	for rows.Next() {
		var d models.AudioChannelDistribution
		err := rows.Scan(&d.Channels, &d.Layout, &d.PlaybackCount, &d.Percentage, &d.AvgCompletion)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan channel row: %w", err)
		}
		channelDist = append(channelDist, d)
		// Count 6+ channels as surround sound
		if d.Channels != "2" && d.Channels != "1" && d.Channels != "" {
			surroundCount += d.PlaybackCount
		}
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating rows: %w", err)
	}

	surroundAdoption := calculateAdoptionRate(surroundCount, total)

	return channelDist, surroundAdoption, nil
}

// getAudioCodecDistribution retrieves audio codec distribution with lossless/atmos statistics
func (db *DB) getAudioCodecDistribution(ctx context.Context, whereClause string, args []interface{}, total int) ([]models.AudioCodecDistribution, float64, float64, int, error) {
	losslessCodecs := map[string]bool{
		"flac": true, "alac": true, "ape": true, "truehd": true,
		"dts-hd ma": true, "pcm": true, "wav": true,
	}

	query := fmt.Sprintf(`
		SELECT
			LOWER(COALESCE(audio_codec, 'aac')) as codec,
			COUNT(*) as playback_count,
			(COUNT(*) * 100.0 / %d) as percentage,
			AVG(COALESCE(audio_bitrate, 0)) as avg_bitrate
		FROM playback_events
		%s
		GROUP BY audio_codec
		ORDER BY playback_count DESC
	`, total, whereClause)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to query codec distribution: %w", err)
	}
	defer rows.Close()

	var codecDist []models.AudioCodecDistribution
	losslessCount := 0
	totalBitrate := 0.0
	bitrateCount := 0
	atmosCount := 0

	for rows.Next() {
		var d models.AudioCodecDistribution
		err := rows.Scan(&d.Codec, &d.PlaybackCount, &d.Percentage, &d.AvgBitrate)
		if err != nil {
			return nil, 0, 0, 0, fmt.Errorf("failed to scan codec row: %w", err)
		}
		d.IsLossless = losslessCodecs[strings.ToLower(d.Codec)]
		codecDist = append(codecDist, d)

		if d.IsLossless {
			losslessCount += d.PlaybackCount
		}
		if d.AvgBitrate > 0 {
			totalBitrate += d.AvgBitrate * float64(d.PlaybackCount)
			bitrateCount += d.PlaybackCount
		}
		if strings.Contains(strings.ToLower(d.Codec), "atmos") {
			atmosCount += d.PlaybackCount
		}
	}

	if err = rows.Err(); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("error iterating codec rows: %w", err)
	}

	losslessAdoption := calculateAdoptionRate(losslessCount, total)

	avgBitrate := 0.0
	if bitrateCount > 0 {
		avgBitrate = totalBitrate / float64(bitrateCount)
	}

	return codecDist, losslessAdoption, avgBitrate, atmosCount, nil
}

// getAudioDownmixEvents retrieves audio downmix events where channels are reduced
func (db *DB) getAudioDownmixEvents(ctx context.Context, whereClause string, args []interface{}) ([]models.AudioDownmixEvent, error) {
	downmixCondition := "audio_channels IS NOT NULL AND stream_audio_channels IS NOT NULL AND audio_channels != stream_audio_channels"
	downmixWhere := appendWhereCondition(whereClause, downmixCondition)

	query := fmt.Sprintf(`
		SELECT
			audio_channels as source_channels,
			stream_audio_channels as stream_channels,
			COUNT(*) as occurrence_count,
			string_agg(DISTINCT platform, ',') as platforms,
			string_agg(DISTINCT username, ',') as users
		FROM playback_events
		%s
		GROUP BY audio_channels, stream_audio_channels
		ORDER BY occurrence_count DESC
		LIMIT 10
	`, downmixWhere)

	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query downmix events: %w", err)
	}
	defer rows.Close()

	var downmixEvents []models.AudioDownmixEvent
	for rows.Next() {
		var dm models.AudioDownmixEvent
		var platformsStr, usersStr string
		err := rows.Scan(&dm.SourceChannels, &dm.StreamChannels, &dm.OccurrenceCount, &platformsStr, &usersStr)
		if err != nil {
			return nil, fmt.Errorf("failed to scan downmix row: %w", err)
		}
		dm.Platforms = parseAggregatedList(platformsStr)
		dm.Users = parseAggregatedList(usersStr)
		downmixEvents = append(downmixEvents, dm)
	}

	return downmixEvents, rows.Err()
}
