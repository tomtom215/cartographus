// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

//go:build nats

package main

import (
	"context"
	"testing"
	"time"
)

// TestNATSComponents_IsRunning tests the IsRunning method.
func TestNATSComponents_IsRunning(t *testing.T) {
	t.Run("nil components", func(t *testing.T) {
		var c *NATSComponents
		if c.IsRunning() {
			t.Error("IsRunning() should return false for nil components")
		}
	})

	t.Run("not running", func(t *testing.T) {
		c := &NATSComponents{}
		if c.IsRunning() {
			t.Error("IsRunning() should return false when not running")
		}
	})

	t.Run("running", func(t *testing.T) {
		c := &NATSComponents{running: true}
		if !c.IsRunning() {
			t.Error("IsRunning() should return true when running")
		}
	})
}

// TestNATSComponents_Shutdown tests the Shutdown method.
func TestNATSComponents_Shutdown(t *testing.T) {
	t.Run("nil components", func(t *testing.T) {
		var c *NATSComponents
		// Should not panic
		c.Shutdown(context.Background())
	})

	t.Run("not running", func(t *testing.T) {
		c := &NATSComponents{}
		// Should not panic
		c.Shutdown(context.Background())
	})

	t.Run("shutdown completes", func(t *testing.T) {
		c := &NATSComponents{
			running:          true,
			shutdownComplete: make(chan struct{}),
		}

		done := make(chan struct{})
		go func() {
			c.Shutdown(context.Background())
			close(done)
		}()

		select {
		case <-done:
			// Good - shutdown completed
		case <-time.After(time.Second):
			t.Error("Shutdown blocked for too long")
		}

		if c.IsRunning() {
			t.Error("Should not be running after shutdown")
		}
	})
}

// TestNATSComponents_Start tests the Start method.
func TestNATSComponents_Start(t *testing.T) {
	t.Run("nil components", func(t *testing.T) {
		var c *NATSComponents
		err := c.Start(context.Background())
		if err != nil {
			t.Errorf("Start() should return nil for nil components, got %v", err)
		}
	})

	t.Run("nil subscriber", func(t *testing.T) {
		c := &NATSComponents{}
		err := c.Start(context.Background())
		if err != nil {
			t.Errorf("Start() should return nil for nil subscriber, got %v", err)
		}
	})
}

// Note: TestMessageHandlerAdapter_Close was removed when MessageHandlerAdapter
// was replaced by Router-based message handling in the Watermill migration.
