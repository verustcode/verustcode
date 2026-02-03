// Package gitlab implements the Git provider interface for GitLab.
// It supports both GitLab.com (SaaS) and self-hosted GitLab instances.
// This implementation uses the official GitLab API client library.
package gitlab

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/pkg/logger"
)

// GitLab API pagination configuration
const defaultPerPage = 100

// Default GitLab SaaS URL
const defaultGitLabURL = "https://gitlab.com"

func init() {
	// Register GitLab provider factory
	provider.Register("gitlab", NewProvider)
}

// GitLabProvider implements the Provider interface for GitLab
type GitLabProvider struct {
	client             *gitlab.Client
	token              string
	baseURL            string
	insecureSkipVerify bool
}

// NewProvider creates a new GitLab provider instance
// Supports both GitLab.com and self-hosted GitLab instances with HTTP/HTTPS
func NewProvider(opts *provider.ProviderOptions) (provider.Provider, error) {
	token := opts.Token
	baseURL := opts.BaseURL

	// Normalize base URL
	if baseURL == "" {
		baseURL = defaultGitLabURL
	}

	// Build client options
	clientOpts := []gitlab.ClientOptionFunc{}

	// Set base URL for self-hosted instances
	if baseURL != defaultGitLabURL {
		clientOpts = append(clientOpts, gitlab.WithBaseURL(baseURL))
	}

	// Configure custom HTTP client for InsecureSkipVerify or custom transport
	if opts.InsecureSkipVerify {
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec // User explicitly enabled insecure mode
				},
			},
		}
		clientOpts = append(clientOpts, gitlab.WithHTTPClient(httpClient))
		logger.Warn("GitLab client configured with InsecureSkipVerify=true, SSL certificate verification is disabled")
	}

	// Create GitLab client with private token authentication
	client, err := gitlab.NewClient(token, clientOpts...)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to create gitlab client",
			Err:      err,
		}
	}

	logger.Info("GitLab provider initialized",
		zap.String("base_url", baseURL),
		zap.Bool("insecure_skip_verify", opts.InsecureSkipVerify),
	)

	return &GitLabProvider{
		client:             client,
		token:              token,
		baseURL:            baseURL,
		insecureSkipVerify: opts.InsecureSkipVerify,
	}, nil
}

// Name returns the provider name
func (p *GitLabProvider) Name() string {
	return "gitlab"
}

// GetBaseURL returns the base URL of the provider
func (p *GitLabProvider) GetBaseURL() string {
	if p.baseURL == "" {
		return defaultGitLabURL
	}
	return p.baseURL
}

// MatchesURL checks if the given repository URL matches GitLab
func (p *GitLabProvider) MatchesURL(repoURL string) bool {
	if repoURL == "" {
		return false
	}

	// Get base URL for comparison
	baseURL := p.GetBaseURL()

	// Normalize URL: remove .git suffix, protocol, trailing slash
	url := strings.TrimSuffix(repoURL, ".git")
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")
	url = strings.TrimSuffix(url, "/")

	// Handle git@ format (git@gitlab.com:group/subgroup/project)
	if strings.Contains(url, ":") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			url = parts[0] + "/" + parts[1]
		}
	}

	// Normalize base URL for comparison
	baseURLNormalized := strings.TrimPrefix(baseURL, "https://")
	baseURLNormalized = strings.TrimPrefix(baseURLNormalized, "http://")
	baseURLNormalized = strings.TrimSuffix(baseURLNormalized, "/")

	// Extract domain from URL (first part before /)
	urlParts := strings.Split(url, "/")
	if len(urlParts) == 0 {
		return false
	}
	urlDomain := urlParts[0]

	// Check if URL domain matches base URL domain
	// For public GitLab: check if domain contains "gitlab.com"
	// For self-hosted: check if domain matches configured base URL
	if baseURL == defaultGitLabURL || baseURL == "" {
		// Public GitLab: check for gitlab.com in domain
		return strings.Contains(urlDomain, "gitlab.com")
	}

	// Self-hosted: check if domain matches base URL domain
	baseParts := strings.Split(baseURLNormalized, "/")
	if len(baseParts) > 0 {
		baseDomain := baseParts[0]
		return urlDomain == baseDomain || strings.HasPrefix(url, baseURLNormalized)
	}

	return false
}

