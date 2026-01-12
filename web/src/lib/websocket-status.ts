// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * WebSocket Status UI Manager
 *
 * Provides visual feedback for WebSocket connection state.
 * Updates the status indicator in the header with:
 * - Connection state (connected, connecting, disconnected, error)
 * - Reconnection attempt count
 * - Accessible status announcements
 */

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'reconnecting' | 'error';

export interface StatusUpdateOptions {
  status: ConnectionStatus;
  reconnectAttempt?: number;
  maxReconnectAttempts?: number;
  message?: string;
}

export class WebSocketStatusUI {
  private statusElement: HTMLElement | null;
  private statusText: HTMLElement | null;
  private currentStatus: ConnectionStatus = 'connecting';

  constructor() {
    this.statusElement = document.getElementById('ws-status');
    this.statusText = this.statusElement?.querySelector('.ws-status-text') ?? null;
  }

  /**
   * Update the WebSocket connection status display
   */
  updateStatus(options: StatusUpdateOptions): void {
    if (!this.statusElement) return;

    this.currentStatus = options.status;

    // Update data attribute for CSS styling
    this.statusElement.setAttribute('data-status', options.status);

    // Update text content
    const displayText = this.getDisplayText(options);
    if (this.statusText) {
      this.statusText.textContent = displayText;
    }

    // Update tooltip
    const tooltipText = this.getTooltipText(options);
    this.statusElement.setAttribute('title', tooltipText);

    // Update aria-label for accessibility
    this.statusElement.setAttribute('aria-label', `WebSocket connection: ${tooltipText}`);

    // Add connected class for legacy compatibility
    this.statusElement.classList.toggle('connected', options.status === 'connected');
    this.statusElement.classList.toggle('disconnected', options.status === 'disconnected');
    this.statusElement.classList.toggle('reconnecting', options.status === 'reconnecting');
    this.statusElement.classList.toggle('error', options.status === 'error');
  }

  /**
   * Get display text for status badge
   */
  private getDisplayText(options: StatusUpdateOptions): string {
    switch (options.status) {
      case 'connected':
        return 'Live';
      case 'connecting':
        return 'Connecting';
      case 'reconnecting':
        if (options.reconnectAttempt !== undefined && options.maxReconnectAttempts !== undefined) {
          return `Retry ${options.reconnectAttempt}/${options.maxReconnectAttempts}`;
        }
        return 'Reconnecting';
      case 'disconnected':
        return 'Offline';
      case 'error':
        return 'Error';
      default:
        return 'Unknown';
    }
  }

  /**
   * Get tooltip text with more details
   */
  private getTooltipText(options: StatusUpdateOptions): string {
    switch (options.status) {
      case 'connected':
        return 'WebSocket: Connected - Real-time updates active';
      case 'connecting':
        return 'WebSocket: Connecting to server...';
      case 'reconnecting':
        if (options.reconnectAttempt !== undefined && options.maxReconnectAttempts !== undefined) {
          return `WebSocket: Reconnecting... (attempt ${options.reconnectAttempt} of ${options.maxReconnectAttempts})`;
        }
        return 'WebSocket: Reconnecting...';
      case 'disconnected':
        return options.message || 'WebSocket: Disconnected - Updates paused';
      case 'error':
        return options.message || 'WebSocket: Connection error';
      default:
        return 'WebSocket: Unknown state';
    }
  }

  /**
   * Get current connection status
   */
  getStatus(): ConnectionStatus {
    return this.currentStatus;
  }

  /**
   * Check if currently connected
   */
  isConnected(): boolean {
    return this.currentStatus === 'connected';
  }
}

// Singleton instance for global access
let wsStatusUI: WebSocketStatusUI | null = null;

export function getWebSocketStatusUI(): WebSocketStatusUI {
  if (!wsStatusUI) {
    wsStatusUI = new WebSocketStatusUI();
  }
  return wsStatusUI;
}
