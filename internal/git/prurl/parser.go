// Package prurl provides PR URL parsing utilities for different Git providers.
// It supports parsing PR/MR URLs from GitHub, GitLab, and other Git hosting services.
package prurl

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// PRInfo contains parsed information from a PR URL
type PRInfo struct {
	// Provider is the Git provider name (github, gitlab, etc.)
	Provider string

	// Host is the full host (e.g., github.com, gitlab.example.com)
	Host string

	// Owner is the repository owner/organization
	Owner string

	// Repo is the repository name
	Repo string

	// Number is the PR/MR number
	Number int

	// OriginalURL is the original URL that was parsed
	OriginalURL string
}

// Parser parses PR URLs from different Git providers
type Parser struct {
	// customHostMappings maps custom hosts to provider names
	customHostMappings map[string]string
}

// NewParser creates a new PR URL parser
func NewParser() *Parser {
	return &Parser{
		customHostMappings: make(map[string]string),
	}
}

// RegisterHost registers a custom host mapping to a provider
// For example: RegisterHost("git.example.com", "github") for GitHub Enterprise
func (p *Parser) RegisterHost(host, provider string) {
	p.customHostMappings[strings.ToLower(host)] = provider
}

// Parse parses a PR URL and returns PRInfo
// Supported formats:
// - GitHub: https://github.com/owner/repo/pull/123
// - GitLab: https://gitlab.com/owner/repo/-/merge_requests/123
// - GitHub Enterprise: https://github.example.com/owner/repo/pull/123
func (p *Parser) Parse(prURL string) (*PRInfo, error) {
	// 清理 URL
	prURL = strings.TrimSpace(prURL)
	if prURL == "" {
		return nil, fmt.Errorf("empty PR URL")
	}

	// 解析 URL
	parsedURL, err := url.Parse(prURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	// 获取 host
	host := strings.ToLower(parsedURL.Host)
	if host == "" {
		return nil, fmt.Errorf("missing host in URL")
	}

	// 确定 provider
	provider := p.detectProvider(host, parsedURL.Path)
	if provider == "" {
		return nil, fmt.Errorf("unsupported Git provider for host: %s", host)
	}

	// 根据 provider 解析路径
	var info *PRInfo
	switch provider {
	case "github":
		info, err = p.parseGitHubURL(parsedURL)
	case "gitlab":
		info, err = p.parseGitLabURL(parsedURL)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}

	info.Provider = provider
	info.Host = host
	info.OriginalURL = prURL

	return info, nil
}

// detectProvider determines the Git provider from host and path
func (p *Parser) detectProvider(host, path string) string {
	// 检查自定义映射（优先级最高）
	if provider, ok := p.customHostMappings[host]; ok {
		return provider
	}

	// 检查已知的公共主机
	switch {
	case strings.Contains(host, "github"):
		return "github"
	case strings.Contains(host, "gitlab"):
		return "gitlab"
	case strings.Contains(host, "bitbucket"):
		return "bitbucket"
	}

	// 通过路径模式检测（用于自部署实例，路径模式优先）
	if strings.Contains(path, "/pull/") {
		return "github"
	}
	if strings.Contains(path, "/-/merge_requests/") || strings.Contains(path, "/merge_requests/") {
		return "gitlab"
	}

	return ""
}

// RegisterHostsFromConfig registers custom host mappings from provider configurations
// This should be called during application initialization with the Git provider configs
// Example: For a GitLab self-hosted at git.example.com, call RegisterHost("git.example.com", "gitlab")
func (p *Parser) RegisterHostsFromConfig(providers []struct {
	Type string
	URL  string
}) {
	for _, prov := range providers {
		if prov.URL == "" {
			continue
		}

		// Extract host from URL
		host := extractHostFromURL(prov.URL)
		if host != "" && host != "github.com" && host != "gitlab.com" {
			p.RegisterHost(host, prov.Type)
		}
	}
}

// extractHostFromURL extracts the host part from a URL
func extractHostFromURL(rawURL string) string {
	// Remove protocol
	host := rawURL
	if strings.HasPrefix(host, "https://") {
		host = strings.TrimPrefix(host, "https://")
	} else if strings.HasPrefix(host, "http://") {
		host = strings.TrimPrefix(host, "http://")
	}

	// Remove trailing path
	if idx := strings.Index(host, "/"); idx > 0 {
		host = host[:idx]
	}

	return strings.ToLower(host)
}

// parseGitHubURL parses a GitHub PR URL
// Format: https://github.com/owner/repo/pull/123
func (p *Parser) parseGitHubURL(u *url.URL) (*PRInfo, error) {
	// 使用正则匹配路径
	// 支持: /owner/repo/pull/123 或 /owner/repo/pull/123/files 等
	pattern := regexp.MustCompile(`^/([^/]+)/([^/]+)/pull/(\d+)`)
	matches := pattern.FindStringSubmatch(u.Path)

	if len(matches) != 4 {
		return nil, fmt.Errorf("invalid GitHub PR URL format: %s", u.Path)
	}

	prNumber, err := strconv.Atoi(matches[3])
	if err != nil {
		return nil, fmt.Errorf("invalid PR number: %s", matches[3])
	}

	return &PRInfo{
		Owner:  matches[1],
		Repo:   matches[2],
		Number: prNumber,
	}, nil
}

// parseGitLabURL parses a GitLab MR URL
// Format: https://gitlab.com/owner/repo/-/merge_requests/123
// Or: https://gitlab.com/group/subgroup/repo/-/merge_requests/123
func (p *Parser) parseGitLabURL(u *url.URL) (*PRInfo, error) {
	// GitLab 支持嵌套组，所以需要更灵活的匹配
	// 支持: /owner/repo/-/merge_requests/123
	// 或: /group/subgroup/repo/-/merge_requests/123
	pattern := regexp.MustCompile(`^/(.+?)/-/merge_requests/(\d+)`)
	matches := pattern.FindStringSubmatch(u.Path)

	if len(matches) != 3 {
		// 尝试不带 /- 的旧格式
		pattern = regexp.MustCompile(`^/(.+?)/merge_requests/(\d+)`)
		matches = pattern.FindStringSubmatch(u.Path)
		if len(matches) != 3 {
			return nil, fmt.Errorf("invalid GitLab MR URL format: %s", u.Path)
		}
	}

	mrNumber, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid MR number: %s", matches[2])
	}

	// 分割路径获取 owner 和 repo
	pathParts := strings.Split(matches[1], "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid GitLab path: %s", matches[1])
	}

	// 最后一个是 repo，其余是 owner/group
	repo := pathParts[len(pathParts)-1]
	owner := strings.Join(pathParts[:len(pathParts)-1], "/")

	return &PRInfo{
		Owner:  owner,
		Repo:   repo,
		Number: mrNumber,
	}, nil
}

