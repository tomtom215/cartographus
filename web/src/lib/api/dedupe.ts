// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Dedupe Audit API Module
 *
 * Dedupe audit operations for data management (ADR-0022).
 */

import type {
    DedupeAuditListResponse,
    DedupeAuditStats,
    DedupeAuditActionRequest,
    DedupeAuditRestoreResponse,
    DedupeAuditFilter,
} from '../types/dedupe';
import { BaseAPIClient } from './client';

/**
 * Dedupe Audit API methods
 */
export class DedupeAPI extends BaseAPIClient {
    async getDedupeAuditStats(): Promise<DedupeAuditStats> {
        const response = await this.fetch<DedupeAuditStats>('/dedupe/audit/stats');
        return response.data;
    }

    async getDedupeAuditEntries(filter: DedupeAuditFilter = {}): Promise<DedupeAuditListResponse> {
        const params = new URLSearchParams();
        if (filter.limit !== undefined) params.append('limit', filter.limit.toString());
        if (filter.offset !== undefined) params.append('offset', filter.offset.toString());
        if (filter.user_id !== undefined) params.append('user_id', filter.user_id.toString());
        if (filter.status) params.append('status', filter.status);
        if (filter.reason) params.append('reason', filter.reason);
        if (filter.layer) params.append('layer', filter.layer);
        if (filter.source) params.append('source', filter.source);
        if (filter.from) params.append('from', filter.from);
        if (filter.to) params.append('to', filter.to);

        const queryString = params.toString();
        const url = queryString ? `/dedupe/audit?${queryString}` : '/dedupe/audit';

        const response = await this.fetch<DedupeAuditListResponse>(url);
        return response.data;
    }

    async confirmDedupeEntry(id: string, request: DedupeAuditActionRequest = {}): Promise<void> {
        await this.fetch(`/dedupe/audit/${id}/confirm`, {
            method: 'POST',
            body: JSON.stringify(request),
        });
    }

    async restoreDedupeEntry(id: string, request: DedupeAuditActionRequest = {}): Promise<DedupeAuditRestoreResponse> {
        const response = await this.fetch<DedupeAuditRestoreResponse>(`/dedupe/audit/${id}/restore`, {
            method: 'POST',
            body: JSON.stringify(request),
        });
        return response.data;
    }

    getDedupeAuditExportUrl(filter: DedupeAuditFilter = {}): string {
        const params = new URLSearchParams();
        if (filter.status) params.append('status', filter.status);
        if (filter.reason) params.append('reason', filter.reason);
        if (filter.source) params.append('source', filter.source);
        const queryString = params.toString();
        return queryString
            ? `${this.baseURL}/dedupe/audit/export?${queryString}`
            : `${this.baseURL}/dedupe/audit/export`;
    }
}
