// Package github implements the Git provider interface for GitHub.
package github

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/google/go-github/v57/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"

	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/pkg/logger"
)

// GitHub provider constants
const (
	// API pagination configuration
	defaultPerPage = 100

	// Default GitHub URL for public GitHub
	defaultGitHubURL = "https://github.com"

	// URL prefixes and suffixes
	gitSuffix   = ".git"
	httpsPrefix = "https://"
	httpPrefix  = "http://"
	gitAtPrefix = "git@"

	// Authentication username for GitHub Personal Access Tokens
	// GitHub recommends using "x-access-token" as username for PAT authentication
	tokenAuthUser = "x-access-token"

	// Path separator used in git@ format URLs (e.g., git@github.com:owner/repo)
	gitAtPathSeparator = ":"
)

func init() {
	// Register GitHub provider factory
	provider.Register("github", NewProvider)
}

// GitHubProvider implements the Provider interface for GitHub
type GitHubProvider struct {
	client             *github.Client
	token              string
	baseURL            string
	insecureSkipVerify bool
}

// isDefaultGitHub returns true if the provider is configured for public GitHub
// (i.e., not GitHub Enterprise)
func (p *GitHubProvider) isDefaultGitHub() bool {
	return p.baseURL == "" || p.baseURL == defaultGitHubURL
}

// normalizeURL removes protocol prefixes, .git suffix, and trailing slashes from a URL.
// It also converts git@ format (git@github.com:owner/repo) to standard path format.
func normalizeURL(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, gitSuffix)

	// Remove protocol prefixes
	url = strings.TrimPrefix(url, httpsPrefix)
	url = strings.TrimPrefix(url, httpPrefix)
	url = strings.TrimPrefix(url, gitAtPrefix)

	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Handle git@ format (git@github.com:owner/repo -> github.com/owner/repo)
	if idx := strings.Index(url, gitAtPathSeparator); idx != -1 {
		url = url[:idx] + "/" + url[idx+1:]
	}

	return url
}

// extractHostFromBaseURL extracts the hostname from the configured baseURL.
// It removes protocol prefixes and trailing slashes.
func (p *GitHubProvider) extractHostFromBaseURL() string {
	if p.isDefaultGitHub() {
		return "github.com"
	}

	host := strings.TrimPrefix(p.baseURL, httpsPrefix)
	host = strings.TrimPrefix(host, httpPrefix)
	host = strings.TrimSuffix(host, "/")

	return host
}

// buildCloneURL constructs a clone URL for the repository.
// If withAuth is true and a token is configured, the token is embedded in the URL.
func (p *GitHubProvider) buildCloneURL(owner, repo string, withAuth bool) string {
	host := p.extractHostFromBaseURL()

	if withAuth && p.token != "" {
		// Authenticated URL: https://x-access-token:TOKEN@host/owner/repo.git
		return fmt.Sprintf("%s%s:%s@%s/%s/%s%s", httpsPrefix, tokenAuthUser, p.token, host, owner, repo, gitSuffix)
	}

	// Anonymous URL: https://host/owner/repo.git
	return fmt.Sprintf("%s%s/%s/%s%s", httpsPrefix, host, owner, repo, gitSuffix)
}

// buildRepoURL constructs a repository URL without authentication information.
// This is useful for operations that don't require embedded credentials.
func (p *GitHubProvider) buildRepoURL(owner, repo string) string {
	host := p.extractHostFromBaseURL()
	return fmt.Sprintf("%s%s/%s/%s%s", httpsPrefix, host, owner, repo, gitSuffix)
}

// NewProvider creates a new GitHub provider instance
func NewProvider(opts *provider.ProviderOptions) (provider.Provider, error) {
	ctx := context.Background()

	token := opts.Token
	baseURL := opts.BaseURL

	var httpClient *http.Client

	if token != "" {
		// Authenticated mode: use OAuth2 token
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)

		// Create HTTP client with optional insecure skip verify
		httpClient = oauth2.NewClient(ctx, ts)
		if opts.InsecureSkipVerify {
			transport := httpClient.Transport.(*oauth2.Transport)
			if transport.Base == nil {
				transport.Base = &http.Transport{}
			}
			if t, ok := transport.Base.(*http.Transport); ok {
				if t.TLSClientConfig == nil {
					t.TLSClientConfig = &tls.Config{}
				}
				t.TLSClientConfig.InsecureSkipVerify = true
			}
		}
	} else {
		// Anonymous mode: use plain HTTP client for public repositories
		transport := &http.Transport{}
		if opts.InsecureSkipVerify {
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}
		httpClient = &http.Client{
			Transport: transport,
		}
	}

	var client *github.Client
	var err error

	if baseURL != "" && baseURL != defaultGitHubURL {
		// GitHub Enterprise
		client, err = github.NewClient(httpClient).WithEnterpriseURLs(baseURL, baseURL)
		if err != nil {
			return nil, &provider.ProviderError{
				Provider: "github",
				Message:  "failed to create enterprise client",
				Err:      err,
			}
		}
	} else {
		client = github.NewClient(httpClient)
	}

	return &GitHubProvider{
		client:             client,
		token:              token,
		baseURL:            baseURL,
		insecureSkipVerify: opts.InsecureSkipVerify,
	}, nil
}

