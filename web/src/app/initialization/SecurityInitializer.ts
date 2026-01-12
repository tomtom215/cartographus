// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Security Managers Initializer
 *
 * Initializes security-related managers: SecurityAlerts, DetectionRules,
 * DedupeAudit, CrossPlatform, DataGovernance, Confirmation, ErrorBoundary,
 * DataFreshness, FilterPreset, Backup, ServerManagement, Settings.
 */

import type { API } from '../../lib/api';
import type { ToastManager } from '../../lib/toast';
import type { FilterManager } from '../../lib/filters';
import type { ThemeManager } from '../ThemeManager';
import type { NavigationManager } from '../NavigationManager';
import { SecurityAlertsManager } from '../SecurityAlertsManager';
import { DetectionRulesManager } from '../DetectionRulesManager';
import { DedupeAuditManager } from '../DedupeAuditManager';
import { CrossPlatformManager } from '../CrossPlatformManager';
import { DataGovernanceManager } from '../DataGovernanceManager';
import { ConfirmationDialogManager } from '../ConfirmationDialogManager';
import { DataFreshnessManager } from '../DataFreshnessManager';
import { ErrorBoundaryManager } from '../ErrorBoundaryManager';
import { FilterPresetManager } from '../FilterPresetManager';
import { BackupRestoreManager } from '../BackupRestoreManager';
import { ServerManagementManager } from '../ServerManagementManager';
import { SettingsManager } from '../SettingsManager';
import { DataExportManager } from '../DataExportManager';

export interface SecurityManagers {
    securityAlertsManager: SecurityAlertsManager;
    detectionRulesManager: DetectionRulesManager;
    dedupeAuditManager: DedupeAuditManager;
    crossPlatformManager: CrossPlatformManager;
    dataGovernanceManager: DataGovernanceManager;
    confirmationDialogManager: ConfirmationDialogManager;
    dataFreshnessManager: DataFreshnessManager;
    errorBoundaryManager: ErrorBoundaryManager;
    filterPresetManager: FilterPresetManager;
    backupRestoreManager: BackupRestoreManager;
    serverManagementManager: ServerManagementManager;
    settingsManager: SettingsManager;
    dataExportManager: DataExportManager;
}

export interface SecurityInitConfig {
    api: API;
    toastManager: ToastManager;
    filterManager: FilterManager;
    themeManager: ThemeManager;
    navigationManager: NavigationManager;
    loadData: () => Promise<void>;
    triggerSync: () => Promise<void>;
}

/**
 * Initialize security and governance managers
 */
export function initializeSecurityManagers(config: SecurityInitConfig): SecurityManagers {
    // Security alerts manager (ADR-0020)
    const securityAlertsManager = new SecurityAlertsManager(config.api);
    securityAlertsManager.init();

    // Detection rules manager (ADR-0020)
    const detectionRulesManager = new DetectionRulesManager(config.api);
    detectionRulesManager.init();

    // Dedupe audit manager (ADR-0022)
    const dedupeAuditManager = new DedupeAuditManager(config.api);
    dedupeAuditManager.init();

    // Cross-platform manager
    const crossPlatformManager = new CrossPlatformManager(config.api);
    // Initialized lazily when user navigates to the tab

    // Set callback for when cross-platform view is shown
    config.navigationManager.setCrossPlatformShowCallback(() => {
        if (!crossPlatformManager.isInitialized()) {
            crossPlatformManager.init('cross-platform-container');
        }
    });

    // Data governance manager
    const dataGovernanceManager = new DataGovernanceManager(config.api);
    // Initialized lazily when user navigates to the tab

    // Set callback for when data governance view is shown
    config.navigationManager.setDataGovernanceShowCallback(() => {
        dataGovernanceManager.init();
    });

    // Confirmation dialog manager
    const confirmationDialogManager = new ConfirmationDialogManager();
    confirmationDialogManager.init();

    // Data freshness manager
    const dataFreshnessManager = new DataFreshnessManager();
    dataFreshnessManager.setRefreshCallback(config.loadData);
    dataFreshnessManager.init();

    // Error boundary manager
    const errorBoundaryManager = new ErrorBoundaryManager();
    errorBoundaryManager.setRetryCallback(config.loadData);
    errorBoundaryManager.init();

    // Filter preset manager
    const filterPresetManager = new FilterPresetManager();
    filterPresetManager.setFilterManager(config.filterManager);
    filterPresetManager.setToastManager(config.toastManager);
    filterPresetManager.init();

    // Backup/restore manager
    const backupRestoreManager = new BackupRestoreManager(config.api);
    backupRestoreManager.setToastManager(config.toastManager);
    backupRestoreManager.setConfirmationManager(confirmationDialogManager);
    backupRestoreManager.init();

    // Server management manager
    const serverManagementManager = new ServerManagementManager(config.api);
    serverManagementManager.setToastManager(config.toastManager);
    serverManagementManager.init();

    // Settings manager
    const settingsManager = new SettingsManager();
    settingsManager.setToastManager(config.toastManager);
    settingsManager.setThemeManager(config.themeManager);
    settingsManager.init();

    // Data export manager
    const dataExportManager = new DataExportManager(config.filterManager, config.toastManager);

    return {
        securityAlertsManager,
        detectionRulesManager,
        dedupeAuditManager,
        crossPlatformManager,
        dataGovernanceManager,
        confirmationDialogManager,
        dataFreshnessManager,
        errorBoundaryManager,
        filterPresetManager,
        backupRestoreManager,
        serverManagementManager,
        settingsManager,
        dataExportManager
    };
}
