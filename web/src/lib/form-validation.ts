// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * Form Validation Utility - WCAG 2.1 AA Compliant
 *
 * Provides accessible, real-time form validation with:
 * - ARIA attributes for screen readers (aria-invalid, aria-describedby)
 * - Live region announcements for validation state changes
 * - Debounced validation to avoid noise
 * - Consistent error message patterns
 * - Password policy validation (NIST SP 800-63B)
 *
 * Usage:
 *   const validator = new FormValidator(formElement);
 *   validator.addField('username', {
 *     required: true,
 *     minLength: 3,
 *     pattern: /^[a-zA-Z0-9_]+$/,
 *     messages: {
 *       required: 'Username is required',
 *       minLength: 'Username must be at least 3 characters',
 *       pattern: 'Username can only contain letters, numbers, and underscores'
 *     }
 *   });
 *   validator.validate(); // Returns boolean
 */

import { createLogger } from './logger';

const logger = createLogger('FormValidation');

/**
 * Validation rule types
 */
export interface ValidationRule {
  required?: boolean;
  minLength?: number;
  maxLength?: number;
  min?: number;
  max?: number;
  pattern?: RegExp;
  email?: boolean;
  password?: boolean; // NIST SP 800-63B compliant
  match?: string; // Field name to match (e.g., password confirmation)
  custom?: (value: string, formData: FormData) => string | null; // Custom validator
  messages?: Partial<ValidationMessages>;
}

export interface ValidationMessages {
  required: string;
  minLength: string;
  maxLength: string;
  min: string;
  max: string;
  pattern: string;
  email: string;
  password: string;
  match: string;
  custom: string;
}

export interface FieldState {
  element: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement;
  rules: ValidationRule;
  errorElement: HTMLElement | null;
  isValid: boolean;
  isTouched: boolean;
  errorMessage: string;
}

export interface ValidationResult {
  isValid: boolean;
  errors: Map<string, string>;
  firstInvalidField: string | null;
}

/**
 * Default validation messages
 */
const DEFAULT_MESSAGES: ValidationMessages = {
  required: 'This field is required',
  minLength: 'Must be at least {min} characters',
  maxLength: 'Must be no more than {max} characters',
  min: 'Must be at least {min}',
  max: 'Must be no more than {max}',
  pattern: 'Invalid format',
  email: 'Please enter a valid email address',
  password: 'Password does not meet requirements',
  match: 'Fields do not match',
  custom: 'Invalid value',
};

/**
 * Common password patterns to reject (NIST SP 800-63B)
 */
const COMMON_PASSWORDS = new Set([
  'password', 'password123', 'password1', '123456', '12345678',
  'qwerty', 'abc123', 'monkey', 'master', 'dragon', 'letmein',
  'login', 'admin', 'welcome', 'passw0rd', 'shadow', 'sunshine',
  'princess', '123123', 'football', 'iloveyou', 'trustno1',
]);

/**
 * Debounce utility for validation
 */
function debounce<T extends (...args: Parameters<T>) => void>(
  fn: T,
  delay: number
): (...args: Parameters<T>) => void {
  let timeoutId: ReturnType<typeof setTimeout> | null = null;
  return (...args: Parameters<T>) => {
    if (timeoutId) clearTimeout(timeoutId);
    timeoutId = setTimeout(() => fn(...args), delay);
  };
}

/**
 * FormValidator class - manages form validation state and accessibility
 */
export class FormValidator {
  private form: HTMLFormElement;
  private fields: Map<string, FieldState> = new Map();
  private liveRegion: HTMLElement | null = null;
  private validateOnBlur: boolean;
  private validateOnInput: boolean;
  private debounceMs: number;

  constructor(
    form: HTMLFormElement,
    options: {
      validateOnBlur?: boolean;
      validateOnInput?: boolean;
      debounceMs?: number;
    } = {}
  ) {
    this.form = form;
    this.validateOnBlur = options.validateOnBlur ?? true;
    this.validateOnInput = options.validateOnInput ?? false;
    this.debounceMs = options.debounceMs ?? 300;

    this.setupLiveRegion();
    this.setupFormSubmit();
  }