// Name returns the provider name
func (p *GitHubProvider) Name() string {
	return "github"
}

// GetBaseURL returns the base URL of the provider
func (p *GitHubProvider) GetBaseURL() string {
	if p.isDefaultGitHub() {
		return defaultGitHubURL
	}
	return p.baseURL
}

// MatchesURL checks if the given repository URL matches GitHub
func (p *GitHubProvider) MatchesURL(repoURL string) bool {
	if repoURL == "" {
		return false
	}

	// Normalize URL using helper function
	url := normalizeURL(repoURL)

	// Extract domain from URL (first part before /)
	urlParts := strings.Split(url, "/")
	if len(urlParts) == 0 {
		return false
	}
	urlDomain := urlParts[0]

	// Check if URL domain matches base URL domain
	// For public GitHub: check if domain contains "github.com"
	// For self-hosted: check if domain matches configured base URL
	if p.isDefaultGitHub() {
		// Public GitHub: check for github.com in domain
		return strings.Contains(urlDomain, "github.com")
	}

	// Self-hosted: check if domain matches configured host
	configuredHost := p.extractHostFromBaseURL()
	baseParts := strings.Split(configuredHost, "/")
	if len(baseParts) > 0 {
		baseDomain := baseParts[0]
		return urlDomain == baseDomain || strings.HasPrefix(url, configuredHost)
	}

	return false
}

