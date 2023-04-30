package main

import "context"

type github interface {
	GetPullStatus(ctx context.Context, url string) (interface{}, error)
}