// Clone clones a repository using git command
func (p *GitLabProvider) Clone(ctx context.Context, owner, repo, destPath string, opts *provider.CloneOptions) error {
	logger.Info("Cloning repository",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("dest", destPath),
	)

	// Build clone URL with token
	// Supports both HTTP and HTTPS based on baseURL
	cloneURL := p.buildCloneURL(owner, repo)

	// Build git clone command
	args := []string{"clone"}
	if opts != nil {
		if opts.Depth > 0 {
			args = append(args, "--depth", fmt.Sprintf("%d", opts.Depth))
		}
		if opts.Branch != "" {
			args = append(args, "--branch", opts.Branch)
		}
	}
	args = append(args, cloneURL, destPath)

	// Configure git to skip SSL verification if needed
	cmd := exec.CommandContext(ctx, "git", args...)
	// Prevent interactive prompts for credentials
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	if p.insecureSkipVerify {
		cmd.Env = append(cmd.Environ(), "GIT_SSL_NO_VERIFY=true")
	}
	cmd.Stdout = io.Discard

	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		stderrOutput := stderrBuf.String()
		logger.Error("Failed to clone repository",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.String("stderr", stderrOutput),
		)

		errMsg := "failed to clone repository"
		if strings.Contains(stderrOutput, "Authentication failed") ||
			strings.Contains(stderrOutput, "could not read Username") {
			errMsg = "authentication failed: check your SC_GITLAB_TOKEN"
		} else if strings.Contains(stderrOutput, "Repository not found") ||
			strings.Contains(stderrOutput, "not found") {
			errMsg = "repository not found: check that the repository exists and your token has access"
		} else if strings.Contains(stderrOutput, "SSL certificate problem") {
			errMsg = "SSL certificate verification failed: consider setting insecure_skip_verify: true for self-signed certificates"
		}

		return &provider.ProviderError{
			Provider: "gitlab",
			Message:  errMsg,
			Err:      fmt.Errorf("%w: %s", err, stderrOutput),
		}
	}

	logger.Info("Repository cloned successfully",
		zap.String("owner", owner),
		zap.String("repo", repo),
	)
	return nil
}

// buildCloneURL builds the Git clone URL WITHOUT embedded credentials
// P1-2 Security improvement: Token is passed via credential helper, not in URL
func (p *GitLabProvider) buildCloneURL(owner, repo string) string {
	// Parse base URL to extract protocol and host
	baseURL := p.baseURL
	protocol := "https"
	host := "gitlab.com"

	if strings.HasPrefix(baseURL, "https://") {
		host = strings.TrimPrefix(baseURL, "https://")
		host = strings.TrimSuffix(host, "/")
	} else if strings.HasPrefix(baseURL, "http://") {
		protocol = "http"
		host = strings.TrimPrefix(baseURL, "http://")
		host = strings.TrimSuffix(host, "/")
	}

	// Remove any path components from host (e.g., /api/v4)
	if idx := strings.Index(host, "/"); idx > 0 {
		host = host[:idx]
	}

	// Return URL without token embedded (token passed via credential helper)
	return fmt.Sprintf("%s://%s/%s/%s.git", protocol, host, owner, repo)
}

// GetPRRef returns the Git ref for a MR number
// GitLab uses refs/merge-requests/{mr}/head format
// Note: Self-hosted GitLab requires 'merge_request_fetchable' setting enabled
func (p *GitLabProvider) GetPRRef(prNumber int) string {
	return fmt.Sprintf("refs/merge-requests/%d/head", prNumber)
}

