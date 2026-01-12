// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus
/**
 * FriendsManager
 *
 * Manages the Plex friends and sharing UI, allowing users to:
 * - View and manage Plex friends
 * - Share libraries with friends
 * - Manage Plex Home managed users
 * - Update sharing permissions and content restrictions
 */

import type { API } from '../lib/api';
import type { ToastManager } from '../lib/toast';
import type {
    PlexFriend,
    PlexSharedServer,
    PlexManagedUser,
    PlexLibrarySectionForSharing,
} from '../lib/types/plex';
import { createLogger } from '../lib/logger';

const logger = createLogger('FriendsManager');

// Re-export types for backwards compatibility
export type { PlexFriend, PlexSharedServer, PlexManagedUser };
export type PlexLibrarySection = PlexLibrarySectionForSharing;

type TabType = 'friends' | 'sharing' | 'managed';

// ============================================================================
// FriendsManager Class
// ============================================================================

export class FriendsManager {
    private api: API;
    private toast: ToastManager;
    private container: HTMLElement | null = null;
    private currentTab: TabType = 'friends';
    private friends: PlexFriend[] = [];
    private sharedServers: PlexSharedServer[] = [];
    private managedUsers: PlexManagedUser[] = [];
    private libraries: PlexLibrarySectionForSharing[] = [];
    private isLoading = false;

    constructor(api: API, toast: ToastManager) {
        this.api = api;
        this.toast = toast;
        logger.info('FriendsManager initialized');
    }

    /**
     * Initialize the FriendsManager and render the UI
     */
    async initialize(containerId: string): Promise<void> {
        this.container = document.getElementById(containerId);
        if (!this.container) {
            logger.error('Container not found:', containerId);
            return;
        }

        this.render();
        await this.loadData();
    }

    /**
     * Load all data from the API
     */
    private async loadData(): Promise<void> {
        this.isLoading = true;
        this.updateLoadingState();

        try {
            const [friendsRes, sharingRes, usersRes, librariesRes] = await Promise.all([
                this.api.getPlexFriends(),
                this.api.getPlexSharedServers(),
                this.api.getPlexManagedUsers(),
                this.api.getPlexLibrariesForSharing(),
            ]);

            this.friends = friendsRes.friends || [];
            this.sharedServers = sharingRes.sharedServers || [];
            this.managedUsers = usersRes.users || [];
            this.libraries = librariesRes.sections || [];

            logger.info('Data loaded:', {
                friends: this.friends.length,
                sharedServers: this.sharedServers.length,
                managedUsers: this.managedUsers.length,
                libraries: this.libraries.length,
            });
        } catch (error) {
            logger.error('Failed to load data:', error);
            this.toast.error('Failed to load Plex friends data');
        } finally {
            this.isLoading = false;
            this.render();
        }
    }

    /**
     * Render the main UI
     */
    private render(): void {
        if (!this.container) return;

        this.container.innerHTML = `
            <div class="friends-manager">
                <div class="friends-header">
                    <h2>Plex Friends & Sharing</h2>
                    <p class="friends-description">
                        Manage your Plex friends, library sharing, and home users.
                    </p>
                </div>

                <div class="friends-tabs">
                    <button class="tab-button ${this.currentTab === 'friends' ? 'active' : ''}" data-tab="friends">
                        <span class="tab-icon">&#x1F465;</span>
                        Friends (${this.friends.length})
                    </button>
                    <button class="tab-button ${this.currentTab === 'sharing' ? 'active' : ''}" data-tab="sharing">
                        <span class="tab-icon">&#x1F4E4;</span>
                        Sharing (${this.sharedServers.length})
                    </button>
                    <button class="tab-button ${this.currentTab === 'managed' ? 'active' : ''}" data-tab="managed">
                        <span class="tab-icon">&#x1F46A;</span>
                        Managed Users (${this.managedUsers.length})
                    </button>
                </div>

                <div class="friends-content">
                    ${this.renderTabContent()}
                </div>
            </div>
        `;

        this.attachEventListeners();
    }

