// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Wrapped Reports API Module
 *
 * API methods for the Annual Wrapped Reports feature.
 * Provides endpoints for viewing, generating, and sharing wrapped reports.
 */

import type {
    WrappedReport,
    WrappedServerStats,
    WrappedLeaderboardEntry,
    WrappedGenerateRequest,
    WrappedGenerateResponse,
} from '../types/wrapped';
import { BaseAPIClient } from './client';

/**
 * Wrapped Reports API methods
 */
export class WrappedAPI extends BaseAPIClient {
    // ========================================================================
    // Wrapped Report Retrieval
    // ========================================================================

    /**
     * Get server-wide wrapped statistics for a year
     * @param year The year to get statistics for
     */
    async getWrappedServerStats(year: number): Promise<WrappedServerStats> {
        const response = await this.fetch<WrappedServerStats>(`/wrapped/${year}`);
        return response.data;
    }

    /**
     * Get a wrapped report for a specific user
     * @param year The year for the report
     * @param userID The user ID to get the report for
     * @param generate If true, generate the report if it doesn't exist
     */
    async getWrappedUserReport(year: number, userID: number, generate: boolean = false): Promise<WrappedReport> {
        const params = new URLSearchParams();
        if (generate) {
            params.append('generate', 'true');
        }
        const queryString = params.toString();
        const url = queryString ? `/wrapped/${year}/user/${userID}?${queryString}` : `/wrapped/${year}/user/${userID}`;
        const response = await this.fetch<WrappedReport>(url);
        return response.data;
    }

    /**
     * Get the wrapped leaderboard for a year
     * @param year The year for the leaderboard
     * @param limit Maximum number of entries (default: 10)
     */
    async getWrappedLeaderboard(year: number, limit: number = 10): Promise<WrappedLeaderboardEntry[]> {
        const params = new URLSearchParams();
        params.append('limit', limit.toString());
        const response = await this.fetch<WrappedLeaderboardEntry[]>(`/wrapped/${year}/leaderboard?${params}`);
        return response.data;
    }

    /**
     * Get a shared wrapped report by share token
     * @param token The share token for the report
     */
    async getWrappedByShareToken(token: string): Promise<WrappedReport> {
        const response = await this.fetch<WrappedReport>(`/wrapped/share/${token}`);
        return response.data;
    }

    // ========================================================================
    // Wrapped Report Generation (Admin)
    // ========================================================================

    /**
     * Trigger wrapped report generation
     * @param request Generation request with year, optional user ID, and force flag
     */
    async generateWrappedReports(request: WrappedGenerateRequest): Promise<WrappedGenerateResponse> {
        const response = await this.fetch<WrappedGenerateResponse>(
            `/wrapped/${request.year}/generate`,
            {
                method: 'POST',
                body: JSON.stringify(request),
            }
        );
        return response.data;
    }
}
