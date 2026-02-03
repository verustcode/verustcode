package output

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/internal/prompt"
)

// mockProvider is a mock implementation of provider.Provider
type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockProvider) GetBaseURL() string {
	args := m.Called()
	return args.String(0)
}

func (m *mockProvider) Clone(ctx context.Context, owner, repo, destPath string, opts *provider.CloneOptions) error {
	args := m.Called(ctx, owner, repo, destPath, opts)
	return args.Error(0)
}

func (m *mockProvider) ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *provider.CloneOptions) error {
	args := m.Called(ctx, owner, repo, prNumber, destPath, opts)
	return args.Error(0)
}

func (m *mockProvider) GetPRRef(prNumber int) string {
	args := m.Called(prNumber)
	return args.String(0)
}

func (m *mockProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*provider.PullRequest, error) {
	args := m.Called(ctx, owner, repo, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.PullRequest), args.Error(1)
}

func (m *mockProvider) ListPullRequests(ctx context.Context, owner, repo string) ([]*provider.PullRequest, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*provider.PullRequest), args.Error(1)
}

func (m *mockProvider) PostComment(ctx context.Context, owner, repo string, opts *provider.CommentOptions, body string) error {
	args := m.Called(ctx, owner, repo, opts, body)
	return args.Error(0)
}

func (m *mockProvider) ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*provider.Comment, error) {
	args := m.Called(ctx, owner, repo, prNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*provider.Comment), args.Error(1)
}

func (m *mockProvider) DeleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	args := m.Called(ctx, owner, repo, commentID)
	return args.Error(0)
}

func (m *mockProvider) UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error {
	args := m.Called(ctx, owner, repo, commentID, prNumber, body)
	return args.Error(0)
}

func (m *mockProvider) ParseWebhook(r *http.Request, secret string) (*provider.WebhookEvent, error) {
	args := m.Called(r, secret)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.WebhookEvent), args.Error(1)
}

func (m *mockProvider) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	args := m.Called(ctx, owner, repo, url, secret, events)
	return args.String(0), args.Error(1)
}

func (m *mockProvider) DeleteWebhook(ctx context.Context, owner, repo, webhookID string) error {
	args := m.Called(ctx, owner, repo, webhookID)
	return args.Error(0)
}

func (m *mockProvider) ValidateToken(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockProvider) ParseRepoPath(repoURL string) (owner, repo string, err error) {
	args := m.Called(repoURL)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockProvider) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	args := m.Called(ctx, owner, repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *mockProvider) MatchesURL(repoURL string) bool {
	args := m.Called(repoURL)
	return args.Bool(0)
}

func TestNewCommentChannel(t *testing.T) {
	channel := NewCommentChannel()

	assert.Equal(t, "comment", channel.Name())
	assert.False(t, channel.Overwrite)
	assert.Equal(t, DefaultCommentMarkerPrefix, channel.MarkerPrefix)
	assert.Equal(t, "markdown", channel.Format)
}

func TestNewCommentChannelWithConfig(t *testing.T) {
	tests := []struct {
		name         string
		overwrite    bool
		markerPrefix string
		format       string
		expected     *CommentChannel
	}{
		{
			name:         "full config",
			overwrite:    true,
			markerPrefix: "custom_marker",
			format:       "json",
			expected: &CommentChannel{
				Overwrite:    true,
				MarkerPrefix: "custom_marker",
				Format:       "json",
			},
		},
		{
			name:         "empty marker prefix uses default",
			overwrite:    false,
			markerPrefix: "",
			format:       "markdown",
			expected: &CommentChannel{
				Overwrite:    false,
				MarkerPrefix: DefaultCommentMarkerPrefix,
				Format:       "markdown",
			},
		},
		{
			name:         "empty format uses default",
			overwrite:    true,
			markerPrefix: "test",
			format:       "",
			expected: &CommentChannel{
				Overwrite:    true,
				MarkerPrefix: "test",
				Format:       "markdown",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := NewCommentChannelWithConfig(tt.overwrite, tt.markerPrefix, tt.format)
			assert.Equal(t, tt.expected.Overwrite, channel.Overwrite)
			assert.Equal(t, tt.expected.MarkerPrefix, channel.MarkerPrefix)
			assert.Equal(t, tt.expected.Format, channel.Format)
		})
	}
}

func TestCommentChannel_Publish_NoPRNumber(t *testing.T) {
	channel := NewCommentChannel()
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test"},
	}
	opts := &PublishOptions{}

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
}

func TestCommentChannel_Publish_NoProvider(t *testing.T) {
	channel := NewCommentChannel()
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test"},
	}
	opts := &PublishOptions{
		PRNumber: 123,
	}

	err := channel.Publish(context.Background(), result, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider not configured")
}

func TestCommentChannel_Publish_InvalidURL(t *testing.T) {
	channel := NewCommentChannel()
	mockProv := new(mockProvider)
	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test"},
	}
	opts := &PublishOptions{
		PRNumber: 123,
		RepoURL:  "invalid-url",
		Provider: mockProv,
	}

	err := channel.Publish(context.Background(), result, opts)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse repo URL")
}

func TestCommentChannel_Publish_Success(t *testing.T) {
	channel := NewCommentChannel()
	mockProv := new(mockProvider)
	mockProv.On("Name").Return("github")

	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{
		PRNumber: 123,
		RepoURL:  "https://github.com/test/repo",
		Provider: mockProv,
		MetadataConfig: &config.OutputMetadataConfig{
			ShowAgent: boolPtr(true),
			ShowModel: boolPtr(true),
		},
		AgentName: "cursor",
		ModelName: "gpt-4",
	}

	mockProv.On("PostComment", mock.Anything, "test", "repo", mock.AnythingOfType("*provider.CommentOptions"), mock.AnythingOfType("string")).Return(nil)

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
	mockProv.AssertExpectations(t)
}

