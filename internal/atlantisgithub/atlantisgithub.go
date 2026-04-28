package atlantisgithub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cresta/gogit"
	"github.com/cresta/gogithub"
)

func CheckOutTerraformRepo(ctx context.Context, gitHubClient gogithub.GitHub, cloner *gogit.Cloner, repo string) (*gogit.Repository, error) {
	token, err := gitHubClient.GetAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}
	// https://docs.github.com/en/developers/apps/building-github-apps/authenticating-with-github-apps#http-based-git-access-by-an-installation
	githubRepoURL := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, repo)
	repository, err := cloner.Clone(ctx, githubRepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repo: %w", err)
	}
	return repository, nil
}

// GetLatestCommitSHA fetches the latest commit SHA for a given branch from GitHub API
func GetLatestCommitSHA(ctx context.Context, gitHubClient gogithub.GitHub, repo string, branch string) (string, error) {
	token, err := gitHubClient.GetAccessToken(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	// Parse repo into owner and repo name
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid repo format, expected 'owner/repo', got: %s", repo)
	}
	owner := parts[0]
	repoName := parts[1]

	// Use GitHub REST API to get the latest commit SHA for the branch
	// GET /repos/{owner}/{repo}/git/ref/{ref}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/ref/heads/%s", owner, repoName, branch)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch branch ref: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch branch ref: status %d, body: %s", resp.StatusCode, string(body))
	}

	var refResponse struct {
		Ref    string `json:"ref"`
		NodeID string `json:"node_id"`
		URL    string `json:"url"`
		Object struct {
			SHA  string `json:"sha"`
			Type string `json:"type"`
			URL  string `json:"url"`
		} `json:"object"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&refResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if refResponse.Object.SHA == "" {
		return "", fmt.Errorf("no SHA found in response for branch %s", branch)
	}

	return refResponse.Object.SHA, nil
}
