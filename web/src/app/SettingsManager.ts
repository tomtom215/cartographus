// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * SettingsManager - User preferences and settings management
 *
 * Features:
 * - Settings panel visibility control
 * - Theme settings (dark/light/high-contrast/system)
 * - Colorblind mode toggle
 * - Map visualization settings
 * - View mode settings (2D/3D)
 * - Auto-refresh settings
 * - Export/Import settings
 * - Clear all settings
 */

import type { ToastManager } from '../lib/toast';
import type { ThemeManager, Theme } from './ThemeManager';
import { createLogger } from '../lib/logger';
import { SafeStorage } from '../lib/utils/SafeStorage';

const logger = createLogger('SettingsManager');

export interface UserSettings {
    theme: Theme;
    colorblindMode: boolean;
    visualizationMode: 'points' | 'heatmap' | 'clusters';
    viewMode: '2d' | '3d';
    autoRefresh: boolean;
    autoRefreshInterval: number;
    sidebarCollapsed: boolean;
    arcOverlayEnabled: boolean;
}

const DEFAULT_SETTINGS: UserSettings = {
    theme: 'dark',
    colorblindMode: false,
    visualizationMode: 'points',
    viewMode: '2d',
    autoRefresh: false,
    autoRefreshInterval: 60000,
    sidebarCollapsed: false,
    arcOverlayEnabled: false
};

const SETTINGS_STORAGE_KEYS = [
    'theme',
    'colorblind-mode',
    'map-visualization-mode',
    'view-mode',
    'auto-refresh-enabled',
    'auto-refresh-interval',
    'sidebar-collapsed',
    'arc-overlay-enabled',
    'filter-presets',
    'stats-previous',
    'stats-sparkline-history',
    'last-data-update',
    'map-config',
    'onboarding-completed'
];

export class SettingsManager {
    private panel: HTMLElement | null = null;
    private overlay: HTMLElement | null = null;
    private isOpen: boolean = false;
    private toastManager: ToastManager | null = null;
    private themeManager: ThemeManager | null = null;

    // Event handler references for cleanup
    private settingsButtonClickHandler: (() => void) | null = null;
    private closeButtonClickHandler: (() => void) | null = null;
    private overlayClickHandler: (() => void) | null = null;
    private documentKeydownHandler: ((e: KeyboardEvent) => void) | null = null;
    private panelClickHandler: ((e: Event) => void) | null = null;
    private panelChangeHandler: ((e: Event) => void) | null = null;

    // Callbacks for applying settings
    private onThemeChange: ((theme: Theme) => void) | null = null;
    private onVisualizationModeChange: ((mode: string) => void) | null = null;
    private onViewModeChange: ((mode: string) => void) | null = null;
    private onAutoRefreshChange: ((enabled: boolean) => void) | null = null;

    constructor() {
        // Empty constructor - use init() to set up
    }

    /**
     * Initialize the settings manager
     */
    init(): void {
        this.panel = document.getElementById('settings-panel');
        this.overlay = document.getElementById('settings-overlay');

        if (!this.panel) {
            logger.error('Settings panel not found');
            return;
        }

        this.setupEventListeners();
        this.loadCurrentSettings();
        logger.debug('SettingsManager initialized');
    }

    /**
     * Set toast manager reference
     */
    setToastManager(toast: ToastManager): void {
        this.toastManager = toast;
    }

    /**
     * Set theme manager reference
     */
    setThemeManager(theme: ThemeManager): void {
        this.themeManager = theme;
    }

    /**
     * Set callback for theme changes
     */
    setOnThemeChange(callback: (theme: Theme) => void): void {
        this.onThemeChange = callback;
    }

    /**
     * Set callback for visualization mode changes
     */
    setOnVisualizationModeChange(callback: (mode: string) => void): void {
        this.onVisualizationModeChange = callback;
    }

    /**
     * Set callback for view mode changes
     */
    setOnViewModeChange(callback: (mode: string) => void): void {
        this.onViewModeChange = callback;
    }

    /**
     * Set callback for auto-refresh changes
     */
    setOnAutoRefreshChange(callback: (enabled: boolean) => void): void {
        this.onAutoRefreshChange = callback;
    }

