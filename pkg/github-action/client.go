package githubaction

import (
	"context"
	"errors"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func NewGithubClientFromEnv() (*github.Client, error) {
	ghToken := GetInputOrDefault("GH-TOKEN", "")
	if ghToken == "" {
		return nil, errors.New("missing input gh-token")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: ghToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	return github.NewClient(tc), nil
}
