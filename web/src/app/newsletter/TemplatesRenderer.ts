// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * TemplatesRenderer - Newsletter Template Management
 *
 * CRUD operations for newsletter templates with live preview.
 */

import { createLogger } from '../../lib/logger';
import type { API } from '../../lib/api';
import type {
    NewsletterTemplate,
    NewsletterType,
    CreateTemplateRequest,
    UpdateTemplateRequest,
} from '../../lib/types/newsletter';
import { NEWSLETTER_TYPES } from '../../lib/types/newsletter';
import { BaseNewsletterRenderer, NewsletterConfig } from './BaseNewsletterRenderer';

const logger = createLogger('NewsletterTemplatesRenderer');

export class TemplatesRenderer extends BaseNewsletterRenderer {
    private templates: NewsletterTemplate[] = [];
    private filterType: NewsletterType | '' = '';
    private filterActive: boolean | undefined = undefined;

    constructor(api: API, config: NewsletterConfig) {
        super(api, config);
    }

    // =========================================================================
    // Public API
    // =========================================================================

    setupEventListeners(): void {
        // Refresh button
        document.getElementById('newsletter-templates-refresh-btn')?.addEventListener('click', () => {
            this.load();
        });

        // Create button
        document.getElementById('newsletter-templates-create-btn')?.addEventListener('click', () => {
            this.showCreateModal();
        });

        // Type filter
        document.getElementById('newsletter-templates-type-filter')?.addEventListener('change', (e) => {
            this.filterType = (e.target as HTMLSelectElement).value as NewsletterType | '';
            this.load();
        });

        // Active filter
        document.getElementById('newsletter-templates-active-filter')?.addEventListener('change', (e) => {
            const value = (e.target as HTMLSelectElement).value;
            this.filterActive = value === '' ? undefined : value === 'true';
            this.load();
        });

        // Search
        document.getElementById('newsletter-templates-search')?.addEventListener('input', this.debounce((e: Event) => {
            this.filterBySearch((e.target as HTMLInputElement).value);
        }, 300));
    }

    async load(): Promise<void> {
        try {
            const result = await this.api.getNewsletterTemplates({
                type: this.filterType || undefined,
                active: this.filterActive,
                limit: this.config.maxTableEntries,
            });

            this.templates = result.templates || [];
            this.renderTable();
        } catch (error) {
            logger.error('Failed to load newsletter templates:', error);
        }
    }

    // =========================================================================
    // Table Rendering
    // =========================================================================

    private renderTable(): void {
        const tbody = document.getElementById('newsletter-templates-tbody');
        if (!tbody) return;

        if (this.templates.length === 0) {
            tbody.innerHTML = '<tr><td colspan="7" class="table-empty">No templates found. Create your first template to get started.</td></tr>';
            return;
        }

        tbody.innerHTML = this.templates.map(template => `
            <tr class="newsletter-template-row" data-id="${template.id}">
                <td class="template-name">
                    <div class="template-name-cell">
                        <span class="template-title">${this.escapeHtml(template.name)}</span>
                        ${template.is_built_in ? '<span class="built-in-badge">Built-in</span>' : ''}
                    </div>
                    ${template.description ? `<span class="template-description">${this.escapeHtml(template.description)}</span>` : ''}
                </td>
                <td class="template-type">${this.renderTypeBadge(template.type)}</td>
                <td class="template-subject">${this.escapeHtml(template.subject)}</td>
                <td class="template-version">v${template.version}</td>
                <td class="template-status">
                    <span class="status-indicator ${template.is_active ? 'active' : 'inactive'}">
                        ${template.is_active ? 'Active' : 'Inactive'}
                    </span>
                </td>
                <td class="template-updated">${this.formatTimeAgo(new Date(template.updated_at))}</td>
                <td class="template-actions">
                    <button class="btn-action btn-preview" data-action="preview" data-id="${template.id}" title="Preview">
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>
                            <circle cx="12" cy="12" r="3"/>
                        </svg>
                    </button>
                    <button class="btn-action btn-edit" data-action="edit" data-id="${template.id}" title="Edit" ${template.is_built_in ? 'disabled' : ''}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                            <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                        </svg>
                    </button>
                    <button class="btn-action btn-delete" data-action="delete" data-id="${template.id}" title="Delete" ${template.is_built_in ? 'disabled' : ''}>
                        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <polyline points="3 6 5 6 21 6"/>
                            <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                        </svg>
                    </button>
                </td>
            </tr>
        `).join('');

        // Add action handlers
        this.attachTableActions();
    }