    /**
     * Set up event listeners with stored references for cleanup
     */
    private setupEventListeners(): void {
        // Settings button in header
        const settingsButton = document.getElementById('settings-button');
        if (settingsButton) {
            this.settingsButtonClickHandler = () => this.open();
            settingsButton.addEventListener('click', this.settingsButtonClickHandler);
        }

        // Close button
        const closeButton = this.panel?.querySelector('.settings-close');
        if (closeButton) {
            this.closeButtonClickHandler = () => this.close();
            closeButton.addEventListener('click', this.closeButtonClickHandler);
        }

        // Overlay click to close
        if (this.overlay) {
            this.overlayClickHandler = () => this.close();
            this.overlay.addEventListener('click', this.overlayClickHandler);
        }

        // Escape key (document level - must be cleaned up!)
        this.documentKeydownHandler = (e: KeyboardEvent) => {
            if (e.key === 'Escape' && this.isOpen) {
                this.close();
            }
        };
        document.addEventListener('keydown', this.documentKeydownHandler);

        // Delegated click handler for panel (theme options, viz options, view options, buttons)
        if (this.panel) {
            this.panelClickHandler = (e: Event) => {
                const target = e.target as HTMLElement;
                const clickedElement = target.closest('[data-theme], [data-viz-mode], [data-view-mode], [id]') as HTMLElement;
                if (!clickedElement) return;

                // Theme options
                const theme = clickedElement.dataset.theme as Theme;
                if (theme) {
                    this.setTheme(theme);
                    return;
                }

                // Visualization mode options
                const vizMode = clickedElement.dataset.vizMode;
                if (vizMode) {
                    this.setVisualizationMode(vizMode);
                    return;
                }

                // View mode options
                const viewMode = clickedElement.dataset.viewMode;
                if (viewMode) {
                    this.setViewMode(viewMode);
                    return;
                }

                // Clear settings button
                if (clickedElement.id === 'btn-clear-settings') {
                    this.confirmClearSettings();
                    return;
                }

                // Export settings button
                if (clickedElement.id === 'btn-export-settings') {
                    this.exportSettings();
                    return;
                }

                // Import settings button
                if (clickedElement.id === 'btn-import-settings') {
                    this.triggerImport();
                }
            };
            this.panel.addEventListener('click', this.panelClickHandler);

            // Delegated change handler for toggles and inputs
            this.panelChangeHandler = (e: Event) => {
                const target = e.target as HTMLInputElement;

                // Colorblind mode toggle
                if (target.id === 'colorblind-settings-toggle') {
                    this.setColorblindMode(target.checked);
                    return;
                }

                // Auto-refresh toggle
                if (target.id === 'auto-refresh-toggle-settings') {
                    this.setAutoRefresh(target.checked);
                    return;
                }

                // Import file input
                if (target.id === 'settings-import-input' && target.files?.[0]) {
                    this.importSettings(target.files[0]);
                }
            };
            this.panel.addEventListener('change', this.panelChangeHandler);
        }
    }

    /**
     * Remove event listeners for cleanup
     */
    private removeEventListeners(): void {
        // Settings button
        const settingsButton = document.getElementById('settings-button');
        if (settingsButton && this.settingsButtonClickHandler) {
            settingsButton.removeEventListener('click', this.settingsButtonClickHandler);
        }

        // Close button
        const closeButton = this.panel?.querySelector('.settings-close');
        if (closeButton && this.closeButtonClickHandler) {
            closeButton.removeEventListener('click', this.closeButtonClickHandler);
        }

        // Overlay
        if (this.overlay && this.overlayClickHandler) {
            this.overlay.removeEventListener('click', this.overlayClickHandler);
        }

        // Document keydown (critical to clean up!)
        if (this.documentKeydownHandler) {
            document.removeEventListener('keydown', this.documentKeydownHandler);
        }

        // Panel delegated handlers
        if (this.panel) {
            if (this.panelClickHandler) {
                this.panel.removeEventListener('click', this.panelClickHandler);
            }
            if (this.panelChangeHandler) {
                this.panel.removeEventListener('change', this.panelChangeHandler);
            }
        }
    }

