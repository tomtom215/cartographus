# ADR-0002: Frontend Technology Stack Selection

**Date**: 2025-11-16 (MapLibre migration: 2025-11-25)
**Status**: Accepted

---

## Context

Cartographus requires a frontend capable of:

1. **Interactive Maps**: Rendering 10,000+ geographic points with clustering
2. **3D Globe Visualization**: WebGL-powered globe view with scatter plots
3. **Rich Analytics**: 47 charts across 6 analytics pages
4. **Real-Time Updates**: Live WebSocket notifications
5. **Offline Tile Support**: Self-hosted map tiles via PMTiles
6. **Cross-Browser Compatibility**: Chromium, Firefox, WebKit support

### Build Requirements

- **Fast Build Times**: Developer productivity with hot reload
- **Small Bundle Size**: Optimal loading performance
- **Tree Shaking**: Eliminate unused code
- **TypeScript**: Type safety with strict mode

### Alternatives Considered

| Framework/Tool | Pros | Cons |
|----------------|------|------|
| **React + Vite** | Ecosystem, components | Heavy runtime, larger bundle |
| **Vue + Vite** | Reactive, lightweight | Additional learning curve |
| **Vanilla TS + esbuild** | No framework overhead, fast builds | More manual work |
| **Webpack** | Mature, flexible | Complex config, slower builds |

---

## Decision

Use a **minimal vanilla TypeScript stack** with specialized visualization libraries:

### Core Stack

| Component | Library | Version | Purpose |
|-----------|---------|---------|---------|
| **Language** | TypeScript | 5.9.3 | Type safety with strict mode |
| **Bundler** | esbuild | 0.27.2 | Sub-second builds, tree shaking |
| **Maps** | MapLibre GL JS | 5.15.0 | Open-source WebGL maps |
| **3D Globe** | deck.gl | 9.2.5 | WebGL 3D visualization |
| **Charts** | ECharts | 6.0.0 | 47 analytics charts |
| **Tiles** | PMTiles | 4.3.2 | Self-hosted vector tiles |
| **E2E Testing** | Playwright | 1.57.0 | Cross-browser testing |

### Key Factors

1. **No Framework Runtime**: Smaller bundle, faster initial load
2. **esbuild Performance**: 50-100x faster than Webpack
3. **MapLibre GL JS**: Mapbox-compatible, no API key required
4. **deck.gl Integration**: MapboxOverlay with ScatterplotLayer for globe visualization
5. **ECharts Flexibility**: Declarative charting with 47 chart types

---

## Consequences

### Positive

- **Sub-50ms Build Times**: esbuild achieves ~40ms incremental builds
- **~300KB Initial Bundle**: Minimal framework overhead
- **1300+ E2E Tests**: Cross-browser coverage with Playwright (75 spec files)
- **WebGL Performance**: Smooth 60fps rendering for maps and globe
- **Self-Hosted Tiles**: No external tile service dependencies

### Negative

- **Manual State Management**: No framework reactivity primitives
- **Custom Component Patterns**: More boilerplate for UI components
- **TypeScript Discipline Required**: Strict mode catches issues but requires attention

### Neutral

- **No Virtual DOM**: Direct DOM manipulation (appropriate for visualization-heavy app)
- **ESM Modules**: Modern browser requirement (IE11 not supported)

---

## Implementation

### Directory Structure

```
web/
├── src/
│   ├── index.ts              # Entry point (~2200 lines)
│   ├── service-worker.ts     # Offline support
│   ├── app/                  # Application managers (70+ files)
│   │   ├── ThemeManager.ts
│   │   ├── ViewportManager.ts
│   │   ├── NavigationManager.ts
│   │   └── ...
│   └── lib/                  # Core libraries
│       ├── map.ts            # MapLibre GL JS integration (MapManager)
│       ├── globe-deckgl.ts   # deck.gl globe visualization (GlobeManagerDeckGL)
│       ├── charts/           # ECharts initialization (ChartManager)
│       ├── filters.ts        # Filter management (FilterManager)
│       ├── websocket.ts      # WebSocket manager
│       ├── api.ts            # Fetch wrapper
│       ├── types/            # TypeScript interfaces
│       └── utils/            # Shared utilities
├── public/
│   ├── index.html
│   └── tiles/               # PMTiles files
├── dist/                    # Build output
├── package.json
└── tsconfig.json
├── playwright.config.ts     # E2E test configuration (at project root)
```

### Build Configuration

```json
// package.json scripts (simplified)
{
  "scripts": {
    "build": "npm run build:critical && npm run build:css && npm run build:app && npm run build:sw",
    "build:app": "npx esbuild src/index.ts --bundle --outdir=dist --splitting --format=esm --minify --sourcemap",
    "build:sw": "npx esbuild src/service-worker.ts --bundle --outfile=public/service-worker.js --minify",
    "watch": "npm run watch:css & npm run watch:app & npm run watch:sw",
    "dev": "npx esbuild src/index.ts --bundle --outdir=dist --splitting --format=esm --sourcemap --servedir=dist --serve=8081"
  }
}
```

