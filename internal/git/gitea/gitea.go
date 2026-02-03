// Package gitea implements the Git provider interface for Gitea.
// It supports both Gitea.com (cloud hosting) and self-hosted Gitea instances.
// This implementation uses the official Gitea Go SDK.
package gitea

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"code.gitea.io/sdk/gitea"
	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/pkg/logger"
)

// Gitea API pagination configuration
const defaultPerPage = 100

// Default Gitea cloud URL
const defaultGiteaURL = "https://gitea.com"

func init() {
	// Register Gitea provider factory
	provider.Register("gitea", NewProvider)
}

// GiteaProvider implements the Provider interface for Gitea
type GiteaProvider struct {
	client             *gitea.Client
	token              string
	baseURL            string
	insecureSkipVerify bool
}

// NewProvider creates a new Gitea provider instance
// Supports both Gitea.com and self-hosted Gitea instances with HTTP/HTTPS
func NewProvider(opts *provider.ProviderOptions) (provider.Provider, error) {
	token := opts.Token
	baseURL := opts.BaseURL

	// Normalize base URL
	if baseURL == "" {
		baseURL = defaultGiteaURL
	}

	// Build client options
	clientOpts := []gitea.ClientOption{
		gitea.SetToken(token),
	}

	// Configure custom HTTP client for InsecureSkipVerify
	if opts.InsecureSkipVerify {
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec // User explicitly enabled insecure mode
				},
			},
		}
		clientOpts = append(clientOpts, gitea.SetHTTPClient(httpClient))
		logger.Warn("Gitea client configured with InsecureSkipVerify=true, SSL certificate verification is disabled")
	}

	// Create Gitea client
	client, err := gitea.NewClient(baseURL, clientOpts...)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to create gitea client",
			Err:      err,
		}
	}

	logger.Info("Gitea provider initialized",
		zap.String("base_url", baseURL),
		zap.Bool("insecure_skip_verify", opts.InsecureSkipVerify),
	)

	return &GiteaProvider{
		client:             client,
		token:              token,
		baseURL:            baseURL,
		insecureSkipVerify: opts.InsecureSkipVerify,
	}, nil
}

// Name returns the provider name
func (p *GiteaProvider) Name() string {
	return "gitea"
}

// GetBaseURL returns the base URL of the provider
func (p *GiteaProvider) GetBaseURL() string {
	if p.baseURL == "" {
		return defaultGiteaURL
	}
	return p.baseURL
}

