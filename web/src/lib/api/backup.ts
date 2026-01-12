// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Backup API Module
 *
 * Backup/restore operations and retention policy management.
 */

import type {
    Backup,
    BackupStats,
    CreateBackupRequest,
    RestoreBackupRequest,
    RestoreResult,
    RetentionPolicy,
    RetentionPreview,
    RetentionApplyResult,
    ScheduleConfig,
    SetScheduleConfigRequest,
} from '../types/backup';
import { BaseAPIClient } from './client';

/**
 * Backup API methods
 */
export class BackupAPI extends BaseAPIClient {
    // ========================================================================
    // Backup Operations
    // ========================================================================

    async listBackups(): Promise<Backup[]> {
        return this.fetchSimple<Backup[]>('/backups');
    }

    async getBackupStats(): Promise<BackupStats> {
        return this.fetchSimple<BackupStats>('/backup/stats');
    }

    async createBackup(request: CreateBackupRequest = {}): Promise<Backup> {
        const response = await this.fetch<Backup>('/backup', {
            method: 'POST',
            body: JSON.stringify(request)
        });
        return response.data;
    }

    async deleteBackup(backupId: string): Promise<void> {
        await this.fetch<{ deleted: boolean }>(`/backups/delete?id=${encodeURIComponent(backupId)}`, {
            method: 'DELETE'
        });
    }

    getBackupDownloadUrl(backupId: string): string {
        return `${this.baseURL}/backups/download?id=${encodeURIComponent(backupId)}`;
    }

    async validateBackup(backupId: string): Promise<{ is_valid: boolean; issues?: string[] }> {
        const response = await this.fetch<{ is_valid: boolean; issues?: string[] }>(
            `/backups/validate?id=${encodeURIComponent(backupId)}`,
            { method: 'POST' }
        );
        return response.data;
    }

    async restoreBackup(backupId: string, options: RestoreBackupRequest = {}): Promise<RestoreResult> {
        const response = await this.fetch<RestoreResult>(
            `/backups/restore?id=${encodeURIComponent(backupId)}`,
            {
                method: 'POST',
                body: JSON.stringify(options)
            }
        );
        return response.data;
    }

    // ========================================================================
    // Retention Policy (Task 29)
    // ========================================================================

    async getRetentionPolicy(): Promise<RetentionPolicy> {
        return this.fetchSimple<RetentionPolicy>('/backup/retention');
    }

    async setRetentionPolicy(policy: RetentionPolicy): Promise<RetentionPolicy> {
        const response = await this.fetch<RetentionPolicy>('/backup/retention', {
            method: 'PUT',
            body: JSON.stringify(policy)
        });
        return response.data;
    }

    async getRetentionPreview(): Promise<RetentionPreview> {
        return this.fetchSimple<RetentionPreview>('/backup/retention/preview');
    }

    async applyRetention(): Promise<RetentionApplyResult> {
        const response = await this.fetch<RetentionApplyResult>('/backup/retention/apply', {
            method: 'POST'
        });
        return response.data;
    }

    async cleanupCorruptedBackups(): Promise<{ deleted_count: number; freed_bytes: number }> {
        const response = await this.fetch<{ deleted_count: number; freed_bytes: number }>('/backup/cleanup', {
            method: 'POST'
        });
        return response.data;
    }

    // ========================================================================
    // Schedule Configuration
    // ========================================================================

    async getScheduleConfig(): Promise<ScheduleConfig> {
        return this.fetchSimple<ScheduleConfig>('/backup/schedule');
    }

    async setScheduleConfig(config: SetScheduleConfigRequest): Promise<ScheduleConfig> {
        const response = await this.fetch<ScheduleConfig>('/backup/schedule', {
            method: 'PUT',
            body: JSON.stringify(config)
        });
        return response.data;
    }

    async triggerScheduledBackup(): Promise<Backup> {
        const response = await this.fetch<Backup>('/backup/schedule/trigger', {
            method: 'POST'
        });
        return response.data;
    }
}
