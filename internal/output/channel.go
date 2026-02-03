// Package output provides output channels for publishing review results.
// It supports multiple output formats and destinations including files
// and Git platform comments.
package output

import (
	"context"
	"fmt"

	"github.com/verustcode/verustcode/consts"
	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/dsl"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/prompt"
	"github.com/verustcode/verustcode/internal/store"
	"github.com/verustcode/verustcode/pkg/errors"
)

// GetDefaultFormat returns the default format for a channel type.
// - webhook: defaults to "json" (structured data for API integration)
// - file, comment: defaults to "markdown" (human readable)
func GetDefaultFormat(channelType string) string {
	switch channelType {
	case "webhook":
		return consts.OutputFormatJSON
	default:
		return consts.OutputFormatMarkdown
	}
}

// GetEffectiveFormat returns the effective format for a channel config.
// If format is specified, use it; otherwise use the smart default.
func GetEffectiveFormat(cfg *dsl.OutputItemConfig) string {
	if cfg.Format != "" {
		return cfg.Format
	}
	return GetDefaultFormat(cfg.Type)
}

// Channel defines the interface for output channels
type Channel interface {
	// Name returns the channel name
	Name() string

	// Publish publishes the review result to this channel
	Publish(ctx context.Context, result *prompt.ReviewResult, opts *PublishOptions) error
}

// PublishOptions provides options for publishing
type PublishOptions struct {
	// ReviewID is a unique identifier for this review
	ReviewID string

	// RepoURL is the repository URL
	RepoURL string

	// Ref is the branch/tag/commit
	Ref string

	// PRNumber is the PR/MR number (for Git comments)
	PRNumber int

	// PRInfo contains PR/MR information (URL, title, etc.)
	// This is populated when available from provider API
	PRInfo *provider.PullRequest

	// OutputDir is the output directory (for file channels)
	OutputDir string

	// FileName is the output file name (for file channels)
	FileName string

	// Overwrite allows overwriting existing files
	Overwrite bool

	// Provider is the Git provider instance (for comment channels)
	Provider provider.Provider

	// CommentMode specifies the comment mode (append or overwrite)
	CommentMode string

	// CommentMarker is the marker prefix for identifying VerustCode comments
	CommentMarker string

	// RepoPath is the local repository path (for extracting workspace name)
	RepoPath string

	// MetadataConfig controls output metadata (from global config)
	MetadataConfig *config.OutputMetadataConfig

	// AgentName is the agent name used
	AgentName string

	// ModelName is the model name used
	ModelName string
}

// Registry holds registered channel factories
var Registry = make(map[string]ChannelFactory)

// ChannelFactory creates a Channel instance
type ChannelFactory func(s store.Store) Channel

// Register registers a channel factory
func Register(name string, factory ChannelFactory) {
	Registry[name] = factory
}

// Create creates a channel by name
func Create(name string, s store.Store) (Channel, error) {
	factory, ok := Registry[name]
	if !ok {
		return nil, errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("unknown output channel: %s", name))
	}
	return factory(s), nil
}

// CreateAll creates all specified channels
func CreateAll(names []string, s store.Store) ([]Channel, error) {
	channels := make([]Channel, 0, len(names))
	for _, name := range names {
		ch, err := Create(name, s)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

// Publisher publishes to multiple channels
type Publisher struct {
	channels []Channel
}

// NewPublisher creates a new Publisher
func NewPublisher(channels ...Channel) *Publisher {
	return &Publisher{
		channels: channels,
	}
}

// NewPublisherFromNames creates a Publisher from channel names
func NewPublisherFromNames(names []string, s store.Store) (*Publisher, error) {
	channels, err := CreateAll(names, s)
	if err != nil {
		return nil, err
	}
	return NewPublisher(channels...), nil
}

// Add adds a channel to the publisher
func (p *Publisher) Add(ch Channel) {
	p.channels = append(p.channels, ch)
}

// Publish publishes to all channels
func (p *Publisher) Publish(ctx context.Context, result *prompt.ReviewResult, opts *PublishOptions) error {
	var errs []error

	for _, ch := range p.channels {
		if err := ch.Publish(ctx, result, opts); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", ch.Name(), err))
		}
	}

	if len(errs) > 0 {
		// Return combined error
		return fmt.Errorf("publish errors: %v", errs)
	}

	return nil
}

// ChannelNames returns the names of all registered channels
func ChannelNames() []string {
	names := make([]string, 0, len(Registry))
	for name := range Registry {
		names = append(names, name)
	}
	return names
}

// CreateFromConfig creates a Channel from an OutputItemConfig
func CreateFromConfig(cfg *dsl.OutputItemConfig, s store.Store) (Channel, error) {
	// Get effective format (explicit or smart default)
	format := GetEffectiveFormat(cfg)

	switch cfg.Type {
	case "file":
		return NewFileChannelWithConfig(format, cfg.Dir, cfg.Overwrite), nil
	case "comment":
		return NewCommentChannelWithConfig(cfg.Overwrite, cfg.MarkerPrefix, format), nil
	case "webhook":
		return NewWebhookChannelWithConfig(cfg.URL, cfg.HeaderSecret, cfg.Timeout, cfg.MaxRetries, format, s), nil
	default:
		return nil, errors.New(errors.ErrCodeConfigInvalid,
			fmt.Sprintf("unknown output type: %s", cfg.Type))
	}
}

// CreateAllFromConfig creates all channels from an OutputConfig
func CreateAllFromConfig(cfg *dsl.OutputConfig, s store.Store) ([]Channel, error) {
	if cfg == nil || len(cfg.Channels) == 0 {
		return nil, errors.New(errors.ErrCodeConfigInvalid,
			"at least one output channel must be configured (file, comment, or webhook)")
	}

	channels := make([]Channel, 0, len(cfg.Channels))
	for _, item := range cfg.Channels {
		ch, err := CreateFromConfig(&item, s)
		if err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

// NewPublisherFromConfig creates a Publisher from OutputConfig
func NewPublisherFromConfig(cfg *dsl.OutputConfig, s store.Store) (*Publisher, error) {
	channels, err := CreateAllFromConfig(cfg, s)
	if err != nil {
		return nil, err
	}
	return NewPublisher(channels...), nil
}

func init() {
	// Register unified channels
	Register("file", func(s store.Store) Channel { return NewUnifiedFileChannel() })
	Register("comment", func(s store.Store) Channel { return NewCommentChannel() })
	Register("webhook", func(s store.Store) Channel { return NewWebhookChannel(s) })
}
