package prurl

import (
	"testing"
)

func TestParse_GitHub(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantErr  bool
		provider string
		owner    string
		repo     string
		number   int
	}{
		{
			name:     "standard GitHub PR URL",
			url:      "https://github.com/owner/repo/pull/123",
			wantErr:  false,
			provider: "github",
			owner:    "owner",
			repo:     "repo",
			number:   123,
		},
		{
			name:     "GitHub PR URL with trailing slash",
			url:      "https://github.com/owner/repo/pull/456/",
			wantErr:  false,
			provider: "github",
			owner:    "owner",
			repo:     "repo",
			number:   456,
		},
		{
			name:     "GitHub PR URL with files tab",
			url:      "https://github.com/owner/repo/pull/789/files",
			wantErr:  false,
			provider: "github",
			owner:    "owner",
			repo:     "repo",
			number:   789,
		},
		{
			name:     "GitHub PR URL with commits tab",
			url:      "https://github.com/microsoft/vscode/pull/12345/commits",
			wantErr:  false,
			provider: "github",
			owner:    "microsoft",
			repo:     "vscode",
			number:   12345,
		},
		{
			name:     "GitHub PR URL with hyphenated owner and repo",
			url:      "https://github.com/my-org/my-repo/pull/1",
			wantErr:  false,
			provider: "github",
			owner:    "my-org",
			repo:     "my-repo",
			number:   1,
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parser.Parse(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error: %v", err)
				return
			}

			if info.Provider != tt.provider {
				t.Errorf("Provider = %v, want %v", info.Provider, tt.provider)
			}
			if info.Owner != tt.owner {
				t.Errorf("Owner = %v, want %v", info.Owner, tt.owner)
			}
			if info.Repo != tt.repo {
				t.Errorf("Repo = %v, want %v", info.Repo, tt.repo)
			}
			if info.Number != tt.number {
				t.Errorf("Number = %v, want %v", info.Number, tt.number)
			}
		})
	}
}

func TestParse_GitLab(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantErr  bool
		provider string
		owner    string
		repo     string
		number   int
	}{
		{
			name:     "standard GitLab MR URL",
			url:      "https://gitlab.com/owner/repo/-/merge_requests/123",
			wantErr:  false,
			provider: "gitlab",
			owner:    "owner",
			repo:     "repo",
			number:   123,
		},
		{
			name:     "GitLab MR URL with nested group",
			url:      "https://gitlab.com/group/subgroup/repo/-/merge_requests/456",
			wantErr:  false,
			provider: "gitlab",
			owner:    "group/subgroup",
			repo:     "repo",
			number:   456,
		},
		{
			name:     "GitLab MR URL with deeply nested group",
			url:      "https://gitlab.com/org/team/project/repo/-/merge_requests/789",
			wantErr:  false,
			provider: "gitlab",
			owner:    "org/team/project",
			repo:     "repo",
			number:   789,
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parser.Parse(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Parse() unexpected error: %v", err)
				return
			}

			if info.Provider != tt.provider {
				t.Errorf("Provider = %v, want %v", info.Provider, tt.provider)
			}
			if info.Owner != tt.owner {
				t.Errorf("Owner = %v, want %v", info.Owner, tt.owner)
			}
			if info.Repo != tt.repo {
				t.Errorf("Repo = %v, want %v", info.Repo, tt.repo)
			}
			if info.Number != tt.number {
				t.Errorf("Number = %v, want %v", info.Number, tt.number)
			}
		})
	}
}

func TestParse_Invalid(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{
			name: "empty URL",
			url:  "",
		},
		{
			name: "invalid URL",
			url:  "not-a-url",
		},
		{
			name: "missing PR number",
			url:  "https://github.com/owner/repo/pull/",
		},
		{
			name: "non-PR GitHub URL",
			url:  "https://github.com/owner/repo",
		},
		{
			name: "GitHub issues URL (not PR)",
			url:  "https://github.com/owner/repo/issues/123",
		},
		{
			name: "unsupported provider",
			url:  "https://bitbucket.org/owner/repo/pull-requests/123",
		},
	}

	parser := NewParser()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.url)
			if err == nil {
				t.Errorf("Parse() expected error for invalid URL: %s", tt.url)
			}
		})
	}
}

