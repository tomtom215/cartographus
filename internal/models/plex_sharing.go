// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

/*
plex_sharing.go - Plex Sharing API Models

This file provides request and response types for the Plex friends and sharing
API endpoints with validation using go-playground/validator.

Features:
  - Friend invitation with email validation
  - Library sharing with section IDs
  - Managed user creation with restriction profiles
  - API response wrappers for consistent formatting
*/

package models

import "time"

// ============================================================================
// Friend Management
// ============================================================================

// PlexFriendResponse represents a friend in API responses
type PlexFriendResponse struct {
	ID                int64  `json:"id"`
	UUID              string `json:"uuid"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	Thumb             string `json:"thumb"`
	Title             string `json:"title"`
	Server            bool   `json:"server"`
	Home              bool   `json:"home"`
	AllowSync         bool   `json:"allowSync"`
	AllowCameraUpload bool   `json:"allowCameraUpload"`
	AllowChannels     bool   `json:"allowChannels"`
	SharedSections    []int  `json:"sharedSections"`
	FilterMovies      string `json:"filterMovies,omitempty"`
	FilterTelevision  string `json:"filterTelevision,omitempty"`
	FilterMusic       string `json:"filterMusic,omitempty"`
	Status            string `json:"status"` // "accepted", "pending", "pending_received"
}

// PlexFriendsListResponse is the response for listing friends
type PlexFriendsListResponse struct {
	Friends []PlexFriendResponse `json:"friends"`
	Total   int                  `json:"total"`
}

// PlexInviteFriendRequest represents a friend invitation request
type PlexInviteFriendRequest struct {
	Email             string `json:"email" validate:"required,email"`
	AllowSync         bool   `json:"allowSync"`
	AllowCameraUpload bool   `json:"allowCameraUpload"`
	AllowChannels     bool   `json:"allowChannels"`
}

// ============================================================================
// Library Sharing
// ============================================================================

// PlexSharedServerResponse represents a shared server entry in API responses
type PlexSharedServerResponse struct {
	ID                int64     `json:"id"`
	UserID            int64     `json:"userId"`
	Username          string    `json:"username"`
	Email             string    `json:"email"`
	Thumb             string    `json:"thumb"`
	InvitedEmail      string    `json:"invitedEmail,omitempty"`
	AcceptedAt        time.Time `json:"acceptedAt,omitempty"`
	AllowSync         bool      `json:"allowSync"`
	AllowCameraUpload bool      `json:"allowCameraUpload"`
	AllowChannels     bool      `json:"allowChannels"`
	FilterMovies      string    `json:"filterMovies,omitempty"`
	FilterTelevision  string    `json:"filterTelevision,omitempty"`
	FilterMusic       string    `json:"filterMusic,omitempty"`
	SharedSections    []int     `json:"sharedSections"`
}

// PlexSharedServersListResponse is the response for listing shared servers
type PlexSharedServersListResponse struct {
	SharedServers []PlexSharedServerResponse `json:"sharedServers"`
	Total         int                        `json:"total"`
}

// PlexShareLibrariesRequest represents a library sharing request
type PlexShareLibrariesRequest struct {
	Email             string `json:"email" validate:"required,email"`
	LibrarySectionIDs []int  `json:"librarySectionIds" validate:"required,min=1,dive,min=1"`
	AllowSync         bool   `json:"allowSync"`
	AllowCameraUpload bool   `json:"allowCameraUpload"`
	AllowChannels     bool   `json:"allowChannels"`
	FilterMovies      string `json:"filterMovies,omitempty"`
	FilterTelevision  string `json:"filterTelevision,omitempty"`
	FilterMusic       string `json:"filterMusic,omitempty"`
}

// PlexUpdateSharingRequest represents a request to update sharing settings
type PlexUpdateSharingRequest struct {
	LibrarySectionIDs []int  `json:"librarySectionIds" validate:"required,min=1,dive,min=1"`
	AllowSync         *bool  `json:"allowSync,omitempty"`
	AllowCameraUpload *bool  `json:"allowCameraUpload,omitempty"`
	AllowChannels     *bool  `json:"allowChannels,omitempty"`
	FilterMovies      string `json:"filterMovies,omitempty"`
	FilterTelevision  string `json:"filterTelevision,omitempty"`
	FilterMusic       string `json:"filterMusic,omitempty"`
}

// ============================================================================
// Managed Users (Plex Home)
// ============================================================================

// PlexManagedUserResponse represents a managed user in API responses
type PlexManagedUserResponse struct {
	ID                 int64  `json:"id"`
	UUID               string `json:"uuid"`
	Username           string `json:"username"`
	Title              string `json:"title"`
	Thumb              string `json:"thumb,omitempty"`
	Restricted         bool   `json:"restricted"`
	RestrictionProfile string `json:"restrictionProfile"` // "little_kid", "older_kid", "teen", ""
	Home               bool   `json:"home"`
	HomeAdmin          bool   `json:"homeAdmin"`
	Guest              bool   `json:"guest"`
	Protected          bool   `json:"protected"` // Has PIN
}

// PlexManagedUsersListResponse is the response for listing managed users
type PlexManagedUsersListResponse struct {
	Users []PlexManagedUserResponse `json:"users"`
	Total int                       `json:"total"`
}

// PlexCreateManagedUserRequest represents a request to create a managed user
type PlexCreateManagedUserRequest struct {
	Name               string `json:"name" validate:"required,min=1,max=50"`
	RestrictionProfile string `json:"restrictionProfile" validate:"omitempty,oneof=little_kid older_kid teen"`
}

// PlexUpdateManagedUserRequest represents a request to update a managed user
type PlexUpdateManagedUserRequest struct {
	RestrictionProfile string `json:"restrictionProfile" validate:"required,oneof=little_kid older_kid teen ''"`
}

// ============================================================================
// Library Information
// ============================================================================

// PlexLibrarySectionResponse represents a library section for sharing UI
type PlexLibrarySectionResponse struct {
	ID        int    `json:"id"`
	Key       string `json:"key"`
	Type      string `json:"type"` // "movie", "show", "artist", "photo"
	Title     string `json:"title"`
	Thumb     string `json:"thumb,omitempty"`
	ItemCount int    `json:"itemCount"`
}

// PlexLibrarySectionsListResponse is the response for listing library sections
type PlexLibrarySectionsListResponse struct {
	Sections []PlexLibrarySectionResponse `json:"sections"`
	Total    int                          `json:"total"`
}

// ============================================================================
// Server Identity
// ============================================================================

// PlexServerIdentityResponse represents server identity information
type PlexServerIdentityResponse struct {
	MachineIdentifier string `json:"machineIdentifier"`
	Version           string `json:"version"`
	Platform          string `json:"platform"`
	PlatformVersion   string `json:"platformVersion"`
	FriendlyName      string `json:"friendlyName"`
}

// ============================================================================
// Error Response
// ============================================================================

// PlexErrorResponse represents an error from the Plex API
type PlexErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}