    /**
     * Load current settings into UI
     */
    private loadCurrentSettings(): void {
        // Load theme
        const currentTheme = SafeStorage.getItem('theme') as Theme || 'dark';
        this.updateThemeUI(currentTheme);

        // Load colorblind mode (in settings panel)
        const colorblindMode = SafeStorage.getItem('colorblind-mode') === 'true';
        const settingsColorblindToggle = document.getElementById('colorblind-settings-toggle') as HTMLInputElement;
        if (settingsColorblindToggle) {
            settingsColorblindToggle.checked = colorblindMode;
        }

        // Load visualization mode
        const vizMode = SafeStorage.getItem('map-visualization-mode') || 'points';
        this.updateVisualizationModeUI(vizMode);

        // Load view mode
        const viewMode = SafeStorage.getItem('view-mode') || '2d';
        this.updateViewModeUI(viewMode);

        // Load auto-refresh
        const autoRefresh = SafeStorage.getItem('auto-refresh-enabled') === 'true';
        const autoRefreshToggle = document.getElementById('auto-refresh-toggle-settings') as HTMLInputElement;
        if (autoRefreshToggle) {
            autoRefreshToggle.checked = autoRefresh;
        }
    }

    /**
     * Update theme option UI
     */
    private updateThemeUI(theme: Theme): void {
        this.panel?.querySelectorAll('.theme-option').forEach(option => {
            const optionTheme = (option as HTMLElement).dataset.theme;
            option.classList.toggle('active', optionTheme === theme);
        });
    }

    /**
     * Update visualization mode UI
     */
    private updateVisualizationModeUI(mode: string): void {
        this.panel?.querySelectorAll('.viz-mode-option').forEach(option => {
            const optionMode = (option as HTMLElement).dataset.vizMode;
            option.classList.toggle('active', optionMode === mode);
        });
    }

    /**
     * Update view mode UI
     */
    private updateViewModeUI(mode: string): void {
        this.panel?.querySelectorAll('.view-mode-option').forEach(option => {
            const optionMode = (option as HTMLElement).dataset.viewMode;
            option.classList.toggle('active', optionMode === mode);
        });
    }

    /**
     * Open settings panel
     */
    open(): void {
        if (!this.panel) return;

        this.isOpen = true;
        this.panel.style.display = 'block';
        this.panel.classList.add('visible');
        this.panel.setAttribute('aria-hidden', 'false');

        if (this.overlay) {
            this.overlay.style.display = 'block';
            this.overlay.classList.add('visible');
        }

        // Reload current settings
        this.loadCurrentSettings();

        // Focus first interactive element
        const firstFocusable = this.panel.querySelector('button, [tabindex="0"], input') as HTMLElement;
        setTimeout(() => firstFocusable?.focus(), 100);
    }

    /**
     * Close settings panel
     */
    close(): void {
        if (!this.panel) return;

        this.isOpen = false;
        this.panel.classList.remove('visible');
        this.panel.setAttribute('aria-hidden', 'true');

        if (this.overlay) {
            this.overlay.classList.remove('visible');
        }

        setTimeout(() => {
            if (this.panel && !this.isOpen) {
                this.panel.style.display = 'none';
            }
            if (this.overlay && !this.isOpen) {
                this.overlay.style.display = 'none';
            }
        }, 200);
    }

    /**
     * Set theme
     */
    private setTheme(theme: Theme): void {
        // Update storage and DOM
        SafeStorage.setItem('theme', theme);
        document.documentElement.setAttribute('data-theme', theme);

        // Update theme icon
        const themeIcon = document.getElementById('theme-icon');
        if (themeIcon) {
            const icons = {
                dark: '☀️',
                light: '🌙',
                'high-contrast': '◐'
            };
            themeIcon.textContent = icons[theme] || '☀️';
        }

        // Update theme button title/aria-label
        const themeButton = document.getElementById('theme-toggle');
        if (themeButton) {
            const labels = {
                'dark': 'Switch to light mode',
                'light': 'Switch to high contrast mode',
                'high-contrast': 'Switch to dark mode',
            };
            themeButton.setAttribute('aria-label', labels[theme]);
            themeButton.setAttribute('title', labels[theme]);
        }

        this.updateThemeUI(theme);
        this.onThemeChange?.(theme);
    }

    /**
     * Set colorblind mode
     */
    private setColorblindMode(enabled: boolean): void {
        SafeStorage.setItem('colorblind-mode', String(enabled));
        document.documentElement.classList.toggle('colorblind-mode', enabled);

        if (this.themeManager) {
            // ThemeManager may have its own colorblind handling
        }
    }

    /**
     * Set visualization mode
     */
    private setVisualizationMode(mode: string): void {
        SafeStorage.setItem('map-visualization-mode', mode);
        this.updateVisualizationModeUI(mode);
        this.onVisualizationModeChange?.(mode);
    }