    /**
     * Render content for the current tab
     */
    private renderTabContent(): string {
        if (this.isLoading) {
            return `
                <div class="loading-state">
                    <div class="spinner"></div>
                    <p>Loading...</p>
                </div>
            `;
        }

        switch (this.currentTab) {
            case 'friends':
                return this.renderFriendsTab();
            case 'sharing':
                return this.renderSharingTab();
            case 'managed':
                return this.renderManagedTab();
            default:
                return '';
        }
    }

    /**
     * Render the friends tab
     */
    private renderFriendsTab(): string {
        const inviteForm = `
            <div class="invite-form">
                <h3>Invite Friend</h3>
                <div class="form-row">
                    <input type="email" id="invite-email" placeholder="friend@example.com" />
                    <button class="btn btn-primary" id="invite-btn">Send Invite</button>
                </div>
                <div class="form-options">
                    <label><input type="checkbox" id="invite-sync" /> Allow Sync</label>
                    <label><input type="checkbox" id="invite-camera" /> Allow Camera Upload</label>
                    <label><input type="checkbox" id="invite-channels" /> Allow Channels</label>
                </div>
            </div>
        `;

        if (this.friends.length === 0) {
            return inviteForm + `
                <div class="empty-state">
                    <p>No friends yet. Invite someone to share your Plex library!</p>
                </div>
            `;
        }

        const friendsList = this.friends.map(friend => `
            <div class="friend-card" data-friend-id="${friend.id}">
                <div class="friend-avatar">
                    ${friend.thumb ? `<img src="${friend.thumb}" alt="${friend.username}" />` : `<div class="avatar-placeholder">${friend.username.charAt(0).toUpperCase()}</div>`}
                </div>
                <div class="friend-info">
                    <h4>${friend.title || friend.username}</h4>
                    <p class="friend-email">${friend.email}</p>
                    <div class="friend-badges">
                        ${friend.status === 'pending' ? '<span class="badge pending">Pending</span>' : ''}
                        ${friend.status === 'pending_received' ? '<span class="badge pending-received">Invite Received</span>' : ''}
                        ${friend.home ? '<span class="badge home">Home User</span>' : ''}
                        ${friend.server ? '<span class="badge server">Has Server</span>' : ''}
                    </div>
                    <div class="friend-permissions">
                        ${friend.allowSync ? '<span class="perm">Sync</span>' : ''}
                        ${friend.allowCameraUpload ? '<span class="perm">Camera</span>' : ''}
                        ${friend.allowChannels ? '<span class="perm">Channels</span>' : ''}
                    </div>
                </div>
                <div class="friend-actions">
                    <button class="btn btn-danger btn-sm remove-friend" data-friend-id="${friend.id}">Remove</button>
                </div>
            </div>
        `).join('');

        return inviteForm + `
            <div class="friends-list">
                <h3>Your Friends</h3>
                ${friendsList}
            </div>
        `;
    }

