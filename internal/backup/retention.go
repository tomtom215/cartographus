// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package backup

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/tomtom215/cartographus/internal/logging"
)

// getCompletedBackupsSorted returns completed backups sorted by creation time (newest first)
func (m *Manager) getCompletedBackupsSorted() []*Backup {
	var completedBackups []*Backup
	for _, b := range m.metadata.Backups {
		if b.Status == StatusCompleted {
			completedBackups = append(completedBackups, b)
		}
	}

	sort.Slice(completedBackups, func(i, j int) bool {
		return completedBackups[i].CreatedAt.After(completedBackups[j].CreatedAt)
	})

	return completedBackups
}

// addMinCountToKeepSet adds minimum count backups to the keep set
func addMinCountToKeepSet(keepSet map[string]bool, backups []*Backup, minCount int) {
	for i := 0; i < minCount && i < len(backups); i++ {
		keepSet[backups[i].ID] = true
	}
}

// addRecentBackupsToKeepSet adds recent backups to the keep set
func addRecentBackupsToKeepSet(keepSet map[string]bool, backups []*Backup, hours int, now time.Time) {
	cutoff := now.Add(-time.Duration(hours) * time.Hour)
	for _, b := range backups {
		if b.CreatedAt.After(cutoff) {
			keepSet[b.ID] = true
		}
	}
}

// addBackupsToKeepSet adds a list of backups to the keep set
func addBackupsToKeepSet(keepSet map[string]bool, backups []*Backup) {
	for _, b := range backups {
		keepSet[b.ID] = true
	}
}

// applyRetentionRules applies all retention rules and returns the set of backup IDs to keep
func (m *Manager) applyRetentionRules(backups []*Backup, policy RetentionPolicy, now time.Time) map[string]bool {
	keepSet := make(map[string]bool)

	// Rule 1: Always keep MinCount backups
	addMinCountToKeepSet(keepSet, backups, policy.MinCount)

	// Rule 2: Keep all backups from the last KeepRecentHours
	if policy.KeepRecentHours > 0 {
		addRecentBackupsToKeepSet(keepSet, backups, policy.KeepRecentHours, now)
	}

	// Rule 3: Keep at least one backup per day for KeepDailyForDays
	if policy.KeepDailyForDays > 0 {
		dailyBackups := m.selectDailyBackups(backups, policy.KeepDailyForDays, now)
		addBackupsToKeepSet(keepSet, dailyBackups)
	}

	// Rule 4: Keep at least one backup per week for KeepWeeklyForWeeks
	if policy.KeepWeeklyForWeeks > 0 {
		weeklyBackups := m.selectWeeklyBackups(backups, policy.KeepWeeklyForWeeks, now)
		addBackupsToKeepSet(keepSet, weeklyBackups)
	}

	// Rule 5: Keep at least one backup per month for KeepMonthlyForMonths
	if policy.KeepMonthlyForMonths > 0 {
		monthlyBackups := m.selectMonthlyBackups(backups, policy.KeepMonthlyForMonths, now)
		addBackupsToKeepSet(keepSet, monthlyBackups)
	}

	return keepSet
}

// shouldDeleteByAge returns true if backup should be deleted due to age
func shouldDeleteByAge(b *Backup, policy RetentionPolicy, now time.Time) bool {
	if policy.MaxAgeDays <= 0 {
		return false
	}
	cutoff := now.AddDate(0, 0, -policy.MaxAgeDays)
	return b.CreatedAt.Before(cutoff)
}

// shouldDeleteByCount returns true if backup should be deleted due to count limits
func shouldDeleteByCount(keepSetSize int, policy RetentionPolicy) bool {
	return policy.MaxCount > 0 && keepSetSize >= policy.MaxCount
}

// identifyBackupsToDelete identifies backups that should be deleted based on retention policy
func (m *Manager) identifyBackupsToDelete(backups []*Backup, keepSet map[string]bool, policy RetentionPolicy, now time.Time) []*Backup {
	var toDelete []*Backup

	for _, b := range backups {
		// Skip if backup is explicitly kept
		if keepSet[b.ID] {
			continue
		}

		// Delete if too old
		if shouldDeleteByAge(b, policy, now) {
			toDelete = append(toDelete, b)
			continue
		}

		// Delete if exceeding max count
		if shouldDeleteByCount(len(keepSet), policy) {
			toDelete = append(toDelete, b)
		}
	}

	return toDelete
}

