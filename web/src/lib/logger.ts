// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Logger - Production-Safe Logging
 *
 * Provides a centralized logging utility that can be configured
 * to disable verbose logging in production while preserving
 * error and warning logs.
 *
 * Features:
 * - Environment-aware logging levels
 * - Preserves error/warn in production
 * - Strips debug/log in production via esbuild drop
 * - Consistent log formatting
 * - Performance-safe (no-op in production)
 *
 * Usage:
 *   import { logger } from './lib/logger';
 *   logger.debug('Debug message');  // Stripped in production
 *   logger.log('Info message');     // Stripped in production
 *   logger.warn('Warning');         // Preserved in production
 *   logger.error('Error');          // Preserved in production
 *
 * Reference: UI/UX Audit Task
 * @see /docs/working/UI_UX_AUDIT.md
 */

/**
 * Log levels from least to most severe
 */
export enum LogLevel {
  DEBUG = 0,
  LOG = 1,
  WARN = 2,
  ERROR = 3,
  NONE = 4,
}

/**
 * Environment detection
 */
const isDevelopment = typeof process !== 'undefined'
  ? process.env.NODE_ENV !== 'production'
  : !window.location.hostname.includes('localhost') === false;

/**
 * Default log level based on environment
 * Production: WARN (only warnings and errors)
 * Development: DEBUG (all logs)
 */
const DEFAULT_LOG_LEVEL = isDevelopment ? LogLevel.DEBUG : LogLevel.WARN;

/**
 * Logger configuration
 */
interface LoggerConfig {
  level: LogLevel;
  prefix?: string;
  timestamps?: boolean;
}

/**
 * Logger class for environment-aware logging
 */
class Logger {
  private level: LogLevel;
  private prefix: string;
  private timestamps: boolean;

  constructor(config: Partial<LoggerConfig> = {}) {
    this.level = config.level ?? DEFAULT_LOG_LEVEL;
    this.prefix = config.prefix ?? '[Cartographus]';
    this.timestamps = config.timestamps ?? isDevelopment;
  }

  /**
   * Set the log level
   */
  setLevel(level: LogLevel): void {
    this.level = level;
  }

  /**
   * Get the current log level
   */
  getLevel(): LogLevel {
    return this.level;
  }

  /**
   * Format a log message with optional prefix and timestamp
   */
  private format(...args: unknown[]): unknown[] {
    const parts: unknown[] = [];

    if (this.prefix) {
      parts.push(this.prefix);
    }

    if (this.timestamps) {
      parts.push(new Date().toISOString());
    }

    return [...parts, ...args];
  }

  /**
   * Debug level logging - stripped in production builds
   * Use for verbose debugging information
   */
  debug(...args: unknown[]): void {
    if (this.level <= LogLevel.DEBUG) {
      // eslint-disable-next-line no-console
      console.debug(...this.format(...args));
    }
  }

  /**
   * Log level logging - stripped in production builds
   * Use for general information
   */
  log(...args: unknown[]): void {
    if (this.level <= LogLevel.LOG) {
      // eslint-disable-next-line no-console
      console.log(...this.format(...args));
    }
  }

  /**
   * Info alias for log - stripped in production builds
   */
  info(...args: unknown[]): void {
    this.log(...args);
  }

  /**
   * Warning level logging - preserved in production
   * Use for non-critical issues
   */
  warn(...args: unknown[]): void {
    if (this.level <= LogLevel.WARN) {
      // eslint-disable-next-line no-console
      console.warn(...this.format(...args));
    }
  }

  /**
   * Error level logging - preserved in production
   * Use for errors that need attention
   */
  error(...args: unknown[]): void {
    if (this.level <= LogLevel.ERROR) {
      // eslint-disable-next-line no-console
      console.error(...this.format(...args));
    }
  }

  /**
   * Group console output - development only
   */
  group(label?: string): void {
    if (this.level <= LogLevel.DEBUG) {
      // eslint-disable-next-line no-console
      console.group(label);
    }
  }

  /**
   * End console group - development only
   */
  groupEnd(): void {
    if (this.level <= LogLevel.DEBUG) {
      // eslint-disable-next-line no-console
      console.groupEnd();
    }
  }

  /**
   * Time measurement start - development only
   */
  time(label?: string): void {
    if (this.level <= LogLevel.DEBUG) {
      // eslint-disable-next-line no-console
      console.time(label);
    }
  }

  /**
   * Time measurement end - development only
   */
  timeEnd(label?: string): void {
    if (this.level <= LogLevel.DEBUG) {
      // eslint-disable-next-line no-console
      console.timeEnd(label);
    }
  }

  /**
   * Create a child logger with a specific prefix
   */
  child(prefix: string): Logger {
    return new Logger({
      level: this.level,
      prefix: `${this.prefix}[${prefix}]`,
      timestamps: this.timestamps,
    });
  }
}

/**
 * Default logger instance
 * Import this for most logging needs
 */
export const logger = new Logger();

/**
 * Create a named logger for a specific module
 *
 * @example
 * const log = createLogger('ChartManager');
 * log.debug('Chart initialized');
 */
export function createLogger(name: string): Logger {
  return logger.child(name);
}

/**
 * Export Logger class for custom instances
 */
export { Logger };

/**
 * Quick check if we're in development mode
 */
export const isDevMode = isDevelopment;
