package main

import (
	"bytes"
	"os"
	"text/template"

	"github.com/pkg/errors"
)

func runTemplate(input string, cfg *config) string {
	if len(cfg.TemplateParameters) == 0 || len(input) == 0 {
		return input
	}

	parsed, err := template.New("any").Parse(input)
	if err != nil {
		lg.Err(errors.Wrapf(err, "can not parse template - %v", err))
		return input
	}

	var buf bytes.Buffer

	if err = parsed.Execute(&buf, cfg.TemplateParameters); err != nil {
		lg.Err(errors.Wrapf(err, "can not parse template - %v", err))
		return input
	}

	return buf.String()
}

func readFile(filename string) ([]byte, error) {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return fileContent, nil
}
