# Documentation Screenshots

This directory contains screenshots captured for the README and documentation.

## Directory Structure

```
screenshots/
├── README.md           # This file
├── captured/           # Output directory for captured screenshots
│   ├── *.png           # Screenshot images
│   ├── results.json    # Capture results and metadata
│   ├── METADATA.txt    # Generation metadata
│   └── MANIFEST.md     # Usage documentation
└── .gitkeep            # Keep directory in git
```

## Capturing Screenshots

### Using GitHub Actions (Recommended)

Screenshots are captured automatically via the GitHub Actions workflow:

1. Go to **Actions** > **Documentation Screenshots**
2. Click **Run workflow**
3. Select the branch to capture from
4. Choose the theme mode (dark, light, or both)
5. Wait for the workflow to complete
6. Download the ZIP from the workflow artifacts

### Running Locally

To capture screenshots locally:

```bash
# From the web/ directory
cd web

# Install dependencies
npm ci

# Install Playwright browsers
npx playwright install chromium

# Start the application (in another terminal)
cd .. && ./cartographus

# Run screenshot capture
npx playwright test --config=playwright.screenshots.config.ts
```

## Screenshot List

The following screenshots are captured for documentation:

| ID | Name | Description | Auth Required |
|----|------|-------------|---------------|
| `login-page` | Login Page | Authentication page with login form | No |
| `live-activity` | Live Activity | Real-time playback monitoring | Yes |
| `server-dashboard` | Server Dashboard | Server statistics and health | Yes |
| `data-governance` | Data Governance | Audit and governance controls | Yes |
| `recently-added` | Recently Added | Content discovery view | Yes |
| `analytics-overview` | Analytics Overview | Main analytics dashboard | Yes |
| `map-view` | 2D Map View | Interactive map with locations | Yes |
| `globe-view` | 3D Globe View | 3D globe visualization | Yes |

## Using Screenshots in Documentation

Reference screenshots in markdown:

```markdown
![Map View](docs/screenshots/map-view.png)
![Analytics Dashboard](docs/screenshots/analytics-overview.png)
![3D Globe](docs/screenshots/globe-view.png)
```

## Configuration

Screenshot capture is configured in:

- `playwright.screenshots.config.ts` - Playwright configuration
- `scripts/capture-screenshots.ts` - Screenshot definitions and capture logic

### Key Settings

| Setting | Value | Description |
|---------|-------|-------------|
| Viewport | 1920x1080 | Fixed viewport for consistency |
| Browser | Chromium | With SwiftShader for WebGL |
| Color Scheme | Configurable | Dark, light, or both |
| Animations | Disabled | For deterministic captures |

### Theme Modes

The workflow supports three theme modes via the `theme` input:

| Mode | Description | Output Files |
|------|-------------|--------------|
| `dark` | Dark theme only (default) | `login-page.png`, `map-view.png`, etc. |
| `light` | Light theme only | `login-page.png`, `map-view.png`, etc. |
| `both` | Both themes captured | `login-page-dark.png`, `login-page-light.png`, etc. |

When `both` is selected:
- Each screenshot is captured twice (once per theme)
- Filenames include theme suffix: `{id}-dark.png` and `{id}-light.png`
- Total screenshot count doubles

To capture with a specific theme locally:

```bash
# Dark theme (default)
SCREENSHOT_THEME=dark npx playwright test --config=playwright.screenshots.config.ts

# Light theme
SCREENSHOT_THEME=light npx playwright test --config=playwright.screenshots.config.ts

# Both themes
SCREENSHOT_THEME=both npx playwright test --config=playwright.screenshots.config.ts
```

## Troubleshooting

### WebGL Issues

If screenshots fail with WebGL errors:

1. Ensure Playwright is using the correct Chrome args:
   - `--use-gl=swiftshader`
   - `--enable-webgl`
   - `--ignore-gpu-blocklist`

2. Clean WebGL resources between captures (done automatically)

### Authentication Issues

If login fails:

1. Check the application is running with `AUTH_MODE=none` or `AUTH_MODE=basic`
2. Verify credentials in the workflow inputs
3. Check server logs for authentication errors

### Missing Elements

If screenshots fail waiting for elements:

1. Verify the application is fully loaded
2. Check for JavaScript errors in the console
3. Increase timeouts in the configuration

## Demo Data Seeding

By default, the screenshot workflow seeds the database with realistic demo data to ensure screenshots display meaningful content instead of empty dashboards.

### How It Works

The workflow uses the DuckDB [fakeit](https://github.com/tobilg/duckdb-fakeit) community extension to generate:

| Data | Count | Description |
|------|-------|-------------|
| Playback Events | 500 | Across 30 days, weighted towards evenings |
| Unique Users | 50 | With realistic names and email addresses |
| Geographic Locations | ~100 | Global distribution (US, EU, APAC, etc.) |
| Media Items | 32 | Movies, TV episodes, and music tracks |
| Platforms | 13 | Various Plex clients and devices |

### Seed Data Features

- **Reproducible**: Uses fixed random seed (0.42) for deterministic generation
- **Realistic distribution**:
  - 60% evening playbacks (6pm-midnight)
  - 70% complete watches (90%+ progress)
  - 40% North American, 30% European locations
- **Varied content**: Mix of movies, TV shows, and music
- **Quality spread**: 4K (40%), 1080p (40%), 720p/480p (20%)

### Disabling Seed Data

To capture screenshots without demo data (empty state), set `seed-data: false` in the workflow dispatch:

```yaml
# Workflow inputs
seed-data: false  # Disable demo data generation
```

### Local Seeding

To seed demo data locally:

```bash
# Requires DuckDB CLI or use the application's database
duckdb /path/to/cartographus.duckdb < web/scripts/seed-screenshot-data.sql
```

### Seed Script Location

- `web/scripts/seed-screenshot-data.sql` - SQL script with fakeit extension

## CI/CD Integration

The screenshot workflow (`screenshots.yml`) is designed for:

- **On-demand execution**: Manual trigger only
- **Branch selection**: Capture from any branch
- **Theme selection**: Dark, light, or both themes
- **Demo data seeding**: Realistic playback data (enabled by default)
- **Artifact generation**: ZIP file with all screenshots
- **Summary reporting**: Results in workflow summary
