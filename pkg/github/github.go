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

	pr, _, err := client.PullRequests.Get(ctx, owner, repo, prNum)
	if err != nil {
		return nil, err
	}

	//reviews, _, err := client.PullRequests.ListReviews(ctx, owner, repo, prNum, nil)
	//if err != nil {
	//	return nil, err
	//}

	checks, err := g.getChecks(ctx, client, owner, repo, pr.GetHead().GetSHA())
	if err != nil {
		return nil, err
	}

	if !pr.GetMergeable() {
		return &CanMerge{
			StatusText: StatusTextFail,
			Reason:     fmt.Sprintf("Mergable - false. MergableState - %v", pr.GetMergeableState()),
			Checks:     checks,
		}, nil
	}

	if pr.GetMergeableState() == "clean" {
		return &CanMerge{
			Checks:     checks,
			StatusText: StatusTextSuccess,
		}, nil
	}

	if pr.GetMergeableState() == "unstable" {
		return &CanMerge{
			Checks:     checks,
			StatusText: StatusTextWorkflowRunning,
		}, nil
	}

	if pr.GetMergeableState() == "blocked" || pr.GetMergeableState() == "dirty" {
		allChecksSuccess := true

		for _, c := range checks {
			if c.Conclusion != "success" {
				allChecksSuccess = false
				break
			}
		}

		if !allChecksSuccess { // looks like it requires intervention from us
			return &CanMerge{
				Checks:     checks,
				StatusText: StatusTextFail,
			}, nil
		}

		return &CanMerge{ // it means that ci is passing, but there are branch constreins or review requested
			Checks:     checks,
			StatusText: StatusTextSuccess,
		}, nil
	}

	return &CanMerge{
		StatusText: "Unknown",
		Reason:     "Unknown",
		Checks:     checks,
	}, nil
}

func (g *Github) getChecks(
	ctx context.Context,
	client *github.Client,
	owner,
	repo string,
	sha string,
) ([]*CheckStatus, error) {
	opts := github.ListCheckRunsOptions{}
	checkRuns, _, err := client.Checks.ListCheckRunsForRef(
		ctx,
		owner,
		repo,
		sha,
		&opts,
	)
	if err != nil {
		return nil, err
	}

	checksStatus := make([]*CheckStatus, len(checkRuns.CheckRuns))
	for i, checkRun := range checkRuns.CheckRuns {
		checksStatus[i] = &CheckStatus{
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