  /**
   * Create a live region for screen reader announcements
   */
  private setupLiveRegion(): void {
    // Check if live region already exists
    this.liveRegion = this.form.querySelector('[data-validation-live-region]');
    if (!this.liveRegion) {
      this.liveRegion = document.createElement('div');
      this.liveRegion.setAttribute('aria-live', 'polite');
      this.liveRegion.setAttribute('aria-atomic', 'true');
      this.liveRegion.setAttribute('data-validation-live-region', 'true');
      this.liveRegion.className = 'sr-only';
      this.form.appendChild(this.liveRegion);
    }
  }

  /**
   * Setup form submit handler
   */
  private setupFormSubmit(): void {
    this.form.addEventListener('submit', (e) => {
      const result = this.validate();
      if (!result.isValid) {
        e.preventDefault();
        this.focusFirstInvalidField();
        this.announceErrors(result.errors);
      }
    });
  }

  /**
   * Add a field to be validated
   */
  addField(
    name: string,
    rules: ValidationRule
  ): this {
    const element = this.form.querySelector<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>(
      `[name="${name}"], #${name}`
    );

    if (!element) {
      logger.warn(`Field not found: ${name}`);
      return this;
    }

    // Create or find error element
    let errorElement = this.findOrCreateErrorElement(element, name);

    const state: FieldState = {
      element,
      rules,
      errorElement,
      isValid: true,
      isTouched: false,
      errorMessage: '',
    };

    this.fields.set(name, state);

    // Setup accessibility attributes
    this.setupFieldAccessibility(element, errorElement, name);

    // Setup event listeners
    this.setupFieldListeners(name, state);

    return this;
  }

  /**
   * Find or create an error element for a field
   */
  private findOrCreateErrorElement(
    element: HTMLElement,
    name: string
  ): HTMLElement {
    const errorId = `${name}-error`;
    let errorElement = document.getElementById(errorId);

    if (!errorElement) {
      errorElement = document.createElement('div');
      errorElement.id = errorId;
      errorElement.className = 'field-error';
      errorElement.setAttribute('role', 'alert');
      errorElement.setAttribute('aria-live', 'polite');

      // Insert after the input element
      const parent = element.parentElement;
      if (parent) {
        parent.insertBefore(errorElement, element.nextSibling);
      }
    }

    return errorElement;
  }

  /**
   * Setup accessibility attributes on a field
   */
  private setupFieldAccessibility(
    element: HTMLElement,
    errorElement: HTMLElement,
    _name: string
  ): void {
    // Link input to error message
    const existingDescribedBy = element.getAttribute('aria-describedby');
    const errorId = errorElement.id;

    if (existingDescribedBy) {
      if (!existingDescribedBy.includes(errorId)) {
        element.setAttribute('aria-describedby', `${existingDescribedBy} ${errorId}`);
      }
    } else {
      element.setAttribute('aria-describedby', errorId);
    }

    // Initialize as valid
    element.setAttribute('aria-invalid', 'false');
  }

  /**
   * Setup event listeners for a field
   */
  private setupFieldListeners(name: string, state: FieldState): void {
    const { element } = state;

    // Mark as touched on blur
    element.addEventListener('blur', () => {
      state.isTouched = true;
      if (this.validateOnBlur) {
        this.validateField(name);
      }
    });

    // Validate on input with debounce
    if (this.validateOnInput) {
      const debouncedValidate = debounce(() => {
        if (state.isTouched) {
          this.validateField(name);
        }
      }, this.debounceMs);

      element.addEventListener('input', debouncedValidate);
    }

    // Clear error state when user starts typing (immediate feedback)
    element.addEventListener('input', () => {
      if (!state.isValid && state.isTouched) {
        // Revalidate to potentially clear the error
        this.validateField(name);
      }
    });
  }

  /**
   * Validate a single field
   */
  validateField(name: string): boolean {
    const state = this.fields.get(name);
    if (!state) return true;

    const { element, rules } = state;
    const value = element.value.trim();
    const error = this.getFieldError(value, rules, name);

    state.isValid = !error;
    state.errorMessage = error || '';

    this.updateFieldUI(state);

    return state.isValid;
  }