// Clone clones a repository using git command
func (p *GitHubProvider) Clone(ctx context.Context, owner, repo, destPath string, opts *provider.CloneOptions) error {
	logger.Info("Cloning repository",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.String("dest", destPath),
	)

	// Build clone URL using helper function
	// withAuth=true embeds token in URL if available
	cloneURL := p.buildCloneURL(owner, repo, true)

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

	// Execute git clone
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = io.Discard

	// Prevent interactive prompts for credentials
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")

	// Capture stderr to analyze error messages
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

		// Check for common error patterns and provide helpful messages
		errMsg := "failed to clone repository"
		if strings.Contains(stderrOutput, "Write access to repository not granted") ||
			strings.Contains(stderrOutput, "Permission denied") ||
			strings.Contains(stderrOutput, "Authentication failed") {
			errMsg = "authentication or permission denied: check that your SC_GITHUB_TOKEN has 'repo' scope for private repositories"
		} else if strings.Contains(stderrOutput, "Repository not found") {
			errMsg = "repository not found: check that the repository exists and your token has access"
		}

		return &provider.ProviderError{
			Provider: "github",
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

// GetPRRef returns the Git ref for a PR number
// GitHub uses refs/pull/{pr}/head format
func (p *GitHubProvider) GetPRRef(prNumber int) string {
	return fmt.Sprintf("refs/pull/%d/head", prNumber)
}

// ClonePR clones a specific PR using refs/pull/<pr>/head
// This method works for both fork and non-fork PRs
func (p *GitHubProvider) ClonePR(ctx context.Context, owner, repo string, prNumber int, destPath string, opts *provider.CloneOptions) error {
	logger.Info("Cloning PR using refs",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("pr_number", prNumber),
		zap.String("dest", destPath),
	)

	// Build repository URL WITHOUT embedded token (P1-2 Security improvement)
	repoURL := p.buildRepoURL(owner, repo)

	// Use helper function for the 4-step clone flow
	// P1-2 Security improvement: Pass token separately instead of embedding in URL
	// Note: Token can be empty for public repositories (anonymous access)
	err := provider.ClonePRWithRefs(ctx, &provider.ClonePRParams{
		ProviderName:       "github",
		RepoURL:            repoURL,
		Token:              p.token, // Empty token is allowed for public repos
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
func (p *GitHubProvider) GetPullRequest(ctx context.Context, owner, repo string, number int) (*provider.PullRequest, error) {
	pr, _, err := p.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		logger.Error("Failed to get pull request",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("number", number),
		)
		return nil, &provider.ProviderError{
			Provider: "github",
			Message:  "failed to get pull request",
			Err:      err,
		}
	}

	return &provider.PullRequest{
		Number:      pr.GetNumber(),
		Title:       pr.GetTitle(),
		Description: pr.GetBody(),
		State:       pr.GetState(),
		HeadBranch:  pr.GetHead().GetRef(),
		HeadSHA:     pr.GetHead().GetSHA(),
		BaseBranch:  pr.GetBase().GetRef(),
		BaseSHA:     pr.GetBase().GetSHA(),
		Author:      pr.GetUser().GetLogin(),
		URL:         pr.GetHTMLURL(),
	}, nil
}

// ListPullRequests lists open pull requests
func (p *GitHubProvider) ListPullRequests(ctx context.Context, owner, repo string) ([]*provider.PullRequest, error) {
	prs, _, err := p.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		logger.Error("Failed to list pull requests",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
		)
		return nil, &provider.ProviderError{
			Provider: "github",
			Message:  "failed to list pull requests",
			Err:      err,
		}
	}

	result := make([]*provider.PullRequest, len(prs))
	for i, pr := range prs {
		result[i] = &provider.PullRequest{
			Number:      pr.GetNumber(),
			Title:       pr.GetTitle(),
			Description: pr.GetBody(),
			State:       pr.GetState(),
			HeadBranch:  pr.GetHead().GetRef(),
			HeadSHA:     pr.GetHead().GetSHA(),
			BaseBranch:  pr.GetBase().GetRef(),
			BaseSHA:     pr.GetBase().GetSHA(),
			Author:      pr.GetUser().GetLogin(),
			URL:         pr.GetHTMLURL(),
		}
	}

	return result, nil
}

// PostComment posts a comment on a PR or commit
func (p *GitHubProvider) PostComment(ctx context.Context, owner, repo string, opts *provider.CommentOptions, body string) error {
	if opts.PRNumber > 0 {
		// Post PR comment
		comment := &github.IssueComment{Body: &body}
		_, _, err := p.client.Issues.CreateComment(ctx, owner, repo, opts.PRNumber, comment)
		if err != nil {
			logger.Error("Failed to post PR comment",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("repo", repo),
				zap.Int("pr", opts.PRNumber),
			)
			return &provider.ProviderError{
				Provider: "github",
				Message:  "failed to post PR comment",
				Err:      err,
			}
		}
	} else if opts.CommitSHA != "" {
		// Post commit comment
		comment := &github.RepositoryComment{Body: &body}
		_, _, err := p.client.Repositories.CreateComment(ctx, owner, repo, opts.CommitSHA, comment)
		if err != nil {
			logger.Error("Failed to post commit comment",
				zap.Error(err),
				zap.String("owner", owner),
				zap.String("repo", repo),
				zap.String("commit", opts.CommitSHA),
			)
			return &provider.ProviderError{
				Provider: "github",
				Message:  "failed to post commit comment",
				Err:      err,
			}
		}
	}

	return nil
}

// ListComments lists comments on a PR
func (p *GitHubProvider) ListComments(ctx context.Context, owner, repo string, prNumber int) ([]*provider.Comment, error) {
	comments, _, err := p.client.Issues.ListComments(ctx, owner, repo, prNumber, &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{PerPage: defaultPerPage},
	})
	if err != nil {
		logger.Error("Failed to list PR comments",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int("pr", prNumber),
		)
		return nil, &provider.ProviderError{
			Provider: "github",
			Message:  "failed to list PR comments",
			Err:      err,
		}
	}

	result := make([]*provider.Comment, len(comments))
	for i, c := range comments {
		result[i] = &provider.Comment{
			ID:        c.GetID(),
			Body:      c.GetBody(),
			Author:    c.GetUser().GetLogin(),
			CreatedAt: c.GetCreatedAt().Format("2006-01-02T15:04:05Z"),
		}
	}

	return result, nil
}

// DeleteComment deletes a comment by ID
func (p *GitHubProvider) DeleteComment(ctx context.Context, owner, repo string, commentID int64) error {
	_, err := p.client.Issues.DeleteComment(ctx, owner, repo, commentID)
	if err != nil {
		logger.Error("Failed to delete comment",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int64("comment_id", commentID),
		)
		return &provider.ProviderError{
			Provider: "github",
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
// Note: prNumber is not needed for GitHub as comments are identified by ID only
func (p *GitHubProvider) UpdateComment(ctx context.Context, owner, repo string, commentID int64, prNumber int, body string) error {
	comment := &github.IssueComment{Body: &body}
	_, _, err := p.client.Issues.EditComment(ctx, owner, repo, commentID, comment)
	if err != nil {
		logger.Error("Failed to update comment",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.Int64("comment_id", commentID),
		)
		return &provider.ProviderError{
			Provider: "github",
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
func (p *GitHubProvider) ParseWebhook(r *http.Request, secret string) (*provider.WebhookEvent, error) {
	var body []byte
	var err error

	// Validate signature if secret is provided, using go-github's ValidatePayload
	// ValidatePayload reads the body and validates it, returning the validated payload
	if secret != "" {
		body, err = github.ValidatePayload(r, []byte(secret))
		if err != nil {
			logger.Warn("Failed to validate webhook payload",
				zap.Error(err),
			)
			return nil, &provider.ProviderError{
				Provider: "github",
				Message:  "invalid webhook signature",
				Err:      err,
			}
		}
	} else {
		// No secret provided, read body directly
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, &provider.ProviderError{
				Provider: "github",
				Message:  "failed to read webhook body",
				Err:      err,
			}
		}
	}

	// Parse event type
	eventType := r.Header.Get("X-GitHub-Event")

	event := &provider.WebhookEvent{
		Provider:   "github",
		RawPayload: body,
	}

	switch eventType {
	case "push":
		var payload github.PushEvent
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, &provider.ProviderError{
				Provider: "github",
				Message:  "failed to parse push event",
				Err:      err,
			}
		}
		event.Type = provider.EventTypePush
		event.Owner = payload.GetRepo().GetOwner().GetLogin()
		event.Repo = payload.GetRepo().GetName()
		event.Ref = strings.TrimPrefix(payload.GetRef(), "refs/heads/")
		event.CommitSHA = payload.GetAfter()
		event.Sender = payload.GetSender().GetLogin()

	case "pull_request":
		var payload github.PullRequestEvent
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, &provider.ProviderError{
				Provider: "github",
				Message:  "failed to parse pull_request event",
				Err:      err,
			}
		}
		pr := payload.GetPullRequest()
		event.Type = provider.EventTypePullRequest
		event.Owner = payload.GetRepo().GetOwner().GetLogin()
		event.Repo = payload.GetRepo().GetName()
		event.Ref = pr.GetHead().GetRef()
		event.CommitSHA = pr.GetHead().GetSHA()
		event.PRNumber = pr.GetNumber()
		// GitHub actions are already in lowercase: opened, synchronize, reopened, closed, etc.
		// Normalize to lowercase to ensure consistency
		event.Action = strings.ToLower(payload.GetAction())
		event.Sender = payload.GetSender().GetLogin()

		// Extract complete PR information from webhook payload
		event.PRTitle = pr.GetTitle()
		event.PRDescription = pr.GetBody()
		if pr.GetBase() != nil {
			event.BaseCommitSHA = pr.GetBase().GetSHA()
		}

		// Extract changed files if available in payload
		// Note: GitHub webhook may not always include changed_files in the payload
		// If not available, we can fetch it later via API if needed
		if pr.ChangedFiles != nil && *pr.ChangedFiles > 0 {
			// GitHub webhook payload doesn't directly include changed_files list
			// We would need to make an API call to get the list, but for now
			// we'll leave it empty and can fetch later if needed
			event.ChangedFiles = []string{}
		}

		logger.Info("Parsed GitHub pull_request webhook",
			zap.String("action", event.Action),
			zap.String("owner", event.Owner),
			zap.String("repo", event.Repo),
			zap.Int("pr_number", event.PRNumber),
			zap.String("pr_title", event.PRTitle),
			zap.String("base_sha", event.BaseCommitSHA),
		)

	default:
		return nil, &provider.ProviderError{
			Provider: "github",
			Message:  fmt.Sprintf("unsupported event type: %s", eventType),
		}
	}

	return event, nil
}

// CreateWebhook creates a webhook for the repository
func (p *GitHubProvider) CreateWebhook(ctx context.Context, owner, repo, url, secret string, events []string) (string, error) {
	hook := &github.Hook{
		Config: map[string]interface{}{
			"url":          url,
			"content_type": "json",
			"secret":       secret,
		},
		Events: events,
		Active: github.Bool(true),
	}

	created, _, err := p.client.Repositories.CreateHook(ctx, owner, repo, hook)
	if err != nil {
		logger.Error("Failed to create webhook",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
		)
		return "", &provider.ProviderError{
			Provider: "github",
			Message:  "failed to create webhook",
			Err:      err,
		}
	}

	return fmt.Sprintf("%d", created.GetID()), nil
}

// DeleteWebhook deletes a webhook from the repository
func (p *GitHubProvider) DeleteWebhook(ctx context.Context, owner, repo, webhookID string) error {
	var id int64
	fmt.Sscanf(webhookID, "%d", &id)

	_, err := p.client.Repositories.DeleteHook(ctx, owner, repo, id)
	if err != nil {
		logger.Error("Failed to delete webhook",
			zap.Error(err),
			zap.String("owner", owner),
			zap.String("repo", repo),
			zap.String("webhook_id", webhookID),
		)
		return &provider.ProviderError{
			Provider: "github",
			Message:  "failed to delete webhook",
			Err:      err,
		}
	}

	return nil
}

// ValidateToken validates the GitHub token
func (p *GitHubProvider) ValidateToken(ctx context.Context) error {
	_, _, err := p.client.Users.Get(ctx, "")
	if err != nil {
		return &provider.ProviderError{
			Provider: "github",
			Message:  "invalid token",
			Err:      err,
		}
	}
	return nil
}

// ListBranches lists all branches for a repository
func (p *GitHubProvider) ListBranches(ctx context.Context, owner, repo string) ([]string, error) {
	allBranches := []string{}
	page := 1

	for {
		branches, resp, err := p.client.Repositories.ListBranches(ctx, owner, repo, &github.BranchListOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: defaultPerPage,
			},
		})
		if err != nil {
			// Check for 401 error (Bad credentials) and retry with anonymous access
			if resp != nil && resp.StatusCode == 401 && p.token != "" {
				logger.Warn("Token invalid, retrying with anonymous access for public repository",
					zap.String("owner", owner),
					zap.String("repo", repo),
				)
				// Create anonymous client and retry
				anonymousClient := github.NewClient(nil)
				branches, resp, err = anonymousClient.Repositories.ListBranches(ctx, owner, repo, &github.BranchListOptions{
					ListOptions: github.ListOptions{
						Page:    page,
						PerPage: defaultPerPage,
					},
				})
			}

			// Check for 403 error (rate limit exceeded)
			if err != nil && resp != nil && resp.StatusCode == 403 {
				logger.Error("GitHub API rate limit exceeded",
					zap.String("owner", owner),
					zap.String("repo", repo),
				)
				return nil, &provider.ProviderError{
					Provider: "github",
					Message:  "GitHub API rate limit exceeded. Please configure a valid GITHUB_TOKEN environment variable for higher rate limits",
					Err:      err,
				}
			}

			if err != nil {
				logger.Error("Failed to list branches",
					zap.Error(err),
					zap.String("owner", owner),
					zap.String("repo", repo),
				)
				return nil, &provider.ProviderError{
					Provider: "github",
					Message:  "failed to list branches",
					Err:      err,
				}
			}
		}

		for _, branch := range branches {
			if branch.Name != nil {
				allBranches = append(allBranches, *branch.Name)
			}
		}

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	logger.Info("Listed branches",
		zap.String("owner", owner),
		zap.String("repo", repo),
		zap.Int("count", len(allBranches)),
	)

	return allBranches, nil
}

// ParseRepoPath parses owner and repo from a repository URL.
// GitHub uses a two-level structure: owner/repo
// Supported formats:
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo.git
//   - github.com/owner/repo
//   - owner/repo
func (p *GitHubProvider) ParseRepoPath(repoURL string) (owner, repo string, err error) {
	if repoURL == "" {
		return "", "", &provider.ProviderError{
			Provider: "github",
			Message:  "empty repository URL",
		}
	}

	// Normalize URL using helper function
	url := normalizeURL(repoURL)

	// Split by /
	parts := strings.Split(url, "/")

	// Remove empty parts
	var cleanParts []string
	for _, p := range parts {
		if p != "" {
			cleanParts = append(cleanParts, p)
		}
	}

	// GitHub uses two-level: owner/repo
	// If URL includes domain: github.com/owner/repo -> parts[0]=github.com, parts[1]=owner, parts[2]=repo
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
			Provider: "github",
			Message:  fmt.Sprintf("invalid repository URL format: %s", repoURL),
		}
	}
}
