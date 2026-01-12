// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Newsletter Types
 *
 * Types for the Newsletter Generator system.
 * Matches the Go types in internal/models/newsletter.go
 */

/** Newsletter type enumeration */
export type NewsletterType =
  | 'recently_added'
  | 'coming_soon'
  | 'weekly_digest'
  | 'monthly_recap'
  | 'watchlist_updates'
  | 'trending_content'
  | 'personalized_recommendations';

/** Delivery channel enumeration */
export type DeliveryChannel =
  | 'email'
  | 'discord'
  | 'slack'
  | 'telegram'
  | 'webhook'
  | 'in_app';

/** Delivery status enumeration */
export type DeliveryStatus =
  | 'pending'
  | 'sending'
  | 'delivered'
  | 'failed'
  | 'partial'
  | 'canceled';

/** Time frame unit for newsletter content */
export type TimeFrameUnit = 'hours' | 'days' | 'weeks' | 'months';

/** Template variable definition */
export interface TemplateVariable {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'array' | 'object';
  description?: string;
  default_value?: unknown;
  required?: boolean;
}

/** Template configuration */
export interface TemplateConfig {
  time_frame: number;
  time_frame_unit: TimeFrameUnit;
  max_items: number;
  include_movies: boolean;
  include_shows: boolean;
  include_stats: boolean;
  include_top_content: boolean;
  custom_settings?: Record<string, unknown>;
}

/** Newsletter template */
export interface NewsletterTemplate {
  id: string;
  name: string;
  description?: string;
  type: NewsletterType;
  subject: string;
  body_html: string;
  body_text?: string;
  variables?: TemplateVariable[];
  default_config?: TemplateConfig;
  version: number;
  is_built_in: boolean;
  is_active: boolean;
  created_by: string;
  created_at: string;
  updated_by?: string;
  updated_at: string;
}

/** Channel-specific configuration */
export interface ChannelConfig {
  enabled: boolean;
  settings?: Record<string, unknown>;
  webhook_url?: string;
  bot_token?: string;
  chat_id?: string;
  channel_id?: string;
}

/** Newsletter schedule */
export interface NewsletterSchedule {
  id: string;
  name: string;
  description?: string;
  template_id: string;
  template_name?: string;
  recipients: string[];
  cron_expression: string;
  timezone: string;
  config?: TemplateConfig;
  channels: DeliveryChannel[];
  channel_configs?: Record<DeliveryChannel, ChannelConfig>;
  is_enabled: boolean;
  last_run_at?: string;
  next_run_at?: string;
  last_run_status?: DeliveryStatus;
  run_count: number;
  success_count: number;
  failure_count: number;
  created_by: string;
  created_at: string;
  updated_by?: string;
  updated_at: string;
}

/** Recipient delivery details */
export interface RecipientDeliveryDetail {
  recipient_id: string;
  status: DeliveryStatus;
  delivered_at?: string;
  error?: string;
}

/** Content statistics for delivery */
export interface DeliveryContentStats {
  movies_count: number;
  shows_count: number;
  episodes_count: number;
  total_items: number;
}

/** Newsletter delivery record */
export interface NewsletterDelivery {
  id: string;
  schedule_id?: string;
  schedule_name?: string;
  template_id: string;
  template_name?: string;
  template_version: number;
  channel: DeliveryChannel;
  status: DeliveryStatus;
  recipients_total: number;
  recipients_delivered: number;
  recipients_failed: number;
  recipient_details?: RecipientDeliveryDetail[];
  content_summary?: string;
  content_stats?: DeliveryContentStats;
  rendered_subject?: string;
  rendered_body_size?: number;
  started_at: string;
  completed_at?: string;
  duration_ms?: number;
  error_message?: string;
  error_details?: Record<string, unknown>;
  triggered_by: string;
  triggered_by_user_id?: string;
}

/** User newsletter preferences */
export interface NewsletterUserPreferences {
  user_id: string;
  username: string;
  global_opt_out: boolean;
  global_opt_out_at?: string;
  schedule_preferences?: Record<string, boolean>;
  preferred_channel?: DeliveryChannel;
  preferred_email?: string;
  language?: string;
  updated_at: string;
}

/** Newsletter audit entry */
export interface NewsletterAuditEntry {
  id: string;
  timestamp: string;
  actor_id: string;
  actor_username?: string;
  action: string;
  resource_type: string;
  resource_id: string;
  resource_name?: string;
  details?: Record<string, unknown>;
  ip_address?: string;
  user_agent?: string;
}

/** Newsletter statistics */
export interface NewsletterStats {
  total_templates: number;
  active_templates: number;
  total_schedules: number;
  enabled_schedules: number;
  total_deliveries: number;
  successful_deliveries: number;
  failed_deliveries: number;
  deliveries_by_channel: Record<string, number>;
  deliveries_by_type: Record<string, number>;
  last_7_days_deliveries: number;
  last_30_days_deliveries: number;
}

// ============================================================================
// Request/Response Types
// ============================================================================

/** Create template request */
export interface CreateTemplateRequest {
  name: string;
  description?: string;
  type: NewsletterType;
  subject: string;
  body_html: string;
  body_text?: string;
  default_config?: TemplateConfig;
}

/** Update template request (partial) */
export interface UpdateTemplateRequest {
  name?: string;
  description?: string;
  subject?: string;
  body_html?: string;
  body_text?: string;
  default_config?: TemplateConfig;
  is_active?: boolean;
}