func TestBuildClonePath(t *testing.T) {
	// New format: {provider}-{owner}-{repo} (without branch or PR number)
	// This allows multiple PRs from the same repo to share the same directory
	tests := []struct {
		name       string
		info       *PRInfo
		baseBranch string
		want       string
	}{
		{
			name: "simple GitHub PR",
			info: &PRInfo{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
			},
			baseBranch: "main",
			want:       "github-owner-repo",
		},
		{
			name: "GitHub PR with hyphenated names",
			info: &PRInfo{
				Provider: "github",
				Owner:    "my-org",
				Repo:     "my-repo",
			},
			baseBranch: "develop",
			want:       "github-my-org-my-repo",
		},
		{
			name: "GitLab MR with nested group",
			info: &PRInfo{
				Provider: "gitlab",
				Owner:    "group/subgroup",
				Repo:     "repo",
			},
			baseBranch: "master",
			want:       "gitlab-group-subgroup-repo",
		},
		{
			name: "branch with slash - branch is ignored in new format",
			info: &PRInfo{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
			},
			baseBranch: "feature/new-feature",
			want:       "github-owner-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.BuildClonePath(tt.baseBranch)
			if got != tt.want {
				t.Errorf("BuildClonePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildClonePathForPR(t *testing.T) {
	// New format: {provider}-{owner}-{repo} (without PR number)
	// This allows multiple PRs from the same repo to share the same directory
	tests := []struct {
		name string
		info *PRInfo
		want string
	}{
		{
			name: "simple GitHub PR",
			info: &PRInfo{
				Provider: "github",
				Owner:    "owner",
				Repo:     "repo",
				Number:   123,
			},
			want: "github-owner-repo",
		},
		{
			name: "GitHub PR with hyphenated names",
			info: &PRInfo{
				Provider: "github",
				Owner:    "my-org",
				Repo:     "my-repo",
				Number:   456,
			},
			want: "github-my-org-my-repo",
		},
		{
			name: "GitLab MR with nested group",
			info: &PRInfo{
				Provider: "gitlab",
				Owner:    "group/subgroup",
				Repo:     "repo",
				Number:   789,
			},
			want: "gitlab-group-subgroup-repo",
		},
		{
			name: "deeply nested GitLab group",
			info: &PRInfo{
				Provider: "gitlab",
				Owner:    "org/team/project",
				Repo:     "repo",
				Number:   100,
			},
			want: "gitlab-org-team-project-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.BuildClonePathForPR()
			if got != tt.want {
				t.Errorf("BuildClonePathForPR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPRInfo_String(t *testing.T) {
	info := &PRInfo{
		Provider: "github",
		Owner:    "owner",
		Repo:     "repo",
		Number:   123,
	}

	want := "owner/repo#123 (github)"
	got := info.String()

	if got != want {
		t.Errorf("String() = %v, want %v", got, want)
	}
}

func TestDefaultParser(t *testing.T) {
	// 测试默认解析器的便捷函数
	info, err := Parse("https://github.com/owner/repo/pull/123")
	if err != nil {
		t.Errorf("Parse() unexpected error: %v", err)
		return
	}

	if info.Owner != "owner" || info.Repo != "repo" || info.Number != 123 {
		t.Errorf("Parse() returned unexpected result: %+v", info)
	}
}

func TestRegisterHost(t *testing.T) {
	parser := NewParser()

	// 注册自定义 GitHub Enterprise 主机
	parser.RegisterHost("github.mycompany.com", "github")

	info, err := parser.Parse("https://github.mycompany.com/team/project/pull/42")
	if err != nil {
		t.Errorf("Parse() unexpected error: %v", err)
		return
	}

	if info.Provider != "github" {
		t.Errorf("Provider = %v, want github", info.Provider)
	}
	if info.Owner != "team" {
		t.Errorf("Owner = %v, want team", info.Owner)
	}
	if info.Repo != "project" {
		t.Errorf("Repo = %v, want project", info.Repo)
	}
	if info.Number != 42 {
		t.Errorf("Number = %v, want 42", info.Number)
	}
}
