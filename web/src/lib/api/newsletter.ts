// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Newsletter API Module
 *
 * Newsletter Generator API operations for templates, schedules, deliveries, and preferences.
 */

import type {
    NewsletterTemplate,
    NewsletterSchedule,
    NewsletterDelivery,
    NewsletterStats,
    NewsletterUserPreferences,
    NewsletterAuditEntry,
    CreateTemplateRequest,
    UpdateTemplateRequest,
    CreateScheduleRequest,
    UpdateScheduleRequest,
    PreviewNewsletterRequest,
    PreviewNewsletterResponse,
    ListTemplatesResponse,
    ListSchedulesResponse,
    ListDeliveriesResponse,
    NewsletterFilter,
    NewsletterAuditFilter,
} from '../types/newsletter';
import { BaseAPIClient } from './client';

/**
 * Newsletter API methods
 */
export class NewsletterAPI extends BaseAPIClient {
    // =========================================================================
    // Templates
    // =========================================================================

    /**
     * List newsletter templates with optional filtering
     */
    async getTemplates(filter: NewsletterFilter = {}): Promise<ListTemplatesResponse> {
        const params = new URLSearchParams();
        if (filter.type) params.append('type', filter.type);
        if (filter.active !== undefined) params.append('active', String(filter.active));
        if (filter.limit !== undefined) params.append('limit', filter.limit.toString());
        if (filter.offset !== undefined) params.append('offset', filter.offset.toString());

        const queryString = params.toString();
        const url = queryString ? `/newsletter/templates?${queryString}` : '/newsletter/templates';

        const response = await this.fetch<ListTemplatesResponse>(url);
        return response.data;
    }

    /**
     * Get a specific template by ID
     */
    async getTemplate(id: string): Promise<NewsletterTemplate> {
        const response = await this.fetch<NewsletterTemplate>(`/newsletter/templates/${id}`);
        return response.data;
    }

