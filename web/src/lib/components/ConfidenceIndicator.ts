// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * ConfidenceIndicator - Visual confidence score display component
 *
 * Displays a confidence score (0.0-1.0) as a color-coded progress bar
 * with optional percentage and label text.
 */

import { escapeHtml } from '../sanitize';

/**
 * Confidence level thresholds
 */
const CONFIDENCE_THRESHOLDS = {
  high: 0.8,    // >= 80%
  medium: 0.5,  // >= 50%
  low: 0        // < 50%
};

/**
 * Options for rendering a confidence indicator
 */
export interface ConfidenceIndicatorOptions {
  /** Confidence value between 0.0 and 1.0 */
  confidence: number;
  /** Show the percentage value */
  showValue?: boolean;
  /** Show the confidence level label (High/Medium/Low) */
  showLabel?: boolean;
  /** Custom label text */
  label?: string;
  /** Size variant */
  size?: 'small' | 'medium' | 'large';
  /** Whether to animate the bar fill */
  animated?: boolean;
}

/**
 * ConfidenceIndicator class for displaying confidence scores
 */
export class ConfidenceIndicator {
  private options: ConfidenceIndicatorOptions;
  private element: HTMLElement | null = null;
  private fillElement: HTMLElement | null = null;

  constructor(options: ConfidenceIndicatorOptions) {
    this.options = {
      showValue: true,
      showLabel: false,
      size: 'medium',
      animated: true,
      ...options
    };

    // Clamp confidence to valid range
    this.options.confidence = Math.max(0, Math.min(1, this.options.confidence));
  }

  /**
   * Render the confidence indicator and return the HTML element
   */
  render(): HTMLElement {
    const { confidence, showValue, showLabel, label, size } = this.options;
    const level = this.getConfidenceLevel();
    const percentage = Math.round(confidence * 100);

    // Create container
    this.element = document.createElement('div');
    this.element.className = `confidence-indicator confidence-indicator--${size}`;
    this.element.setAttribute('role', 'meter');
    this.element.setAttribute('aria-valuenow', String(percentage));
    this.element.setAttribute('aria-valuemin', '0');
    this.element.setAttribute('aria-valuemax', '100');
    this.element.setAttribute('aria-label', `Confidence: ${percentage}%`);

    // Build HTML
    let html = `
      <div class="confidence-bar">
        <div class="confidence-bar-fill confidence-${level}" style="width: ${percentage}%"></div>
      </div>
    `;

    if (showValue) {
      html += `<span class="confidence-value">${percentage}%</span>`;
    }

    if (showLabel) {
      const labelText = label || this.getConfidenceLabelText(level);
      html += `<span class="confidence-label">${escapeHtml(labelText)}</span>`;
    }

    this.element.innerHTML = html;

    // Store reference to fill element for updates
    this.fillElement = this.element.querySelector('.confidence-bar-fill');

    return this.element;
  }

  /**
   * Get the confidence level based on value
   */
  private getConfidenceLevel(): 'high' | 'medium' | 'low' {
    const { confidence } = this.options;
    if (confidence >= CONFIDENCE_THRESHOLDS.high) return 'high';
    if (confidence >= CONFIDENCE_THRESHOLDS.medium) return 'medium';
    return 'low';
  }

  /**
   * Get display text for confidence level
   */
  private getConfidenceLabelText(level: 'high' | 'medium' | 'low'): string {
    switch (level) {
      case 'high': return 'High';
      case 'medium': return 'Medium';
      case 'low': return 'Low';
    }
  }

  /**
   * Update the confidence value
   */
  setConfidence(confidence: number): void {
    this.options.confidence = Math.max(0, Math.min(1, confidence));
    const level = this.getConfidenceLevel();
    const percentage = Math.round(this.options.confidence * 100);

    if (this.fillElement) {
      // Update width
      this.fillElement.style.width = `${percentage}%`;

      // Update level class
      this.fillElement.classList.remove('confidence-high', 'confidence-medium', 'confidence-low');
      this.fillElement.classList.add(`confidence-${level}`);
    }

    if (this.element) {
      this.element.setAttribute('aria-valuenow', String(percentage));
      this.element.setAttribute('aria-label', `Confidence: ${percentage}%`);

      // Update value display if present
      const valueEl = this.element.querySelector('.confidence-value');
      if (valueEl) {
        valueEl.textContent = `${percentage}%`;
      }

      // Update label if present
      const labelEl = this.element.querySelector('.confidence-label');
      if (labelEl && !this.options.label) {
        labelEl.textContent = this.getConfidenceLabelText(level);
      }
    }
  }

  /**
   * Get the current confidence value
   */
  getConfidence(): number {
    return this.options.confidence;
  }

  /**
   * Get the current confidence level
   */
  getLevel(): 'high' | 'medium' | 'low' {
    return this.getConfidenceLevel();
  }

  /**
   * Destroy the indicator and cleanup
   */
  destroy(): void {
    this.element = null;
    this.fillElement = null;
  }
}

/**
 * Create a simple confidence indicator HTML string (for use in templates)
 */
export function renderConfidenceIndicatorHTML(
  confidence: number,
  options?: Partial<ConfidenceIndicatorOptions>
): string {
  const clampedConfidence = Math.max(0, Math.min(1, confidence));
  const percentage = Math.round(clampedConfidence * 100);
  const level = clampedConfidence >= 0.8 ? 'high' : clampedConfidence >= 0.5 ? 'medium' : 'low';

  const showValue = options?.showValue !== false;
  const showLabel = options?.showLabel === true;
  const label = options?.label;

  let html = `
    <div class="confidence-indicator" role="meter" aria-valuenow="${percentage}" aria-valuemin="0" aria-valuemax="100" aria-label="Confidence: ${percentage}%">
      <div class="confidence-bar">
        <div class="confidence-bar-fill confidence-${level}" style="width: ${percentage}%"></div>
      </div>
  `;

  if (showValue) {
    html += `<span class="confidence-value">${percentage}%</span>`;
  }

  if (showLabel) {
    const labelText = label || (level === 'high' ? 'High' : level === 'medium' ? 'Medium' : 'Low');
    html += `<span class="confidence-label">${escapeHtml(labelText)}</span>`;
  }

  html += '</div>';

  return html;
}

/**
 * Get confidence level from a value
 */
export function getConfidenceLevel(confidence: number): 'high' | 'medium' | 'low' {
  if (confidence >= CONFIDENCE_THRESHOLDS.high) return 'high';
  if (confidence >= CONFIDENCE_THRESHOLDS.medium) return 'medium';
  return 'low';
}

/**
 * Get a human-readable description for a confidence level
 */
export function getConfidenceDescription(confidence: number): string {
  const percentage = Math.round(confidence * 100);
  const level = getConfidenceLevel(confidence);

  switch (level) {
    case 'high':
      return `High confidence (${percentage}%) - This match is very likely correct.`;
    case 'medium':
      return `Medium confidence (${percentage}%) - This match is probable but should be verified.`;
    case 'low':
      return `Low confidence (${percentage}%) - This match is uncertain and requires manual review.`;
  }
}

export default ConfidenceIndicator;
