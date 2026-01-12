// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
import { API, TautulliServerInfoData } from './api';
import { createLogger } from './logger';

const logger = createLogger('ServerDashboard');

export class ServerDashboardManager {
    private api: API;

    constructor(api: API) {
        this.api = api;
    }

    async init(): Promise<void> {
        await this.loadServerInfo();
    }

    destroy(): void {
        // Cleanup event listeners if needed
    }

    private async loadServerInfo(): Promise<void> {
        try {
            const serverInfo = await this.api.getTautulliServerInfo();
            this.renderServerInfo(serverInfo);
        } catch (error) {
            logger.error('Failed to load server info:', error);
            this.showError();
        }
    }

    private renderServerInfo(data: TautulliServerInfoData): void {
        // Update server status indicator (green for online)
        const statusEl = document.getElementById('server-status');
        if (statusEl) {
            statusEl.style.color = '#4ade80'; // Green
            statusEl.title = 'Server Online';
        } else {
            logger.error('Server status element not found');
        }

        // Update server details with null checks
        const serverName = document.getElementById('server-name');
        if (serverName) {
            serverName.textContent = data.plex_server_name || '-';
        }

        const serverVersion = document.getElementById('server-version');
        if (serverVersion) {
            serverVersion.textContent = data.plex_server_version || '-';
        }

        const serverPlatform = document.getElementById('server-platform');
        if (serverPlatform) {
            serverPlatform.textContent = data.platform || '-';
        }

        const serverPlatformVersion = document.getElementById('server-platform-version');
        if (serverPlatformVersion) {
            serverPlatformVersion.textContent = data.platform_version || '-';
        }

        const serverMachineId = document.getElementById('server-machine-id');
        if (serverMachineId) {
            serverMachineId.textContent = data.machine_identifier || '-';
        }

        // Show update notice if available
        const updateNotice = document.getElementById('server-update-available');
        const updateVersion = document.getElementById('server-update-version');

        if (updateNotice && updateVersion) {
            if (data.update_available === 1 && data.update_version) {
                updateNotice.style.display = 'block';
                updateVersion.textContent = `Version ${data.update_version} is available`;
                if (data.update_release_date) {
                    updateVersion.textContent += ` (Released: ${new Date(data.update_release_date).toLocaleDateString()})`;
                }
            } else {
                updateNotice.style.display = 'none';
            }
        }
    }

    private showError(): void {
        // Update status indicator to show error (if element exists)
        const statusEl = document.getElementById('server-status');
        if (statusEl) {
            statusEl.style.color = '#ef4444'; // Red
            statusEl.title = 'Error Loading Server Info';
        }

        // Set all server details to error state (without destroying DOM)
        const serverName = document.getElementById('server-name');
        if (serverName) serverName.textContent = 'Error';

        const serverVersion = document.getElementById('server-version');
        if (serverVersion) serverVersion.textContent = 'Error';

        const serverPlatform = document.getElementById('server-platform');
        if (serverPlatform) serverPlatform.textContent = 'Error';

        const serverPlatformVersion = document.getElementById('server-platform-version');
        if (serverPlatformVersion) serverPlatformVersion.textContent = 'Error';

        const serverMachineId = document.getElementById('server-machine-id');
        if (serverMachineId) serverMachineId.textContent = 'Error';

        // Hide update notice on error
        const updateNotice = document.getElementById('server-update-available');
        if (updateNotice) {
            updateNotice.style.display = 'none';
        }

        // Show error message with retry button (append, don't replace)
        const serverHealth = document.getElementById('server-health');
        if (serverHealth) {
            // Check if error message already exists to avoid duplicates
            const existingError = serverHealth.querySelector('.server-error-message');
            if (!existingError) {
                const errorDiv = document.createElement('div');
                errorDiv.className = 'server-error-message';
                errorDiv.style.cssText = 'margin-top: 20px; padding: 15px; background: rgba(239, 68, 68, 0.1); border: 1px solid #ef4444; border-radius: 4px;';
                errorDiv.innerHTML = `
                    <p style="margin: 0 0 10px 0; color: #ef4444; font-weight: 500;">Failed to load server information</p>
                    <button id="server-retry" style="padding: 8px 16px; background: #ef4444; color: white; border: none; border-radius: 4px; cursor: pointer;">Retry</button>
                `;
                serverHealth.appendChild(errorDiv);

                // Attach retry listener
                const retryButton = document.getElementById('server-retry');
                if (retryButton) {
                    retryButton.addEventListener('click', () => {
                        // Remove error message
                        errorDiv.remove();
                        // Retry loading
                        this.loadServerInfo();
                    });
                }
            }
        }
    }
}
