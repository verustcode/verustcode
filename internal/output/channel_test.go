package output

import (
	"context"
	"testing"

	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
)

// mockChannel is a mock implementation of Channel for testing
type mockChannel struct {
	name         string
	publishErr   error
	publishCalls int
}

func (m *mockChannel) Name() string {
	return m.name
}

func (m *mockChannel) Publish(ctx context.Context, result *prompt.ReviewResult, opts *PublishOptions) error {
	m.publishCalls++
	return m.publishErr
}

// TestRegisterAndCreate tests channel registration and creation
func TestRegisterAndCreate(t *testing.T) {
	// Save original registry and restore after test
	originalRegistry := make(map[string]ChannelFactory)
	for k, v := range Registry {
		originalRegistry[k] = v
	}
	defer func() {
		Registry = originalRegistry
	}()

	t.Run("register and create channel", func(t *testing.T) {
		// Register a test channel
		Register("test-channel", func(s store.Store) Channel {
			return &mockChannel{name: "test-channel"}
		})

		// Create the channel
		ch, err := Create("test-channel", nil)
		if err != nil {
			t.Fatalf("Create() unexpected error: %v", err)
		}

		if ch == nil {
			t.Fatal("Create() returned nil channel")
		}

		if ch.Name() != "test-channel" {
			t.Errorf("Channel name = %s, want 'test-channel'", ch.Name())
		}
	})

	t.Run("create unknown channel", func(t *testing.T) {
		_, err := Create("non-existent-channel", nil)
		if err == nil {
			t.Error("Create() expected error for unknown channel, got nil")
		}
	})
}

// TestCreateAll tests creating multiple channels
func TestCreateAll(t *testing.T) {
	t.Run("create registered channels", func(t *testing.T) {
		// file, comment, webhook are registered in init()
		channels, err := CreateAll([]string{"file", "comment"}, nil)
		if err != nil {
			t.Fatalf("CreateAll() unexpected error: %v", err)
		}

		if len(channels) != 2 {
			t.Errorf("CreateAll() returned %d channels, want 2", len(channels))
		}
	})

	t.Run("create with unknown channel", func(t *testing.T) {
		_, err := CreateAll([]string{"file", "unknown-channel"}, nil)
		if err == nil {
			t.Error("CreateAll() expected error for unknown channel, got nil")
		}
	})

	t.Run("create empty list", func(t *testing.T) {
		channels, err := CreateAll([]string{}, nil)
		if err != nil {
			t.Fatalf("CreateAll() unexpected error: %v", err)
		}

		if len(channels) != 0 {
			t.Errorf("CreateAll() returned %d channels, want 0", len(channels))
		}
	})
}

// TestChannelNames tests getting all registered channel names
func TestChannelNames(t *testing.T) {
	names := ChannelNames()

	// At minimum, we should have the 3 registered channels
	if len(names) < 3 {
		t.Errorf("ChannelNames() returned %d names, want at least 3", len(names))
	}

	// Check that expected channels are registered
	expectedChannels := map[string]bool{
		"file":    false,
		"comment": false,
		"webhook": false,
	}

	for _, name := range names {
		if _, exists := expectedChannels[name]; exists {
			expectedChannels[name] = true
		}
	}

	for name, found := range expectedChannels {
		if !found {
			t.Errorf("Expected channel %s not found in registry", name)
		}
	}
}

// TestNewPublisher tests creating a new publisher
func TestNewPublisher(t *testing.T) {
	t.Run("with channels", func(t *testing.T) {
		ch1 := &mockChannel{name: "ch1"}
		ch2 := &mockChannel{name: "ch2"}

		publisher := NewPublisher(ch1, ch2)

		if publisher == nil {
			t.Fatal("NewPublisher() returned nil")
		}

		if len(publisher.channels) != 2 {
			t.Errorf("Publisher has %d channels, want 2", len(publisher.channels))
		}
	})

	t.Run("empty channels", func(t *testing.T) {
		publisher := NewPublisher()

		if publisher == nil {
			t.Fatal("NewPublisher() returned nil")
		}

		if len(publisher.channels) != 0 {
			t.Errorf("Publisher has %d channels, want 0", len(publisher.channels))
		}
	})
}

