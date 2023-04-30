package github

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v38/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type Github struct {
	apiKey string
}

func NewGithub(apiKey string) *Github {
	return &Github{
		apiKey: apiKey,
	}
}

func (g *Github) GetPullStatus(ctx context.Context, url string) (interface{}, error) {
	owner, repo, prNum, err := g.parsePRURL(url)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing PR URL")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: g.apiKey})
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	// Fetch the GitHub checks status for the PR
	checksStatus, err := g.getPRChecksStatus(ctx, client, owner, repo, prNum)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching PR checks status")
	}

	for _, status := range checksStatus {
		fmt.Printf("%s: %s || %s\n", status.Name, status.State, status.Conclusion)
	}

	// action_required

	return nil, nil
}

func (g *Github) getPRChecksStatus(
	ctx context.Context,
	client *github.Client,
	owner, repo string, prNum int,
) ([]CheckStatus, error) {
	commitSHA, _, err := client.PullRequests.Get(ctx, owner, repo, prNum)
	if err != nil {
		return nil, err
	}

	opts := github.ListCheckRunsOptions{}
	checkRuns, _, err := client.Checks.ListCheckRunsForRef(
		ctx,
		owner,
		repo,
		commitSHA.GetHead().GetSHA(),
		&opts,
	)
	if err != nil {
		return nil, err
	}

	checksStatus := make([]CheckStatus, len(checkRuns.CheckRuns))
	for i, checkRun := range checkRuns.CheckRuns {
		checksStatus[i] = CheckStatus{
			Name:       checkRun.GetName(),
			State:      checkRun.GetStatus(),
			Conclusion: checkRun.GetConclusion(),
		}
	}

	return checksStatus, nil
}

func (g *Github) parsePRURL(prURL string) (string, string, int, error) {
	u, err := url.Parse(prURL)
	if err != nil {
		return "", "", 0, err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) < 5 {
		return "", "", 0, fmt.Errorf("Invalid PR URL")
	}

	owner := parts[1]
	repo := parts[2]
	prNum, err := strconv.Atoi(parts[4])
	if err != nil {
		return "", "", 0, fmt.Errorf("Invalid PR number in URL")
	}

	return owner, repo, prNum, nil
}