// ClonePR clones a specific MR using refs/merge-requests/<mr>/head
// This method works for both fork and non-fork MRs
func (p *GitLabProvider) ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *provider.CloneOptions) error {
	logger.Info("Cloning MR using refs",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("mr_number", prNumber),
		zap.String("dest", destPath),
	)

	// Use helper function for the 4-step clone flow
	// P1-2 Security improvement: Pass token separately instead of embedding in URL
	err := provider.ClonePRWithRefs(ctx, &provider.ClonePRParams{
		ProviderName:       "gitlab",
		RepoURL:            p.buildCloneURL(owner, repo),
		Token:              p.token,
		PRRef:              p.GetPRRef(prNumber),
		PRNumber:           prNumber,
		DestPath:           destPath,
		InsecureSkipVerify: p.insecureSkipVerify,
	})
	if err != nil {
		logger.Error("Failed to clone MR",
			zap.Error(err),
			zap.Int("mr_number", prNumber),
		)
		return err
	}

	logger.Info("MR cloned successfully",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("mr_number", prNumber),
		zap.String("dest", destPath),
	)
	return nil
}

// projectPath returns the GitLab project path
func projectPath(owner, repo string) string {
	return owner + "/" + repo
}

// GetPullRequest retrieves merge request details
func (p *GitLabProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*provider.PullRequest, error) {
	// Official GitLab API uses int64 for MR IID
	mr, _, err := p.client.MergeRequests.GetMergeRequest(projectPath(owner, repo), int64(number), nil)
	if err != nil {
		logger.Error("Failed to get merge request",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("number", number),
		)
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to get merge request",
			Err:      err,
		}
	}

	// Get BaseSHA from DiffRefs if available
	// In official library, DiffRefs is embedded struct, not pointer
	baseSHA := ""
	if mr.DiffRefs.BaseSha != "" {
		baseSHA = mr.DiffRefs.BaseSha
	}

	// Get author username - in official library, Author is embedded struct
	authorUsername := mr.Author.Username

	return &provider.PullRequest{
		Number:      int(mr.IID),
		Title:       mr.Title,
		Description: mr.Description,
		State:       mr.State,
		HeadBranch:  mr.SourceBranch,
		HeadSHA:     mr.SHA,
		BaseBranch:  mr.TargetBranch,
		BaseSHA:     baseSHA,
		Author:      authorUsername,
		URL:         mr.WebURL,
	}, nil
}

// ListPullRequests lists open merge requests
func (p *GitLabProvider) ListPullRequests(ctx context.Context, owner, repo string) ([]*provider.PullRequest, error) {
	state := "opened"
	mrs, _, err := p.client.MergeRequests.ListProjectMergeRequests(projectPath(owner, repo), &gitlab.ListProjectMergeRequestsOptions{
		State: &state,
		ListOptions: gitlab.ListOptions{
			PerPage: defaultPerPage,
		},
	})
	if err != nil {
		logger.Error("Failed to list merge requests",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
		)
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to list merge requests",
			Err:      err,
		}
	}

	result := make([]*provider.PullRequest, len(mrs))
	for i, mr := range mrs {
		// BasicMergeRequest doesn't have DiffRefs, leave BaseSHA empty
		// It can be fetched later via GetPullRequest if needed
		baseSHA := ""

		// Get author username - in official library, Author is embedded struct
		authorUsername := mr.Author.Username

		result[i] = &provider.PullRequest{
			Number:      int(mr.IID),
			Title:       mr.Title,
			Description: mr.Description,
			State:       mr.State,
			HeadBranch:  mr.SourceBranch,
			HeadSHA:     mr.SHA,
			BaseBranch:  mr.TargetBranch,
			BaseSHA:     baseSHA,
			Author:      authorUsername,
			URL:         mr.WebURL,
		}
	}

	return result, nil
}

