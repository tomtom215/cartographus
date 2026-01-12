// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Cross-Platform UI Components
 *
 * Reusable components for the cross-platform linking feature.
 */

export {
  PlatformBadge,
  renderPlatformBadges,
  getPlatformConfig,
  isValidPlatform,
  getAllPlatforms,
  type Platform,
  type PlatformBadgeOptions
} from './PlatformBadge';

export {
  ConfidenceIndicator,
  renderConfidenceIndicatorHTML,
  getConfidenceLevel,
  getConfidenceDescription,
  type ConfidenceIndicatorOptions
} from './ConfidenceIndicator';

export {
  ExternalIdLookup,
  validateExternalId,
  formatExternalId,
  getExternalIdConfig,
  type ExternalIdType,
  type ExternalIdLookupOptions,
  type LookupCallback
} from './ExternalIdLookup';

export {
  UserSelector,
  linkedUserToSelectorItem,
  type UserSelectorItem,
  type UserSelectCallback,
  type UserSelectorOptions
} from './UserSelector';