// getKeptBackups returns all backups that are in the keep set
func getKeptBackups(backups []*Backup, keepSet map[string]bool) []*Backup {
	var kept []*Backup
	for _, b := range backups {
		if keepSet[b.ID] {
			kept = append(kept, b)
		}
	}
	return kept
}

// sortBackupsByAge sorts backups by creation time (oldest first)
func sortBackupsByAge(backups []*Backup) {
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.Before(backups[j].CreatedAt)
	})
}

// selectExcessBackups selects backups to delete from excess, respecting MinCount
func selectExcessBackups(kept []*Backup, excess, minCount int, keepSet map[string]bool) []*Backup {
	var toDelete []*Backup

	for i := 0; i < excess; i++ {
		// Don't delete if it's in the MinCount protection
		if i >= len(kept)-minCount {
			break
		}
		toDelete = append(toDelete, kept[i])
		delete(keepSet, kept[i].ID)
	}

	return toDelete
}

// enforceMaxCount ensures the kept backups don't exceed MaxCount
func (m *Manager) enforceMaxCount(backups []*Backup, keepSet map[string]bool, policy RetentionPolicy) []*Backup {
	if policy.MaxCount <= 0 {
		return nil
	}

	kept := getKeptBackups(backups, keepSet)
	if len(kept) <= policy.MaxCount {
		return nil
	}

	// Sort by date (oldest first) and mark excess for deletion
	sortBackupsByAge(kept)
	excess := len(kept) - policy.MaxCount

	return selectExcessBackups(kept, excess, policy.MinCount, keepSet)
}

// collectBackupsToDelete gathers all backups to delete based on retention policy
func (m *Manager) collectBackupsToDelete(completedBackups []*Backup, policy RetentionPolicy, now time.Time) []*Backup {
	// Apply retention rules to identify backups to keep
	keepSet := m.applyRetentionRules(completedBackups, policy, now)

	// Identify backups to delete based on age and count
	toDelete := m.identifyBackupsToDelete(completedBackups, keepSet, policy, now)

	// Enforce MaxCount if still exceeded
	toDelete = append(toDelete, m.enforceMaxCount(completedBackups, keepSet, policy)...)

	return toDelete
}

// deleteBackups deletes the provided backups and logs the results
func (m *Manager) deleteBackups(toDelete []*Backup) (deletedCount int, deletedSize int64) {
	for _, b := range toDelete {
		if err := m.deleteBackupLocked(b); err != nil {
			logging.Warn().Err(err).Str("backup_id", b.ID).Msg("Failed to delete backup")
		} else {
			deletedCount++
			deletedSize += b.FileSize
		}
	}
	return deletedCount, deletedSize
}

// logRetentionResults logs the results of retention policy application
func logRetentionResults(deletedCount int, deletedSize int64) {
	if deletedCount > 0 {
		logging.Info().
			Int("deleted_count", deletedCount).
			Float64("deleted_mb", float64(deletedSize)/(1024*1024)).
			Msg("Retention policy applied")
	}
}

// ApplyRetentionPolicy applies the retention policy to clean up old backups
func (m *Manager) ApplyRetentionPolicy(_ context.Context) error {
	if m.metadata == nil || len(m.metadata.Backups) == 0 {
		return nil
	}

	m.metadataMu.Lock()
	defer m.metadataMu.Unlock()

	completedBackups := m.getCompletedBackupsSorted()
	if len(completedBackups) == 0 {
		return nil
	}

	policy := m.cfg.Retention
	now := time.Now()

	toDelete := m.collectBackupsToDelete(completedBackups, policy, now)
	deletedCount, deletedSize := m.deleteBackups(toDelete)
	logRetentionResults(deletedCount, deletedSize)

	return m.saveMetadataLocked()
}

// shouldPreferBackup returns true if the candidate backup should be preferred over the existing one
func shouldPreferBackup(candidate, existing *Backup) bool {
	// Prefer full backups over partial backups
	if candidate.Type == TypeFull && existing.Type != TypeFull {
		return true
	}
	// If same type, prefer the more recent one
	if candidate.Type == existing.Type && candidate.CreatedAt.After(existing.CreatedAt) {
		return true
	}
	return false
}