// PostComment posts a comment on a MR or commit
func (p *GitLabProvider) PostComment(ctx context.Context, owner, repo string, opts *provider.CommentOptions, body string) error {
	pid := projectPath(owner, repo)

	if opts.PRNumber > 0 {
		// Post MR comment (note) - official API uses int64
		_, _, err := p.client.Notes.CreateMergeRequestNote(pid, int64(opts.PRNumber), &gitlab.CreateMergeRequestNoteOptions{
			Body: &body,
		})
		if err != nil {
			logger.Error("Failed to post MR comment",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("repo", repo),
				zap.Int("mr", opts.PRNumber),
			)
			return &provider.ProviderError{
				Provider: "gitlab",
				Message:  "failed to post MR comment",
				Err:      err,
			}
		}
		logger.Info("MR comment posted successfully",
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("mr", opts.PRNumber),
		)
	} else if opts.CommitSHA != "" {
		// Post commit comment using discussions API
		_, _, err := p.client.Discussions.CreateCommitDiscussion(pid, opts.CommitSHA, &gitlab.CreateCommitDiscussionOptions{
			Body: &body,
		})
		if err != nil {
			logger.Error("Failed to post commit comment",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("repo", repo),
				zap.String("commit", opts.CommitSHA),
			)
			return &provider.ProviderError{
				Provider: "gitlab",
				Message:  "failed to post commit comment",
				Err:      err,
			}
		}
	}

	return nil
}

// ListComments lists comments (notes) on a MR
func (p *GitLabProvider) ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*provider.Comment, error) {
	pid := projectPath(owner, repo)

	// Official API uses int64 for MR IID
	notes, _, err := p.client.Notes.ListMergeRequestNotes(pid, int64(prNumber), &gitlab.ListMergeRequestNotesOptions{
		ListOptions: gitlab.ListOptions{PerPage: defaultPerPage},
	})
	if err != nil {
		logger.Error("Failed to list MR comments",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("mr", prNumber),
		)
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to list MR comments",
			Err:      err,
		}
	}

	result := make([]*provider.Comment, 0, len(notes))
	for _, note := range notes {
		// Skip system notes (e.g., "merged branch X into Y")
		if note.System {
			continue
		}

		// Get author username - in official library, Author is embedded struct
		authorUsername := note.Author.Username

		createdAt := ""
		if note.CreatedAt != nil {
			createdAt = note.CreatedAt.Format("2006-01-02T15:04:05Z")
		}

		result = append(result, &provider.Comment{
			ID:        note.ID,
			Body:      note.Body,
			Author:    authorUsername,
			CreatedAt: createdAt,
		})
	}

	return result, nil
}

// DeleteComment deletes a comment (note) by ID
func (p *GitLabProvider) DeleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	pid := projectPath(owner, repo)

	// GitLab Notes API requires the MR IID to delete a note
	// We need to iterate through MRs to find the note containing this comment ID

	// List all MRs (both open and closed) and check each one
	state := "all"
	mrs, _, err := p.client.MergeRequests.ListProjectMergeRequests(pid, &gitlab.ListProjectMergeRequestsOptions{
		State: &state,
		ListOptions: gitlab.ListOptions{
			PerPage: defaultPerPage,
		},
	})
	if err != nil {
		logger.Error("Failed to list MRs for comment deletion",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
		)
		return &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to list MRs for comment deletion",
			Err:      err,
		}
	}

	for _, mr := range mrs {
		// Official API uses int64 for MR IID
		notes, _, err := p.client.Notes.ListMergeRequestNotes(pid, mr.IID, &gitlab.ListMergeRequestNotesOptions{
			ListOptions: gitlab.ListOptions{PerPage: defaultPerPage},
		})
		if err != nil {
			continue
		}

		for _, note := range notes {
			if note.ID == commentID {
				// Found the note, delete it
				_, err := p.client.Notes.DeleteMergeRequestNote(pid, mr.IID, note.ID)
				if err != nil {
					logger.Error("Failed to delete MR comment",
						zap.Error(err),
						zap.String("owner", owner),
						zap.String("repo", repo),
						zap.Int64("comment_id", commentID),
					)
					return &provider.ProviderError{
						Provider: "gitlab",
						Message:  "failed to delete MR comment",
						Err:      err,
					}
				}

				logger.Info("Deleted MR comment",
					zap.String("owner", owner),
					zap.String("repo", repo),
					zap.Int64("comment_id", commentID),
				)
				return nil
			}
		}
	}

	return &provider.ProviderError{
		Provider: "gitlab",
		Message:  "comment not found",
	}
}

