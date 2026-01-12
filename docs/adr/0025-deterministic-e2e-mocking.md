# ADR-0025: Deterministic E2E Test Mocking

**Date**: 2026-01-01
**Status**: Accepted

---

## Context

The E2E test suite was experiencing persistent flaky failures in CI environments due to non-deterministic API mocking. Over 50+ commits attempted to fix these issues with various approaches:

### Previous Implementation Problems

1. **Race Conditions with page.route()**: Routes were registered using `page.route()` after page creation but before navigation. This created a race window where early requests could bypass interception.

2. **Conflicting Route Registrations**: Multiple files (`auth-mocks.ts` and `api-mocks.ts`) registered overlapping routes for the same endpoints (e.g., `/api/v1/users`, `/api/v1/stats`).

3. **Playwright's LIFO Ordering**: Routes are evaluated in Last-In-First-Out order. With multiple registration points and timing variations, this caused unpredictable matching behavior.

4. **CI-Specific Timing Issues**: SwiftShader (software WebGL) and container startup delays exacerbated timing windows, causing ~15% of tests to hit the real Tautulli API (30+ second responses).

### Failed Approaches

- Sequential `page.route()` calls with stabilization delays (50ms, 150ms, 200ms)
- Glob patterns (`**/api/v1/**`)
- Regex patterns (`/\/api\/v1\//`)
- URL matcher functions
- Consolidated single handler with diagnostic logging

All approaches failed because they addressed symptoms, not root cause: **routes must be registered before page creation**.

---

## Decision

Implement **Browser Context-level routing** using `context.route()` instead of `page.route()`. This architectural change ensures:

1. Routes are registered ONCE on the browser context BEFORE any page is created
2. All pages in the context inherit the routes automatically
3. No race between route registration and navigation
4. Single source of truth for all API mocking

### Key Architectural Changes

1. **New unified mock-server.ts**: Single file containing all API handlers with explicit priority ordering
2. **Context-level routing in fixtures.ts**: Custom fixture creates mocked context before page
3. **Removed conflicting registrations**: auth-mocks.ts now only contains error filtering utilities
4. **auth.setup.ts updated**: Uses setupMockServer() on context before creating page

---

## Consequences

### Positive

- **100% Deterministic Mocking**: Routes are guaranteed to be active before any request
- **Fast Tests**: Mock responses return in <50ms vs 30+ seconds for real API
- **Single Source of Truth**: All API mocking in mock-server.ts, easy to maintain
- **No More Flaky Failures**: Eliminates the entire class of route interception bugs
- **Simplified Debugging**: Request logging shows exactly what's intercepted
- **Isolation**: resetMockData() between tests ensures clean state

### Negative

- **Breaking Change for Custom Mocking**: Tests using page.route() for custom mocking need migration
- **Context Per Test**: Each test gets a new context (slight overhead, but necessary for isolation)
- **Learning Curve**: Developers need to understand context-level vs page-level routing

### Neutral

- **Existing API Mock Data**: Same mock data patterns, just centralized location
- **Test Structure**: Tests still use same page-level interactions

---

## Implementation

### Technical Details

**Architecture (v2 - Express Proxy):**

The implementation uses a hybrid architecture for 100% deterministic behavior:

1. Express mock server runs on port 3900 (started by `global-setup.ts`)
2. Playwright intercepts API requests at the browser context level
3. Requests are proxied to Express server via real HTTP fetch
4. Express handles concurrency properly (no CDP race conditions)
5. Fallback to inline handlers if Express is unavailable

**Route Registration Flow:**
```
1. global-setup.ts: startMockServer() starts Express on port 3900
2. fixtures.ts: test.extend() defines mockContext fixture
3. mockContext fixture: browser.newContext() creates fresh context
4. mockContext fixture: setupMockServer(context) registers context-level routes
5. Routes proxy to Express server for deterministic responses
6. page fixture: mockContext.newPage() creates page with pre-registered routes
7. Test: page.goto('/') - all requests are already intercepted
```

**Handler Priority:**

