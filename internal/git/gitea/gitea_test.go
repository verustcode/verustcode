package gitea

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/verustcode/verustcode/internal/git/provider"
)

func TestParseRepoPath(t *testing.T) {
	p := &GiteaProvider{
		baseURL: "https://gitea.com",
	}

	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "simple owner/repo format",
			repoURL:   "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "full HTTPS URL",
			repoURL:   "https://gitea.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "full HTTPS URL with .git suffix",
			repoURL:   "https://gitea.com/owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "full HTTP URL",
			repoURL:   "http://gitea.example.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "URL with trailing slash",
			repoURL:   "https://gitea.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "URL with extra path",
			repoURL:   "https://gitea.com/owner/repo/pulls/123",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "git@ format",
			repoURL:   "git@gitea.com:owner/repo.git",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "domain only with owner/repo",
			repoURL:   "gitea.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "hyphenated names",
			repoURL:   "https://gitea.com/my-org/my-repo",
			wantOwner: "my-org",
			wantRepo:  "my-repo",
			wantErr:   false,
		},
		{
			name:    "empty URL",
			repoURL: "",
			wantErr: true,
		},
		{
			name:    "invalid format - single segment",
			repoURL: "owner",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := p.ParseRepoPath(tt.repoURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseRepoPath() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseRepoPath() unexpected error: %v", err)
				return
			}

			if owner != tt.wantOwner {
				t.Errorf("Owner = %v, want %v", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("Repo = %v, want %v", repo, tt.wantRepo)
			}
		})
	}
}

func TestBuildCloneURL(t *testing.T) {
	// Note: buildCloneURL no longer embeds token in URL for security reasons.
	// Token is now passed separately via Git credential helper.
	tests := []struct {
		name    string
		baseURL string
		owner   string
		repo    string
		wantURL string
	}{
		{
			name:    "default gitea.com",
			baseURL: "https://gitea.com",
			owner:   "owner",
			repo:    "repo",
			wantURL: "https://gitea.com/owner/repo.git",
		},
		{
			name:    "self-hosted HTTPS",
			baseURL: "https://gitea.example.com",
			owner:   "myorg",
			repo:    "myrepo",
			wantURL: "https://gitea.example.com/myorg/myrepo.git",
		},
		{
			name:    "self-hosted HTTP",
			baseURL: "http://gitea.local:3000",
			owner:   "dev",
			repo:    "project",
			wantURL: "http://gitea.local:3000/dev/project.git",
		},
		{
			name:    "URL with trailing slash",
			baseURL: "https://gitea.com/",
			owner:   "owner",
			repo:    "repo",
			wantURL: "https://gitea.com/owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &GiteaProvider{
				baseURL: tt.baseURL,
			}

			got := p.buildCloneURL(tt.owner, tt.repo)
			if got != tt.wantURL {
				t.Errorf("buildCloneURL() = %v, want %v", got, tt.wantURL)
			}
		})
	}
}

func TestGetPRRef(t *testing.T) {
	p := &GiteaProvider{}

	tests := []struct {
		prNumber int
		want     string
	}{
		{prNumber: 1, want: "refs/pull/1/head"},
		{prNumber: 123, want: "refs/pull/123/head"},
		{prNumber: 9999, want: "refs/pull/9999/head"},
	}

	for _, tt := range tests {
		got := p.GetPRRef(tt.prNumber)
		if got != tt.want {
			t.Errorf("GetPRRef(%d) = %v, want %v", tt.prNumber, got, tt.want)
		}
	}
}

