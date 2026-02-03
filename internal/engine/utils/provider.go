// Package utils provides utility functions for the engine.
// This file contains Git provider detection utilities.
package utils

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/verustcode/verustcode/internal/config"
	"github.com/verustcode/verustcode/internal/git/provider"
	"github.com/verustcode/verustcode/pkg/logger"
)

// DetectProviderFromURL detects the Git provider from a repository URL.
// Returns "github", "gitlab", or empty string if unknown.
func DetectProviderFromURL(repoURL string) string {
	if repoURL == "" {
		return ""
	}

	// Check for GitHub
	if strings.Contains(repoURL, "github.com") || strings.Contains(repoURL, "github.") {
		return "github"
	}

	// Check for GitLab
	if strings.Contains(repoURL, "gitlab.com") || strings.Contains(repoURL, "gitlab.") {
		return "gitlab"
	}

	return ""
}

// GetOrCreateProviderForURL 获取或创建匹配 URL 的 provider
// 优先使用已配置的 providers，如果没有匹配则尝试创建匿名 provider
//
// 参数：
//   - repoURL: 仓库 URL
//   - configuredProviders: 已配置的 provider 配置列表
//
// 返回：
//   - provider.Provider: 匹配的 provider 实例
//   - error: 如果无法找到或创建 provider
func GetOrCreateProviderForURL(repoURL string, configuredProviders []config.ProviderConfig) (provider.Provider, error) {
	// 1. 先尝试已配置的 providers
	for _, pc := range configuredProviders {
		opts := &provider.ProviderOptions{
			Token:              pc.Token,
			BaseURL:            pc.URL,
			InsecureSkipVerify: pc.InsecureSkipVerify,
		}

		p, err := provider.Create(pc.Type, opts)
		if err != nil {
			logger.Warn("Failed to create provider",
				zap.String("type", pc.Type),
				zap.Error(err),
			)
			continue
		}

		if p.MatchesURL(repoURL) {
			logger.Debug("Using configured provider",
				zap.String("provider_type", pc.Type),
				zap.String("repo_url", repoURL),
			)
			return p, nil
		}
	}

	// 2. 没有配置的 provider 匹配，尝试创建匿名 provider
	providerType := DetectProviderFromURL(repoURL)
	if providerType == "" {
		return nil, fmt.Errorf("unsupported repository URL: %s", repoURL)
	}

	// 创建匿名 provider（无 token = 公开访问）
	anonymousProvider, err := provider.Create(providerType, &provider.ProviderOptions{
		Token:              "",
		BaseURL:            "",
		InsecureSkipVerify: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create anonymous %s provider: %w", providerType, err)
	}

	if !anonymousProvider.MatchesURL(repoURL) {
		return nil, fmt.Errorf("provider %s does not match URL: %s", providerType, repoURL)
	}

	logger.Info("Using anonymous provider for public repository",
		zap.String("provider_type", providerType),
		zap.String("repo_url", repoURL),
	)

	return anonymousProvider, nil
}
