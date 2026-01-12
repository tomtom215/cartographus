// Cartographus - Media Server Analytics and Geographic Visualization
// Copyright 2026 Tom F. (tomtom215)
// SPDX-License-Identifier: AGPL-3.0-or-later
// https://github.com/tomtom215/cartographus

package delivery

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/tomtom215/cartographus/internal/models"
)

func TestNewManager(t *testing.T) {
	logger := zerolog.Nop()
	config := DefaultManagerConfig()

	manager := NewManager(&logger, config)
	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	if manager.maxRetries != config.MaxRetries {
		t.Errorf("maxRetries = %d, want %d", manager.maxRetries, config.MaxRetries)
	}
	if manager.parallelism != config.Parallelism {
		t.Errorf("parallelism = %d, want %d", manager.parallelism, config.Parallelism)
	}
}

func TestNewManager_DefaultConfig(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name         string
		config       ManagerConfig
		wantRetries  int
		wantDelay    time.Duration
		wantMaxDelay time.Duration
		wantParallel int
	}{
		{
			name:         "zero values get defaults",
			config:       ManagerConfig{},
			wantRetries:  3,
			wantDelay:    1 * time.Second,
			wantMaxDelay: 30 * time.Second,
			wantParallel: 10,
		},
		{
			name: "negative values get defaults",
			config: ManagerConfig{
				MaxRetries:  -1,
				BaseDelay:   -1,
				MaxDelay:    -1,
				Parallelism: -1,
			},
			wantRetries:  3,
			wantDelay:    1 * time.Second,
			wantMaxDelay: 30 * time.Second,
			wantParallel: 10,
		},
		{
			name: "custom values preserved",
			config: ManagerConfig{
				MaxRetries:  5,
				BaseDelay:   2 * time.Second,
				MaxDelay:    60 * time.Second,
				Parallelism: 20,
			},
			wantRetries:  5,
			wantDelay:    2 * time.Second,
			wantMaxDelay: 60 * time.Second,
			wantParallel: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(&logger, tt.config)
			if manager.maxRetries != tt.wantRetries {
				t.Errorf("maxRetries = %d, want %d", manager.maxRetries, tt.wantRetries)
			}
			if manager.baseDelay != tt.wantDelay {
				t.Errorf("baseDelay = %v, want %v", manager.baseDelay, tt.wantDelay)
			}
			if manager.maxDelay != tt.wantMaxDelay {
				t.Errorf("maxDelay = %v, want %v", manager.maxDelay, tt.wantMaxDelay)
			}
			if manager.parallelism != tt.wantParallel {
				t.Errorf("parallelism = %d, want %d", manager.parallelism, tt.wantParallel)
			}
		})
	}
}

func TestDefaultManagerConfig(t *testing.T) {
	config := DefaultManagerConfig()

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}
	if config.BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want 1s", config.BaseDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", config.MaxDelay)
	}
	if config.Parallelism != 10 {
		t.Errorf("Parallelism = %d, want 10", config.Parallelism)
	}
}

func TestManager_GetAvailableChannels(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewManager(&logger, DefaultManagerConfig())

	channels := manager.GetAvailableChannels()
	if len(channels) == 0 {
		t.Fatal("GetAvailableChannels returned empty list")
	}

	expectedChannels := []models.DeliveryChannel{
		models.DeliveryChannelEmail,
		models.DeliveryChannelDiscord,
		models.DeliveryChannelSlack,
		models.DeliveryChannelTelegram,
		models.DeliveryChannelWebhook,
		models.DeliveryChannelInApp,
	}

	for _, expected := range expectedChannels {
		found := false
		for _, ch := range channels {
			if ch == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected channel %s not found", expected)
		}
	}
}