    /**
     * Render the sharing tab
     */
    private renderSharingTab(): string {
        const shareForm = `
            <div class="share-form">
                <h3>Share Libraries</h3>
                <div class="form-row">
                    <input type="email" id="share-email" placeholder="friend@example.com" />
                </div>
                <div class="library-selection">
                    <h4>Select Libraries to Share:</h4>
                    ${this.libraries.map(lib => `
                        <label class="library-checkbox">
                            <input type="checkbox" name="share-library" value="${lib.id}" />
                            <span class="library-icon library-${lib.type}"></span>
                            ${lib.title}
                        </label>
                    `).join('')}
                </div>
                <div class="form-options">
                    <label><input type="checkbox" id="share-sync" /> Allow Sync</label>
                    <label><input type="checkbox" id="share-camera" /> Allow Camera Upload</label>
                    <label><input type="checkbox" id="share-channels" /> Allow Channels</label>
                </div>
                <button class="btn btn-primary" id="share-btn">Share Libraries</button>
            </div>
        `;

        if (this.sharedServers.length === 0) {
            return shareForm + `
                <div class="empty-state">
                    <p>No shared servers yet. Share your libraries with friends!</p>
                </div>
            `;
        }

        const sharedList = this.sharedServers.map(shared => `
            <div class="shared-card" data-shared-id="${shared.id}">
                <div class="shared-avatar">
                    ${shared.thumb ? `<img src="${shared.thumb}" alt="${shared.username}" />` : `<div class="avatar-placeholder">${(shared.username || shared.invitedEmail || '?').charAt(0).toUpperCase()}</div>`}
                </div>
                <div class="shared-info">
                    <h4>${shared.username || shared.invitedEmail}</h4>
                    <p class="shared-email">${shared.email || shared.invitedEmail}</p>
                    ${shared.acceptedAt ? `<p class="shared-date">Accepted: ${new Date(shared.acceptedAt).toLocaleDateString()}</p>` : '<p class="shared-date pending">Invitation pending</p>'}
                    <div class="shared-permissions">
                        ${shared.allowSync ? '<span class="perm">Sync</span>' : ''}
                        ${shared.allowCameraUpload ? '<span class="perm">Camera</span>' : ''}
                        ${shared.allowChannels ? '<span class="perm">Channels</span>' : ''}
                    </div>
                </div>
                <div class="shared-actions">
                    <button class="btn btn-secondary btn-sm edit-sharing" data-shared-id="${shared.id}">Edit</button>
                    <button class="btn btn-danger btn-sm revoke-sharing" data-shared-id="${shared.id}">Revoke</button>
                </div>
            </div>
        `).join('');

        return shareForm + `
            <div class="shared-list">
                <h3>Shared With</h3>
                ${sharedList}
            </div>
        `;
    }

    /**
     * Render the managed users tab
     */
    private renderManagedTab(): string {
        const createForm = `
            <div class="create-user-form">
                <h3>Create Managed User</h3>
                <div class="form-row">
                    <input type="text" id="user-name" placeholder="Name" />
                    <select id="user-restriction">
                        <option value="">No Restrictions</option>
                        <option value="little_kid">Little Kid</option>
                        <option value="older_kid">Older Kid</option>
                        <option value="teen">Teen</option>
                    </select>
                    <button class="btn btn-primary" id="create-user-btn">Create User</button>
                </div>
            </div>
        `;

        if (this.managedUsers.length === 0) {
            return createForm + `
                <div class="empty-state">
                    <p>No managed users yet. Create a managed user for your Plex Home!</p>
                </div>
            `;
        }

        const usersList = this.managedUsers.map(user => `
            <div class="user-card" data-user-id="${user.id}">
                <div class="user-avatar">
                    ${user.thumb ? `<img src="${user.thumb}" alt="${user.username}" />` : `<div class="avatar-placeholder">${user.title.charAt(0).toUpperCase()}</div>`}
                </div>
                <div class="user-info">
                    <h4>${user.title || user.username}</h4>
                    <div class="user-badges">
                        ${user.restricted ? `<span class="badge restriction">${this.formatRestriction(user.restrictionProfile)}</span>` : '<span class="badge unrestricted">Unrestricted</span>'}
                        ${user.homeAdmin ? '<span class="badge admin">Admin</span>' : ''}
                        ${user.protected ? '<span class="badge protected">PIN Protected</span>' : ''}
                        ${user.guest ? '<span class="badge guest">Guest</span>' : ''}
                    </div>
                </div>
                <div class="user-actions">
                    <select class="restriction-select" data-user-id="${user.id}">
                        <option value="" ${!user.restrictionProfile ? 'selected' : ''}>No Restrictions</option>
                        <option value="little_kid" ${user.restrictionProfile === 'little_kid' ? 'selected' : ''}>Little Kid</option>
                        <option value="older_kid" ${user.restrictionProfile === 'older_kid' ? 'selected' : ''}>Older Kid</option>
                        <option value="teen" ${user.restrictionProfile === 'teen' ? 'selected' : ''}>Teen</option>
                    </select>
                    ${!user.homeAdmin ? `<button class="btn btn-danger btn-sm delete-user" data-user-id="${user.id}">Delete</button>` : ''}
                </div>
            </div>
        `).join('');

        return createForm + `
            <div class="users-list">
                <h3>Managed Users</h3>
                ${usersList}
            </div>
        `;
    }