// MatchesURL checks if the given repository URL matches Gitea
func (p *GiteaProvider) MatchesURL(repoURL string) bool {
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

	// Handle git@ format (git@gitea.com:owner/repo)
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
	// For public Gitea: check if domain contains "gitea.com"
	// For self-hosted: check if domain matches configured base URL
	if baseURL == defaultGiteaURL || baseURL == "" {
		// Public Gitea: check for gitea.com in domain
		return strings.Contains(urlDomain, "gitea.com")
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
func (p *GiteaProvider) Clone(ctx context.Context, owner, repo, destPath string, opts *provider.CloneOptions) error {
	logger.Info("Cloning repository",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("dest", destPath),
	)

	// Build clone URL with token
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
			errMsg = "authentication failed: check your SC_GITEA_TOKEN"
		} else if strings.Contains(stderrOutput, "Repository not found") ||
			strings.Contains(stderrOutput, "not found") {
			errMsg = "repository not found: check that the repository exists and your token has access"
		} else if strings.Contains(stderrOutput, "SSL certificate problem") {
			errMsg = "SSL certificate verification failed: consider setting insecure_skip_verify: true for self-signed certificates"
		}

		return &provider.ProviderError{
			Provider: "gitea",
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
func (p *GiteaProvider) buildCloneURL(owner, repo string) string {
	// Parse base URL to extract protocol and host
	baseURL := p.baseURL
	protocol := "https"
	host := "gitea.com"

	if strings.HasPrefix(baseURL, "https://") {
		host = strings.TrimPrefix(baseURL, "https://")
		host = strings.TrimSuffix(host, "/")
	} else if strings.HasPrefix(baseURL, "http://") {
		protocol = "http"
		host = strings.TrimPrefix(baseURL, "http://")
		host = strings.TrimSuffix(host, "/")
	}

	// Remove any path components from host (e.g., /api/v1)
	if idx := strings.Index(host, "/"); idx > 0 {
		host = host[:idx]
	}

	// Return URL without token embedded (token passed via credential helper)
	return fmt.Sprintf("%s://%s/%s/%s.git", protocol, host, owner, repo)
}

// GetPRRef returns the Git ref for a PR number
// Gitea uses refs/pull/{pr}/head format (similar to GitHub)
func (p *GiteaProvider) GetPRRef(prNumber int) string {
	return fmt.Sprintf("refs/pull/%d/head", prNumber)
}

// ClonePR clones a specific PR using refs/pull/<pr>/head
// This method works for both fork and non-fork PRs
func (p *GiteaProvider) ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *provider.CloneOptions) error {
	logger.Info("Cloning PR using refs",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("pr_number", prNumber),
		zap.String("dest", destPath),
	)

	// Use helper function for the 4-step clone flow
	// P1-2 Security improvement: Pass token separately instead of embedding in URL
	err := provider.ClonePRWithRefs(ctx, &provider.ClonePRParams{
		ProviderName:       "gitea",
		RepoURL:            p.buildCloneURL(owner, repo),
		Token:              p.token,
		PRRef:              p.GetPRRef(prNumber),
		PRNumber:           prNumber,
		DestPath:           destPath,
		InsecureSkipVerify: p.insecureSkipVerify,
	})
	if err != nil {
		logger.Error("Failed to clone PR",
			zap.Error(err),
			zap.Int("pr_number", prNumber),
		)
		return err
	}

	logger.Info("PR cloned successfully",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("pr_number", prNumber),
		zap.String("dest", destPath),
	)
	return nil
}

// GetPullRequest retrieves pull request details
func (p *GiteaProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*provider.PullRequest, error) {
	pr, _, err := p.client.GetPullRequest(owner, repo, int64(number))
	if err != nil {
		logger.Error("Failed to get pull request",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("number", number),
		)
		return nil, &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to get pull request",
			Err:      err,
		}
	}

	// Get BaseSHA from MergeBase if available
	baseSHA := ""
	if pr.MergeBase != "" {
		baseSHA = pr.MergeBase
	}

	// Get author username
	authorUsername := ""
	if pr.Poster != nil {
		authorUsername = pr.Poster.UserName
	}

	return &provider.PullRequest{
		Number:      int(pr.Index),
		Title:       pr.Title,
		Description: pr.Body,
		State:       string(pr.State),
		HeadBranch:  pr.Head.Ref,
		HeadSHA:     pr.Head.Sha,
		BaseBranch:  pr.Base.Ref,
		BaseSHA:     baseSHA,
		Author:      authorUsername,
		URL:         pr.HTMLURL,
	}, nil
}

// ListPullRequests lists open pull requests
func (p *GiteaProvider) ListPullRequests(ctx context.Context, owner, repo string) ([]*provider.PullRequest, error) {
	state := gitea.StateOpen
	prs, _, err := p.client.ListRepoPullRequests(owner, repo, gitea.ListPullRequestsOptions{
		State: state,
		ListOptions: gitea.ListOptions{
			PageSize: defaultPerPage,
		},
	})
	if err != nil {
		logger.Error("Failed to list pull requests",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
		)
		return nil, &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to list pull requests",
			Err:      err,
		}
	}

	result := make([]*provider.PullRequest, len(prs))
	for i, pr := range prs {
		baseSHA := ""
		if pr.MergeBase != "" {
			baseSHA = pr.MergeBase
		}

		authorUsername := ""
		if pr.Poster != nil {
			authorUsername = pr.Poster.UserName
		}

		result[i] = &provider.PullRequest{
			Number:      int(pr.Index),
			Title:       pr.Title,
			Description: pr.Body,
			State:       string(pr.State),
			HeadBranch:  pr.Head.Ref,
			HeadSHA:     pr.Head.Sha,
			BaseBranch:  pr.Base.Ref,
			BaseSHA:     baseSHA,
			Author:      authorUsername,
			URL:         pr.HTMLURL,
		}
	}

	return result, nil
}