// UpdateComment updates an existing comment (note) by ID
// prNumber is required for GitLab as notes are scoped to MRs
func (p *GitLabProvider) UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error {
	pid := projectPath(owner, repo)

	// Update the MR note directly using the provided prNumber
	_, _, err := p.client.Notes.UpdateMergeRequestNote(pid, int64(prNumber), commentID, &gitlab.UpdateMergeRequestNoteOptions{
		Body: &body,
	})
	if err != nil {
		logger.Error("Failed to update MR comment",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int64("comment_id", commentID),
			zap.Int("mr", prNumber),
		)
		return &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to update MR comment",
			Err:      err,
		}
	}

	logger.Info("Updated MR comment",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int64("comment_id", commentID),
		zap.Int("mr", prNumber),
	)
	return nil
}

// ParseWebhook parses an incoming webhook request
func (p *GitLabProvider) ParseWebhook(r *http.Request, secret string) (*provider.WebhookEvent, error) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to read webhook body",
			Err:      err,
		}
	}

	// Validate token if secret is provided
	// GitLab uses X-Gitlab-Token header for webhook authentication
	if secret != "" {
		token := r.Header.Get("X-Gitlab-Token")
		if token != secret {
			logger.Warn("Invalid webhook token received",
				zap.String("expected_length", fmt.Sprintf("%d", len(secret))),
				zap.String("received_length", fmt.Sprintf("%d", len(token))),
			)
			return nil, &provider.ProviderError{
				Provider: "gitlab",
				Message:  "invalid webhook token",
			}
		}
	}

	// Parse event type from header
	eventType := r.Header.Get("X-Gitlab-Event")

	// If header is empty, try to infer from request body's object_kind field
	if eventType == "" {
		var fallbackPayload struct {
			ObjectKind string `json:"object_kind"`
		}
		if err := json.Unmarshal(body, &fallbackPayload); err == nil {
			// Map object_kind to X-Gitlab-Event format
			switch fallbackPayload.ObjectKind {
			case "merge_request":
				eventType = "Merge Request Hook"
			case "push":
				eventType = "Push Hook"
			case "tag_push":
				eventType = "Tag Push Hook"
			}
		}
	}

	event := &provider.WebhookEvent{
		Provider:   "gitlab",
		RawPayload: body,
	}

	switch eventType {
	case "Push Hook":
		return p.parsePushEvent(body, event)
	case "Merge Request Hook":
		return p.parseMergeRequestEvent(body, event)
	default:
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  fmt.Sprintf("unsupported event type: %s", eventType),
		}
	}
}