### TypeScript Configuration

```json
// tsconfig.json
{
  "compilerOptions": {
    "strict": true,
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "node",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noImplicitReturns": true
  }
}
```

### MapLibre GL JS Integration

```typescript
// web/src/lib/map.ts
import maplibregl from 'maplibre-gl';
import { Protocol } from 'pmtiles';

export class MapManager {
    private map: maplibregl.Map;

    constructor(containerId: string, onModeChange?: () => void) {
        // Initialize PMTiles protocol
        const protocol = new Protocol();
        maplibregl.addProtocol('pmtiles', protocol.tile);

        // Create MapLibre map instance
        this.map = new maplibregl.Map({
            container: containerId,
            style: createMapStyle(config),
            center: [0, 20],
            zoom: 2,
            maxPitch: 85,
        });

        // Add navigation controls
        this.map.addControl(new maplibregl.NavigationControl(), 'top-right');
    }

    // Update with location statistics
    updateLocations(locations: LocationStats[]): void {
        const features = this.createLocationFeatures(locations);
        this.updateLocationSource(features);
    }
}
```

### deck.gl Globe Integration

```typescript
// web/src/lib/globe-deckgl.ts
import { MapboxOverlay } from '@deck.gl/mapbox';
import { ScatterplotLayer } from '@deck.gl/layers';
import maplibregl from 'maplibre-gl';

export class GlobeManagerDeckGL {
    private overlay: MapboxOverlay | null = null;
    private map: maplibregl.Map | null = null;

    constructor(containerId: string) {
        this.containerId = containerId;
    }

    public initialize(): void {
        // Create MapLibre map with globe projection
        this.map = new maplibregl.Map({
            container: this.containerId,
            style: createMapStyle(config),
            center: [0, 20],
            zoom: 2,
        });

        this.map.on('load', () => {
            // Set globe projection
            if ('setProjection' in this.map) {
                this.map.setProjection({ type: 'globe' });
            }

            // Initialize deck.gl overlay
            this.overlay = new MapboxOverlay({
                interleaved: true,
                layers: [],
            });
            this.map.addControl(this.overlay);
        });
    }

    public updateLocations(locations: LocationStats[]): void {
        const layer = new ScatterplotLayer({
            id: 'playback-locations',
            data: locations,
            getPosition: d => [d.longitude, d.latitude, 0],
            getRadius: d => Math.sqrt(d.playback_count) * 20000,
            getFillColor: d => this.getColorByPlaybackCount(d.playback_count),
            pickable: true,
        });
        this.overlay.setProps({ layers: [layer] });
    }
}
```

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Main entry | `web/src/index.ts` | ~2200 lines, application bootstrap |
| Package config | `web/package.json` | Dependencies and scripts |
| TypeScript config | `web/tsconfig.json` | Strict mode settings |
| Playwright config | `playwright.config.ts` | E2E test configuration (project root) |
| Map manager | `web/src/lib/map.ts` | MapLibre GL JS integration |
| Globe manager | `web/src/lib/globe-deckgl.ts` | deck.gl globe visualization |
| Chart manager | `web/src/lib/charts/ChartManager.ts` | ECharts initialization |
| Filter manager | `web/src/lib/filters.ts` | Filter state management |

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| TypeScript 5.9.3 | `web/package.json` devDependencies | Yes |
| esbuild 0.27.2 | `web/package.json` devDependencies | Yes |
| MapLibre GL JS 5.15.0 | `web/package.json` dependencies | Yes |
| deck.gl 9.2.5 | `web/package.json` dependencies | Yes |
| ECharts 6.0.0 | `web/package.json` dependencies | Yes |
| PMTiles 4.3.2 | `web/package.json` dependencies | Yes |
| Playwright 1.57.0 | `web/package.json` devDependencies | Yes |

### Test Coverage

- E2E tests: `tests/e2e/` directory
- 75 test suites, 1300+ test cases
- Cross-browser: Chromium, Firefox, WebKit
- Visual regression: Screenshot comparisons

---

## Related ADRs

- [ADR-0001](0001-use-duckdb-for-analytics.md): Backend data source for charts
- [ADR-0010](0010-cursor-based-pagination.md): API pagination for large datasets

---

## References

- [TypeScript Handbook](https://www.typescriptlang.org/docs/handbook/)
- [esbuild Documentation](https://esbuild.github.io/)
- [MapLibre GL JS](https://maplibre.org/maplibre-gl-js-docs/)
- [deck.gl Documentation](https://deck.gl/docs)
- [ECharts Examples](https://echarts.apache.org/examples/)
- [PMTiles Specification](https://github.com/protomaps/PMTiles)
