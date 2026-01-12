// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ThemeManager - Handles theme detection, switching, and persistence
 *
 * Manages three theme modes:
 * - dark (default)
 * - light
 * - high-contrast (WCAG 2.1 AAA compliance)
 *
 * Also manages colorblind-safe mode:
 * - Uses blue-orange-teal palette safe for deuteranopia/protanopia
 * - Persisted separately from theme selection
 *
 * Features:
 * - System preference detection (prefers-color-scheme, prefers-contrast)
 * - Theme cycling with localStorage persistence
 * - Colorblind mode toggle with localStorage persistence
 * - Automatic theme switching on system preference changes
 * - Globe theme synchronization
 * - Chart theme synchronization
 * @see /docs/working/UI_UX_AUDIT.md
 */

import type { GlobeManagerDeckGL } from '../lib/globe-deckgl';

export type Theme = 'dark' | 'light' | 'high-contrast';

/**
 * Minimal interface for ChartManager to avoid circular imports
 */
interface ChartManagerInterface {
    updateTheme(theme: Theme): void;
    updateColorblindMode?(enabled: boolean): void;
}

export class ThemeManager {
    private currentTheme: Theme = 'dark';
    private colorblindMode: boolean = false;
    private globeManager: GlobeManagerDeckGL | null = null;
    private chartManager: ChartManagerInterface | null = null;
    /** Bound handler for colorblind toggle */
    private colorblindToggleHandler: (() => void) | null = null;
    /** Bound handler for dark mode media query */
    private darkModeHandler: ((e: MediaQueryListEvent) => void) | null = null;
    /** Bound handler for high contrast media query */
    private highContrastHandler: ((e: MediaQueryListEvent) => void) | null = null;
    /** Media query for dark mode preference */
    private darkModeQuery: MediaQueryList | null = null;
    /** Media query for high contrast preference */
    private highContrastQuery: MediaQueryList | null = null;

    constructor() {
        this.initTheme();
        this.initColorblindMode();
        this.setupColorblindToggle();
    }

    /**
     * Set globe manager reference for theme synchronization
     */
    setGlobeManager(globeManager: GlobeManagerDeckGL | null): void {
        this.globeManager = globeManager;

        // Apply current theme to globe if available
        if (this.globeManager) {
            this.globeManager.updateTheme(this.currentTheme !== 'light');
        }
    }

    /**
     * Set chart manager reference for theme synchronization
     */
    setChartManager(chartManager: ChartManagerInterface): void {
        this.chartManager = chartManager;

        // Apply current theme to charts if available
        if (this.chartManager) {
            this.chartManager.updateTheme(this.currentTheme);
        }
    }

    /**
     * Get current theme
     */
    getCurrentTheme(): Theme {
        return this.currentTheme;
    }

    /**
     * Get colorblind mode status
     */
    isColorblindMode(): boolean {
        return this.colorblindMode;
    }

    /**
     * Initialize colorblind mode from localStorage
     */
    private initColorblindMode(): void {
        const saved = localStorage.getItem('colorblind-mode');
        this.colorblindMode = saved === 'true';
        this.applyColorblindMode(this.colorblindMode);
    }

    /**
     * Setup colorblind toggle button listener
     */
    private setupColorblindToggle(): void {
        const toggle = document.getElementById('colorblind-toggle');
        if (toggle) {
            this.colorblindToggleHandler = () => this.toggleColorblindMode();
            toggle.addEventListener('click', this.colorblindToggleHandler);
        }
    }

    /**
     * Toggle colorblind mode
     */
    toggleColorblindMode(): void {
        this.colorblindMode = !this.colorblindMode;
        this.applyColorblindMode(this.colorblindMode);
        localStorage.setItem('colorblind-mode', this.colorblindMode.toString());

        // Notify charts to update colors
        if (this.chartManager?.updateColorblindMode) {
            this.chartManager.updateColorblindMode(this.colorblindMode);
        }
    }

    /**
     * Apply colorblind mode to document
     */
    private applyColorblindMode(enabled: boolean): void {
        if (enabled) {
            document.documentElement.setAttribute('data-colorblind', 'true');
        } else {
            document.documentElement.removeAttribute('data-colorblind');
        }

        // Update toggle button visual state
        const toggle = document.getElementById('colorblind-toggle');
        const icon = document.getElementById('colorblind-icon');

        if (toggle) {
            toggle.classList.toggle('active', enabled);
            toggle.setAttribute('aria-pressed', enabled.toString());
            toggle.setAttribute('aria-label', enabled ? 'Disable colorblind-safe mode' : 'Enable colorblind-safe mode');
            toggle.setAttribute('title', enabled ? 'Disable colorblind-safe mode' : 'Enable colorblind-safe mode');
        }

        if (icon) {
            // Use different icon to indicate active state
            icon.textContent = enabled ? '👁️‍🗨️' : '👁️';
        }
    }