// parsePushEvent parses a push webhook event
func (p *GitLabProvider) parsePushEvent(body []byte, event *provider.WebhookEvent) (*provider.WebhookEvent, error) {
	var payload struct {
		Ref      string `json:"ref"`
		After    string `json:"after"`
		UserName string `json:"user_name"`
		Project  struct {
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
		} `json:"project"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to parse push event",
			Err:      err,
		}
	}

	parts := strings.SplitN(payload.Project.PathWithNamespace, "/", 2)
	if len(parts) != 2 {
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "invalid project path",
		}
	}

	event.Type = provider.EventTypePush
	event.Owner = parts[0]
	event.Repo = parts[1]
	event.Ref = strings.TrimPrefix(payload.Ref, "refs/heads/")
	event.CommitSHA = payload.After
	event.Sender = payload.UserName

	return event, nil
}

// parseMergeRequestEvent parses a merge request webhook event
func (p *GitLabProvider) parseMergeRequestEvent(body []byte, event *provider.WebhookEvent) (*provider.WebhookEvent, error) {
	var payload struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
		} `json:"project"`
		ObjectAttributes struct {
			IID          int    `json:"iid"`
			Title        string `json:"title"`
			Description  string `json:"description"`
			SourceBranch string `json:"source_branch"`
			TargetBranch string `json:"target_branch"`
			LastCommit   struct {
				ID string `json:"id"`
			} `json:"last_commit"`
			Action   string `json:"action"`
			DiffRefs struct {
				BaseSha string `json:"base_sha"`
				HeadSha string `json:"head_sha"`
			} `json:"diff_refs"`
		} `json:"object_attributes"`
		Changes struct {
			// GitLab can include file changes in webhook
			// We capture them if available
		} `json:"changes"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to parse merge request event",
			Err:      err,
		}
	}

	parts := strings.SplitN(payload.Project.PathWithNamespace, "/", 2)
	if len(parts) != 2 {
		return nil, &provider.ProviderError{
			Provider: "gitlab",
			Message:  "invalid project path",
		}
	}

	attrs := payload.ObjectAttributes
	event.Type = provider.EventTypeMergeRequest
	event.Owner = parts[0]
	event.Repo = parts[1]
	event.Ref = attrs.SourceBranch
	event.CommitSHA = attrs.LastCommit.ID
	event.PRNumber = attrs.IID
	// Normalize GitLab action to unified format
	event.Action = normalizeGitLabAction(attrs.Action)
	event.Sender = payload.User.Username

	// Extract complete MR information
	event.PRTitle = attrs.Title
	event.PRDescription = attrs.Description
	if attrs.DiffRefs.BaseSha != "" {
		event.BaseCommitSHA = attrs.DiffRefs.BaseSha
	}
	// GitLab webhook doesn't include full changed_files list
	event.ChangedFiles = []string{}

	logger.Info("Parsed GitLab merge request webhook",
		zap.String("action", event.Action),
		zap.String("original_action", attrs.Action),
		zap.String("owner", event.Owner),
		zap.String("repo", event.Repo),
		zap.Int("mr_number", event.PRNumber),
		zap.String("mr_title", event.PRTitle),
		zap.String("base_sha", event.BaseCommitSHA),
	)

	return event, nil
}

// CreateWebhook creates a webhook for the project
func (p *GitLabProvider) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	pid := projectPath(owner, repo)

	// Build hook options
	opts := &gitlab.AddProjectHookOptions{
		URL:   &url,
		Token: &secret,
	}

	// Map event names to GitLab webhook options
	for _, event := range events {
		switch event {
		case "push":
			opts.PushEvents = gitlab.Ptr(true)
		case "merge_request":
			opts.MergeRequestsEvents = gitlab.Ptr(true)
		case "note":
			opts.NoteEvents = gitlab.Ptr(true)
		case "tag_push":
			opts.TagPushEvents = gitlab.Ptr(true)
		case "pipeline":
			opts.PipelineEvents = gitlab.Ptr(true)
		}
	}

	hook, _, err := p.client.Projects.AddProjectHook(pid, opts)
	if err != nil {
		logger.Error("Failed to create webhook",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
		)
		return "", &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to create webhook",
			Err:      err,
		}
	}

	logger.Info("Webhook created successfully",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int64("hook_id", hook.ID),
	)

	return fmt.Sprintf("%d", hook.ID), nil
}

// DeleteWebhook deletes a webhook from the project
func (p *GitLabProvider) DeleteWebhook(ctx context.Context, owner, repo, webhookID string) error {
	pid := projectPath(owner, repo)
	var id int64
	fmt.Sscanf(webhookID, "%d", &id)

	_, err := p.client.Projects.DeleteProjectHook(pid, id)
	if err != nil {
		logger.Error("Failed to delete webhook",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.String("webhook_id", webhookID),
		)
		return &provider.ProviderError{
			Provider: "gitlab",
			Message:  "failed to delete webhook",
			Err:      err,
		}
	}

	logger.Info("Webhook deleted successfully",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("webhook_id", webhookID),
	)

	return nil
}

