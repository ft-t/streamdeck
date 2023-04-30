package main

type config struct {
	ApiUrl               string            `json:"apiUrl"`
	BrowserUrl           string            `json:"browserUrl"`
	IntervalSeconds      int               `json:"intervalSeconds"`
	ResponseJSONSelector string            `json:"responseJSONSelector"`
	ResponseMapper       map[string]string `json:"responseMapper"`
	Headers              map[string]string `json:"headers"`
	TemplateParameters   map[string]string `json:"parameters"`
	TitlePrefix          string            `json:"titlePrefix"`
}
