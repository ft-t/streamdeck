package github_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ft-t/streamdeck/pkg/github"
)

func TestGetPrStatus(t *testing.T) {
	gh := github.NewGithub("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")

	ctx := context.TODO()
	//st, err := gh.GetPullStatus(ctx, "https://github.com/hyperledger/aries-framework-go/pull/3551")
	st, err := gh.GetPullStatus(ctx, "https://github.com/bloodyorg/pb/pull/47")
	assert.NoError(t, err)
	assert.NotNil(t, st)
}