// periodKeyFunc generates a key for grouping backups by period
type periodKeyFunc func(*Backup) string

// selectBackupsByPeriod is a generic function to select backups by time period
func (m *Manager) selectBackupsByPeriod(backups []*Backup, cutoff time.Time, keyFunc periodKeyFunc) []*Backup {
	selected := make(map[string]*Backup)

	for _, b := range backups {
		if b.CreatedAt.Before(cutoff) {
			continue
		}

		key := keyFunc(b)
		existing, exists := selected[key]

		if !exists || shouldPreferBackup(b, existing) {
			selected[key] = b
		}
	}

	result := make([]*Backup, 0, len(selected))
	for _, b := range selected {
		result = append(result, b)
	}
	return result
}

// selectDailyBackups selects the best backup from each day
func (m *Manager) selectDailyBackups(backups []*Backup, days int, now time.Time) []*Backup {
	cutoff := now.AddDate(0, 0, -days)
	keyFunc := func(b *Backup) string {
		return b.CreatedAt.Format("2006-01-02")
	}
	return m.selectBackupsByPeriod(backups, cutoff, keyFunc)
}

// selectWeeklyBackups selects the best backup from each week
func (m *Manager) selectWeeklyBackups(backups []*Backup, weeks int, now time.Time) []*Backup {
	cutoff := now.AddDate(0, 0, -weeks*7)
	keyFunc := func(b *Backup) string {
		year, week := b.CreatedAt.ISOWeek()
		return fmt.Sprintf("%d-%02d", year, week)
	}
	return m.selectBackupsByPeriod(backups, cutoff, keyFunc)
}

// selectMonthlyBackups selects the best backup from each month
func (m *Manager) selectMonthlyBackups(backups []*Backup, months int, now time.Time) []*Backup {
	cutoff := now.AddDate(0, -months, 0)
	keyFunc := func(b *Backup) string {
		return b.CreatedAt.Format("2006-01")
	}
	return m.selectBackupsByPeriod(backups, cutoff, keyFunc)
}

// deleteBackupLocked deletes a backup (must be called with lock held)
func (m *Manager) deleteBackupLocked(backup *Backup) error {
	// Delete the backup file
	if fileExists(backup.FilePath) {
		if err := os.Remove(backup.FilePath); err != nil {
			return fmt.Errorf("failed to delete backup file: %w", err)
		}
	}

	// Remove from metadata
	for i, b := range m.metadata.Backups {
		if b.ID == backup.ID {
			m.metadata.Backups = append(m.metadata.Backups[:i], m.metadata.Backups[i+1:]...)
			break
		}
	}

	return nil
}

// isBackupInProgress returns true if the backup is still in progress
func isBackupInProgress(b *Backup) bool {
	return b.Status == StatusInProgress || b.Status == StatusPending
}

