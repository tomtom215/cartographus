# Testing Guide

Comprehensive testing infrastructure for Cartographus with E2E tests and API documentation.

## Table of Contents

- [End-to-End (E2E) Tests](#end-to-end-e2e-tests)
- [Unit Tests](#unit-tests)
- [API Documentation](#api-documentation)
- [Running Tests](#running-tests)

## End-to-End (E2E) Tests

### Overview

E2E tests use **Playwright** to test the full application stack with real browser interactions. Tests cover critical user flows and ensure the application works correctly from a user's perspective.

### Test Coverage

**Total**: 22 test suites, 337+ test cases

**Cross-Browser Coverage**:
- ✅ Chromium (Chrome, Edge)
- ✅ Firefox (with WebGL support)
- ✅ WebKit (Safari)

**Mobile Device Coverage**:
- ✅ iPhone 12 (390x844)
- ✅ Pixel 5 (393x851)
- ✅ iPad (810x1080)
- ✅ 7 responsive breakpoints (320px - 1920px)

#### 1. Authentication Flow (`tests/e2e/01-login.spec.ts`)
- Login form rendering and validation
- Credential validation (valid/invalid)
- JWT token issuance and storage
- Session persistence with "Remember Me"
- Protected route access
- Logout functionality
- Password show/hide toggle

#### 2. Chart Rendering (`tests/e2e/02-charts.spec.ts`)
- Chart rendering on main dashboard
- Chart interactions (hover, tooltips)
- PNG export functionality
- Filter-driven chart updates
- Responsive chart sizing
- Graceful handling of empty data
- Lazy loading performance

#### 3. Map Visualization (`tests/e2e/03-map.spec.ts`)
- MapLibre GL map rendering
- Navigation controls (zoom, pan, compass)
- Marker and cluster rendering
- Popup interactions on marker click
- Cluster expansion on click
- Visualization mode switching (points/clusters/heatmap)
- Heatmap layer toggling
- Filter-driven map updates
- Responsive map behavior
- Performance with large datasets

#### 4. Filter Application (`tests/e2e/04-filters.spec.ts`)
- Filter panel rendering
- Date range presets (24h, 7d, 30d, etc.)
- Custom date range selection
- User filter (multi-select)
- Media type filter (multi-select)
- Multiple simultaneous filters
- URL query parameter sync
- Filter restoration from URL
- Clear all filters
- Debounced filter updates (300ms)
- Stats/charts/map updates when filters change
- Graceful handling of no matching data

#### 5. WebSocket Real-Time Updates (`tests/e2e/05-websocket.spec.ts`)
- WebSocket connection establishment
- Connection status indicators
- Toast notifications for new playbacks
- Stats auto-refresh on new data
- Graceful disconnection handling
- Auto-reconnection with exponential backoff
- Ping/pong connection health monitoring
- Playback notification data parsing
- Fallback when WebSocket unavailable
- Performance with rapid messages
- Clean connection closure on page unload
- Secure WebSocket (wss://) on HTTPS

#### 6. 3D Globe Visualization (`tests/e2e/06-globe-deckgl.spec.ts`)
- deck.gl 3D globe rendering
- Globe rotation and zoom controls
- ScatterplotLayer for playback locations
- ArcLayer for user-server connections
- Globe toggle and controls

#### 7. Enhanced Globe Features (`tests/e2e/07-globe-enhanced.spec.ts`)
- HexagonLayer for density visualization
- TripsLayer for temporal animation
- Timeline widget controls
- Screenshot widget functionality
- Layer toggling and visibility
- Advanced WebGL features

#### 8. Live Activity Monitoring (`tests/e2e/08-live-activity.spec.ts`)
- Real-time playback monitoring
- Live stream indicators
- Activity feed updates
- Stream metadata display

#### 9. Recently Added Content (`tests/e2e/09-recently-added.spec.ts`)
- Content timeline rendering
- Infinite scroll functionality
- Media metadata display
- Filter integration

#### 10. Server Information (`tests/e2e/10-server-info.spec.ts`)
- Server health metrics
- Tautulli connection status
- Database statistics
- System information display

#### 11. Data Export (`tests/e2e/11-data-export.spec.ts`) - 18 tests
- CSV export for playback data
- GeoJSON export for location data
- Export with applied filters
- Export validation and format checking
- Large dataset export handling

#### 12. Analytics Pages (`tests/e2e/12-analytics-pages.spec.ts`) - 15 tests
- Navigation across all 6 analytics pages:
  - **Overview** (6 charts): Trends, media, users, countries, platforms, heatmap
  - **Content** (8 charts): Libraries, popular movies/shows/artists, ratings, years, duration, completion
  - **Users & Behavior** (10 charts): User activity, platforms, players, devices, watch parties, binge watching
  - **Performance** (16 charts): Transcode, resolution, codec, bandwidth, concurrent streams, errors, buffering, quality analytics, bitrate distribution
  - **Geographic** (2 charts): Countries, cities
  - **Advanced** (5 charts): Comparative analytics, temporal heatmap, retention, engagement, predictions
- URL hash navigation and bookmarking
- Browser back/forward navigation
- Active tab state management
- Chart rendering after page switches
- Keyboard navigation between tabs
- Filter state persistence across pages

**Total**: 47 interactive charts with full E2E coverage

#### 13. Mobile & Responsive Design (`tests/e2e/13-mobile-responsive.spec.ts`) - 20+ tests
- **iPhone 12 Testing**: Login responsiveness, touch-friendly inputs (44px min), map rendering, navigation tabs, charts, filter controls, touch interactions, text readability, orientation changes
- **Pixel 5 (Android) Testing**: Touch targets, Android-specific gestures
- **iPad (Tablet) Testing**: Tablet-optimized layout, touch and mouse interactions
- **7 Responsive Breakpoints**: Mobile Small (320px), Mobile Medium (375px), Mobile Large (414px), Tablet Small (768px), Tablet Large (1024px), Desktop Small (1280px), Desktop Large (1920px)
- **Responsive Typography**: Font size scaling across viewports
- **Mobile Navigation**: Horizontal scrolling, touch target spacing
- **Performance on Mobile**: Slow network simulation, lazy loading

#### 14. Theme & Accessibility (`tests/e2e/14-theme-accessibility.spec.ts`) - 25+ tests
- **Theme Switching**: Three theme modes (dark → light → high-contrast → dark)
- **Default Theme**: Dark mode by default
- **Theme Persistence**: localStorage and system preferences
- **ARIA Labels**: Dynamic aria-label updates for theme toggle
- **High-Contrast Mode** (WCAG 2.1 AAA):
  - 21:1 text contrast (#000000 bg, #ffffff text)
  - Enhanced borders (2px) for all interactive elements
- **Keyboard Navigation**:
  - Charts keyboard focusable (tabindex="0")
  - Arrow key navigation within charts (ArrowRight, ArrowLeft, Home, End)
  - Screen reader announcements (aria-live regions)
  - Navigation tab keyboard accessibility
- **Focus Indicators**: Visible focus outlines with 7:1 contrast (#ff6b8a)
- **System Preference Detection**: prefers-color-scheme and prefers-contrast media queries
- **Touch Target Sizes** (WCAG 2.1 SC 2.5.5): All buttons and tabs meet 44x44px minimum

#### 15. Plex Real-Time Integration (`tests/e2e/15-plex-realtime.spec.ts`)
- Plex WebSocket connection establishment
- Real-time playback notifications (playing, paused, stopped, buffering)
- State change handling and deduplication
- New session detection and toast notifications
- Buffering alerts and performance monitoring
- Frontend WebSocket callback registration
- Message parsing and UI updates

#### 16. Plex Transcode Monitoring (`tests/e2e/16-plex-transcode-monitoring.spec.ts`)
- Active transcode session tracking
- Quality transition detection (4K→1080p, 1080p→720p)
- Codec transition monitoring (HEVC→H.264)
- Hardware acceleration detection (Quick Sync, NVENC, VAAPI, VideoToolbox)
- Transcode progress and speed metrics
- Throttling warnings and system load alerts
- Session cards with detailed transcode information

#### 17. Bitrate & Bandwidth Analytics (`tests/e2e/17-bitrate-bandwidth-analytics.spec.ts`)
- Bitrate distribution histogram (median/peak metrics)
- Bandwidth utilization trends (30-day window)
- Network bottleneck identification (constrained sessions)
- Resolution-based bitrate comparison (4K/1080p/720p/SD)
- Chart rendering and accessibility
- API integration and filter support
- Export functionality for bitrate data

#### 18. Buffer Health Monitoring (`tests/e2e/18-buffer-health-monitoring.spec.ts`)
- Real-time buffer fill percentage tracking (0-100%)
- Buffer drain rate calculation and monitoring
- Predictive buffering countdown warnings (10-15s advance notice)
- Three-tier health status (critical <20%, risky 20-50%, healthy >50%)
- Per-session buffer analysis with progress bars
- Automatic toast alerts for critical/risky buffers
- UI panel with session cards and health indicators

#### 19. OAuth 2.0 PKCE Flow (`tests/e2e/19-oauth-flow.spec.ts`)
- OAuth authorization URL generation
- PKCE code challenge/verifier validation
- Authorization code exchange for tokens
- Token refresh functionality
- CSRF protection with state parameter
- Token storage and expiration handling
- Logout with token revocation

#### 20. Cursor Pagination (`tests/e2e/20-cursor-pagination.spec.ts`)
- Cursor-based pagination for large datasets
- Forward/backward navigation
- Cursor token generation and validation
- Performance with 10,000+ records
- Filter persistence across pages
- Graceful handling of invalid cursors

#### 21. Plex Webhooks (`tests/e2e/21-plex-webhooks.spec.ts`)
- Webhook endpoint registration
- Webhook payload parsing and validation
- Event type filtering (play, pause, stop, resume)
- Webhook authentication and security
- Integration with real-time notifications
- Error handling for malformed payloads

#### 22. Documentation Screenshots (`tests/e2e/screenshots.spec.ts`)
- Automated screenshot generation for README
- Chart visualization screenshots (all 47 charts)
- Map and globe visualization screenshots
- Filter panel and UI component screenshots
- Mobile and desktop viewport screenshots
- High-resolution screenshots for documentation

### Running E2E Tests

#### Prerequisites

```bash
# Install frontend dependencies (from root directory)
cd web && npm install && cd ..

# Install Playwright browsers
npx playwright install chromium firefox webkit
```

#### Run Tests

```bash
# Run all E2E tests via Makefile (recommended)
make test-e2e

# Or run Playwright directly from root directory
npx playwright test                         # All browsers (Chromium, Firefox, WebKit)
npx playwright test --project=chromium      # Chromium only
npx playwright test --project=firefox       # Firefox only
npx playwright test --project=webkit        # WebKit (Safari) only

# Interactive modes
npx playwright test --ui                    # UI mode (interactive)
npx playwright test --headed                # Show browser window
npx playwright test --debug                 # Debug mode with Playwright Inspector

# Run specific test file
npx playwright test tests/e2e/12-analytics-pages.spec.ts

# Run with specific browser and headed mode
npx playwright test --project=chromium --headed
```

#### Environment Variables

Set these for authentication tests:

```bash
export ADMIN_USERNAME=admin
export ADMIN_PASSWORD=your_password
export BASE_URL=http://localhost:3857
```

#### CI/CD Integration

E2E tests run automatically on:
- Pull requests
- Pushes to main/develop branches

```yaml
# In .github/workflows/e2e.yml
- name: Run E2E Tests
  run: |
    npm install
    npx playwright install --with-deps chromium
    npx playwright test
```

### Test Configuration

Configuration in `playwright.config.ts`:

```typescript
export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30000, // 30 seconds per test
  expect: { timeout: 5000 },
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
    ...(process.env.CI ? [['github']] : []),
  ],
  use: {
    baseURL: 'http://localhost:3857',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
});
```

### Test Reports

After running tests:

```bash
# View HTML report
npx playwright show-report

# Report location
# playwright-report/index.html
```

### Writing New E2E Tests

Example test structure:

```typescript
import { test, expect } from '@playwright/test';

test.describe('Feature Name', () => {
  test.beforeEach(async ({ page }) => {
    // Setup (login, navigate, etc.)
  });

  test('should do something', async ({ page }) => {
    // Arrange
    await page.goto('/feature');

    // Act
    await page.click('button#action');

    // Assert
    await expect(page.locator('#result')).toBeVisible();
  });
});
```

## Unit Tests

### Go Unit Tests

Run backend unit tests:

```bash
# Run all unit tests
make test

# Or use go directly
go test -v -race ./...

# Run with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### TypeScript Type Checking

```bash
# Check TypeScript types
cd web && npx tsc --noEmit
```

## API Documentation

### OpenAPI/Swagger

API documentation is generated from code using **swaggo**.

#### Generate Documentation

```bash
# Generate Swagger docs from Go comments
make swagger-gen

# Output: docs/swagger/swagger.json
# Output: docs/swagger/swagger.yaml
```

#### View Documentation

```bash
# Start the server
make docker-run

# Access Swagger UI
open http://localhost:3857/swagger/index.html
```

#### API Documentation Structure

```go
// @title Cartographus API
// @version 1.0
// @description Geographic visualization for Plex playback activity
//
// @host localhost:3857
// @BasePath /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in cookie
// @name token
```

Swagger annotations are added to handler functions:

```go
// @Summary Get health status
// @Tags Core
// @Produce json
// @Success 200 {object} models.HealthStatus
// @Router /health [get]
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
  // ...
}
```

### API Endpoints Documentation

All endpoints are documented in:
- README.md (API Documentation section)
- ARCHITECTURE.md (Layer 3: API Service)
- Swagger UI (auto-generated)

## Running All Tests

```bash
# Run unit tests + E2E tests
make test-all
```

## Continuous Integration

### GitHub Actions

Tests run automatically on:

```yaml
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]
```

### Test Matrix

- **Go Unit Tests**: Linux (amd64)
- **E2E Tests**: Chromium (latest)
- **Build Tests**: Multi-arch (amd64, arm64)

## Test Coverage Goals

| Test Type | Current | Target |
|-----------|---------|--------|
| Go Unit Tests | 85%+ (core packages) | 90%+ |
| E2E Tests | Comprehensive | Maintain |
| Integration Tests | Basic | Expand |

### Package-Level Coverage

| Package | Coverage | Status |
|---------|----------|--------|
| internal/config | 97.8% | Excellent |
| internal/cache | 95%+ | Excellent (enhanced with edge cases) |
| internal/middleware | 92.0% | Excellent |
| internal/auth | 90.8% | Excellent |
| internal/sync | 90%+ | Excellent (comprehensive tests added) |
| internal/websocket | 78.0% | Good (room for improvement) |
| internal/api | 75%+ | Good (room for improvement) |
| internal/database | 70%+ | Good (room for improvement) |
| internal/models | N/A | (struct definitions only) |

**Recent Improvements:**
- **internal/sync**: Increased from 0% to 90%+ with comprehensive test suite (1,200+ lines)
  - Added full Tautulli API client tests with HTTP mocking
  - Complete sync manager tests with database and client mocks
  - Tests for retry logic, geolocation fallback, and error handling
  - Benchmark tests for performance validation
- **internal/cache**: Enhanced from 78.9% to 95%+ with edge case coverage
  - Added 25+ new test functions for cleanup, TTL edge cases, stats validation
  - Large dataset tests (10,000 entries)
  - Concurrent access and race condition tests
  - Memory cleanup validation

## Troubleshooting

### E2E Tests Fail Locally

1. **Check server is running:**
   ```bash
   curl http://localhost:3857/api/v1/health
   ```

2. **Install Playwright browsers:**
   ```bash
   npx playwright install chromium
   ```

3. **Check environment variables:**
   ```bash
   echo $ADMIN_USERNAME
   echo $ADMIN_PASSWORD
   ```

### Swagger Generation Fails

1. **Install swag CLI:**
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. **Verify GOPATH/bin in PATH:**
   ```bash
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

3. **Run manually:**
   ```bash
   swag init -g cmd/server/docs.go -o docs/swagger
   ```

## Best Practices

### E2E Tests

1. **Use data-testid attributes** for reliable selectors
2. **Avoid timing-dependent assertions** - use Playwright's auto-waiting
3. **Test user flows, not implementation details**
4. **Keep tests independent** - don't rely on test execution order
5. **Clean up state** in `beforeEach` hooks

#### Common E2E Test Issues and Solutions

Based on extensive debugging of the E2E test suite, here are critical patterns and solutions:

##### ECharts SVG vs Canvas Rendering

ECharts uses different renderers based on device type:

```typescript
// ChartManager.ts uses SVG on touch devices, canvas on desktop
private getOptimalRenderer(dataSize: number = 0): 'canvas' | 'svg' {
    if ('ontouchstart' in window || navigator.maxTouchPoints > 0) {
        return 'svg';  // Touch devices get SVG
    }
    return dataSize > 1000 ? 'canvas' : 'svg';
}
```

**Solution**: Use combined selectors for chart element detection:
```typescript
// WRONG - fails on mobile
await page.locator('#chart-trends canvas').toBeVisible();

// CORRECT - works on all devices
await page.locator('#chart-trends canvas, #chart-trends svg').first().toBeVisible();
```

##### Keyboard Navigation Interception

The NavigationManager intercepts ArrowRight/ArrowLeft keys to switch analytics pages:

```typescript
// NavigationManager.ts lines 321-336
// ArrowRight/ArrowLeft switches between analytics pages, not chart navigation
```

**Solution**: Use Home/End keys for chart-level keyboard navigation tests:
```typescript
// WRONG - switches to next analytics page
await page.keyboard.press('ArrowRight');

// CORRECT - navigates within chart
await page.keyboard.press('Home');
await page.keyboard.press('End');
```

##### Analytics Container Visibility Order

The analytics container must be visible before checking child elements:

```typescript
// WRONG - child may not be visible even when parent is
await page.waitForSelector('#analytics-overview', { state: 'visible' });

// CORRECT - wait for parent container first
await page.waitForSelector('#analytics-container:not([style*="display: none"])', {
    state: 'attached',
    timeout: 10000
});
await page.waitForSelector('#analytics-overview', { state: 'visible', timeout: 10000 });
```

##### Authentication Flow Timing

The authentication flow hides the login container BEFORE showing the app:

```typescript
// WRONG - may fail due to timing
await page.click('button[type="submit"]');
await expect(page.locator('#app')).toBeVisible();

// CORRECT - wait for login form to hide first
await page.click('button[type="submit"]');
await expect(page.locator('#login-container')).not.toBeVisible({ timeout: 15000 });
await expect(page.locator('#app')).toBeVisible({ timeout: 10000 });
```

##### Mobile Viewport Element Access

Elements may be outside the viewport on mobile even with `force: true`:

```typescript
// WRONG - element may be outside viewport
await page.locator('.nav-tab[data-view="analytics"]').click({ force: true });

// CORRECT - scroll into view first
const analyticsTab = page.locator('.nav-tab[data-view="analytics"]');
await analyticsTab.scrollIntoViewIfNeeded();
await analyticsTab.click({ force: true });
```

##### Playwright Locators vs page.evaluate()

Prefer Playwright locators over raw DOM queries for better reliability:

```typescript
// AVOID - raw DOM access can miss dynamic elements
const exists = await page.evaluate(() => {
    return document.getElementById('chart-announcer') !== null;
});

// PREFER - Playwright locators with built-in waiting
await expect(page.locator('#chart-announcer')).toBeAttached();
await expect(page.locator('#chart-announcer')).toHaveAttribute('aria-live', 'assertive');
```

##### Mobile Sidebar Positioning

The sidebar starts at `left: -320px` and requires `.open` class to be visible:

```typescript
// Mobile sidebar test - check for open class
await page.locator('#sidebar').evaluate(el => el.classList.contains('open'));
```

### Unit Tests

1. **Table-driven tests** for multiple scenarios
2. **Mock external dependencies** (database, API clients)
3. **Test error paths** in addition to happy paths
4. **Use subtests** for better organization
5. **Avoid test interdependence**

---

## CI/CD Testing Infrastructure

### Hybrid Runner Architecture

This project uses a **hybrid GitHub Actions infrastructure** combining self-hosted and GitHub-hosted runners for optimal performance and security.

### Self-Hosted Runners (Main Branch)

**Hardware Requirements:**
- Ubuntu 22.04/24.04 LTS
- 8+ CPU cores, 32GB+ RAM
- 250GB+ SSD storage
- Docker 24.0+ with Buildx

**Pre-Installed Dependencies:**
- Go 1.24+ with tools (gotestsum, gocovmerge, benchstat)
- Node.js 20.x LTS
- Docker 24.0+ with Buildx plugin
- Cross-compilation toolchains (ARM64, Windows MinGW, macOS OSXCross)
- Code analysis tools (scc v3.4.0)
- WebGL dependencies (libegl1, libgles2, libgl1-mesa-dri) for screenshot testing
- Playwright browsers (installed per-run via `npx playwright install --with-deps`)

**Workflows Using Self-Hosted:**
- **Lint** (_lint.yml): Go and TypeScript linting
- **Test** (_test.yml): 8861 unit tests with race detection
- **Build** (_build.yml): Frontend builds, Docker multi-arch (amd64, arm64)
- **Analysis** (_analysis.yml): Coverage (75.5%), profiling, benchmarking, code metrics
- **E2E** (_e2e.yml): 338 Playwright tests, integration tests

### GitHub-Hosted Runners (Security Isolation)

**Workflows Using GitHub-Hosted (ubuntu-latest):**
- **Security** (_security.yml): Trivy, Trufflehog, license compliance
- **CodeQL** (_codeql.yml): SAST analysis (Go, TypeScript)
- **Release**: Semantic versioning, GHCR container publishing

**Rationale**: Security scanning tools require privileged operations and benefit from GitHub's isolation guarantees.

### Test Execution Flow

**Pull Requests:**
```
Lint (self-hosted) ─┐
                    ├──► Test (self-hosted) ──► Build (self-hosted)
Security (GitHub)  ─┤
CodeQL (GitHub)    ─┘
```

**Main Branch Merges:**
```
Lint (self-hosted) ─┐
                    ├──► Test (self-hosted) ──► Build (self-hosted) ──┐
Security (GitHub)  ─┤                                                 ├──► E2E (self-hosted)
CodeQL (GitHub)    ─┘                                                 │
                                                                       └──► Analysis (self-hosted)
```

### Test Suite Metrics

| Test Category | Count | Coverage Target | Runner Type |
|--------------|-------|-----------------|-------------|
| **Unit Tests** | 8861 | 75.5% | Self-hosted |
| **Fuzz Tests** | 2 | N/A | Self-hosted (main only) |
| **E2E Tests** | 338 | N/A | Self-hosted (main only) |
| **Integration Tests** | - | N/A | Self-hosted (main only) |
| **Security Scans** | - | CRITICAL/HIGH blocking | GitHub-hosted |

### Performance Benefits

**Self-Hosted Advantages:**
- 50%+ faster build times (persistent Docker layer cache)
- No cold starts (pre-installed dependencies)
- Larger resource limits (32GB RAM vs 7GB on GitHub-hosted)
- Local filesystem caching for Docker builds

**Caching Strategy:**
- **Self-hosted**: Local filesystem cache (`type=local`) for Docker builds
- **GitHub-hosted**: GitHub Actions Cache (`type=gha`) for npm/Go modules
- **DuckDB extensions**: Auto-download in workflows (lightweight, version-flexible)

### Fallback Behavior

**Runner Unavailability:**
- Jobs queue for up to 6 hours if self-hosted runners offline
- Manual intervention required (no automatic fallback to GitHub-hosted)
- Monitor runner status: Settings → Actions → Runners

**To revert to GitHub-hosted runners:**
1. Change `runs-on: [self-hosted, linux, x64]` to `runs-on: ubuntu-latest`
2. Remove conditional steps marked with `if: runner.environment != 'self-hosted'`
3. Update Docker cache strategy from `type=local` to `type=gha`

See [docs/SELF_HOSTED_RUNNER.md](./docs/SELF_HOSTED_RUNNER.md) for complete setup guide.

---

## Resources

- [Playwright Documentation](https://playwright.dev/docs/intro)
- [Go Testing Package](https://pkg.go.dev/testing)
- [Swaggo Documentation](https://github.com/swaggo/swag)
- [OpenAPI Specification](https://swagger.io/specification/)
