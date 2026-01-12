# Phase 0b: Competitive Analysis Report

**Generated**: 2026-01-09
**Phase**: 0b - Competitive Analysis
**Last Updated**: 2026-01-09
**Status**: COMPLETE (Verified and Updated)

---

## Executive Summary

**Market Position**: Cartographus occupies a unique position between **Tautulli's comprehensive analytics** and **Plex Dash's mobile convenience**, with significant differentiation through **real-time geospatial visualization**, **multi-server support (Plex, Jellyfin, Emby)**, and **advanced security detection**.

**Key Differentiators** (Verified as Implemented):
- Real-time geospatial playback visualization (2D + 3D globe)
- Multi-server support: Plex, Jellyfin, Emby, and Tautulli integration
- 7 security detection rules (impossible travel, concurrent streams, device velocity, geo restriction, simultaneous locations, user agent anomaly, VPN usage)
- WebGL 3D visualizations via deck.gl
- Production-grade observability (Prometheus metrics)
- 299 REST API endpoints

**Also Implemented**:
- Recommendation engine (collaborative filtering algorithms)
- Newsletter system with customizable templates

**Roadmap Features** (Not Yet Implemented):
- Request management system (Sonarr/Radarr integration)
- Native mobile apps
- ML-powered anomaly predictions

**Recommendation**: Position as **"The Mission Control for Media Servers"** - combining comprehensive analytics with real-time operational awareness across all major media server platforms.

---

## 1. Competitive Landscape Overview

### 1.1 Competitor Categories

| Category | Competitors | Focus Area |
|----------|-------------|------------|
| **Monitoring & Analytics** | Tautulli | Deep historical analytics, notifications |
| **Official Dashboard** | Plex Dash | Mobile-first, real-time status |
| **Request Management** | Ombi, Overseerr, Jellyseerr | User requests, media discovery |
| **Alternatives** | Varken, Logarr | Alternative monitoring approaches |

### 1.2 Market Gaps Identified

**Opportunity Areas**:
1. **Geospatial Intelligence**: No competitor visualizes playback data on maps - **Cartographus fills this gap**
2. **Multi-Server Monitoring**: Competitors are single-source only - **Cartographus supports Plex + Jellyfin + Emby + Tautulli**
3. **Security Detection**: No competitor offers anomaly detection - **Cartographus has 7 detection rules**
4. **Advanced Visualization**: Basic charts elsewhere vs WebGL 3D visualizations - **Cartographus uses deck.gl**
5. **Real-Time Operations**: Polling-based (30-60s) elsewhere vs sub-second updates - **Cartographus uses WebSockets**
6. **Predictive Analytics**: No ML/AI capabilities anywhere - **Still a roadmap opportunity**

---

## 2. Competitor Analysis

### 2.1 Tautulli

