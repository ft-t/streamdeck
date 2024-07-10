package main

import "context"

type ScriptExecutor interface {
	Execute(
		ctx context.Context,
		script string,
		rawBody string,
		statusCode int,
		headers map[string]string,
		templateVariables map[string]string,
	) (string, error)
}