// validateBackupIntegrity checks if a backup is valid and returns an error if not
func (m *Manager) validateBackupIntegrity(b *Backup) error {
	if !fileExists(b.FilePath) {
		return fmt.Errorf("backup file does not exist")
	}

	actualChecksum, err := m.calculateFileChecksum(b.FilePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if actualChecksum != b.Checksum {
		return fmt.Errorf("checksum mismatch")
	}

	return nil
}

// identifyCorruptedBackups finds all corrupted backups in the metadata
func (m *Manager) identifyCorruptedBackups() []*Backup {
	var corrupted []*Backup

	for _, b := range m.metadata.Backups {
		if isBackupInProgress(b) {
			continue
		}

		if err := m.validateBackupIntegrity(b); err != nil {
			b.Status = StatusCorrupted
			corrupted = append(corrupted, b)
		}
	}

	return corrupted
}

// deleteCorruptedBackups deletes the provided corrupted backups
func (m *Manager) deleteCorruptedBackups(toDelete []*Backup) {
	for _, b := range toDelete {
		if err := m.deleteBackupLocked(b); err != nil {
			logging.Warn().Err(err).Str("backup_id", b.ID).Msg("Failed to delete corrupted backup")
		}
	}
}

// CleanupCorruptedBackups removes backups that fail validation
func (m *Manager) CleanupCorruptedBackups(ctx context.Context) (int, error) {
	m.metadataMu.Lock()
	defer m.metadataMu.Unlock()

	toDelete := m.identifyCorruptedBackups()
	m.deleteCorruptedBackups(toDelete)

	if err := m.saveMetadataLocked(); err != nil {
		return len(toDelete), err
	}

	return len(toDelete), nil
}

// retentionRule defines a retention rule with its selector and reason
type retentionRule struct {
	enabled   bool
	selector  func() []*Backup
	reasonFmt string
}

// addMinCountReasons adds keep reasons for minimum count protection
func addMinCountReasons(keepReasons map[string][]string, backups []*Backup, minCount int) {
	for i := 0; i < minCount && i < len(backups); i++ {
		keepReasons[backups[i].ID] = append(keepReasons[backups[i].ID], "minimum count protection")
	}
}

// addRecentHoursReasons adds keep reasons for recent hours protection
func addRecentHoursReasons(keepReasons map[string][]string, backups []*Backup, hours int, now time.Time) {
	cutoff := now.Add(-time.Duration(hours) * time.Hour)
	for _, b := range backups {
		if b.CreatedAt.After(cutoff) {
			keepReasons[b.ID] = append(keepReasons[b.ID], fmt.Sprintf("within last %d hours", hours))
		}
	}
}

// addPeriodicReasons adds keep reasons for selected backups with a given reason
func addPeriodicReasons(keepReasons map[string][]string, selected []*Backup, reason string) {
	for _, b := range selected {
		keepReasons[b.ID] = append(keepReasons[b.ID], reason)
	}
}

// buildKeepReasons builds a map of backup IDs to reasons why they should be kept
func (m *Manager) buildKeepReasons(backups []*Backup, policy RetentionPolicy, now time.Time) map[string][]string {
	keepReasons := make(map[string][]string)

	// Rule 1: MinCount
	addMinCountReasons(keepReasons, backups, policy.MinCount)

	// Rule 2: Recent hours
	if policy.KeepRecentHours > 0 {
		addRecentHoursReasons(keepReasons, backups, policy.KeepRecentHours, now)
	}

	// Rules 3-5: Periodic backups (daily, weekly, monthly)
	periodicRules := []retentionRule{
		{
			enabled:   policy.KeepDailyForDays > 0,
			selector:  func() []*Backup { return m.selectDailyBackups(backups, policy.KeepDailyForDays, now) },
			reasonFmt: fmt.Sprintf("daily backup for %d days", policy.KeepDailyForDays),
		},
		{
			enabled:   policy.KeepWeeklyForWeeks > 0,
			selector:  func() []*Backup { return m.selectWeeklyBackups(backups, policy.KeepWeeklyForWeeks, now) },
			reasonFmt: fmt.Sprintf("weekly backup for %d weeks", policy.KeepWeeklyForWeeks),
		},
		{
			enabled:   policy.KeepMonthlyForMonths > 0,
			selector:  func() []*Backup { return m.selectMonthlyBackups(backups, policy.KeepMonthlyForMonths, now) },
			reasonFmt: fmt.Sprintf("monthly backup for %d months", policy.KeepMonthlyForMonths),
		},
	}

	for _, rule := range periodicRules {
		if rule.enabled {
			addPeriodicReasons(keepReasons, rule.selector(), rule.reasonFmt)
		}
	}

	return keepReasons
}

// getDeleteReason returns the reason a backup should be deleted
func getDeleteReason(b *Backup, policy RetentionPolicy, now time.Time) string {
	if policy.MaxAgeDays > 0 {
		cutoff := now.AddDate(0, 0, -policy.MaxAgeDays)
		if b.CreatedAt.Before(cutoff) {
			return fmt.Sprintf("older than %d days", policy.MaxAgeDays)
		}
	}
	return "no retention rule matched"
}

// createPreviewItem creates a preview item from a backup
func createPreviewItem(b *Backup) *BackupPreviewItem {
	return &BackupPreviewItem{
		ID:        b.ID,
		Type:      b.Type,
		CreatedAt: b.CreatedAt,
		FileSize:  b.FileSize,
	}
}

// addKeptItem adds a backup to the kept list in the preview
func addKeptItem(preview *RetentionPreview, item *BackupPreviewItem, reasons []string, fileSize int64) {
	item.Reasons = reasons
	preview.WouldKeep = append(preview.WouldKeep, item)
	preview.TotalKeptSize += fileSize
}

// addDeletedItem adds a backup to the deleted list in the preview
func addDeletedItem(preview *RetentionPreview, item *BackupPreviewItem, reason string, fileSize int64) {
	item.Reasons = []string{reason}
	preview.WouldDelete = append(preview.WouldDelete, item)
	preview.TotalDeletedSize += fileSize
}

// updatePreviewCounts updates the count fields in the preview
func updatePreviewCounts(preview *RetentionPreview) {
	preview.KeptCount = len(preview.WouldKeep)
	preview.DeletedCount = len(preview.WouldDelete)
}

// buildPreviewItems builds the preview items from backups and keep reasons
func buildPreviewItems(backups []*Backup, keepReasons map[string][]string, policy RetentionPolicy, now time.Time, preview *RetentionPreview) {
	for _, b := range backups {
		item := createPreviewItem(b)

		if reasons, kept := keepReasons[b.ID]; kept {
			addKeptItem(preview, item, reasons, b.FileSize)
		} else {
			reason := getDeleteReason(b, policy, now)
			addDeletedItem(preview, item, reason, b.FileSize)
		}
	}

	updatePreviewCounts(preview)
}

// createEmptyPreview creates an empty retention preview
func createEmptyPreview() *RetentionPreview {
	return &RetentionPreview{
		WouldDelete: make([]*BackupPreviewItem, 0),
		WouldKeep:   make([]*BackupPreviewItem, 0),
	}
}

// GetRetentionPreview returns a preview of what would be deleted by retention policy
func (m *Manager) GetRetentionPreview() (*RetentionPreview, error) {
	m.metadataMu.RLock()
	defer m.metadataMu.RUnlock()

	preview := createEmptyPreview()

	if m.metadata == nil || len(m.metadata.Backups) == 0 {
		return preview, nil
	}

	policy := m.cfg.Retention
	now := time.Now()
	completedBackups := m.getCompletedBackupsSorted()

	keepReasons := m.buildKeepReasons(completedBackups, policy, now)
	buildPreviewItems(completedBackups, keepReasons, policy, now, preview)

	return preview, nil
}

// RetentionPreview shows what would be affected by retention policy
type RetentionPreview struct {
	WouldDelete      []*BackupPreviewItem `json:"would_delete"`
	WouldKeep        []*BackupPreviewItem `json:"would_keep"`
	DeletedCount     int                  `json:"deleted_count"`
	KeptCount        int                  `json:"kept_count"`
	TotalDeletedSize int64                `json:"total_deleted_size"`
	TotalKeptSize    int64                `json:"total_kept_size"`
}

// BackupPreviewItem represents a backup in the retention preview
type BackupPreviewItem struct {
	ID        string     `json:"id"`
	Type      BackupType `json:"type"`
	CreatedAt time.Time  `json:"created_at"`
	FileSize  int64      `json:"file_size"`
	Reasons   []string   `json:"reasons"`
}

// validateRetentionPolicy validates a retention policy
func validateRetentionPolicy(policy RetentionPolicy) error {
	if policy.MinCount < 1 {
		return fmt.Errorf("min_count must be at least 1")
	}
	if policy.MaxCount > 0 && policy.MaxCount < policy.MinCount {
		return fmt.Errorf("max_count must be >= min_count")
	}
	return nil
}

// updateRetentionPolicyLocked updates the retention policy (must be called with lock held)
func (m *Manager) updateRetentionPolicyLocked(policy RetentionPolicy) {
	m.cfg.Retention = policy
	if m.metadata != nil {
		m.metadata.Retention = policy
	}
}

// SetRetentionPolicy updates the retention policy
func (m *Manager) SetRetentionPolicy(policy RetentionPolicy) error {
	if err := validateRetentionPolicy(policy); err != nil {
		return err
	}

	m.metadataMu.Lock()
	defer m.metadataMu.Unlock()

	m.updateRetentionPolicyLocked(policy)
	return m.saveMetadataLocked()
}

// GetRetentionPolicy returns the current retention policy
func (m *Manager) GetRetentionPolicy() RetentionPolicy {
	return m.cfg.Retention
}