// PostComment posts a comment on a PR or commit
func (p *GiteaProvider) PostComment(ctx context.Context, owner, repo string, opts *provider.CommentOptions, body string) error {
	if opts.PRNumber > 0 {
		// Post PR comment (issue comment)
		_, _, err := p.client.CreateIssueComment(owner, repo, int64(opts.PRNumber), gitea.CreateIssueCommentOption{
			Body: body,
		})
		if err != nil {
			logger.Error("Failed to post PR comment",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("repo", repo),
				zap.Int("pr", opts.PRNumber),
			)
			return &provider.ProviderError{
				Provider: "gitea",
				Message:  "failed to post PR comment",
				Err:      err,
			}
		}
		logger.Info("PR comment posted successfully",
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("pr", opts.PRNumber),
		)
	} else if opts.CommitSHA != "" {
		// Post commit comment
		// Note: Gitea SDK does not have direct support for commit comments
		// We use the repository commit status API as a workaround or skip
		logger.Warn("Commit comments are not fully supported in Gitea SDK",
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.String("commit", opts.CommitSHA),
		)
		return &provider.ProviderError{
			Provider: "gitea",
			Message:  "commit comments are not supported via Gitea SDK",
		}
	}

	return nil
}

// ListComments lists comments on a PR
func (p *GiteaProvider) ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*provider.Comment, error) {
	comments, _, err := p.client.ListIssueComments(owner, repo, int64(prNumber), gitea.ListIssueCommentOptions{
		ListOptions: gitea.ListOptions{PageSize: defaultPerPage},
	})
	if err != nil {
		logger.Error("Failed to list PR comments",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("pr", prNumber),
		)
		return nil, &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to list PR comments",
			Err:      err,
		}
	}

	result := make([]*provider.Comment, len(comments))
	for i, c := range comments {
		authorUsername := ""
		if c.Poster != nil {
			authorUsername = c.Poster.UserName
		}

		createdAt := ""
		if !c.Created.IsZero() {
			createdAt = c.Created.Format("2006-01-02T15:04:05Z")
		}

		result[i] = &provider.Comment{
			ID:        c.ID,
			Body:      c.Body,
			Author:    authorUsername,
			CreatedAt: createdAt,
		}
	}

	return result, nil
}

// DeleteComment deletes a comment by ID
func (p *GiteaProvider) DeleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	_, err := p.client.DeleteIssueComment(owner, repo, commentID)
	if err != nil {
		logger.Error("Failed to delete comment",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int64("comment_id", commentID),
		)
		return &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to delete comment",
			Err:      err,
		}
	}

	logger.Info("Deleted comment",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int64("comment_id", commentID),
	)
	return nil
}

// UpdateComment updates an existing comment by ID
// prNumber is not required for Gitea as comments are identified by ID globally
func (p *GiteaProvider) UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error {
	_, _, err := p.client.EditIssueComment(owner, repo, commentID, gitea.EditIssueCommentOption{
		Body: body,
	})
	if err != nil {
		logger.Error("Failed to update comment",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int64("comment_id", commentID),
		)
		return &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to update comment",
			Err:      err,
		}
	}

	logger.Info("Updated comment",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int64("comment_id", commentID),
	)
	return nil
}

// ParseWebhook parses an incoming webhook request
func (p *GiteaProvider) ParseWebhook(r *http.Request, secret string) (*provider.WebhookEvent, error) {
	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to read webhook body",
			Err:      err,
		}
	}

	// Validate signature if secret is provided
	// Gitea uses X-Gitea-Signature header with HMAC-SHA256
	if secret != "" {
		signature := r.Header.Get("X-Gitea-Signature")
		if signature == "" {
			return nil, &provider.ProviderError{
				Provider: "gitea",
				Message:  "missing webhook signature header (X-Gitea-Signature)",
			}
		}

		// Calculate expected signature
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
			logger.Warn("Invalid webhook signature received",
				zap.String("expected_length", fmt.Sprintf("%d", len(expectedSig))),
				zap.String("received_length", fmt.Sprintf("%d", len(signature))),
			)
			return nil, &provider.ProviderError{
				Provider: "gitea",
				Message:  "invalid webhook signature",
			}
		}
	}

	// Parse event type from header
	eventType := r.Header.Get("X-Gitea-Event")

	event := &provider.WebhookEvent{
		Provider:   "gitea",
		RawPayload: body,
	}

	switch eventType {
	case "push":
		return p.parsePushEvent(body, event)
	case "pull_request":
		return p.parsePullRequestEvent(body, event)
	default:
		return nil, &provider.ProviderError{
			Provider: "gitea",
			Message:  fmt.Sprintf("unsupported event type: %s", eventType),
		}
	}
}