    /**
     * Format restriction profile for display
     */
    private formatRestriction(profile: string): string {
        switch (profile) {
            case 'little_kid': return 'Little Kid';
            case 'older_kid': return 'Older Kid';
            case 'teen': return 'Teen';
            default: return 'No Restrictions';
        }
    }

    /**
     * Update loading state in UI
     */
    private updateLoadingState(): void {
        const content = this.container?.querySelector('.friends-content');
        if (content && this.isLoading) {
            content.innerHTML = `
                <div class="loading-state">
                    <div class="spinner"></div>
                    <p>Loading...</p>
                </div>
            `;
        }
    }

    /**
     * Attach event listeners
     */
    private attachEventListeners(): void {
        if (!this.container) return;

        // Tab switching
        this.container.querySelectorAll('.tab-button').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const tab = (e.currentTarget as HTMLElement).dataset.tab as TabType;
                if (tab) {
                    this.currentTab = tab;
                    this.render();
                }
            });
        });

        // Invite friend
        const inviteBtn = this.container.querySelector('#invite-btn');
        inviteBtn?.addEventListener('click', () => this.handleInviteFriend());

        // Remove friend
        this.container.querySelectorAll('.remove-friend').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const friendId = parseInt((e.currentTarget as HTMLElement).dataset.friendId || '0');
                if (friendId) this.handleRemoveFriend(friendId);
            });
        });

        // Share libraries
        const shareBtn = this.container.querySelector('#share-btn');
        shareBtn?.addEventListener('click', () => this.handleShareLibraries());

        // Revoke sharing
        this.container.querySelectorAll('.revoke-sharing').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const sharedId = parseInt((e.currentTarget as HTMLElement).dataset.sharedId || '0');
                if (sharedId) this.handleRevokeSharing(sharedId);
            });
        });

        // Create managed user
        const createUserBtn = this.container.querySelector('#create-user-btn');
        createUserBtn?.addEventListener('click', () => this.handleCreateManagedUser());

        // Delete managed user
        this.container.querySelectorAll('.delete-user').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const userId = parseInt((e.currentTarget as HTMLElement).dataset.userId || '0');
                if (userId) this.handleDeleteManagedUser(userId);
            });
        });

        // Update restriction profile
        this.container.querySelectorAll('.restriction-select').forEach(select => {
            select.addEventListener('change', (e) => {
                const userId = parseInt((e.currentTarget as HTMLElement).dataset.userId || '0');
                const profile = (e.currentTarget as HTMLSelectElement).value;
                if (userId) this.handleUpdateRestriction(userId, profile);
            });
        });
    }

    // ========================================================================
    // API Action Handlers
    // ========================================================================

    private async handleInviteFriend(): Promise<void> {
        const emailInput = this.container?.querySelector('#invite-email') as HTMLInputElement;
        const syncCheck = this.container?.querySelector('#invite-sync') as HTMLInputElement;
        const cameraCheck = this.container?.querySelector('#invite-camera') as HTMLInputElement;
        const channelsCheck = this.container?.querySelector('#invite-channels') as HTMLInputElement;

        const email = emailInput?.value?.trim();
        if (!email) {
            this.toast.error('Please enter an email address');
            return;
        }

        try {
            await this.api.invitePlexFriend({
                email,
                allowSync: syncCheck?.checked || false,
                allowCameraUpload: cameraCheck?.checked || false,
                allowChannels: channelsCheck?.checked || false,
            });
            this.toast.success('Friend invitation sent!');
            emailInput.value = '';
            await this.loadData();
        } catch (error) {
            logger.error('Failed to invite friend:', error);
            this.toast.error('Failed to send friend invitation');
        }
    }

    private async handleRemoveFriend(friendId: number): Promise<void> {
        if (!confirm('Are you sure you want to remove this friend?')) return;

        try {
            await this.api.removePlexFriend(friendId);
            this.toast.success('Friend removed');
            await this.loadData();
        } catch (error) {
            logger.error('Failed to remove friend:', error);
            this.toast.error('Failed to remove friend');
        }
    }

    private async handleShareLibraries(): Promise<void> {
        const emailInput = this.container?.querySelector('#share-email') as HTMLInputElement;
        const syncCheck = this.container?.querySelector('#share-sync') as HTMLInputElement;
        const cameraCheck = this.container?.querySelector('#share-camera') as HTMLInputElement;
        const channelsCheck = this.container?.querySelector('#share-channels') as HTMLInputElement;

        const email = emailInput?.value?.trim();
        if (!email) {
            this.toast.error('Please enter an email address');
            return;
        }

        const selectedLibraries: number[] = [];
        this.container?.querySelectorAll('input[name="share-library"]:checked').forEach((checkbox) => {
            selectedLibraries.push(parseInt((checkbox as HTMLInputElement).value));
        });

        if (selectedLibraries.length === 0) {
            this.toast.error('Please select at least one library to share');
            return;
        }

        try {
            await this.api.sharePlexLibraries({
                email,
                librarySectionIds: selectedLibraries,
                allowSync: syncCheck?.checked || false,
                allowCameraUpload: cameraCheck?.checked || false,
                allowChannels: channelsCheck?.checked || false,
            });
            this.toast.success('Libraries shared successfully!');
            emailInput.value = '';
            await this.loadData();
        } catch (error) {
            logger.error('Failed to share libraries:', error);
            this.toast.error('Failed to share libraries');
        }
    }

    private async handleRevokeSharing(sharedId: number): Promise<void> {
        if (!confirm('Are you sure you want to revoke sharing for this user?')) return;

        try {
            await this.api.revokePlexSharing(sharedId);
            this.toast.success('Sharing revoked');
            await this.loadData();
        } catch (error) {
            logger.error('Failed to revoke sharing:', error);
            this.toast.error('Failed to revoke sharing');
        }
    }

    private async handleCreateManagedUser(): Promise<void> {
        const nameInput = this.container?.querySelector('#user-name') as HTMLInputElement;
        const restrictionSelect = this.container?.querySelector('#user-restriction') as HTMLSelectElement;

        const name = nameInput?.value?.trim();
        if (!name) {
            this.toast.error('Please enter a name');
            return;
        }

        try {
            await this.api.createPlexManagedUser({
                name,
                restrictionProfile: (restrictionSelect?.value as 'little_kid' | 'older_kid' | 'teen') || undefined,
            });
            this.toast.success('Managed user created!');
            nameInput.value = '';
            await this.loadData();
        } catch (error) {
            logger.error('Failed to create managed user:', error);
            this.toast.error('Failed to create managed user');
        }
    }

    private async handleDeleteManagedUser(userId: number): Promise<void> {
        if (!confirm('Are you sure you want to delete this managed user?')) return;

        try {
            await this.api.deletePlexManagedUser(userId);
            this.toast.success('Managed user deleted');
            await this.loadData();
        } catch (error) {
            logger.error('Failed to delete managed user:', error);
            this.toast.error('Failed to delete managed user');
        }
    }

    private async handleUpdateRestriction(userId: number, profile: string): Promise<void> {
        try {
            await this.api.updatePlexManagedUser(userId, {
                restrictionProfile: profile as 'little_kid' | 'older_kid' | 'teen' | '',
            });
            this.toast.success('Restriction profile updated');
        } catch (error) {
            logger.error('Failed to update restriction:', error);
            this.toast.error('Failed to update restriction profile');
            await this.loadData(); // Reload to reset the select
        }
    }

    /**
     * Cleanup and destroy the manager
     */
    destroy(): void {
        if (this.container) {
            this.container.innerHTML = '';
        }
        logger.info('FriendsManager destroyed');
    }
}

export default FriendsManager;
