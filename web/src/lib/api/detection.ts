// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Detection API Module
 *
 * Security detection alerts, rules, and trust scoring (ADR-0020).
 */

import type {
    DetectionRuleType,
    DetectionAlert,
    DetectionRule,
    UserTrustScore,
    DetectionAlertFilter,
    DetectionAlertsResponse,
    DetectionRulesResponse,
    LowTrustUsersResponse,
    DetectionMetrics,
    DetectionAlertStats,
    UpdateRuleRequest,
    AcknowledgeAlertRequest,
} from '../types/detection';
import { BaseAPIClient } from './client';

/**
 * Detection API methods
 */
export class DetectionAPI extends BaseAPIClient {
    // ========================================================================
    // Alerts
    // ========================================================================

    async getDetectionAlerts(filter: DetectionAlertFilter = {}): Promise<DetectionAlertsResponse> {
        const params = new URLSearchParams();
        if (filter.limit !== undefined) params.append('limit', filter.limit.toString());
        if (filter.offset !== undefined) params.append('offset', filter.offset.toString());
        if (filter.user_id !== undefined) params.append('user_id', filter.user_id.toString());
        if (filter.acknowledged !== undefined) params.append('acknowledged', filter.acknowledged.toString());
        if (filter.severity) params.append('severity', filter.severity);
        if (filter.rule_type) params.append('rule_type', filter.rule_type);
        if (filter.start_date) params.append('start_date', filter.start_date);
        if (filter.end_date) params.append('end_date', filter.end_date);
        if (filter.order_by) params.append('order_by', filter.order_by);
        if (filter.order_direction) params.append('order_direction', filter.order_direction);

        const queryString = params.toString();
        const url = queryString ? `/detection/alerts?${queryString}` : '/detection/alerts';

        const response = await this.fetch<DetectionAlertsResponse>(url);
        return response.data;
    }

    async getDetectionAlert(id: number): Promise<DetectionAlert> {
        const response = await this.fetch<DetectionAlert>(`/detection/alerts/${id}`);
        return response.data;
    }

    async acknowledgeDetectionAlert(id: number, acknowledgedBy: string): Promise<void> {
        const body: AcknowledgeAlertRequest = { acknowledged_by: acknowledgedBy };
        await this.fetch(`/detection/alerts/${id}/acknowledge`, {
            method: 'POST',
            body: JSON.stringify(body),
        });
    }

    // ========================================================================
    // Rules
    // ========================================================================

    async getDetectionRules(): Promise<DetectionRulesResponse> {
        const response = await this.fetch<DetectionRulesResponse>('/detection/rules');
        return response.data;
    }

    async getDetectionRule(ruleType: DetectionRuleType): Promise<DetectionRule> {
        const response = await this.fetch<DetectionRule>(`/detection/rules/${ruleType}`);
        return response.data;
    }

    async updateDetectionRule(ruleType: DetectionRuleType, update: UpdateRuleRequest): Promise<void> {
        await this.fetch(`/detection/rules/${ruleType}`, {
            method: 'PUT',
            body: JSON.stringify(update),
        });
    }

    async setDetectionRuleEnabled(ruleType: DetectionRuleType, enabled: boolean): Promise<void> {
        await this.fetch(`/detection/rules/${ruleType}/enable`, {
            method: 'POST',
            body: JSON.stringify({ enabled }),
        });
    }

    // ========================================================================
    // Trust Scores
    // ========================================================================

    async getUserTrustScore(userId: number): Promise<UserTrustScore> {
        const response = await this.fetch<UserTrustScore>(`/detection/users/${userId}/trust`);
        return response.data;
    }

    async getLowTrustUsers(threshold?: number): Promise<LowTrustUsersResponse> {
        const params = threshold !== undefined ? `?threshold=${threshold}` : '';
        const response = await this.fetch<LowTrustUsersResponse>(`/detection/users/low-trust${params}`);
        return response.data;
    }

    // ========================================================================
    // Metrics & Stats
    // ========================================================================

    async getDetectionMetrics(): Promise<DetectionMetrics> {
        const response = await this.fetch<DetectionMetrics>('/detection/metrics');
        return response.data;
    }

    async getDetectionAlertStats(): Promise<DetectionAlertStats> {
        const response = await this.fetch<DetectionAlertStats>('/detection/stats');
        return response.data;
    }
}
