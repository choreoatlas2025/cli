// SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
// SPDX-License-Identifier: Apache-2.0
//go:build ce
// +build ce

package telemetry

import "context"

// Event telemetry event structure (CE version)
type Event struct {
	Command    string `json:"command"`
	DurationMS int64  `json:"duration_ms"`
	Result     string `json:"result"`
	Version    string `json:"version"`
	Edition    string `json:"edition"`
}

// Client telemetry client (CE empty implementation)
type Client struct{}

// New creates a new telemetry client
func New() *Client { 
	return &Client{} 
}

// NewClient creates a new telemetry client (compatibility)
func NewClient() *Client { 
	return &Client{} 
}

// RecordEvent records an event (empty implementation for CE)
func (c *Client) RecordEvent(ctx context.Context, event Event) error { 
	return nil 
}

// Close closes the client (empty implementation for CE)
func (c *Client) Close() error { 
	return nil 
}

// IsEnabled checks if telemetry is enabled (always false for CE)
func (c *Client) IsEnabled() bool { 
	return false 
}

// Enable enables telemetry (empty implementation for CE)
func (c *Client) Enable(ctx context.Context) error { 
	return nil 
}

// TrackEvent tracks an event (empty implementation for CE)
func (c *Client) TrackEvent(ctx context.Context, name string, kv map[string]any) {}

// Flush flushes telemetry data (empty implementation for CE)
func (c *Client) Flush(ctx context.Context) {}