**Sources**:
- [Tautulli GitHub](https://github.com/Tautulli/Tautulli) (~6.3k stars)
- [Tautulli Documentation](https://docs.tautulli.com/)
- [Tautulli Releases](https://github.com/Tautulli/Tautulli/releases)

#### Strengths

**Comprehensive Analytics**:
- Complete playback history with search/filtering
- Rich analytics using Highcharts graphing
- Global watching history with dynamic column sorting
- Complete library statistics and media file information
- 93 API endpoints

**Notification System**:
- Fully customizable notifications for stream activity
- Recently added media notifications
- Integration with 20+ notification agents
- Custom scripts support

**User Management**:
- Top statistics on home page with configurable duration
- User-level analytics
- Login history tracking

**Mature Ecosystem**:
- Active development (v2.16.x as of January 2026)
- Large user base and community support (~6.3k GitHub stars)
- Extensive API for third-party integrations
- Newsletter system

#### Weaknesses

**No Real-Time Visualization**:
- Static charts and tables
- No geospatial mapping
- Basic Highcharts graphs
- No advanced WebGL visualizations

**Polling-Based Architecture**:
- 30-60 second refresh intervals
- Not true real-time
- No WebSocket support for instant updates

**Limited Predictive Capabilities**:
- Historical analysis only
- No ML-powered recommendations
- No anomaly detection
- No capacity planning predictions

**Desktop-Focused UI**:
- Not mobile-optimized
- Python-based web interface
- No PWA support

**Security Concerns**:
- Recent CVEs (CVE-2025-58760 through CVE-2025-58763) in versions <=2.15.3
- Requires updates to v2.16.x+

**API Limitations**:
- Imgur hosting hits API limits
- Requires external image hosting (Cloudinary recommended)

#### Market Position

**Role**: The gold standard for Plex monitoring and analytics
**Users**: Power users, server administrators, data enthusiasts
**Pricing**: Free and open source
**GitHub Stars**: ~6.3k

---

### 2.2 Plex Dash

**Sources**:
- [Plex Server Status Dashboard](https://support.plex.tv/articles/200871837-status-and-dashboard/)
- [Plex Plans & Pricing](https://www.plex.tv/plans/)
- [Plex Dash App Store](https://apps.apple.com/us/app/plex-dash/id1500797677)

#### Strengths

**Official Integration**:
- Native Plex authentication
- Direct server access
- No third-party dependencies
- Plex Pass exclusive

**Mobile-First Design**:
- Native iOS and Android apps
- Gorgeous Now Playing tiles
- Artwork browser with swipe gestures
- Single-handed operation optimized

**Real-Time Monitoring**:
- Active playback across all servers
- Network, memory, and CPU graphs
- Real-time logs with filtering
- DVR recordings and Sync conversions

**Server Management**:
- Fix artwork directly from app
- Refresh libraries
- Inspect detailed media information
- Access raw server logs

**Modern UI**:
- Data-rich cards with UltraBlur backgrounds
- Unified view across multiple servers

#### Weaknesses

**Limited Analytics**:
- No historical playback data
- Basic current status only
- No trend analysis
- No predictive capabilities

**Missing Critical Features**:
- Cannot manage users
- No IP address visibility
- No bandwidth details per stream
- Cannot see if users are buffering
- No play time in seconds
- No device linking

**UI Problems**:
- Elements hidden behind status bar (reported)
- No desktop/web version
- Mobile-only limitation

**Plex Pass Required** (Updated Pricing as of April 2025):
- Monthly: $6.99 (was $4.99)
- Yearly: $69.99 (was $39.99)
- Lifetime: $249.99 (was $119.99)
- Significant paywall for basic monitoring

**No Geospatial Features**:
- No location-based visualization
- No geographic analytics

#### Market Position

**Role**: Official mobile monitoring tool for Plex Pass subscribers
**Users**: Casual Plex server owners, mobile-first users
**Pricing**: Requires Plex Pass ($6.99/month, $69.99/year, $249.99 lifetime)

---

### 2.3 Ombi

**Sources**:
- [Ombi GitHub](https://github.com/Ombi-app/Ombi)
- [Ombi Official Site](https://ombi.io/)
- [Ombi Documentation](https://docs.ombi.app/)

#### Strengths

**Multi-Source Support**:
- **Plex, Emby, AND Jellyfin** (unique capability)
- Single request system for multiple servers
- Unified user management across platforms

**Request Management**:
- Movies, TV shows (series/season/episodes), music
- Centralized request hub
- Automatic status updates when available

**Integration Ecosystem**:
- Sonarr, Radarr integration
- CouchPotato, SickRage support
- Automatic request fulfillment

**Mobile Support**:
- Native iOS app
- Native Android app

**Open Source**:
- Free software (v4.53.4 as of January 2026)
- Active development

#### Weaknesses

**Less Polished UI**:
- Criticized for dated interface
- Not as modern as Overseerr/Jellyseerr
- Complex setup process

**No Analytics**:
- Request management only
- No playback monitoring
- No geospatial features

**Request-Focused Only**:
- Not a monitoring tool
- No real-time playback tracking
- No server health monitoring

#### Market Position

**Role**: Multi-platform request management system
**Users**: Home server owners running multiple media servers
**Pricing**: Free and open source
**GitHub Stars**: ~3.8k
**Trend**: Losing ground to Jellyseerr for multi-platform users

---

### 2.4 Overseerr

**Sources**:
- [Overseerr GitHub](https://github.com/sct/overseerr) (~4.6k stars)
- [Overseerr Official Site](https://overseerr.dev/)
- [Overseerr API Docs](https://api-docs.overseerr.dev/)

#### Strengths

**Modern UI/UX**:
- React + Next.js architecture
- Polished, contemporary design
- Mobile-friendly responsive design

**Plex Integration**:
- Full Plex authentication
- User access management via Plex
- Library scan integration

**Request System**:
- Customizable request workflows
- Individual season/episode requests
- Granular permission system

**Service Integration**:
- Sonarr/Radarr integration
- Automated request fulfillment

**Active Development**:
- Regularly updated
- Strong community (~4.6k GitHub stars)

#### Weaknesses

**Plex-Only**:
- No Jellyfin support
- No Emby support
- Requires Plex ecosystem

**No Monitoring/Analytics**:
- Request management focused
- No playback analytics
- No server monitoring

**No Geospatial Features**:
- No location tracking
- No geographic visualization

#### Market Position

**Role**: Modern request management for Plex
**Users**: Plex server owners prioritizing UX
**Pricing**: Free and open source
**GitHub Stars**: ~4.6k
**Tech Stack**: React + Next.js + TypeScript

---

### 2.5 Jellyseerr

**Sources**:
- [Jellyseerr GitHub](https://github.com/Fallenbagel/jellyseerr)

#### Overview

Jellyseerr is a fork of Overseerr that adds support for **Jellyfin** and **Emby** in addition to Plex. It maintains most of Overseerr's features while extending compatibility to multiple media servers.

**Key Differentiator**: Multi-platform support (Plex + Jellyfin + Emby) with Overseerr's modern UI.

**Market Position**: Increasingly preferred over Ombi for multi-platform deployments due to more modern interface.

---

## 3. Feature Matrix Comparison

| Feature Category | Cartographus (Current) | Tautulli | Plex Dash | Ombi | Overseerr |
|------------------|-------------------------|----------|-----------|------|-----------|
| **Monitoring & Analytics** |
| Real-Time Playback Monitoring | WebSocket (<1s) | Polling (30-60s) | Real-time | None | None |
| Historical Analytics | 47+ charts | Comprehensive | None | None | None |
| Geospatial Visualization | 2D + 3D globe | None | None | None | None |
| Advanced Charts | 47+ ECharts | Basic Highcharts | Basic | None | None |
| WebGL 3D Visualizations | deck.gl 9.2.5 | None | None | None | None |
| Security Detection | 7 rules | None | None | None | None |
| **Data Sources** |
| Tautulli Integration | Full | 100% | None | None | None |
| Plex Direct Integration | Full | None | 100% | Basic | Full |
| Jellyfin Support | Full | None | None | Full | None |
| Emby Support | Full | None | None | Full | None |
| Multi-Source Dedup | Yes | None | None | None | None |
| **User Experience** |
| Web Interface | Modern | Dated | None | Yes | Modern |
| Mobile Apps | PWA | None | Native | Native | Responsive |
| Dark/Light/High-Contrast | WCAG AAA | Basic | Basic | Basic | Basic |
| Mobile-Responsive | 7 breakpoints | No | Yes | Partial | Yes |
| **Enterprise Features** |
| API Endpoints | 299 | 93 | Limited | Yes | Yes |
| Prometheus Metrics | Yes | None | None | None | None |
| Observability | Full | None | None | None | None |
| **Management** |
| Request Management | Roadmap | None | None | Full | Full |
| User Management | JWT + OAuth | Basic | None | Full | Full |
| **Technical** |
| Self-Hosted | Yes | Yes | No (cloud) | Yes | Yes |
| Open Source | Yes | Yes | No | Yes | Yes |
| Docker Support | Multi-arch | Yes | N/A | Yes | Yes |
| Database | DuckDB | SQLite | N/A | MySQL/SQLite | SQLite |
| Test Coverage | 10601 unit, 1300+ E2E | N/A | N/A | N/A | N/A |

**Legend**:
- Full = Complete implementation
- Basic = Limited functionality
- None = Not supported
- Roadmap = Planned feature

---

## 4. Differentiation Strategy

### 4.1 Unique Selling Propositions (USPs)

**Position**: **"Mission Control for Media Servers"**

#### USP 1: Real-Time Geospatial Intelligence

**What**: WebGL-powered 2D/3D visualization of playback activity on interactive maps

**Why Unique**: **No competitor offers this**
- Tautulli: Static tables and basic charts
- Plex Dash: Current status only, no maps
- Ombi/Overseerr: Request management, no monitoring

**Value Proposition**:
- See **WHERE** your content is being consumed globally
- Identify geographic usage patterns
- Detect unusual access locations (security)
- Optimize CDN and infrastructure placement

**Technical Differentiator**:
- deck.gl 9.2.5 for GPU-accelerated rendering
- Handles 10,000+ location markers at 60 FPS
- Real-time updates via WebSocket (<1s latency)
- MapLibre GL JS 5.15.0 for map rendering

---

#### USP 2: Multi-Server Platform Support

**What**: Unified analytics from Tautulli + Plex + Jellyfin + Emby with intelligent deduplication

**Why Unique**: **Only Ombi supports multi-platform, but for requests only**
- Tautulli: Plex-only via Tautulli
- Plex Dash: Plex-only
- Overseerr: Plex-only
- Ombi: Multi-platform but NO analytics

**Value Proposition** (IMPLEMENTED):
- **Best of all worlds**: Tautulli's rich metadata + Plex/Jellyfin/Emby real-time updates
- **Unified dashboard**: Single view across all media servers
- **Migration flexibility**: Switch servers without losing analytics history
- **Data resilience**: If one source fails, others continue

**Technical Differentiator**:
- Sophisticated merge logic with deduplication
- Per-server circuit breakers for resilience
- WebSocket connections to all platforms
- Real-time session polling with configurable intervals

---

#### USP 3: Security Detection Engine

**What**: 7 detection rules for anomaly detection and security monitoring

**Why Unique**: **No competitor has security detection**

**Detection Rules** (IMPLEMENTED):
1. **Impossible Travel**: Detects physically impossible location changes
2. **Concurrent Streams**: Monitors unusual simultaneous stream counts
3. **Device Velocity**: Tracks rapid device switching patterns
4. **Geo Restriction**: Enforces geographic access policies
5. **Simultaneous Locations**: Detects account sharing across locations
6. **User Agent Anomaly**: Identifies suspicious client patterns
7. **VPN Usage**: Detects VPN/proxy access attempts

**Value Proposition**:
- **Proactive security**: Detect account sharing and suspicious access
- **Configurable rules**: Enable/disable and tune each detector
- **Real-time alerts**: Immediate notification of anomalies
- **Audit trail**: Complete detection history

---

#### USP 4: Production-Grade Observability

**What**: Prometheus metrics, structured logging, comprehensive API

**Why Unique**: **No competitor offers enterprise observability**

**Value Proposition** (IMPLEMENTED):
- **Prometheus metrics**: Export to Grafana for dashboards
- **299 API endpoints**: Comprehensive programmatic access
- **Structured logging**: JSON logs for aggregation
- **Health endpoints**: Kubernetes-ready liveness/readiness probes

---

#### USP 5: Advanced WebGL Visualizations

**What**: 3D globe, hexagonal heatmaps, animated timelines

**Why Unique**: **No competitor uses WebGL for visualization**
- Tautulli: Basic Highcharts
- Plex Dash: Simple graphs
- Others: No visualizations

**Technical Differentiator**:
- deck.gl 9.2.5 ecosystem
- 60 FPS rendering with 100k+ data points
- GPU-accelerated aggregation
- Hardware-accelerated 3D globe

---

### 4.2 Target Market Segmentation

#### Segment 1: Power Users & Data Enthusiasts

**Current**: Using Tautulli + custom scripts
**Pain Point**: No geospatial insights, no security detection
**Value Prop**: Advanced analytics + security detection + multi-server
**Win Strategy**: Superior visualization + detection rules

#### Segment 2: Multi-Server Operators

**Current**: Running Plex + Jellyfin/Emby, using separate tools
**Pain Point**: Fragmented monitoring, no unified view
**Value Prop**: Single pane of glass for all servers
**Win Strategy**: Multi-source support (unique in monitoring space)

#### Segment 3: Security-Conscious Admins

**Current**: No tools for detecting account sharing
**Pain Point**: Unknown access patterns, potential abuse
**Value Prop**: 7 detection rules for security monitoring
**Win Strategy**: Security features no competitor offers

#### Segment 4: Enterprise / Heavy Users

**Current**: Managing large libraries, many users
**Pain Point**: No observability, no metrics export
**Value Prop**: Prometheus metrics, comprehensive API
**Win Strategy**: Production-grade monitoring features

---

## 5. Feature Gap Analysis

### 5.1 Features We Have That Others Don't

**Real-time geospatial visualization** (2D + 3D globe)
**Multi-server support** (Plex + Jellyfin + Emby + Tautulli)
**7 security detection rules** (unique in the market)
**Recommendation engine** (collaborative filtering algorithms)
**Newsletter system** (customizable templates, scheduled delivery)
**WebSocket sub-second updates** (vs 30-60s polling)
**47+ advanced ECharts** across analytics pages
**DuckDB analytics** (spatial queries, window functions)
**WCAG 2.1 AAA accessibility**
**Progressive Web App** with offline support
**Multi-arch Docker** (amd64, arm64)
**Production-grade testing** (10601 unit tests, 1300+ E2E tests)
**299 REST API endpoints**
**Prometheus metrics export**

### 5.2 Features We Need (Competitor Parity)

#### From Tautulli:
- Custom scripts support
- 20+ notification agents (we have Discord + webhook + newsletter)

#### From Plex Dash:
- Native mobile apps (iOS, Android) - we have PWA only
- DVR recording status visualization
- Sync conversion status

#### From Ombi/Overseerr:
- Request management system
- User permission granularity
- Approval workflows
- Sonarr/Radarr integration

#### Roadmap Features:
- Capacity forecasting
- Churn prediction
- Plugin architecture
- Internationalization (i18n)

---

## 6. Strategic Recommendations

### 6.1 Short-Term (Next 3-6 months)

**Priority 1**: Request Management System
- Implement basic request workflow
- Sonarr/Radarr integration
- User permissions
- **Impact**: Compete with Ombi/Overseerr for comprehensive solution

**Priority 2**: Notification Expansion
- Additional notification agents (Slack, email, Telegram, Pushover)
- Customizable notification templates
- **Impact**: Match Tautulli's notification ecosystem

**Priority 3**: Native Mobile Apps (or Enhanced PWA)
- Push notifications for security alerts
- Offline viewing of analytics
- **Impact**: Better mobile experience than PWA alone

### 6.2 Medium-Term (6-12 months)

**Priority 4**: Enhanced ML Capabilities
- Churn prediction models
- Capacity forecasting
- Anomaly prediction (complementing existing detection rules)
- **Impact**: Further differentiation beyond existing recommendation engine

**Priority 5**: Plugin Architecture
- Plugin SDK for community extensions
- Custom dashboard widgets
- **Impact**: Ecosystem growth

### 6.3 Long-Term (12-24 months)

**Priority 6**: Enterprise Features
- Multi-tenancy
- High availability (read replicas)
- RBAC enhancements
- **Impact**: Enterprise adoption

---

## 7. Risk Assessment

### 7.1 Competitive Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **Tautulli adds geospatial viz** | Low | High | Already established as geo leader |
| **Plex Dash adds analytics** | Medium | High | Focus on multi-server support |
| **Overseerr adds monitoring** | Low | Medium | Already differentiated in analytics |
| **New competitor emerges** | Medium | Medium | Open source + fast iteration |
| **Plex blocks third-party APIs** | Low | High | Multi-server reduces dependency |

### 7.2 Technical Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| **API changes in Plex/Jellyfin/Emby** | Medium | High | Circuit breakers, version detection |
| **DuckDB limitations at scale** | Low | Medium | Consider MotherDuck for distributed |
| **Frontend complexity grows** | Medium | Medium | Modular architecture maintained |

---

## 8. Go-to-Market Strategy

### 8.1 Positioning Statement

**For**: Power users and enterprise media server operators
**Who**: Need advanced analytics, real-time monitoring, and security detection across multiple platforms
**Cartographus**: Is a geospatial analytics platform
**That**: Unifies Plex, Jellyfin, and Emby monitoring with security detection and advanced visualization
**Unlike**: Tautulli (Plex-only, no geospatial), Plex Dash (no analytics), Ombi/Overseerr (no monitoring)
**We Provide**: Mission Control for your media infrastructure with multi-server support and security detection

### 8.2 Key Messages

1. **"See Your Media World in Real-Time"** - Geospatial visualization
2. **"One Dashboard, All Your Servers"** - Multi-server support
3. **"Know Who's Watching From Where"** - Security detection
4. **"From Hobbyist to Enterprise"** - Scalability

### 8.3 Launch Strategy

**Phase 1**: Community Growth (Current)
- Open source GitHub presence
- Community documentation
- Reddit/Discord engagement

**Phase 2**: Feature Parity (Next 6 months)
- Request management system
- Enhanced notifications
- Compete with Overseerr + Tautulli combined

**Phase 3**: Differentiation (12+ months)
- ML/prediction models
- Advanced visualizations
- Position as "the only intelligent multi-platform solution"

---

## 9. Success Metrics

### 9.1 Adoption Metrics

- **GitHub Stars**: Target 2,000+ (Overseerr: ~4.6k, Tautulli: ~6.3k)
- **Docker Pulls**: Target 100k+ in first year
- **Active Installs**: Target 5,000+ active deployments

### 9.2 Feature Metrics

- **Multi-Server Adoption**: 40% of users using 2+ servers by end of year
- **Detection Rules Usage**: 60%+ enabling at least one detection rule
- **API Usage**: 30%+ of users leveraging API integrations

### 9.3 Technical Metrics

- **Performance**: p95 API latency < 100ms (currently <30ms)
- **Test Coverage**: Maintain 10601+ unit tests, 1300+ E2E tests
- **Reliability**: 99.9% uptime for self-hosted instances

---

## 10. Conclusion

**Market Opportunity**: **SIGNIFICANT**

Cartographus has achieved meaningful differentiation through:
- **Unique geospatial visualization** (no competitor offers this)
- **Multi-server support** (Plex + Jellyfin + Emby + Tautulli)
- **Security detection** (7 rules, unique in market)
- **Production-grade quality** (10601 unit tests, 299 API endpoints)

**Current Strengths**:
- Real-time WebSocket monitoring (vs polling)
- Unique geospatial visualization
- Multi-platform support (Plex + Jellyfin + Emby + Tautulli)
- Security detection rules (7 implemented)
- Recommendation engine (collaborative filtering)
- Newsletter system (implemented)
- Comprehensive API (299 endpoints)

**Path Forward**:
1. **Add request management** (compete with Overseerr)
2. **Expand notification agents** (compete with Tautulli's 20+ agents)
3. **Enhance ML capabilities** (build on existing recommendation engine)
4. **Native mobile apps** (compete with Plex Dash)

**Positioning**: **"Mission Control for Media Servers"** - the intelligent, geospatial, multi-platform analytics solution that no competitor offers.

---

## Sources

- [Tautulli GitHub](https://github.com/Tautulli/Tautulli) (~6.3k stars)
- [Tautulli Documentation](https://docs.tautulli.com/)
- [Tautulli Releases](https://github.com/Tautulli/Tautulli/releases)
- [Plex Plans & Pricing](https://www.plex.tv/plans/)
- [Plex Dash App Store](https://apps.apple.com/us/app/plex-dash/id1500797677)
- [Ombi GitHub](https://github.com/Ombi-app/Ombi) (~3.8k stars)
- [Ombi Releases](https://github.com/Ombi-app/Ombi/releases)
- [Overseerr GitHub](https://github.com/sct/overseerr) (~4.6k stars)
- [Overseerr Official Site](https://overseerr.dev/)
- [Jellyseerr GitHub](https://github.com/Fallenbagel/jellyseerr)

---

**Document History**:
- 2025-11-24: Initial version
- 2026-01-09: Comprehensive update with verified feature claims, updated competitor information, corrected statistics