func TestNormalizeGiteaAction(t *testing.T) {
	tests := []struct {
		action string
		want   string
	}{
		{action: "opened", want: provider.PREventActionOpened},
		{action: "Opened", want: provider.PREventActionOpened},
		{action: "OPENED", want: provider.PREventActionOpened},
		{action: "synchronized", want: provider.PREventActionSynchronize},
		{action: "synchronize", want: provider.PREventActionSynchronize},
		{action: "reopened", want: provider.PREventActionReopened},
		{action: "closed", want: "closed"},
		{action: "merged", want: "merged"},
		{action: "edited", want: "edited"},
		{action: "unknown_action", want: "unknown_action"},
		{action: "CUSTOM_ACTION", want: "custom_action"},
	}

	for _, tt := range tests {
		t.Run(tt.action, func(t *testing.T) {
			got := normalizeGiteaAction(tt.action)
			if got != tt.want {
				t.Errorf("normalizeGiteaAction(%q) = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}

func TestParseWebhook_PullRequest(t *testing.T) {
	p := &GiteaProvider{}

	payload := map[string]interface{}{
		"action": "opened",
		"number": 42,
		"pull_request": map[string]interface{}{
			"id":     12345,
			"number": 42,
			"title":  "Test PR",
			"body":   "PR description",
			"head": map[string]interface{}{
				"ref": "feature-branch",
				"sha": "abc123def456",
			},
			"base": map[string]interface{}{
				"ref": "main",
				"sha": "def789ghi012",
			},
			"merge_base": "base123456",
		},
		"sender": map[string]interface{}{
			"login": "testuser",
		},
		"repository": map[string]interface{}{
			"name": "testrepo",
			"owner": map[string]interface{}{
				"login": "testowner",
			},
		},
	}

	body, _ := json.Marshal(payload)

	// Create request with headers
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Gitea-Event", "pull_request")
	req.Header.Set("Content-Type", "application/json")

	event, err := p.ParseWebhook(req, "")
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.Type != provider.EventTypePullRequest {
		t.Errorf("Type = %v, want %v", event.Type, provider.EventTypePullRequest)
	}
	if event.Provider != "gitea" {
		t.Errorf("Provider = %v, want gitea", event.Provider)
	}
	if event.Owner != "testowner" {
		t.Errorf("Owner = %v, want testowner", event.Owner)
	}
	if event.Repo != "testrepo" {
		t.Errorf("Repo = %v, want testrepo", event.Repo)
	}
	if event.PRNumber != 42 {
		t.Errorf("PRNumber = %v, want 42", event.PRNumber)
	}
	if event.Action != provider.PREventActionOpened {
		t.Errorf("Action = %v, want %v", event.Action, provider.PREventActionOpened)
	}
	if event.Sender != "testuser" {
		t.Errorf("Sender = %v, want testuser", event.Sender)
	}
	if event.Ref != "feature-branch" {
		t.Errorf("Ref = %v, want feature-branch", event.Ref)
	}
	if event.CommitSHA != "abc123def456" {
		t.Errorf("CommitSHA = %v, want abc123def456", event.CommitSHA)
	}
	if event.PRTitle != "Test PR" {
		t.Errorf("PRTitle = %v, want Test PR", event.PRTitle)
	}
	if event.BaseCommitSHA != "base123456" {
		t.Errorf("BaseCommitSHA = %v, want base123456", event.BaseCommitSHA)
	}
}

func TestParseWebhook_Push(t *testing.T) {
	p := &GiteaProvider{}

	payload := map[string]interface{}{
		"ref":   "refs/heads/main",
		"after": "newsha123456",
		"sender": map[string]interface{}{
			"login": "pusher",
		},
		"repository": map[string]interface{}{
			"name": "repo",
			"owner": map[string]interface{}{
				"login": "owner",
			},
		},
	}

	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Gitea-Event", "push")
	req.Header.Set("Content-Type", "application/json")

	event, err := p.ParseWebhook(req, "")
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}

	if event.Type != provider.EventTypePush {
		t.Errorf("Type = %v, want %v", event.Type, provider.EventTypePush)
	}
	if event.Ref != "main" {
		t.Errorf("Ref = %v, want main", event.Ref)
	}
	if event.CommitSHA != "newsha123456" {
		t.Errorf("CommitSHA = %v, want newsha123456", event.CommitSHA)
	}
}

func TestParseWebhook_SignatureValidation(t *testing.T) {
	p := &GiteaProvider{}
	secret := "mysecretkey"

	payload := map[string]interface{}{
		"ref":   "refs/heads/main",
		"after": "sha123",
		"sender": map[string]interface{}{
			"login": "user",
		},
		"repository": map[string]interface{}{
			"name": "repo",
			"owner": map[string]interface{}{
				"login": "owner",
			},
		},
	}
	body, _ := json.Marshal(payload)

	// Calculate correct signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	t.Run("valid signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		req.Header.Set("X-Gitea-Event", "push")
		req.Header.Set("X-Gitea-Signature", signature)

		_, err := p.ParseWebhook(req, secret)
		if err != nil {
			t.Errorf("ParseWebhook() with valid signature failed: %v", err)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		req.Header.Set("X-Gitea-Event", "push")
		req.Header.Set("X-Gitea-Signature", "invalidsignature")

		_, err := p.ParseWebhook(req, secret)
		if err == nil {
			t.Error("ParseWebhook() with invalid signature should fail")
		}
	})

	t.Run("missing signature when secret is set", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		req.Header.Set("X-Gitea-Event", "push")

		_, err := p.ParseWebhook(req, secret)
		if err == nil {
			t.Error("ParseWebhook() with missing signature should fail when secret is set")
		}
	})

	t.Run("no signature required when secret is empty", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		req.Header.Set("X-Gitea-Event", "push")

		_, err := p.ParseWebhook(req, "")
		if err != nil {
			t.Errorf("ParseWebhook() without secret should succeed: %v", err)
		}
	})
}

func TestParseWebhook_UnsupportedEvent(t *testing.T) {
	p := &GiteaProvider{}

	payload := map[string]interface{}{
		"action": "created",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	req.Header.Set("X-Gitea-Event", "issue_comment")

	_, err := p.ParseWebhook(req, "")
	if err == nil {
		t.Error("ParseWebhook() should fail for unsupported event type")
	}

	provErr, ok := err.(*provider.ProviderError)
	if !ok {
		t.Errorf("expected ProviderError, got %T", err)
	}
	if provErr.Provider != "gitea" {
		t.Errorf("Provider = %v, want gitea", provErr.Provider)
	}
}

func TestName(t *testing.T) {
	p := &GiteaProvider{}
	if got := p.Name(); got != "gitea" {
		t.Errorf("Name() = %v, want gitea", got)
	}
}

func TestGetBaseURL(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"https://gitea.com", "https://gitea.com"},
		{"https://gitea.example.com", "https://gitea.example.com"},
		{"http://localhost:3000", "http://localhost:3000"},
	}

	for _, tt := range tests {
		p := &GiteaProvider{baseURL: tt.baseURL}
		if got := p.GetBaseURL(); got != tt.want {
			t.Errorf("GetBaseURL() = %v, want %v", got, tt.want)
		}
	}
}