// TestNewPublisherFromNames tests creating a publisher from channel names
func TestNewPublisherFromNames(t *testing.T) {
	t.Run("valid names", func(t *testing.T) {
		publisher, err := NewPublisherFromNames([]string{"file", "comment"}, nil)
		if err != nil {
			t.Fatalf("NewPublisherFromNames() unexpected error: %v", err)
		}

		if publisher == nil {
			t.Fatal("NewPublisherFromNames() returned nil")
		}

		if len(publisher.channels) != 2 {
			t.Errorf("Publisher has %d channels, want 2", len(publisher.channels))
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		_, err := NewPublisherFromNames([]string{"file", "invalid-channel"}, nil)
		if err == nil {
			t.Error("NewPublisherFromNames() expected error for invalid channel, got nil")
		}
	})
}

// TestPublisher_Add tests adding channels to a publisher
func TestPublisher_Add(t *testing.T) {
	publisher := NewPublisher()

	ch1 := &mockChannel{name: "ch1"}
	ch2 := &mockChannel{name: "ch2"}

	publisher.Add(ch1)
	if len(publisher.channels) != 1 {
		t.Errorf("Publisher has %d channels after Add, want 1", len(publisher.channels))
	}

	publisher.Add(ch2)
	if len(publisher.channels) != 2 {
		t.Errorf("Publisher has %d channels after Add, want 2", len(publisher.channels))
	}
}

// TestPublisher_Publish tests publishing to multiple channels
func TestPublisher_Publish(t *testing.T) {
	t.Run("all channels succeed", func(t *testing.T) {
		ch1 := &mockChannel{name: "ch1"}
		ch2 := &mockChannel{name: "ch2"}

		publisher := NewPublisher(ch1, ch2)

		result := prompt.NewReviewResult("test-reviewer")
		opts := &PublishOptions{ReviewID: "test-123"}

		err := publisher.Publish(context.Background(), result, opts)
		if err != nil {
			t.Errorf("Publish() unexpected error: %v", err)
		}

		if ch1.publishCalls != 1 {
			t.Errorf("ch1 publish calls = %d, want 1", ch1.publishCalls)
		}

		if ch2.publishCalls != 1 {
			t.Errorf("ch2 publish calls = %d, want 1", ch2.publishCalls)
		}
	})

	t.Run("some channels fail", func(t *testing.T) {
		ch1 := &mockChannel{name: "ch1"}
		ch2 := &mockChannel{name: "ch2", publishErr: context.DeadlineExceeded}

		publisher := NewPublisher(ch1, ch2)

		result := prompt.NewReviewResult("test-reviewer")
		opts := &PublishOptions{ReviewID: "test-123"}

		err := publisher.Publish(context.Background(), result, opts)
		if err == nil {
			t.Error("Publish() expected error when channel fails, got nil")
		}

		// Both channels should still be called
		if ch1.publishCalls != 1 {
			t.Errorf("ch1 publish calls = %d, want 1", ch1.publishCalls)
		}

		if ch2.publishCalls != 1 {
			t.Errorf("ch2 publish calls = %d, want 1", ch2.publishCalls)
		}
	})

	t.Run("empty publisher", func(t *testing.T) {
		publisher := NewPublisher()

		result := prompt.NewReviewResult("test-reviewer")
		opts := &PublishOptions{ReviewID: "test-123"}

		err := publisher.Publish(context.Background(), result, opts)
		if err != nil {
			t.Errorf("Publish() unexpected error for empty publisher: %v", err)
		}
	})
}

// TestCreateFromConfig tests creating channels from DSL config
func TestCreateFromConfig(t *testing.T) {
	t.Run("file type", func(t *testing.T) {
		cfg := &dsl.OutputItemConfig{Type: "file"}
		ch, err := CreateFromConfig(cfg, nil)
		if err != nil {
			t.Fatalf("CreateFromConfig() unexpected error: %v", err)
		}

		if ch == nil {
			t.Fatal("CreateFromConfig() returned nil")
		}
	})

	t.Run("file type with dir", func(t *testing.T) {
		cfg := &dsl.OutputItemConfig{
			Type:      "file",
			Dir:       "/tmp/output",
			Overwrite: true,
		}
		ch, err := CreateFromConfig(cfg, nil)
		if err != nil {
			t.Fatalf("CreateFromConfig() unexpected error: %v", err)
		}

		if ch == nil {
			t.Fatal("CreateFromConfig() returned nil")
		}
	})

	t.Run("comment type append mode", func(t *testing.T) {
		cfg := &dsl.OutputItemConfig{
			Type:      "comment",
			Overwrite: false,
		}
		ch, err := CreateFromConfig(cfg, nil)
		if err != nil {
			t.Fatalf("CreateFromConfig() unexpected error: %v", err)
		}

		if ch == nil {
			t.Fatal("CreateFromConfig() returned nil")
		}

		if ch.Name() != "comment" {
			t.Errorf("Channel name = %s, want 'comment'", ch.Name())
		}
	})

	t.Run("comment type overwrite mode", func(t *testing.T) {
		cfg := &dsl.OutputItemConfig{
			Type:         "comment",
			Overwrite:    true,
			MarkerPrefix: "custom-marker",
		}
		ch, err := CreateFromConfig(cfg, nil)
		if err != nil {
			t.Fatalf("CreateFromConfig() unexpected error: %v", err)
		}

		if ch == nil {
			t.Fatal("CreateFromConfig() returned nil")
		}
	})

	t.Run("webhook type", func(t *testing.T) {
		cfg := &dsl.OutputItemConfig{
			Type:         "webhook",
			URL:          "https://example.com/callback",
			HeaderSecret: "secret-key-12345",
			Timeout:      30,
			MaxRetries:   3,
		}
		ch, err := CreateFromConfig(cfg, nil)
		if err != nil {
			t.Fatalf("CreateFromConfig() unexpected error: %v", err)
		}

		if ch == nil {
			t.Fatal("CreateFromConfig() returned nil")
		}

		if ch.Name() != "webhook" {
			t.Errorf("Channel name = %s, want 'webhook'", ch.Name())
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		cfg := &dsl.OutputItemConfig{Type: "unknown"}
		_, err := CreateFromConfig(cfg, nil)
		if err == nil {
			t.Error("CreateFromConfig() expected error for unknown type, got nil")
		}
	})
}

// TestCreateAllFromConfig tests creating all channels from OutputConfig
func TestCreateAllFromConfig(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		_, err := CreateAllFromConfig(nil, nil)
		if err == nil {
			t.Error("CreateAllFromConfig() expected error for nil config, got nil")
		}
	})

	t.Run("empty list returns error", func(t *testing.T) {
		cfg := &dsl.OutputConfig{Channels: []dsl.OutputItemConfig{}}
		_, err := CreateAllFromConfig(cfg, nil)
		if err == nil {
			t.Error("CreateAllFromConfig() expected error for empty channels, got nil")
		}
	})

	t.Run("multiple channels", func(t *testing.T) {
		cfg := &dsl.OutputConfig{
			Channels: []dsl.OutputItemConfig{
				{Type: "file"},
				{Type: "comment"},
			},
		}
		channels, err := CreateAllFromConfig(cfg, nil)
		if err != nil {
			t.Fatalf("CreateAllFromConfig() unexpected error: %v", err)
		}

		if len(channels) != 2 {
			t.Errorf("CreateAllFromConfig() returned %d channels, want 2", len(channels))
		}
	})

	t.Run("with invalid type", func(t *testing.T) {
		cfg := &dsl.OutputConfig{
			Channels: []dsl.OutputItemConfig{
				{Type: "file"},
				{Type: "invalid-type"},
			},
		}
		_, err := CreateAllFromConfig(cfg, nil)
		if err == nil {
			t.Error("CreateAllFromConfig() expected error for invalid type, got nil")
		}
	})
}

// TestNewPublisherFromConfig tests creating a publisher from OutputConfig
func TestNewPublisherFromConfig(t *testing.T) {
	t.Run("nil config returns error", func(t *testing.T) {
		_, err := NewPublisherFromConfig(nil, nil)
		if err == nil {
			t.Error("NewPublisherFromConfig() expected error for nil config, got nil")
		}
	})

	t.Run("with multiple channels", func(t *testing.T) {
		cfg := &dsl.OutputConfig{
			Channels: []dsl.OutputItemConfig{
				{Type: "file"},
				{Type: "comment"},
			},
		}

		publisher, err := NewPublisherFromConfig(cfg, nil)
		if err != nil {
			t.Fatalf("NewPublisherFromConfig() unexpected error: %v", err)
		}

		if len(publisher.channels) != 2 {
			t.Errorf("Publisher has %d channels, want 2", len(publisher.channels))
		}
	})
}
