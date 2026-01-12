// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package tautulli

// TautulliGeoIP represents the API response from Tautulli's get_geoip_lookup endpoint
type TautulliGeoIP struct {
	Response TautulliGeoIPResponse `json:"response"`
}

type TautulliGeoIPResponse struct {
	Result  string            `json:"result"`
	Message *string           `json:"message,omitempty"`
	Data    TautulliGeoIPData `json:"data"`
}

type TautulliGeoIPData struct {
	City           string  `json:"city"`
	Region         string  `json:"region"`
	Country        string  `json:"country"`
	PostalCode     string  `json:"postal_code"`
	Timezone       string  `json:"timezone"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyRadius int     `json:"accuracy_radius"`
}
