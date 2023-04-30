package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/imroc/req/v3"
	"github.com/pkg/errors"
	"github.com/valyala/fastjson"
	"meow.tf/streamdeck/sdk"
)

var isJobRunning = false

func runTemplate(input string) string {
	if len(globalConfig.TemplateParameters) == 0 || len(input) == 0 {
		return input
	}

	parsed, err := template.New("any").Parse(input)
	if err != nil {
		lg.Err(errors.Wrapf(err, "can not parse template - %v", err))
		return input
	}

	var buf bytes.Buffer

	if err = parsed.Execute(&buf, globalConfig.TemplateParameters); err != nil {
		lg.Err(errors.Wrapf(err, "can not parse template - %v", err))
		return input
	}

	return buf.String()
}

func process() {
	if isJobRunning {
		return
	}

	isJobRunning = true

	for context.Background().Err() == nil {
		interval := 30
		if globalConfig.IntervalSeconds > 0 {
			interval = globalConfig.IntervalSeconds
		}

		func() {
			apiUrl := runTemplate(globalConfig.ApiUrl)
			lg.Debug().Msgf("sending request to %v", apiUrl)
			resp, err := req.C().NewRequest().Get(apiUrl)

			if err != nil {
				sdk.ShowAlert(contextApp)
				lg.Err(errors.Wrap(err, "error sending request")).Send()
				return
			}

			lg.Debug().Msgf("got raw response %v", resp.String())
			parsed, err := fastjson.ParseBytes(resp.Bytes())
			if err != nil {
				sdk.ShowAlert(contextApp)
				lg.Err(errors.Wrap(err, "error parsing request")).Send()
				return
			}

			value := parsed.String()
			if globalConfig.ResponseJSONSelector != "" {
				selectorVal := parsed.Get(globalConfig.ResponseJSONSelector)
				if selectorVal == nil || selectorVal.Type() == fastjson.TypeNull {
					sdk.ShowAlert(contextApp)
					lg.Err(errors.New("no data found by ResponseJSONSelector")).Send()
					return
				}

				if selectorVal.Type() == fastjson.TypeString {
					if s, errSBytes := selectorVal.StringBytes(); errSBytes != nil {
						sdk.ShowAlert(contextApp)
						lg.Err(errors.Wrap(err, "error parsing StringBytes")).Send()
						return
					} else {
						value = string(s)
					}
				} else {
					value = selectorVal.String()
				}
			}

			lg.Debug().Msgf("got raw value %v", string(value))

			handleResponse(value)
		}()

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func handleResponse(response string) {
	var sb strings.Builder
	prefix := runTemplate(globalConfig.TitlePrefix)
	if prefix != "" {
		sb.WriteString(strings.ReplaceAll(prefix, "\\n", "\n") + "\n")
	}

	if len(globalConfig.ResponseMapper) == 0 {
		if response == "" {
			sb.WriteString("!! NO !!\n !! MAPPING !!")
		} else {
			sb.WriteString(response)
		}

		sdk.SetTitle(contextApp, sb.String(), 0)
		sdk.SetImage(contextApp, "", 0)
		sdk.ShowOk(contextApp)

		return
	}

	mapped, ok := globalConfig.ResponseMapper[response]
	def, defaultOk := globalConfig.ResponseMapper["*"]

	if !ok && defaultOk {
		mapped = def
	} else if !ok && !defaultOk {
		sb.WriteString("!! NO !!\n !! MAPPING !!")
		lg.Error().Msgf("response mapper not found for value - %v", response)

		sdk.SetTitle(contextApp, sb.String(), 0)
		sdk.SetImage(contextApp, "", 0)
		sdk.ShowAlert(contextApp)

		return
	}

	if strings.HasPrefix(mapped, "http") || strings.HasSuffix(mapped, ".png") || strings.HasSuffix(mapped, ".svg") {
		if sb.Len() > 0 {
			sdk.SetTitle(contextApp, sb.String(), 0)
		}

		if strings.HasSuffix(mapped, ".png") || strings.HasSuffix(mapped, ".svg") {
			fileData, err := readFile(filepath.Join("images", mapped))

			if err != nil {
				lg.Err(errors.Wrap(err, "image file not found")).Send()
				sdk.SetImage(contextApp, "", 0)
				sdk.ShowAlert(contextApp)
				return
			}

			imageData := ""
			if strings.HasSuffix(mapped, ".png") {
				imageData = fmt.Sprintf("data:image/png;base64, %v", base64.StdEncoding.EncodeToString(fileData))
			} else if strings.HasSuffix(mapped, ".svg") {
				imageData = fmt.Sprintf("data:image/svg+xml;charset=utf8,%v", string(fileData))
			}

			lg.Info().Msgf("seding to image %v", fileData)
			sdk.SetImage(contextApp, imageData, 0)
		}

		sdk.ShowOk(contextApp)
	} else {
		sb.WriteString(mapped)
		sdk.SetTitle(contextApp, sb.String(), 0)
		sdk.SetImage(contextApp, "", 0)
		sdk.ShowOk(contextApp)

		return
	}
}

func readFile(filename string) ([]byte, error) {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return fileContent, nil
}