// ValidateToken validates the GitLab token by fetching current user
func (p *GitLabProvider) ValidateToken(ctx context.Context) error {
	user, _, err := p.client.Users.CurrentUser()
	if err != nil {
		return &provider.ProviderError{
			Provider: "gitlab",
			Message:  "invalid token",
			Err:      err,
		}
	}

	logger.Info("GitLab token validated successfully",
		zap.String("username", user.Username),
	)
	return nil
}

// ListBranches lists all branches for a repository
func (p *GitLabProvider) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	pid := projectPath(owner, repo)
	allBranches := []string{}
	page := int64(1)

	for {
		branches, _, err := p.client.Branches.ListBranches(pid, &gitlab.ListBranchesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: defaultPerPage,
			},
		})
		if err != nil {
			logger.Error("Failed to list branches",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("repo", repo),
			)
			return nil, &provider.ProviderError{
				Provider: "gitlab",
				Message:  "failed to list branches",
				Err:      err,
			}
		}

		for _, branch := range branches {
			if branch.Name != "" {
				allBranches = append(allBranches, branch.Name)
			}
		}

		// Check if there are more pages
		if len(branches) < defaultPerPage {
			break
		}
		page++
	}

	logger.Info("Listed branches",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("count", len(allBranches)),
	)

	return allBranches, nil
}

// normalizeGitLabAction maps GitLab MR action names to unified format
// GitLab uses: open, update, reopen, close, merge
// Unified format: opened, synchronize, reopened, closed, merged
func normalizeGitLabAction(action string) string {
	switch strings.ToLower(action) {
	case "open":
		return provider.PREventActionOpened
	case "update":
		return provider.PREventActionSynchronize
	case "reopen":
		return provider.PREventActionReopened
	case "close", "closed":
		return "closed"
	case "merge", "merged":
		return "merged"
	default:
		// Return lowercase version of unknown actions
		return strings.ToLower(action)
	}
}

// ParseRepoPath parses owner and repo from a repository URL.
// GitLab supports multi-level namespaces: group/subgroup/.../project
// The first path segment is treated as owner, and the rest as repo.
// Supported formats:
//   - https://gitlab.com/group/subgroup/project
//   - https://gitlab.com/group/subgroup/project.git
//   - gitlab.com/group/subgroup/project
//   - group/subgroup/project
func (p *GitLabProvider) ParseRepoPath(repoURL string) (owner, repo string, err error) {
	if repoURL == "" {
		return "", "", &provider.ProviderError{
			Provider: "gitlab",
			Message:  "empty repository URL",
		}
	}

	// Remove .git suffix if present
	url := strings.TrimSuffix(repoURL, ".git")

	// Remove protocol prefix
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "git@")

	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Handle git@ format (git@gitlab.com:group/subgroup/project)
	if strings.Contains(url, ":") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) == 2 {
			url = parts[1]
		}
	}

	// Split by /
	parts := strings.Split(url, "/")

	// Remove empty parts
	var cleanParts []string
	for _, part := range parts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	// Check if first part looks like a domain (contains a dot)
	hasDomain := len(cleanParts) > 0 && strings.Contains(cleanParts[0], ".")

	var pathParts []string
	if hasDomain {
		// Skip the domain part
		if len(cleanParts) > 1 {
			pathParts = cleanParts[1:]
		}
	} else {
		pathParts = cleanParts
	}

	// GitLab supports multi-level namespaces
	// For group/subgroup/project format:
	// - owner is all groups (group/subgroup)
	// - repo is the project name
	if len(pathParts) < 2 {
		return "", "", &provider.ProviderError{
			Provider: "gitlab",
			Message:  fmt.Sprintf("invalid repository URL format: %s", repoURL),
		}
	}

	// Last segment is the repo (project name)
	repo = pathParts[len(pathParts)-1]
	// All segments except the last are the owner (can include subgroups)
	if len(pathParts) > 1 {
		owner = strings.Join(pathParts[:len(pathParts)-1], "/")
	} else {
		owner = pathParts[0]
	}

	return owner, repo, nil
}