  /**
   * Get the error message for a field value
   */
  private getFieldError(
    value: string,
    rules: ValidationRule,
    _fieldName: string
  ): string | null {
    const messages = { ...DEFAULT_MESSAGES, ...rules.messages };

    // Required check
    if (rules.required && !value) {
      return messages.required;
    }

    // Skip other validations if empty and not required
    if (!value) return null;

    // Min length
    if (rules.minLength && value.length < rules.minLength) {
      return messages.minLength.replace('{min}', String(rules.minLength));
    }

    // Max length
    if (rules.maxLength && value.length > rules.maxLength) {
      return messages.maxLength.replace('{max}', String(rules.maxLength));
    }

    // Numeric min
    if (rules.min !== undefined) {
      const numValue = parseFloat(value);
      if (isNaN(numValue) || numValue < rules.min) {
        return messages.min.replace('{min}', String(rules.min));
      }
    }

    // Numeric max
    if (rules.max !== undefined) {
      const numValue = parseFloat(value);
      if (isNaN(numValue) || numValue > rules.max) {
        return messages.max.replace('{max}', String(rules.max));
      }
    }

    // Pattern
    if (rules.pattern && !rules.pattern.test(value)) {
      return messages.pattern;
    }

    // Email
    if (rules.email) {
      const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
      if (!emailPattern.test(value)) {
        return messages.email;
      }
    }

    // Password (NIST SP 800-63B)
    if (rules.password) {
      const passwordError = this.validatePassword(value);
      if (passwordError) {
        return passwordError;
      }
    }

    // Match another field
    if (rules.match) {
      const matchState = this.fields.get(rules.match);
      if (matchState && matchState.element.value !== value) {
        return messages.match;
      }
    }

    // Custom validator
    if (rules.custom) {
      const formData = new FormData(this.form);
      const customError = rules.custom(value, formData);
      if (customError) {
        return customError;
      }
    }

    return null;
  }

  /**
   * Validate password against NIST SP 800-63B requirements
   */
  private validatePassword(password: string): string | null {
    // Minimum 12 characters
    if (password.length < 12) {
      return 'Password must be at least 12 characters';
    }

    // Check for common passwords
    if (COMMON_PASSWORDS.has(password.toLowerCase())) {
      return 'This password is too common. Please choose a stronger password.';
    }

    // Require complexity: uppercase, lowercase, digit, special char
    const hasUppercase = /[A-Z]/.test(password);
    const hasLowercase = /[a-z]/.test(password);
    const hasDigit = /\d/.test(password);
    const hasSpecial = /[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password);

    if (!hasUppercase || !hasLowercase || !hasDigit || !hasSpecial) {
      return 'Password must include uppercase, lowercase, number, and special character';
    }

    // Check for sequential characters
    if (/(.)\1{2,}/.test(password)) {
      return 'Password cannot contain repeated characters (e.g., "aaa")';
    }

    return null;
  }

  /**
   * Update the UI state for a field
   */
  private updateFieldUI(state: FieldState): void {
    const { element, errorElement, isValid, errorMessage, isTouched } = state;

    // Only show error if field has been touched
    const showError = !isValid && isTouched;

    // Update aria-invalid
    element.setAttribute('aria-invalid', String(!isValid));

    // Update error element
    if (errorElement) {
      errorElement.textContent = showError ? errorMessage : '';
      errorElement.style.display = showError ? 'block' : 'none';
    }

    // Update CSS classes
    element.classList.toggle('field-invalid', showError);
    element.classList.toggle('field-valid', isValid && isTouched);
  }

  /**
   * Validate all fields
   */
  validate(): ValidationResult {
    const errors = new Map<string, string>();
    let firstInvalidField: string | null = null;

    for (const [name, state] of this.fields) {
      // Mark all fields as touched when validating entire form
      state.isTouched = true;

      this.validateField(name);

      if (!state.isValid) {
        errors.set(name, state.errorMessage);
        if (!firstInvalidField) {
          firstInvalidField = name;
        }
      }
    }

    return {
      isValid: errors.size === 0,
      errors,
      firstInvalidField,
    };
  }

