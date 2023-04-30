package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/imroc/req/v3"
	"github.com/pkg/errors"
	"github.com/valyala/fastjson"
	"meow.tf/streamdeck/sdk"
)

var isJobRunning = false

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
			lg.Debug().Msgf("sending request to %v", globalConfig.ApiUrl)
			resp, err := req.C().NewRequest().Get(globalConfig.ApiUrl)

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
	if globalConfig.TitlePrefix != "" {
		sb.WriteString(strings.ReplaceAll(globalConfig.TitlePrefix, "\\n", "\n") + "\n")
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

	if strings.HasPrefix(mapped, "http") || strings.HasSuffix(mapped, ".png") {
		if sb.Len() > 0 {
			sdk.SetTitle(contextApp, sb.String(), 0)
		}

		if strings.HasSuffix(mapped, ".png") {
			fileData, err := encodeFileToBase64(filepath.Join("images", mapped))
			if err != nil {
				lg.Err(errors.Wrap(err, "image file not found")).Send()
				sdk.SetImage(contextApp, "", 0)
				sdk.ShowAlert(contextApp)
				return
			}
			fileData = fmt.Sprintf("data:image/png;base64, %v", fileData)
			lg.Info().Msgf("seding to image %v", fileData)
			sdk.SetImage(contextApp, fileData, 0)
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

func encodeFileToBase64(filename string) (string, error) {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(fileContent)
	return encoded, nil
}
