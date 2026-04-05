package external

import (
	"errors"
	"strconv"
	"strings"
)

func isGitHubHost(host string) bool {
	switch strings.ToLower(host) {
	case githubHost, githubWWWHost:
		return true
	default:
		return false
	}
}

func isStackOverflowHost(host string) bool {
	switch strings.ToLower(host) {
	case stackOverflowHost, stackOverflowWHost:
		return true
	default:
		return false
	}
}

func parseGitHubRepoPath(rawPath string) (string, string, error) {
	const requiredGitHubPathParts = 2

	parts := strings.Split(strings.Trim(rawPath, "/"), "/")
	if len(parts) != requiredGitHubPathParts {
		return "", "", errors.New("invalid github repository path")
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", "", errors.New("empty github owner or repo")
	}
	if strings.HasSuffix(repo, ".git") {
		return "", "", errors.New("git suffix is not allowed")
	}
	return owner, repo, nil
}

func parseStackOverflowQuestionPath(rawPath string) (int64, error) {
	parts := strings.Split(strings.Trim(rawPath, "/"), "/")
	if len(parts) < 2 || parts[0] != "questions" {
		return 0, errors.New("invalid stackoverflow question path")
	}
	questionID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || questionID <= 0 {
		return 0, errors.New("invalid stackoverflow question id")
	}
	return questionID, nil
}
