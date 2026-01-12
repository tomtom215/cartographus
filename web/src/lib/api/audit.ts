// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Audit Log API Module
 *
 * Security audit log operations.
 */

import type {
    AuditEvent,
    AuditEventsResponse,
    AuditStats,
    AuditEventFilter,
    AuditTypesResponse,
    AuditSeveritiesResponse,
} from '../types/audit';
import { BaseAPIClient } from './client';

/**
 * Audit Log API methods
 */
export class AuditAPI extends BaseAPIClient {
    async getAuditEvents(filter: AuditEventFilter = {}): Promise<AuditEventsResponse> {
        const params = new URLSearchParams();
        if (filter.limit !== undefined) params.append('limit', filter.limit.toString());
        if (filter.offset !== undefined) params.append('offset', filter.offset.toString());
        if (filter.types) {
            filter.types.forEach(t => params.append('type', t));
        }
        if (filter.severities) {
            filter.severities.forEach(s => params.append('severity', s));
        }
        if (filter.outcomes) {
            filter.outcomes.forEach(o => params.append('outcome', o));
        }
        if (filter.actor_id) params.append('actor_id', filter.actor_id);
        if (filter.actor_type) params.append('actor_type', filter.actor_type);
        if (filter.target_id) params.append('target_id', filter.target_id);
        if (filter.target_type) params.append('target_type', filter.target_type);
        if (filter.source_ip) params.append('source_ip', filter.source_ip);
        if (filter.start_time) params.append('start_time', filter.start_time);
        if (filter.end_time) params.append('end_time', filter.end_time);
        if (filter.search) params.append('search', filter.search);
        if (filter.correlation_id) params.append('correlation_id', filter.correlation_id);
        if (filter.request_id) params.append('request_id', filter.request_id);
        if (filter.order_by) params.append('order_by', filter.order_by);
        if (filter.order_direction) params.append('order_direction', filter.order_direction);

        const queryString = params.toString();
        const url = queryString ? `/audit/events?${queryString}` : '/audit/events';

        const response = await this.fetch<AuditEventsResponse>(url);
        return response.data;
    }

    async getAuditEvent(id: string): Promise<AuditEvent> {
        const response = await this.fetch<AuditEvent>(`/audit/events/${id}`);
        return response.data;
    }

    async getAuditStats(): Promise<AuditStats> {
        const response = await this.fetch<AuditStats>('/audit/stats');
        return response.data;
    }

    async getAuditTypes(): Promise<string[]> {
        const response = await this.fetch<AuditTypesResponse>('/audit/types');
        return response.data.types;
    }

    async getAuditSeverities(): Promise<string[]> {
        const response = await this.fetch<AuditSeveritiesResponse>('/audit/severities');
        return response.data.severities;
    }

    getAuditExportUrl(format: 'json' | 'cef' = 'json', filter: AuditEventFilter = {}): string {
        const params = new URLSearchParams();
        params.append('format', format);
        if (filter.types) {
            filter.types.forEach(t => params.append('type', t));
        }
        if (filter.start_time) params.append('start_time', filter.start_time);
        if (filter.end_time) params.append('end_time', filter.end_time);
        return `${this.baseURL}/audit/export?${params.toString()}`;
    }
}