/** Create schedule request */
export interface CreateScheduleRequest {
  name: string;
  description?: string;
  template_id: string;
  recipients: string[];
  cron_expression: string;
  timezone: string;
  config?: TemplateConfig;
  channels: DeliveryChannel[];
  channel_configs?: Record<DeliveryChannel, ChannelConfig>;
  is_enabled: boolean;
}

/** Update schedule request (partial) */
export interface UpdateScheduleRequest {
  name?: string;
  description?: string;
  template_id?: string;
  recipients?: string[];
  cron_expression?: string;
  timezone?: string;
  config?: TemplateConfig;
  channels?: DeliveryChannel[];
  channel_configs?: Record<DeliveryChannel, ChannelConfig>;
  is_enabled?: boolean;
}

/** Preview newsletter request */
export interface PreviewNewsletterRequest {
  template_id: string;
  config?: TemplateConfig;
}

/** Preview newsletter response */
export interface PreviewNewsletterResponse {
  subject: string;
  body_html: string;
  body_text?: string;
  data?: unknown;
}

/** List templates response */
export interface ListTemplatesResponse {
  templates: NewsletterTemplate[];
  total_count: number;
}

/** List schedules response */
export interface ListSchedulesResponse {
  schedules: NewsletterSchedule[];
  total_count: number;
}

/** List deliveries response */
export interface ListDeliveriesResponse {
  deliveries: NewsletterDelivery[];
  pagination: {
    limit: number;
    has_more: boolean;
    total_count?: number;
  };
}

/** List audit log response */
export interface ListAuditLogResponse {
  entries: NewsletterAuditEntry[];
  total_count: number;
}

/** Newsletter filter options */
export interface NewsletterFilter {
  type?: NewsletterType;
  active?: boolean;
  enabled?: boolean;
  template_id?: string;
  schedule_id?: string;
  status?: DeliveryStatus;
  channel?: DeliveryChannel;
  limit?: number;
  offset?: number;
}

/** Newsletter audit filter */
export interface NewsletterAuditFilter {
  resource_type?: string;
  resource_id?: string;
  actor_id?: string;
  action?: string;
  limit?: number;
  offset?: number;
}

// ============================================================================
// Display/UI Types
// ============================================================================

/** Newsletter type display configuration */
export interface NewsletterTypeConfig {
  label: string;
  description: string;
  icon: string;
  color: string;
}

/** Delivery channel display configuration */
export interface ChannelDisplayConfig {
  label: string;
  icon: string;
  color: string;
  requiresConfig: boolean;
}

/** Newsletter type display names and icons */
export const NEWSLETTER_TYPES: Record<NewsletterType, NewsletterTypeConfig> = {
  recently_added: {
    label: 'Recently Added',
    description: 'New content added to your library',
    icon: '\u2795',
    color: '#27ae60',
  },
  coming_soon: {
    label: 'Coming Soon',
    description: 'Upcoming releases and premieres',
    icon: '\u23F0',
    color: '#3498db',
  },
  weekly_digest: {
    label: 'Weekly Digest',
    description: 'Weekly summary of activity and content',
    icon: '\u{1F4C5}',
    color: '#9b59b6',
  },
  monthly_recap: {
    label: 'Monthly Recap',
    description: 'Monthly statistics and highlights',
    icon: '\u{1F4CA}',
    color: '#e74c3c',
  },
  watchlist_updates: {
    label: 'Watchlist Updates',
    description: 'Updates to items on your watchlist',
    icon: '\u2B50',
    color: '#f39c12',
  },
  trending_content: {
    label: 'Trending Content',
    description: 'Popular content on your server',
    icon: '\u{1F525}',
    color: '#e67e22',
  },
  personalized_recommendations: {
    label: 'Personalized Recommendations',
    description: 'Content suggestions based on viewing history',
    icon: '\u{1F3AF}',
    color: '#1abc9c',
  },
};

/** Channel display configuration */
export const CHANNEL_CONFIG: Record<DeliveryChannel, ChannelDisplayConfig> = {
  email: {
    label: 'Email',
    icon: '\u2709',
    color: '#3498db',
    requiresConfig: true,
  },
  discord: {
    label: 'Discord',
    icon: '\u{1F4AC}',
    color: '#5865F2',
    requiresConfig: true,
  },
  slack: {
    label: 'Slack',
    icon: '\u{1F4E2}',
    color: '#4A154B',
    requiresConfig: true,
  },
  telegram: {
    label: 'Telegram',
    icon: '\u2708',
    color: '#0088cc',
    requiresConfig: true,
  },
  webhook: {
    label: 'Webhook',
    icon: '\u{1F310}',
    color: '#95a5a6',
    requiresConfig: true,
  },
  in_app: {
    label: 'In-App',
    icon: '\u{1F514}',
    color: '#27ae60',
    requiresConfig: false,
  },
};

/** Delivery status display configuration */
export const STATUS_COLORS: Record<DeliveryStatus, string> = {
  pending: '#f39c12',
  sending: '#3498db',
  delivered: '#27ae60',
  failed: '#e74c3c',
  partial: '#e67e22',
  canceled: '#95a5a6',
};

/** Status display names */
export const STATUS_NAMES: Record<DeliveryStatus, string> = {
  pending: 'Pending',
  sending: 'Sending',
  delivered: 'Delivered',
  failed: 'Failed',
  partial: 'Partial',
  canceled: 'Canceled',
};