func TestCommentChannel_Publish_OverwriteMode(t *testing.T) {
	channel := NewCommentChannelWithConfig(true, "test_marker", "markdown")
	mockProv := new(mockProvider)
	mockProv.On("Name").Return("github")

	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{
		PRNumber: 123,
		RepoURL:  "https://github.com/test/repo",
		Provider: mockProv,
	}

	existingComment := &provider.Comment{
		ID:   1,
		Body: "[test_marker:test-reviewer]",
	}

	mockProv.On("ListComments", mock.Anything, "test", "repo", 123).Return([]*provider.Comment{existingComment}, nil)
	mockProv.On("UpdateComment", mock.Anything, "test", "repo", int64(1), 123, mock.AnythingOfType("string")).Return(nil)

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
	mockProv.AssertExpectations(t)
}

func TestCommentChannel_Publish_OverwriteMode_NoExistingComment(t *testing.T) {
	channel := NewCommentChannelWithConfig(true, "test_marker", "markdown")
	mockProv := new(mockProvider)
	mockProv.On("Name").Return("github")

	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{
		PRNumber: 123,
		RepoURL:  "https://github.com/test/repo",
		Provider: mockProv,
	}

	mockProv.On("ListComments", mock.Anything, "test", "repo", 123).Return([]*provider.Comment{}, nil)
	mockProv.On("PostComment", mock.Anything, "test", "repo", mock.AnythingOfType("*provider.CommentOptions"), mock.AnythingOfType("string")).Return(nil)

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
	mockProv.AssertExpectations(t)
}

func TestCommentChannel_Publish_OverwriteMode_MultipleComments(t *testing.T) {
	channel := NewCommentChannelWithConfig(true, "test_marker", "markdown")
	mockProv := new(mockProvider)
	mockProv.On("Name").Return("github")

	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{
		PRNumber: 123,
		RepoURL:  "https://github.com/test/repo",
		Provider: mockProv,
	}

	existingComments := []*provider.Comment{
		{ID: 1, Body: "[test_marker:test-reviewer]"},
		{ID: 2, Body: "[test_marker:test-reviewer]"},
		{ID: 3, Body: "[test_marker:test-reviewer]"},
	}

	mockProv.On("ListComments", mock.Anything, "test", "repo", 123).Return(existingComments, nil)
	mockProv.On("UpdateComment", mock.Anything, "test", "repo", int64(1), 123, mock.AnythingOfType("string")).Return(nil)
	mockProv.On("DeleteComment", mock.Anything, "test", "repo", int64(2)).Return(nil)
	mockProv.On("DeleteComment", mock.Anything, "test", "repo", int64(3)).Return(nil)

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
	mockProv.AssertExpectations(t)
}

func TestCommentChannel_Publish_JSONFormat(t *testing.T) {
	channel := NewCommentChannelWithConfig(false, "test_marker", "json")
	mockProv := new(mockProvider)
	mockProv.On("Name").Return("github")

	result := &prompt.ReviewResult{
		ReviewerID: "test-reviewer",
		Data:       map[string]any{"summary": "Test summary"},
	}
	opts := &PublishOptions{
		PRNumber: 123,
		RepoURL:  "https://github.com/test/repo",
		Provider: mockProv,
	}

	mockProv.On("PostComment", mock.Anything, "test", "repo", mock.AnythingOfType("*provider.CommentOptions"), mock.MatchedBy(func(body string) bool {
		return strings.Contains(body, "[test_marker:test-reviewer]") && strings.Contains(body, "```json")
	})).Return(nil)

	err := channel.Publish(context.Background(), result, opts)
	assert.NoError(t, err)
	mockProv.AssertExpectations(t)
}

func TestParseRepoURL_HTTPS(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		owner    string
		repo     string
		hasError bool
	}{
		{
			name:     "standard HTTPS",
			url:      "https://github.com/test/repo",
			owner:    "test",
			repo:     "repo",
			hasError: false,
		},
		{
			name:     "HTTPS with .git",
			url:      "https://github.com/test/repo.git",
			owner:    "test",
			repo:     "repo",
			hasError: false,
		},
		{
			name:     "HTTP",
			url:      "http://gitlab.com/test/repo",
			owner:    "test",
			repo:     "repo",
			hasError: false,
		},
		{
			name:     "invalid HTTPS",
			url:      "https://github.com/test",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseRepoURL(tt.url)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.owner, owner)
				assert.Equal(t, tt.repo, repo)
			}
		})
	}
}

func TestParseRepoURL_SSH(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		owner    string
		repo     string
		hasError bool
	}{
		{
			name:     "standard SSH",
			url:      "git@github.com:test/repo.git",
			owner:    "test",
			repo:     "repo",
			hasError: false,
		},
		{
			name:     "SSH without .git",
			url:      "git@github.com:test/repo",
			owner:    "test",
			repo:     "repo",
			hasError: false,
		},
		{
			name:     "invalid SSH",
			url:      "git@github.com",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseRepoURL(tt.url)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.owner, owner)
				assert.Equal(t, tt.repo, repo)
			}
		})
	}
}

func TestParseRepoURL_Simple(t *testing.T) {
	owner, repo, err := parseRepoURL("test/repo")
	assert.NoError(t, err)
	assert.Equal(t, "test", owner)
	assert.Equal(t, "repo", repo)
}

func TestParseRepoURL_Invalid(t *testing.T) {
	_, _, err := parseRepoURL("invalid")
	assert.Error(t, err)
}