    private attachTableActions(): void {
        const tbody = document.getElementById('newsletter-templates-tbody');
        if (!tbody) return;

        tbody.querySelectorAll('[data-action]').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const action = (e.currentTarget as HTMLElement).getAttribute('data-action');
                const id = (e.currentTarget as HTMLElement).getAttribute('data-id');
                if (!id) return;

                switch (action) {
                    case 'preview':
                        await this.showPreviewModal(id);
                        break;
                    case 'edit':
                        await this.showEditModal(id);
                        break;
                    case 'delete':
                        await this.confirmDelete(id);
                        break;
                }
            });
        });
    }

    private filterBySearch(search: string): void {
        const rows = document.querySelectorAll('.newsletter-template-row');
        const searchLower = search.toLowerCase();

        rows.forEach(row => {
            const name = row.querySelector('.template-name')?.textContent?.toLowerCase() || '';
            const subject = row.querySelector('.template-subject')?.textContent?.toLowerCase() || '';
            const visible = name.includes(searchLower) || subject.includes(searchLower);
            (row as HTMLElement).style.display = visible ? '' : 'none';
        });
    }

    // =========================================================================
    // Create Modal
    // =========================================================================

    private showCreateModal(): void {
        const content = this.buildTemplateForm(null);
        const modal = this.createModal('newsletter-template-form-modal', 'Create Newsletter Template', content);

        this.showModal(modal);
        this.attachFormHandlers(modal, null);
    }

    // =========================================================================
    // Edit Modal
    // =========================================================================

    private async showEditModal(id: string): Promise<void> {
        try {
            const template = await this.api.getNewsletterTemplate(id);
            const content = this.buildTemplateForm(template);
            const modal = this.createModal('newsletter-template-form-modal', 'Edit Newsletter Template', content);

            this.showModal(modal);
            this.attachFormHandlers(modal, template);
        } catch (error) {
            logger.error('Failed to load template:', error);
        }
    }

    // =========================================================================
    // Preview Modal
    // =========================================================================

    private async showPreviewModal(id: string): Promise<void> {
        try {
            const template = await this.api.getNewsletterTemplate(id);
            const preview = await this.api.previewNewsletterTemplate({ template_id: id });

            const content = `
                <div class="newsletter-preview-container">
                    <div class="preview-header">
                        <strong>Subject:</strong> ${this.escapeHtml(preview.subject)}
                    </div>
                    <div class="preview-tabs">
                        <button class="preview-tab active" data-tab="html">HTML</button>
                        <button class="preview-tab" data-tab="text">Plain Text</button>
                    </div>
                    <div class="preview-content">
                        <div class="preview-panel preview-html active">
                            <iframe id="preview-html-frame" sandbox="allow-same-origin" class="preview-iframe"></iframe>
                        </div>
                        <div class="preview-panel preview-text" style="display: none;">
                            <pre class="preview-text-content">${this.escapeHtml(preview.body_text || 'No plain text version available')}</pre>
                        </div>
                    </div>
                </div>
            `;

            const modal = this.createModal('newsletter-preview-modal', `Preview: ${template.name}`, content);
            this.showModal(modal);

            // Set iframe content
            const iframe = document.getElementById('preview-html-frame') as HTMLIFrameElement;
            if (iframe) {
                const doc = iframe.contentDocument;
                if (doc) {
                    doc.open();
                    doc.write(preview.body_html);
                    doc.close();
                }
            }

            // Tab switching
            modal.querySelectorAll('.preview-tab').forEach(tab => {
                tab.addEventListener('click', (e) => {
                    const tabName = (e.currentTarget as HTMLElement).getAttribute('data-tab');
                    modal.querySelectorAll('.preview-tab').forEach(t => t.classList.remove('active'));
                    modal.querySelectorAll('.preview-panel').forEach(p => (p as HTMLElement).style.display = 'none');
                    (e.currentTarget as HTMLElement).classList.add('active');
                    const panel = modal.querySelector(`.preview-${tabName}`);
                    if (panel) (panel as HTMLElement).style.display = 'block';
                });
            });
        } catch (error) {
            logger.error('Failed to preview template:', error);
        }
    }

    // =========================================================================
    // Delete Confirmation
    // =========================================================================

    private async confirmDelete(id: string): Promise<void> {
        const template = this.templates.find(t => t.id === id);
        if (!template) return;

        const content = `
            <p>Are you sure you want to delete the template "<strong>${this.escapeHtml(template.name)}</strong>"?</p>
            <p class="warning-text">This action cannot be undone. Any schedules using this template will be affected.</p>
            <div class="modal-actions">
                <button type="button" class="btn btn-secondary" data-action="cancel">Cancel</button>
                <button type="button" class="btn btn-danger" data-action="confirm">Delete</button>
            </div>
        `;

        const modal = this.createModal('newsletter-delete-confirm-modal', 'Delete Template', content);
        this.showModal(modal);

        modal.querySelector('[data-action="cancel"]')?.addEventListener('click', () => {
            this.closeModal('newsletter-delete-confirm-modal');
        });

        modal.querySelector('[data-action="confirm"]')?.addEventListener('click', async () => {
            try {
                await this.api.deleteNewsletterTemplate(id);
                this.closeModal('newsletter-delete-confirm-modal');
                await this.load();
            } catch (error) {
                logger.error('Failed to delete template:', error);
            }
        });
    }

    // =========================================================================
    // Form Building
    // =========================================================================

    private buildTemplateForm(template: NewsletterTemplate | null): string {
        const typeOptions = Object.entries(NEWSLETTER_TYPES)
            .map(([value, config]) => `
                <option value="${value}" ${template?.type === value ? 'selected' : ''}>
                    ${config.icon} ${config.label}
                </option>
            `)
            .join('');

        return `
            <form id="newsletter-template-form" class="newsletter-form">
                <div class="form-row">
                    <div class="form-group">
                        <label for="template-name">Name *</label>
                        <input type="text" id="template-name" name="name" required
                               value="${template ? this.escapeHtml(template.name) : ''}"
                               placeholder="e.g., Weekly Media Digest">
                    </div>
                    <div class="form-group">
                        <label for="template-type">Type *</label>
                        <select id="template-type" name="type" required>
                            <option value="">Select type...</option>
                            ${typeOptions}
                        </select>
                    </div>
                </div>

                <div class="form-group">
                    <label for="template-description">Description</label>
                    <input type="text" id="template-description" name="description"
                           value="${template?.description ? this.escapeHtml(template.description) : ''}"
                           placeholder="Brief description of this template">
                </div>

                <div class="form-group">
                    <label for="template-subject">Subject Line *</label>
                    <input type="text" id="template-subject" name="subject" required
                           value="${template ? this.escapeHtml(template.subject) : ''}"
                           placeholder="e.g., Your Weekly Media Update - {{date}}">
                    <small class="form-hint">Use {{variable}} syntax for dynamic values</small>
                </div>

                <div class="form-group">
                    <label for="template-body-html">HTML Body *</label>
                    <textarea id="template-body-html" name="body_html" required rows="15"
                              placeholder="Enter HTML template content...">${template?.body_html ? this.escapeHtml(template.body_html) : ''}</textarea>
                    <small class="form-hint">
                        Available variables: {{date}}, {{server_name}}, {{items}}, {{stats}}, {{username}}, etc.
                    </small>
                </div>

                <div class="form-group">
                    <label for="template-body-text">Plain Text Body (Optional)</label>
                    <textarea id="template-body-text" name="body_text" rows="8"
                              placeholder="Enter plain text version...">${template?.body_text ? this.escapeHtml(template.body_text) : ''}</textarea>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label>
                            <input type="checkbox" id="template-active" name="is_active"
                                   ${template?.is_active !== false ? 'checked' : ''}>
                            Active
                        </label>
                    </div>
                </div>

                <div class="form-actions">
                    <button type="button" class="btn btn-secondary" data-action="cancel">Cancel</button>
                    <button type="button" class="btn btn-secondary" data-action="preview">Preview</button>
                    <button type="submit" class="btn btn-primary">${template ? 'Update' : 'Create'} Template</button>
                </div>
            </form>
        `;
    }

    private attachFormHandlers(modal: HTMLElement, template: NewsletterTemplate | null): void {
        const form = modal.querySelector('#newsletter-template-form') as HTMLFormElement;
        if (!form) return;

        // Cancel
        form.querySelector('[data-action="cancel"]')?.addEventListener('click', () => {
            this.closeModal('newsletter-template-form-modal');
        });

        // Preview
        form.querySelector('[data-action="preview"]')?.addEventListener('click', async () => {
            if (template?.id) {
                await this.showPreviewModal(template.id);
            }
        });

        // Submit
        form.addEventListener('submit', async (e) => {
            e.preventDefault();

            const formData = new FormData(form);
            const data: CreateTemplateRequest | UpdateTemplateRequest = {
                name: formData.get('name') as string,
                description: formData.get('description') as string || undefined,
                type: formData.get('type') as NewsletterType,
                subject: formData.get('subject') as string,
                body_html: formData.get('body_html') as string,
                body_text: formData.get('body_text') as string || undefined,
            };

            try {
                if (template) {
                    await this.api.updateNewsletterTemplate(template.id, {
                        ...data,
                        is_active: form.querySelector('#template-active')?.hasAttribute('checked'),
                    });
                } else {
                    await this.api.createNewsletterTemplate(data as CreateTemplateRequest);
                }

                this.closeModal('newsletter-template-form-modal');
                await this.load();
            } catch (error) {
                logger.error('Failed to save template:', error);
            }
        });
    }
}