    /**
     * Create a new newsletter template
     */
    async createTemplate(req: CreateTemplateRequest): Promise<NewsletterTemplate> {
        const response = await this.fetch<NewsletterTemplate>('/newsletter/templates', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(req),
        });
        return response.data;
    }

    /**
     * Update an existing template
     */
    async updateTemplate(id: string, req: UpdateTemplateRequest): Promise<NewsletterTemplate> {
        const response = await this.fetch<NewsletterTemplate>(`/newsletter/templates/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(req),
        });
        return response.data;
    }

    /**
     * Delete a template
     */
    async deleteTemplate(id: string): Promise<void> {
        await this.fetch<void>(`/newsletter/templates/${id}`, {
            method: 'DELETE',
        });
    }

    /**
     * Preview a template with sample data
     */
    async previewTemplate(req: PreviewNewsletterRequest): Promise<PreviewNewsletterResponse> {
        const response = await this.fetch<PreviewNewsletterResponse>('/newsletter/templates/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(req),
        });
        return response.data;
    }

    // =========================================================================
    // Schedules
    // =========================================================================

    /**
     * List newsletter schedules with optional filtering
     */
    async getSchedules(filter: NewsletterFilter = {}): Promise<ListSchedulesResponse> {
        const params = new URLSearchParams();
        if (filter.template_id) params.append('template_id', filter.template_id);
        if (filter.enabled !== undefined) params.append('enabled', String(filter.enabled));
        if (filter.limit !== undefined) params.append('limit', filter.limit.toString());
        if (filter.offset !== undefined) params.append('offset', filter.offset.toString());

        const queryString = params.toString();
        const url = queryString ? `/newsletter/schedules?${queryString}` : '/newsletter/schedules';

        const response = await this.fetch<ListSchedulesResponse>(url);
        return response.data;
    }

    /**
     * Get a specific schedule by ID
     */
    async getSchedule(id: string): Promise<NewsletterSchedule> {
        const response = await this.fetch<NewsletterSchedule>(`/newsletter/schedules/${id}`);
        return response.data;
    }

    /**
     * Create a new newsletter schedule
     */
    async createSchedule(req: CreateScheduleRequest): Promise<NewsletterSchedule> {
        const response = await this.fetch<NewsletterSchedule>('/newsletter/schedules', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(req),
        });
        return response.data;
    }

    /**
     * Update an existing schedule
     */
    async updateSchedule(id: string, req: UpdateScheduleRequest): Promise<NewsletterSchedule> {
        const response = await this.fetch<NewsletterSchedule>(`/newsletter/schedules/${id}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(req),
        });
        return response.data;
    }

    /**
     * Delete a schedule
     */
    async deleteSchedule(id: string): Promise<void> {
        await this.fetch<void>(`/newsletter/schedules/${id}`, {
            method: 'DELETE',
        });
    }

    /**
     * Trigger immediate delivery of a schedule
     */
    async triggerSchedule(id: string): Promise<NewsletterDelivery> {
        const response = await this.fetch<NewsletterDelivery>(`/newsletter/schedules/${id}/trigger`, {
            method: 'POST',
        });
        return response.data;
    }

    /**
     * Enable a schedule
     */
    async enableSchedule(id: string): Promise<NewsletterSchedule> {
        return this.updateSchedule(id, { is_enabled: true });
    }

    /**
     * Disable a schedule
     */
    async disableSchedule(id: string): Promise<NewsletterSchedule> {
        return this.updateSchedule(id, { is_enabled: false });
    }

    // =========================================================================
    // Deliveries
    // =========================================================================

    /**
     * List newsletter deliveries with optional filtering
     */
    async getDeliveries(filter: NewsletterFilter = {}): Promise<ListDeliveriesResponse> {
        const params = new URLSearchParams();
        if (filter.schedule_id) params.append('schedule_id', filter.schedule_id);
        if (filter.status) params.append('status', filter.status);
        if (filter.limit !== undefined) params.append('limit', filter.limit.toString());
        if (filter.offset !== undefined) params.append('offset', filter.offset.toString());

        const queryString = params.toString();
        const url = queryString ? `/newsletter/deliveries?${queryString}` : '/newsletter/deliveries';

        const response = await this.fetch<ListDeliveriesResponse>(url);
        return response.data;
    }

    /**
     * Get a specific delivery by ID
     */
    async getDelivery(id: string): Promise<NewsletterDelivery> {
        const response = await this.fetch<NewsletterDelivery>(`/newsletter/deliveries/${id}`);
        return response.data;
    }

    // =========================================================================
    // Statistics
    // =========================================================================

    /**
     * Get newsletter statistics
     */
    async getStats(): Promise<NewsletterStats> {
        const response = await this.fetch<NewsletterStats>('/newsletter/stats');
        return response.data;
    }

    // =========================================================================
    // Audit Log
    // =========================================================================

    /**
     * Get newsletter audit log entries
     */
    async getAuditLog(filter: NewsletterAuditFilter = {}): Promise<{ entries: NewsletterAuditEntry[]; total_count: number }> {
        const params = new URLSearchParams();
        if (filter.resource_type) params.append('resource_type', filter.resource_type);
        if (filter.resource_id) params.append('resource_id', filter.resource_id);
        if (filter.actor_id) params.append('actor_id', filter.actor_id);
        if (filter.action) params.append('action', filter.action);
        if (filter.limit !== undefined) params.append('limit', filter.limit.toString());
        if (filter.offset !== undefined) params.append('offset', filter.offset.toString());

        const queryString = params.toString();
        const url = queryString ? `/newsletter/audit?${queryString}` : '/newsletter/audit';

        const response = await this.fetch<{ entries: NewsletterAuditEntry[]; total_count: number }>(url);
        return response.data;
    }

    // =========================================================================
    // User Preferences
    // =========================================================================

    /**
     * Get current user's newsletter preferences
     */
    async getUserPreferences(): Promise<NewsletterUserPreferences> {
        const response = await this.fetch<NewsletterUserPreferences>('/user/newsletter/preferences');
        return response.data;
    }

    /**
     * Update current user's newsletter preferences
     */
    async updateUserPreferences(prefs: Partial<NewsletterUserPreferences>): Promise<NewsletterUserPreferences> {
        const response = await this.fetch<NewsletterUserPreferences>('/user/newsletter/preferences', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(prefs),
        });
        return response.data;
    }

    /**
     * Unsubscribe from all newsletters
     */
    async unsubscribe(): Promise<void> {
        await this.fetch<void>('/user/newsletter/unsubscribe', {
            method: 'POST',
        });
    }
}
