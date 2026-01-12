// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package eventprocessor provides common error definitions.
package eventprocessor

import "errors"

// ErrNATSNotEnabled is returned when NATS features are used without the nats build tag.
var ErrNATSNotEnabled = errors.New("NATS event processing not enabled (build with -tags nats)")

// ErrNilPublisher is returned when attempting to create a publisher with nil input.
var ErrNilPublisher = errors.New("publisher cannot be nil")

// ErrStreamNotFound is returned when the NATS stream doesn't exist.
var ErrStreamNotFound = errors.New("stream not found")

// ErrInvalidConfig is returned when configuration is invalid.
var ErrInvalidConfig = errors.New("invalid configuration")