Routes are processed in this order:
1. **Express proxy** (primary): Requests are forwarded to the Express mock server on port 3900
2. **Inline handlers** (fallback): If Express is unavailable, routes in `API_HANDLERS` array are used (first match wins)

More specific patterns come before general patterns:

```typescript
// Specific first
{ pattern: /\/api\/v1\/analytics\/trends/, handler: trendsHandler },
{ pattern: /\/api\/v1\/analytics\/media/, handler: mediaHandler },
// Catch-all last
{ pattern: /\/api\/v1\/analytics/, handler: generalAnalyticsHandler },
```

### Code References

| Component | File | Notes |
|-----------|------|-------|
| Express Mock Server | `tests/e2e/mock-api-server.ts` | Standalone Express server (port 3900) |
| Global Setup | `tests/e2e/global-setup.ts` | Starts Express server before tests |
| Playwright Mock Server | `tests/e2e/fixtures/mock-server.ts` | Context-level routing + Express proxy |
| Fixtures | `tests/e2e/fixtures.ts` | Test fixtures with mocked context |
| Auth Setup | `tests/e2e/auth.setup.ts` | Authentication with mocking |
| Error Filters | `tests/e2e/fixtures/auth-mocks.ts` | Console/page error filtering only |
| Test Helpers | `tests/e2e/fixtures/helpers.ts` | Navigation and assertion helpers |
| Constants | `tests/e2e/fixtures/constants.ts` | Timeouts, selectors, test data |

### Configuration

```typescript
// fixtures.ts - Default options
autoMockApi: true,      // Mock all /api/v1/ endpoints
autoMockTiles: true,    // Mock map tiles and geocoder
logRequests: process.env.E2E_VERBOSE === 'true',  // Log with E2E_VERBOSE=true
```

### Dependencies

- `@playwright/test` ^1.57.0
- `express` ^5.2.1 (for standalone mock server)
- `cors` ^2.8.5 (for Express CORS middleware)

---

## Verification

### Verified Claims

| Claim | Source | Verified |
|-------|--------|----------|
| context.route() persists across pages | Playwright docs | Yes |
| Routes are evaluated LIFO | Playwright source | Yes |
| Context routes apply before page creation | Test verification | Yes |

### Test Coverage

- All 1300+ E2E tests use the new mocking infrastructure
- No changes to test assertions required
- Coverage maintained at existing levels

### Verification Commands

```bash
# List all tests (verifies TypeScript compilation)
npx playwright test --list

# Run single test to verify mocking
npx playwright test 02-charts.spec.ts:60 --headed

# Check mock request logs
E2E_VERBOSE=true npx playwright test 02-charts.spec.ts:60
```

---

## Migration Guide

### For Tests Using Default Mocking

No changes required. The `test` fixture from fixtures.ts automatically provides mocked context.

```typescript
// Works unchanged
import { test, expect } from './fixtures';

test('my test', async ({ page }) => {
  await page.goto('/');
  // API already mocked
});
```

### For Tests Needing Custom Mocking

Use testWithRealApi and register custom routes on the page:

```typescript
import { testWithRealApi as test, expect } from './fixtures';

test('custom mock', async ({ page }) => {
  await page.route('/api/v1/custom', route => {
    route.fulfill({ status: 200, body: '{}' });
  });
  await page.goto('/');
});
```

### For Adding New API Endpoints

Add handler to mock-server.ts in the appropriate position:

```typescript
// In API_HANDLERS array
{
  pattern: /\/api\/v1\/new-endpoint/,
  handler: async () => jsonResponse({ data: 'mock' })
},
```

---

## Related ADRs

- [ADR-0019](0019-testcontainers-integration-testing.md): testcontainers for integration tests (complementary approach)
- [ADR-0003](0003-authentication-architecture.md): Auth architecture (mocked in E2E tests)

---

## References

- [Playwright Network Documentation](https://playwright.dev/docs/network)
- [Playwright Route API](https://playwright.dev/docs/api/class-browsercontext#browser-context-route)
- [Browser Context vs Page Routing](https://playwright.dev/docs/network#modify-requests)