// parsePushEvent parses a push webhook event
func (p *GiteaProvider) parsePushEvent(body []byte, event *provider.WebhookEvent) (*provider.WebhookEvent, error) {
	var payload struct {
		Ref    string `json:"ref"`
		After  string `json:"after"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
		Repository struct {
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
			Name string `json:"name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to parse push event",
			Err:      err,
		}
	}

	event.Type = provider.EventTypePush
	event.Owner = payload.Repository.Owner.Login
	event.Repo = payload.Repository.Name
	event.Ref = strings.TrimPrefix(payload.Ref, "refs/heads/")
	event.CommitSHA = payload.After
	event.Sender = payload.Sender.Login

	return event, nil
}

// parsePullRequestEvent parses a pull_request webhook event
func (p *GiteaProvider) parsePullRequestEvent(body []byte, event *provider.WebhookEvent) (*provider.WebhookEvent, error) {
	var payload struct {
		Action      string `json:"action"`
		Number      int64  `json:"number"`
		PullRequest struct {
			ID        int64  `json:"id"`
			Number    int64  `json:"number"`
			Title     string `json:"title"`
			Body      string `json:"body"`
			MergeBase string `json:"merge_base"`
			Head      struct {
				Ref string `json:"ref"`
				Sha string `json:"sha"`
			} `json:"head"`
			Base struct {
				Ref string `json:"ref"`
				Sha string `json:"sha"`
			} `json:"base"`
		} `json:"pull_request"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
		Repository struct {
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
			Name string `json:"name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, &provider.ProviderError{
			Provider: "gitea",
			Message:  "failed to parse pull_request event",
			Err:      err,
		}
	}

	pr := payload.PullRequest
	event.Type = provider.EventTypePullRequest
	event.Owner = payload.Repository.Owner.Login
	event.Repo = payload.Repository.Name
	event.Ref = pr.Head.Ref
	event.CommitSHA = pr.Head.Sha
	event.PRNumber = int(payload.Number)
	// Normalize Gitea action to unified format
	event.Action = normalizeGiteaAction(payload.Action)
	event.Sender = payload.Sender.Login

	// Extract complete PR information
	event.PRTitle = pr.Title
	event.PRDescription = pr.Body
	if pr.MergeBase != "" {
		event.BaseCommitSHA = pr.MergeBase
	} else if pr.Base.Sha != "" {
		event.BaseCommitSHA = pr.Base.Sha
	}
	// Gitea webhook doesn't include full changed_files list
	event.ChangedFiles = []string{}

	logger.Info("Parsed Gitea pull_request webhook",
		zap.String("action", event.Action),
		zap.String("original_action", payload.Action),
		zap.String("owner", event.Owner),
		zap.String("repo", event.Repo),
		zap.Int("pr_number", event.PRNumber),
		zap.String("pr_title", event.PRTitle),
		zap.String("base_sha", event.BaseCommitSHA),
	)

	return event, nil
}

