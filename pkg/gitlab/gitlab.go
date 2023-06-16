package gitlab

import (
	"context"
	"fmt"
	"strings"

	"github.com/imroc/req/v3"
	"github.com/pkg/errors"

	"github.com/ft-t/streamdeck/pkg/types"
)

type Gitlab struct {
	accessToken string
	cl          *
}

func (g *Gitlab) GetPullStatus(ctx context.Context, url string) (*types.CanMerge, error) {
	splitURL := strings.Split(url, "/")
	projectID := splitURL[3]
	mergeRequestID := splitURL[5]

	url = fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/merge_requests/%v/pipelines?access_token=%s", projectID, mergeRequest["id"], accessToken)


	// Send request to GitLab API to get merge request details
	searchUrl := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/merge_requests?search=%s&access_token=%s", projectID, mergeRequestID, g.accessToken)
	resp, err := req.R().Get(searchUrl)
	if err != nil {
		return nil, errors.WithStack(err)
	}


}