    /**
     * Initialize theme from localStorage or system preferences
     */
    private initTheme(): void {
        const savedTheme = localStorage.getItem('theme') as Theme | null;
        this.darkModeQuery = window.matchMedia('(prefers-color-scheme: dark)');
        const prefersLight = window.matchMedia('(prefers-color-scheme: light)').matches;
        this.highContrastQuery = window.matchMedia('(prefers-contrast: more)');

        // Priority: saved preference > high contrast preference > system preference > dark mode (default)
        // Default to dark mode for media apps (matches user expectations and reduces eye strain)
        // Note: In headless browsers, both prefersDark and prefersLight may be false, so we default to dark
        const systemTheme = this.darkModeQuery.matches ? 'dark' : (prefersLight ? 'light' : 'dark');
        this.currentTheme = savedTheme || (this.highContrastQuery.matches ? 'high-contrast' : systemTheme);
        this.applyTheme(this.currentTheme);

        // Listen for system theme changes (store handler for cleanup)
        this.darkModeHandler = (e) => {
            if (!localStorage.getItem('theme')) {
                this.currentTheme = e.matches ? 'dark' : 'light';
                this.applyTheme(this.currentTheme);
            }
        };
        this.darkModeQuery.addEventListener('change', this.darkModeHandler);

        // Listen for high contrast preference changes (store handler for cleanup)
        this.highContrastHandler = (e) => {
            if (!localStorage.getItem('theme') && e.matches) {
                this.currentTheme = 'high-contrast';
                this.applyTheme(this.currentTheme);
            }
        };
        this.highContrastQuery.addEventListener('change', this.highContrastHandler);
    }

    /**
     * Apply theme to document
     * Also notifies globe and chart managers for synchronized theming
     */
    private applyTheme(theme: Theme): void {
        if (theme === 'light') {
            document.documentElement.setAttribute('data-theme', 'light');
        } else if (theme === 'high-contrast') {
            document.documentElement.setAttribute('data-theme', 'high-contrast');
        } else {
            document.documentElement.removeAttribute('data-theme');
        }

        const themeIcon = document.getElementById('theme-icon');
        const themeButton = document.getElementById('theme-toggle');
        if (themeIcon) {
            // Cycle: dark → light → high-contrast → dark
            const icons = {
                'dark': '☀️',
                'light': '🌓',
                'high-contrast': '⚫',
            };
            themeIcon.textContent = icons[theme];
        }

        if (themeButton) {
            const labels = {
                'dark': 'Switch to light mode',
                'light': 'Switch to high contrast mode',
                'high-contrast': 'Switch to dark mode',
            };
            themeButton.setAttribute('aria-label', labels[theme]);
            themeButton.setAttribute('title', labels[theme]);
        }

        // Synchronize globe theme
        if (this.globeManager) {
            // Globe uses dark theme for both dark and high-contrast modes
            this.globeManager.updateTheme(theme !== 'light');
        }

        // Synchronize chart theme
        if (this.chartManager) {
            this.chartManager.updateTheme(theme);
        }
    }

    /**
     * Toggle theme in cycle: dark → light → high-contrast → dark
     */
    toggleTheme(): void {
        // Cycle through: dark → light → high-contrast → dark
        if (this.currentTheme === 'dark') {
            this.currentTheme = 'light';
        } else if (this.currentTheme === 'light') {
            this.currentTheme = 'high-contrast';
        } else {
            this.currentTheme = 'dark';
        }
        this.applyTheme(this.currentTheme);
        localStorage.setItem('theme', this.currentTheme);
    }

    /**
     * Cleanup resources to prevent memory leaks
     * Removes event listeners and clears manager references
     */
    destroy(): void {
        // Remove colorblind toggle listener
        if (this.colorblindToggleHandler) {
            const toggle = document.getElementById('colorblind-toggle');
            toggle?.removeEventListener('click', this.colorblindToggleHandler);
            this.colorblindToggleHandler = null;
        }

        // Remove dark mode media query listener
        if (this.darkModeQuery && this.darkModeHandler) {
            this.darkModeQuery.removeEventListener('change', this.darkModeHandler);
            this.darkModeHandler = null;
            this.darkModeQuery = null;
        }

        // Remove high contrast media query listener
        if (this.highContrastQuery && this.highContrastHandler) {
            this.highContrastQuery.removeEventListener('change', this.highContrastHandler);
            this.highContrastHandler = null;
            this.highContrastQuery = null;
        }

        // Clear manager references
        this.globeManager = null;
        this.chartManager = null;
    }
}
