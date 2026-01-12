// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliStreamData represents the API response from Tautulli's get_stream_data endpoint
type TautulliStreamData struct {
	Response TautulliStreamDataResponse `json:"response"`
}

type TautulliStreamDataResponse struct {
	Result  string                 `json:"result"`
	Message *string                `json:"message,omitempty"`
	Data    TautulliStreamDataInfo `json:"data"`
}

type TautulliStreamDataInfo struct {
	SessionKey              string `json:"session_key"`
	TranscodeDecision       string `json:"transcode_decision"`
	VideoDecision           string `json:"video_decision"`
	AudioDecision           string `json:"audio_decision"`
	SubtitleDecision        string `json:"subtitle_decision,omitempty"`
	Container               string `json:"container"`
	Bitrate                 int    `json:"bitrate"`
	VideoCodec              string `json:"video_codec"`
	VideoResolution         string `json:"video_resolution"`
	VideoWidth              int    `json:"video_width"`
	VideoHeight             int    `json:"video_height"`
	VideoFramerate          string `json:"video_framerate"`
	VideoBitrate            int    `json:"video_bitrate"`
	AudioCodec              string `json:"audio_codec"`
	AudioChannels           int    `json:"audio_channels"`
	AudioChannelLayout      string `json:"audio_channel_layout,omitempty"`
	AudioBitrate            int    `json:"audio_bitrate"`
	AudioSampleRate         int    `json:"audio_sample_rate"`
	StreamContainer         string `json:"stream_container,omitempty"`
	StreamContainerDecision string `json:"stream_container_decision,omitempty"`
	StreamBitrate           int    `json:"stream_bitrate,omitempty"`
	StreamVideoCodec        string `json:"stream_video_codec,omitempty"`
	StreamVideoResolution   string `json:"stream_video_resolution,omitempty"`
	StreamVideoBitrate      int    `json:"stream_video_bitrate,omitempty"`
	StreamVideoWidth        int    `json:"stream_video_width,omitempty"`
	StreamVideoHeight       int    `json:"stream_video_height,omitempty"`
	StreamVideoFramerate    string `json:"stream_video_framerate,omitempty"`
	StreamAudioCodec        string `json:"stream_audio_codec,omitempty"`
	StreamAudioChannels     int    `json:"stream_audio_channels,omitempty"`
	StreamAudioBitrate      int    `json:"stream_audio_bitrate,omitempty"`
	StreamAudioSampleRate   int    `json:"stream_audio_sample_rate,omitempty"`
	SubtitleCodec           string `json:"subtitle_codec,omitempty"`
	Optimized               int    `json:"optimized,omitempty"`
	Throttled               int    `json:"throttled,omitempty"`
}