// BuildClonePath generates a clone directory name based on PR info
// Format: {provider}-{owner}-{repo}
// This format allows multiple PRs from the same repo to share the same directory,
// enabling sequential processing with PR switching via fetch + checkout.
func (info *PRInfo) BuildClonePath(baseBranch string) string {
	// Clean owner, replace / with - (handle GitLab nested groups)
	owner := strings.ReplaceAll(info.Owner, "/", "-")

	// Note: baseBranch parameter is kept for API compatibility but no longer used
	// The new format doesn't include branch info to support repo-level sharing
	_ = baseBranch

	return fmt.Sprintf("%s-%s-%s", info.Provider, owner, info.Repo)
}

// BuildClonePathForPR generates a clone directory name for PR
// Format: {provider}-{owner}-{repo}
// This format allows multiple PRs from the same repo to share the same directory,
// enabling sequential processing with PR switching via fetch + checkout.
// Note: PR number is no longer included in the path to support repo-level sharing.
func (info *PRInfo) BuildClonePathForPR() string {
	// Clean owner, replace / with - (handle GitLab nested groups)
	owner := strings.ReplaceAll(info.Owner, "/", "-")

	return fmt.Sprintf("%s-%s-%s", info.Provider, owner, info.Repo)
}

// String returns a human-readable string representation
func (info *PRInfo) String() string {
	return fmt.Sprintf("%s/%s#%d (%s)", info.Owner, info.Repo, info.Number, info.Provider)
}

// DefaultParser is the default PR URL parser instance
var DefaultParser = NewParser()

// Parse is a convenience function using the default parser
func Parse(prURL string) (*PRInfo, error) {
	return DefaultParser.Parse(prURL)
}
