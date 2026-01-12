// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package detection provides real-time streaming activity detection rules
// for identifying suspicious playback patterns such as impossible travel,
// concurrent stream abuse, and device velocity anomalies.
//
// Detection Architecture:
//
//	MediaEvent -> Detection Engine -> Alert -> Notification System
//	               |                    |
//	               v                    v
//	         Rule Evaluators     WebSocket/Discord/Webhook
//
// The detection engine integrates with the existing NATS JetStream event
// pipeline via Watermill handlers. Each playback event is evaluated against
// configured detection rules, generating alerts when violations are detected.
//
// Supported Detection Rules:
//   - Impossible Travel: Detects when a user streams from geographically
//     distant locations faster than physically possible
//   - Concurrent Streams: Enforces per-user stream limits
//   - Device Velocity: Flags devices appearing from multiple IPs rapidly
//   - Geo Restrictions: Blocks streaming from specified countries (future)
//
// Trust Scoring:
// Each user maintains a trust score (0-100) that decreases with violations
// and gradually recovers over time. Low trust scores can trigger automatic
// restrictions.
//
// See ADR-0020 (pending) for architectural decisions.
package detection