// CreateWebhook creates a webhook for the repository
func (p *GiteaProvider) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	// Map event names to Gitea webhook event types
	giteaEvents := make([]string, 0, len(events))
	for _, event := range events {
		switch event {
		case "push":
			giteaEvents = append(giteaEvents, "push")
		case "pull_request":
			giteaEvents = append(giteaEvents, "pull_request")
		case "create":
			giteaEvents = append(giteaEvents, "create")
		case "delete":
			giteaEvents = append(giteaEvents, "delete")
		case "issue_comment":
			giteaEvents = append(giteaEvents, "issue_comment")
		}
	}

	hook, _, err := p.client.CreateRepoHook(owner, repo, gitea.CreateHookOption{
		Type: gitea.HookTypeGitea,
		Config: map[string]string{
			"url":          url,
			"content_type": "json",
			"secret":       secret,
		},
		Events: giteaEvents,
		Active: true,
	})
	if err != nil {
		logger.Error("Failed to create webhook",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
		)
		return "", &provider.ProviderError{
			Provider: "gitea",
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

// DeleteWebhook deletes a webhook from the repository
func (p *GiteaProvider) DeleteWebhook(ctx context.Context, owner, repo, webhookID string) error {
	var id int64
	fmt.Sscanf(webhookID, "%d", &id)

	_, err := p.client.DeleteRepoHook(owner, repo, id)
	if err != nil {
		logger.Error("Failed to delete webhook",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.String("webhook_id", webhookID),
		)
		return &provider.ProviderError{
			Provider: "gitea",
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

// ValidateToken validates the Gitea token by fetching current user
func (p *GiteaProvider) ValidateToken(ctx context.Context) error {
	user, _, err := p.client.GetMyUserInfo()
	if err != nil {
		return &provider.ProviderError{
			Provider: "gitea",
			Message:  "invalid token",
			Err:      err,
		}
	}

	logger.Info("Gitea token validated successfully",
		zap.String("username", user.UserName),
	)
	return nil
}

// normalizeGiteaAction maps Gitea PR action names to unified format
// Gitea uses: opened, synchronized, closed, reopened, edited, etc.
// These are mostly GitHub-compatible but we normalize for consistency
func normalizeGiteaAction(action string) string {
	switch strings.ToLower(action) {
	case "opened":
		return provider.PREventActionOpened
	case "synchronized", "synchronize":
		return provider.PREventActionSynchronize
	case "reopened":
		return provider.PREventActionReopened
	case "closed":
		return "closed"
	case "merged":
		return "merged"
	case "edited":
		return "edited"
	default:
		// Return lowercase version of unknown actions
		return strings.ToLower(action)
	}
}

// ListBranches lists all branches for a repository
func (p *GiteaProvider) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	allBranches := []string{}
	page := 1

	for {
		branches, _, err := p.client.ListRepoBranches(owner, repo, gitea.ListRepoBranchesOptions{
			ListOptions: gitea.ListOptions{
				Page:     page,
				PageSize: defaultPerPage,
			},
		})
		if err != nil {
			logger.Error("Failed to list branches",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("repo", repo),
			)
			return nil, &provider.ProviderError{
				Provider: "gitea",
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

// ParseRepoPath parses owner and repo from a repository URL.
// Gitea uses a two-level structure: owner/repo (similar to GitHub)
// Supported formats:
//   - https://gitea.com/owner/repo
//   - https://gitea.com/owner/repo.git
//   - gitea.com/owner/repo
//   - owner/repo
func (p *GiteaProvider) ParseRepoPath(repoURL string) (owner, repo string, err error) {
	if repoURL == "" {
		return "", "", &provider.ProviderError{
			Provider: "gitea",
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

	// Handle git@ format (git@gitea.com:owner/repo)
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

	// Gitea uses two-level: owner/repo
	// If URL includes domain: gitea.com/owner/repo -> parts[0]=gitea.com, parts[1]=owner, parts[2]=repo
	// If just path: owner/repo -> parts[0]=owner, parts[1]=repo
	switch len(cleanParts) {
	case 2:
		// owner/repo format
		return cleanParts[0], cleanParts[1], nil
	case 3:
		// domain/owner/repo format
		return cleanParts[1], cleanParts[2], nil
	default:
		if len(cleanParts) > 3 {
			// domain/owner/repo/extra... - still take owner and repo
			return cleanParts[1], cleanParts[2], nil
		}
		return "", "", &provider.ProviderError{
			Provider: "gitea",
			Message:  fmt.Sprintf("invalid repository URL format: %s", repoURL),
		}
	}
}
