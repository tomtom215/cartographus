// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

// Package vpn provides VPN IP detection and lookup services for Cartographus.
//
// This package enables detection of connections from known VPN providers to
// improve geolocation accuracy and flag potentially misleading analytics data.
// VPN connections can significantly skew geographic analytics since the apparent
// location reflects the VPN server, not the actual user location.
//
// # Overview
//
// The VPN detection system consists of several components:
//
//   - Lookup: In-memory IP lookup using hash maps for O(1) exact IP matching
//   - Importer: Imports VPN server data from gluetun format (24+ providers)
//   - Store: DuckDB-backed persistence for VPN IP data
//   - Service: High-level API combining all components
//
// # Data Source
//
// The primary data source is the gluetun project's servers.json file:
// https://github.com/qdm12/gluetun/blob/master/internal/storage/servers.json
//
// This file contains 10,000+ IP addresses across 24+ VPN providers including:
// NordVPN, ExpressVPN, Mullvad, ProtonVPN, Surfshark, and many more.
//
// # Usage
//
// Basic usage:
//
//	// Create and initialize the service
//	svc, err := vpn.NewService(db, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Initialize (creates tables, loads existing data)
//	if err := svc.Initialize(ctx); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Import VPN data from file
//	result, err := svc.ImportFromFile(ctx, "servers.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Imported %d IPs from %d providers", result.IPsImported, result.ProvidersImported)
//
//	// Check if an IP is a VPN
//	if svc.IsVPN("198.51.100.1") {
//	    log.Println("VPN detected!")
//	}
//
//	// Get detailed VPN information
//	result := svc.LookupIP("198.51.100.1")
//	if result.IsVPN {
//	    log.Printf("Provider: %s, Server: %s, %s",
//	        result.ProviderDisplayName,
//	        result.ServerCity,
//	        result.ServerCountry)
//	}
//
// # Integration with Detection Engine
//
// The VPN detection integrates with the Cartographus detection engine via
// the VPNUsageDetector rule type. This allows automatic alerting when users
// stream via VPN connections:
//
//	// Create VPN usage detector
//	detector := detection.NewVPNUsageDetector(vpnService)
//
//	// Add to detection engine
//	engine.RegisterDetector(detector)
//
// # Data Model
//
// The gluetun JSON format is:
//
//	{
//	    "provider_name": {
//	        "version": 1,
//	        "timestamp": 1721997873,
//	        "servers": [
//	            {
//	                "vpn": "wireguard",
//	                "country": "Austria",
//	                "region": "Europe",
//	                "city": "Vienna",
//	                "hostname": "at.vpn.airdns.org",
//	                "ips": ["203.0.113.1", "2001:db8::1"]
//	            }
//	        ]
//	    }
//	}
//
// # Performance
//
// The lookup system uses hash maps for O(1) exact IP matching:
//   - Separate maps for IPv4 and IPv6 addresses
//   - Typical memory usage: ~2MB per 10,000 IPs
//   - Lookup time: <1 microsecond per IP
//
// # Future Enhancements
//
// Planned features:
//   - CIDR range support for more comprehensive coverage
//   - Automatic updates from gluetun repository
//   - Additional data sources (IP2Proxy, ipapi.is)
//   - ASN-based detection for VPN provider networks
package vpn