    /**
     * Set view mode
     */
    private setViewMode(mode: string): void {
        SafeStorage.setItem('view-mode', mode);
        this.updateViewModeUI(mode);
        this.onViewModeChange?.(mode);
    }

    /**
     * Set auto-refresh
     */
    private setAutoRefresh(enabled: boolean): void {
        SafeStorage.setItem('auto-refresh-enabled', String(enabled));
        this.onAutoRefreshChange?.(enabled);
    }

    /**
     * Confirm and clear all settings
     */
    private async confirmClearSettings(): Promise<void> {
        const confirmed = confirm('Are you sure you want to clear all settings? This will reset everything to defaults and reload the page.');

        if (confirmed) {
            this.clearAllSettings();
        }
    }

    /**
     * Clear all settings from localStorage
     */
    private clearAllSettings(): void {
        SETTINGS_STORAGE_KEYS.forEach(key => {
            SafeStorage.removeItem(key);
        });

        this.toastManager?.success('All settings cleared. Reloading...');

        // Reload page to apply defaults
        setTimeout(() => {
            window.location.reload();
        }, 1000);
    }

    /**
     * Export all settings to JSON file
     */
    private exportSettings(): void {
        const settings: Record<string, string | null> = {};

        SETTINGS_STORAGE_KEYS.forEach(key => {
            const value = SafeStorage.getItem(key);
            if (value !== null) {
                settings[key] = value;
            }
        });

        const blob = new Blob([JSON.stringify(settings, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);

        const a = document.createElement('a');
        a.href = url;
        a.download = `cartographus-settings-${new Date().toISOString().split('T')[0]}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        this.toastManager?.success('Settings exported successfully');
    }

    /**
     * Trigger import file dialog
     */
    private triggerImport(): void {
        const input = document.getElementById('settings-import-input') as HTMLInputElement;
        input?.click();
    }

    /**
     * Import settings from JSON file
     */
    private async importSettings(file: File): Promise<void> {
        try {
            const text = await file.text();
            const settings = JSON.parse(text);

            if (typeof settings !== 'object' || settings === null) {
                throw new Error('Invalid settings format');
            }

            // Apply each setting
            Object.entries(settings).forEach(([key, value]) => {
                if (SETTINGS_STORAGE_KEYS.includes(key) && typeof value === 'string') {
                    SafeStorage.setItem(key, value);
                }
            });

            this.toastManager?.success('Settings imported successfully. Reloading...');

            // Reload to apply
            setTimeout(() => {
                window.location.reload();
            }, 1000);
        } catch (error) {
            logger.error('Failed to import settings:', error);
            this.toastManager?.error('Failed to import settings. Invalid file format.');
        }

        // Reset file input
        const input = document.getElementById('settings-import-input') as HTMLInputElement;
        if (input) {
            input.value = '';
        }
    }

    /**
     * Get current settings
     */
    getCurrentSettings(): UserSettings {
        return {
            theme: (SafeStorage.getItem('theme') as Theme) || DEFAULT_SETTINGS.theme,
            colorblindMode: SafeStorage.getItem('colorblind-mode') === 'true',
            visualizationMode: (SafeStorage.getItem('map-visualization-mode') as UserSettings['visualizationMode']) || DEFAULT_SETTINGS.visualizationMode,
            viewMode: (SafeStorage.getItem('view-mode') as UserSettings['viewMode']) || DEFAULT_SETTINGS.viewMode,
            autoRefresh: SafeStorage.getItem('auto-refresh-enabled') === 'true',
            autoRefreshInterval: parseInt(SafeStorage.getItem('auto-refresh-interval') || String(DEFAULT_SETTINGS.autoRefreshInterval), 10),
            sidebarCollapsed: SafeStorage.getItem('sidebar-collapsed') === 'true',
            arcOverlayEnabled: SafeStorage.getItem('arc-overlay-enabled') === 'true'
        };
    }

    /**
     * Check if panel is open
     */
    isPanelOpen(): boolean {
        return this.isOpen;
    }

    /**
     * Clean up resources and event listeners
     */
    destroy(): void {
        this.removeEventListeners();
        this.settingsButtonClickHandler = null;
        this.closeButtonClickHandler = null;
        this.overlayClickHandler = null;
        this.documentKeydownHandler = null;
        this.panelClickHandler = null;
        this.panelChangeHandler = null;
        this.panel = null;
        this.overlay = null;
    }
}

export default SettingsManager;