  /**
   * Focus the first invalid field
   */
  focusFirstInvalidField(): void {
    for (const [, state] of this.fields) {
      if (!state.isValid) {
        state.element.focus();
        break;
      }
    }
  }

  /**
   * Announce errors to screen readers
   */
  private announceErrors(errors: Map<string, string>): void {
    if (!this.liveRegion || errors.size === 0) return;

    const errorCount = errors.size;
    const announcement = errorCount === 1
      ? `Form has 1 error: ${errors.values().next().value}`
      : `Form has ${errorCount} errors. Please correct the highlighted fields.`;

    this.liveRegion.textContent = announcement;
  }

  /**
   * Reset all field states
   */
  reset(): void {
    for (const [_name, state] of this.fields) {
      state.isValid = true;
      state.isTouched = false;
      state.errorMessage = '';
      this.updateFieldUI(state);
    }
  }

  /**
   * Set a custom error on a field (e.g., from server response)
   */
  setError(name: string, message: string): void {
    const state = this.fields.get(name);
    if (!state) return;

    state.isValid = false;
    state.isTouched = true;
    state.errorMessage = message;
    this.updateFieldUI(state);
  }

  /**
   * Clear error on a field
   */
  clearError(name: string): void {
    const state = this.fields.get(name);
    if (!state) return;

    state.isValid = true;
    state.errorMessage = '';
    this.updateFieldUI(state);
  }

  /**
   * Check if a specific field is valid
   */
  isFieldValid(name: string): boolean {
    const state = this.fields.get(name);
    return state ? state.isValid : true;
  }

  /**
   * Get the current value of a field
   */
  getFieldValue(name: string): string {
    const state = this.fields.get(name);
    return state ? state.element.value : '';
  }

  /**
   * Destroy the validator and clean up
   */
  destroy(): void {
    this.fields.clear();
    if (this.liveRegion && this.liveRegion.parentElement) {
      this.liveRegion.parentElement.removeChild(this.liveRegion);
    }
  }
}

/**
 * Helper function to create a validator for a form
 */
export function createFormValidator(
  formSelector: string,
  options?: {
    validateOnBlur?: boolean;
    validateOnInput?: boolean;
    debounceMs?: number;
  }
): FormValidator | null {
  const form = document.querySelector<HTMLFormElement>(formSelector);
  if (!form) {
    logger.warn(`Form not found: ${formSelector}`);
    return null;
  }
  return new FormValidator(form, options);
}

/**
 * Quick validation helpers
 */
export const validators = {
  required: (message?: string): ValidationRule => ({
    required: true,
    messages: message ? { required: message } : undefined,
  }),

  email: (message?: string): ValidationRule => ({
    email: true,
    messages: message ? { email: message } : undefined,
  }),

  minLength: (min: number, message?: string): ValidationRule => ({
    minLength: min,
    messages: message ? { minLength: message } : undefined,
  }),

  maxLength: (max: number, message?: string): ValidationRule => ({
    maxLength: max,
    messages: message ? { maxLength: message } : undefined,
  }),

  pattern: (regex: RegExp, message?: string): ValidationRule => ({
    pattern: regex,
    messages: message ? { pattern: message } : undefined,
  }),

  password: (): ValidationRule => ({
    required: true,
    password: true,
  }),

  passwordConfirm: (passwordFieldName: string): ValidationRule => ({
    required: true,
    match: passwordFieldName,
    messages: { match: 'Passwords do not match' },
  }),

  username: (): ValidationRule => ({
    required: true,
    minLength: 3,
    maxLength: 50,
    pattern: /^[a-zA-Z0-9_-]+$/,
    messages: {
      required: 'Username is required',
      minLength: 'Username must be at least 3 characters',
      maxLength: 'Username cannot exceed 50 characters',
      pattern: 'Username can only contain letters, numbers, underscores, and hyphens',
    },
  }),

  /**
   * Combine multiple validation rules
   */
  combine: (...rules: ValidationRule[]): ValidationRule => {
    return rules.reduce((acc, rule) => ({
      ...acc,
      ...rule,
      messages: { ...acc.messages, ...rule.messages },
    }), {});
  },
};