func TestManager_ValidateChannelConfigs(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewManager(&logger, DefaultManagerConfig())

	tests := []struct {
		name     string
		channels []models.DeliveryChannel
		configs  map[models.DeliveryChannel]*models.ChannelConfig
		wantErr  bool
	}{
		{
			name:     "valid email config",
			channels: []models.DeliveryChannel{models.DeliveryChannelEmail},
			configs: map[models.DeliveryChannel]*models.ChannelConfig{
				models.DeliveryChannelEmail: {
					SMTPHost: "smtp.example.com",
					SMTPPort: 587,
					SMTPFrom: "noreply@example.com",
				},
			},
			wantErr: false,
		},
		{
			name:     "valid discord config",
			channels: []models.DeliveryChannel{models.DeliveryChannelDiscord},
			configs: map[models.DeliveryChannel]*models.ChannelConfig{
				models.DeliveryChannelDiscord: {
					DiscordWebhookURL: "https://discord.com/api/webhooks/123/abc",
				},
			},
			wantErr: false,
		},
		{
			name:     "unknown channel",
			channels: []models.DeliveryChannel{models.DeliveryChannel("unknown")},
			configs:  map[models.DeliveryChannel]*models.ChannelConfig{},
			wantErr:  true,
		},
		{
			name:     "invalid email config",
			channels: []models.DeliveryChannel{models.DeliveryChannelEmail},
			configs: map[models.DeliveryChannel]*models.ChannelConfig{
				models.DeliveryChannelEmail: {
					SMTPHost: "",
					SMTPPort: 0,
				},
			},
			wantErr: true,
		},
		{
			name:     "inapp needs no config",
			channels: []models.DeliveryChannel{models.DeliveryChannelInApp},
			configs:  map[models.DeliveryChannel]*models.ChannelConfig{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ValidateChannelConfigs(tt.channels, tt.configs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateChannelConfigs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManager_calculateBackoff(t *testing.T) {
	logger := zerolog.Nop()
	config := ManagerConfig{
		MaxRetries:  5,
		BaseDelay:   1 * time.Second,
		MaxDelay:    10 * time.Second,
		Parallelism: 1,
	}
	manager := NewManager(&logger, config)

	tests := []struct {
		name       string
		attempt    int
		lastResult *DeliveryResult
		minDelay   time.Duration
		maxDelay   time.Duration
	}{
		{
			name:       "first retry",
			attempt:    1,
			lastResult: nil,
			minDelay:   1 * time.Second,
			maxDelay:   1 * time.Second,
		},
		{
			name:       "second retry",
			attempt:    2,
			lastResult: nil,
			minDelay:   2 * time.Second,
			maxDelay:   2 * time.Second,
		},
		{
			name:       "third retry",
			attempt:    3,
			lastResult: nil,
			minDelay:   4 * time.Second,
			maxDelay:   4 * time.Second,
		},
		{
			name:       "capped at max delay",
			attempt:    5,
			lastResult: nil,
			minDelay:   10 * time.Second,
			maxDelay:   10 * time.Second,
		},
		{
			name:    "server specified retry-after",
			attempt: 1,
			lastResult: &DeliveryResult{
				RetryAfter: func() *time.Duration { d := 5 * time.Second; return &d }(),
			},
			minDelay: 5 * time.Second,
			maxDelay: 5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := manager.calculateBackoff(tt.attempt, tt.lastResult)
			if delay < tt.minDelay || delay > tt.maxDelay {
				t.Errorf("calculateBackoff() = %v, want between %v and %v", delay, tt.minDelay, tt.maxDelay)
			}
		})
	}
}

// MockChannel implements Channel interface for testing
type MockChannel struct {
	name             models.DeliveryChannel
	supportsHTML     bool
	maxContentLength int
	validateError    error
	sendResult       *DeliveryResult
	sendError        error
	sendCallCount    int32
}

func (m *MockChannel) Name() models.DeliveryChannel {
	return m.name
}

func (m *MockChannel) SupportsHTML() bool {
	return m.supportsHTML
}

func (m *MockChannel) MaxContentLength() int {
	return m.maxContentLength
}

func (m *MockChannel) Validate(config *models.ChannelConfig) error {
	return m.validateError
}

func (m *MockChannel) Send(ctx context.Context, params *SendParams) (*DeliveryResult, error) {
	atomic.AddInt32(&m.sendCallCount, 1)
	if m.sendError != nil {
		return nil, m.sendError
	}
	return m.sendResult, nil
}

func TestDeliveryRequest_Fields(t *testing.T) {
	req := &DeliveryRequest{
		DeliveryID: "delivery-123",
		ScheduleID: "schedule-456",
		Template: &models.NewsletterTemplate{
			ID:   "template-789",
			Name: "Test Template",
		},
		Recipients: []models.NewsletterRecipient{
			{Type: "email", Target: "test@example.com"},
		},
		Channels:        []models.DeliveryChannel{models.DeliveryChannelEmail},
		RenderedSubject: "Test Subject",
		RenderedHTML:    "<h1>Test</h1>",
		RenderedText:    "Test",
	}

	if req.DeliveryID != "delivery-123" {
		t.Errorf("DeliveryID = %q, want %q", req.DeliveryID, "delivery-123")
	}
	if req.ScheduleID != "schedule-456" {
		t.Errorf("ScheduleID = %q, want %q", req.ScheduleID, "schedule-456")
	}
	if len(req.Recipients) != 1 {
		t.Errorf("Recipients count = %d, want 1", len(req.Recipients))
	}
	if req.Template.Name != "Test Template" {
		t.Errorf("Template.Name = %q, want %q", req.Template.Name, "Test Template")
	}
	if len(req.Channels) != 1 {
		t.Errorf("Channels count = %d, want 1", len(req.Channels))
	}
	if req.RenderedSubject != "Test Subject" {
		t.Errorf("RenderedSubject = %q, want %q", req.RenderedSubject, "Test Subject")
	}
	if req.RenderedHTML != "<h1>Test</h1>" {
		t.Errorf("RenderedHTML = %q, want %q", req.RenderedHTML, "<h1>Test</h1>")
	}
	if req.RenderedText != "Test" {
		t.Errorf("RenderedText = %q, want %q", req.RenderedText, "Test")
	}
}

func TestDeliveryReport_Fields(t *testing.T) {
	now := time.Now()
	report := &DeliveryReport{
		DeliveryID:           "delivery-123",
		Status:               models.DeliveryStatusDelivered,
		TotalRecipients:      10,
		SuccessfulDeliveries: 8,
		FailedDeliveries:     2,
		StartedAt:            now,
		CompletedAt:          now.Add(5 * time.Second),
		DurationMS:           5000,
	}

	if report.DeliveryID != "delivery-123" {
		t.Errorf("DeliveryID = %q, want %q", report.DeliveryID, "delivery-123")
	}
	if report.Status != models.DeliveryStatusDelivered {
		t.Errorf("Status = %v, want %v", report.Status, models.DeliveryStatusDelivered)
	}
	if report.TotalRecipients != 10 {
		t.Errorf("TotalRecipients = %d, want 10", report.TotalRecipients)
	}
	if report.SuccessfulDeliveries != 8 {
		t.Errorf("SuccessfulDeliveries = %d, want 8", report.SuccessfulDeliveries)
	}
	if report.FailedDeliveries != 2 {
		t.Errorf("FailedDeliveries = %d, want 2", report.FailedDeliveries)
	}
	if !report.StartedAt.Equal(now) {
		t.Errorf("StartedAt = %v, want %v", report.StartedAt, now)
	}
	if !report.CompletedAt.Equal(now.Add(5 * time.Second)) {
		t.Errorf("CompletedAt = %v, want %v", report.CompletedAt, now.Add(5*time.Second))
	}
	if report.DurationMS != 5000 {
		t.Errorf("DurationMS = %d, want 5000", report.DurationMS)
	}
}

func TestFormatForChannel(t *testing.T) {
	emailChannel := NewEmailChannel()
	discordChannel := NewDiscordChannel()

	subject := "Test Subject"
	htmlContent := "<h1>Hello</h1><p>World</p>"
	textContent := "Hello World"

	// Email supports HTML - both HTML and text are preserved
	bodyHTML, bodyText := FormatForChannel(emailChannel, subject, htmlContent, textContent)
	if bodyHTML != htmlContent {
		t.Errorf("Email HTML should preserve content, got %q", bodyHTML)
	}
	if bodyText != textContent {
		t.Errorf("Email text should preserve content, got %q", bodyText)
	}

	// Discord does not support HTML - when text is provided, HTML is kept but text is used
	// The FormatForChannel function preserves HTML for potential future use
	bodyHTML, bodyText = FormatForChannel(discordChannel, subject, htmlContent, textContent)
	// HTML is preserved but will be truncated to Discord's limit
	if len(bodyHTML) > discordChannel.MaxContentLength() {
		t.Errorf("Discord HTML should be truncated to %d, got %d", discordChannel.MaxContentLength(), len(bodyHTML))
	}
	// Text content should be preserved
	if bodyText != textContent {
		t.Errorf("Discord text should use text content, got %q", bodyText)
	}
}

func TestFormatForChannel_Truncation(t *testing.T) {
	slackChannel := NewSlackChannel() // 3000 char limit

	subject := "Test"
	htmlContent := ""
	// Create text content longer than Slack's limit
	longText := make([]byte, 4000)
	for i := range longText {
		longText[i] = 'x'
	}
	textContent := string(longText)

	_, bodyText := FormatForChannel(slackChannel, subject, htmlContent, textContent)
	if len(bodyText) > 3000 {
		t.Errorf("Slack content should be truncated to 3000, got %d", len(bodyText))
	}
}

// Mock store for InApp testing
type MockInAppStore struct {
	mu            sync.Mutex
	notifications []InAppNotification
	err           error
}

func (m *MockInAppStore) CreateNotification(ctx context.Context, notification *InAppNotification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.notifications = append(m.notifications, *notification)
	return nil
}

func (m *MockInAppStore) getNotifications() []InAppNotification {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]InAppNotification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func TestManager_SetInAppStore(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewManager(&logger, DefaultManagerConfig())

	mockStore := &MockInAppStore{}
	manager.SetInAppStore(mockStore)

	// Verify the store was set by checking the channel
	channel, ok := manager.registry.Get(models.DeliveryChannelInApp)
	if !ok {
		t.Fatal("InApp channel not found")
	}

	inAppChannel, ok := channel.(*InAppChannel)
	if !ok {
		t.Fatal("Channel is not InAppChannel")
	}

	// Just verify it's set (can't compare directly due to interface type)
	if inAppChannel.store == nil {
		t.Error("Store was not set")
	}
}

func TestManager_Deliver_InApp(t *testing.T) {
	logger := zerolog.Nop()
	config := ManagerConfig{
		MaxRetries:  1,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Parallelism: 2,
	}
	manager := NewManager(&logger, config)

	// Set up mock store
	mockStore := &MockInAppStore{}
	manager.SetInAppStore(mockStore)

	req := &DeliveryRequest{
		DeliveryID:      "test-delivery-123",
		ScheduleID:      "test-schedule-456",
		Recipients:      []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
		Channels:        []models.DeliveryChannel{models.DeliveryChannelInApp},
		RenderedSubject: "Test Newsletter",
		RenderedHTML:    "<h1>Hello</h1>",
		RenderedText:    "Hello",
		Metadata: &DeliveryMetadata{
			DeliveryID: "test-delivery-123",
			ScheduleID: "test-schedule-456",
			ServerName: "Test Server",
		},
	}

	report, err := manager.Deliver(context.Background(), req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	if report.DeliveryID != "test-delivery-123" {
		t.Errorf("DeliveryID = %q, want %q", report.DeliveryID, "test-delivery-123")
	}
	if report.TotalRecipients != 1 {
		t.Errorf("TotalRecipients = %d, want 1", report.TotalRecipients)
	}
	if report.SuccessfulDeliveries != 1 {
		t.Errorf("SuccessfulDeliveries = %d, want 1", report.SuccessfulDeliveries)
	}
	if report.FailedDeliveries != 0 {
		t.Errorf("FailedDeliveries = %d, want 0", report.FailedDeliveries)
	}
	if report.Status != models.DeliveryStatusDelivered {
		t.Errorf("Status = %v, want %v", report.Status, models.DeliveryStatusDelivered)
	}
	notifications := mockStore.getNotifications()
	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}
}

func TestManager_Deliver_MultipleRecipients(t *testing.T) {
	logger := zerolog.Nop()
	config := ManagerConfig{
		MaxRetries:  1,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Parallelism: 5,
	}
	manager := NewManager(&logger, config)

	// Set up mock store
	mockStore := &MockInAppStore{}
	manager.SetInAppStore(mockStore)

	req := &DeliveryRequest{
		DeliveryID: "test-delivery-multi",
		Recipients: []models.NewsletterRecipient{
			{Type: "user", Target: "user-1"},
			{Type: "user", Target: "user-2"},
			{Type: "user", Target: "user-3"},
		},
		Channels:        []models.DeliveryChannel{models.DeliveryChannelInApp},
		RenderedSubject: "Multi-recipient Test",
		RenderedText:    "Hello everyone",
	}

	report, err := manager.Deliver(context.Background(), req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	if report.TotalRecipients != 3 {
		t.Errorf("TotalRecipients = %d, want 3", report.TotalRecipients)
	}
	if report.SuccessfulDeliveries != 3 {
		t.Errorf("SuccessfulDeliveries = %d, want 3", report.SuccessfulDeliveries)
	}
	notifications := mockStore.getNotifications()
	if len(notifications) != 3 {
		t.Errorf("Expected 3 notifications, got %d", len(notifications))
	}
}

func TestManager_Deliver_PartialFailure(t *testing.T) {
	logger := zerolog.Nop()
	config := ManagerConfig{
		MaxRetries:  0,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Parallelism: 2,
	}
	manager := NewManager(&logger, config)

	// Mix: some to InApp (will succeed), some to unknown recipient type (will fail)
	mockStore := &MockInAppStore{}
	manager.SetInAppStore(mockStore)

	req := &DeliveryRequest{
		DeliveryID: "test-delivery-partial",
		Recipients: []models.NewsletterRecipient{
			{Type: "user", Target: "user-1"},     // Will succeed
			{Type: "email", Target: "bad@email"}, // Will fail (wrong type for InApp)
		},
		Channels:        []models.DeliveryChannel{models.DeliveryChannelInApp},
		RenderedSubject: "Partial Test",
		RenderedText:    "Hello",
	}

	report, err := manager.Deliver(context.Background(), req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	if report.Status != models.DeliveryStatusPartial {
		t.Errorf("Status = %v, want %v", report.Status, models.DeliveryStatusPartial)
	}
	if report.SuccessfulDeliveries != 1 {
		t.Errorf("SuccessfulDeliveries = %d, want 1", report.SuccessfulDeliveries)
	}
	if report.FailedDeliveries != 1 {
		t.Errorf("FailedDeliveries = %d, want 1", report.FailedDeliveries)
	}
}

func TestManager_Deliver_AllFailed(t *testing.T) {
	logger := zerolog.Nop()
	config := ManagerConfig{
		MaxRetries:  0,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Parallelism: 2,
	}
	manager := NewManager(&logger, config)

	// Don't set InApp store - all deliveries will fail
	req := &DeliveryRequest{
		DeliveryID: "test-delivery-fail",
		Recipients: []models.NewsletterRecipient{
			{Type: "user", Target: "user-1"},
			{Type: "user", Target: "user-2"},
		},
		Channels:        []models.DeliveryChannel{models.DeliveryChannelInApp},
		RenderedSubject: "Fail Test",
		RenderedText:    "Hello",
	}

	report, err := manager.Deliver(context.Background(), req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	if report.Status != models.DeliveryStatusFailed {
		t.Errorf("Status = %v, want %v", report.Status, models.DeliveryStatusFailed)
	}
	if report.SuccessfulDeliveries != 0 {
		t.Errorf("SuccessfulDeliveries = %d, want 0", report.SuccessfulDeliveries)
	}
	if report.FailedDeliveries != 2 {
		t.Errorf("FailedDeliveries = %d, want 2", report.FailedDeliveries)
	}
}

func TestManager_Deliver_EmptyRequest(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewManager(&logger, DefaultManagerConfig())

	req := &DeliveryRequest{
		DeliveryID: "test-delivery-empty",
		Recipients: []models.NewsletterRecipient{},
		Channels:   []models.DeliveryChannel{},
	}

	report, err := manager.Deliver(context.Background(), req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	if report.TotalRecipients != 0 {
		t.Errorf("TotalRecipients = %d, want 0", report.TotalRecipients)
	}
	if report.Status != models.DeliveryStatusDelivered {
		t.Errorf("Status = %v, want %v (empty = success)", report.Status, models.DeliveryStatusDelivered)
	}
}

func TestManager_Deliver_UnknownChannel(t *testing.T) {
	logger := zerolog.Nop()
	config := ManagerConfig{
		MaxRetries:  0,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Parallelism: 1,
	}
	manager := NewManager(&logger, config)

	req := &DeliveryRequest{
		DeliveryID: "test-unknown-channel",
		Recipients: []models.NewsletterRecipient{
			{Type: "user", Target: "user-1"},
		},
		Channels:        []models.DeliveryChannel{models.DeliveryChannel("unknown")},
		RenderedSubject: "Unknown Channel Test",
		RenderedText:    "Hello",
	}

	report, err := manager.Deliver(context.Background(), req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	if report.FailedDeliveries != 1 {
		t.Errorf("FailedDeliveries = %d, want 1", report.FailedDeliveries)
	}
	if len(report.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(report.Results))
	}
	if report.Results[0].ErrorCode != ErrorCodeInvalidConfig {
		t.Errorf("ErrorCode = %q, want %q", report.Results[0].ErrorCode, ErrorCodeInvalidConfig)
	}
}

func TestManager_Deliver_ContextCanceled(t *testing.T) {
	logger := zerolog.Nop()
	config := ManagerConfig{
		MaxRetries:  5,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    1 * time.Second,
		Parallelism: 1,
	}
	manager := NewManager(&logger, config)

	// Use a store that returns transient errors to trigger retries
	failingStore := &MockInAppStore{err: context.Canceled}
	manager.SetInAppStore(failingStore)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	req := &DeliveryRequest{
		DeliveryID: "test-canceled",
		Recipients: []models.NewsletterRecipient{
			{Type: "user", Target: "user-1"},
		},
		Channels:        []models.DeliveryChannel{models.DeliveryChannelInApp},
		RenderedSubject: "Cancel Test",
		RenderedText:    "Hello",
	}

	report, err := manager.Deliver(ctx, req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	// Should fail due to canceled context
	if report.FailedDeliveries == 0 {
		t.Error("Expected failures due to canceled context")
	}
}

func TestManager_Deliver_DurationTracking(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewManager(&logger, DefaultManagerConfig())

	mockStore := &MockInAppStore{}
	manager.SetInAppStore(mockStore)

	req := &DeliveryRequest{
		DeliveryID:   "test-duration",
		Recipients:   []models.NewsletterRecipient{{Type: "user", Target: "user-1"}},
		Channels:     []models.DeliveryChannel{models.DeliveryChannelInApp},
		RenderedText: "Hello",
	}

	report, err := manager.Deliver(context.Background(), req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	if report.StartedAt.IsZero() {
		t.Error("StartedAt should not be zero")
	}
	if report.CompletedAt.IsZero() {
		t.Error("CompletedAt should not be zero")
	}
	if !report.CompletedAt.After(report.StartedAt) && !report.CompletedAt.Equal(report.StartedAt) {
		t.Error("CompletedAt should be >= StartedAt")
	}
	if report.DurationMS < 0 {
		t.Error("DurationMS should be >= 0")
	}
}

func TestManager_Deliver_MultipleChannels(t *testing.T) {
	logger := zerolog.Nop()
	config := ManagerConfig{
		MaxRetries:  0,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    10 * time.Millisecond,
		Parallelism: 5,
	}
	manager := NewManager(&logger, config)

	mockStore := &MockInAppStore{}
	manager.SetInAppStore(mockStore)

	req := &DeliveryRequest{
		DeliveryID: "test-multi-channel",
		Recipients: []models.NewsletterRecipient{
			{Type: "user", Target: "user-1"},
		},
		Channels: []models.DeliveryChannel{
			models.DeliveryChannelInApp, // Will succeed
			models.DeliveryChannelEmail, // Will fail (no SMTP config)
		},
		ChannelConfigs: map[models.DeliveryChannel]*models.ChannelConfig{
			models.DeliveryChannelEmail: nil, // Invalid config
		},
		RenderedSubject: "Multi-channel Test",
		RenderedText:    "Hello",
	}

	report, err := manager.Deliver(context.Background(), req)
	if err != nil {
		t.Fatalf("Deliver returned error: %v", err)
	}

	// 1 recipient * 2 channels = 2 total
	if report.TotalRecipients != 2 {
		t.Errorf("TotalRecipients = %d, want 2", report.TotalRecipients)
	}
	// InApp should succeed, Email should fail
	if report.SuccessfulDeliveries != 1 {
		t.Errorf("SuccessfulDeliveries = %d, want 1", report.SuccessfulDeliveries)
	}
	if report.FailedDeliveries != 1 {
		t.Errorf("FailedDeliveries = %d, want 1", report.FailedDeliveries)
	}
}
