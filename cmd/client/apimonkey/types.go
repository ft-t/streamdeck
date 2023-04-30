package main

type config struct {
	ApiUrl               string                 `json:"apiUrl"`
	BrowserUrl           string                 `json:"browserUrl"`
	IntervalSeconds      int                    `json:"intervalSeconds"`
	ResponseJSONSelector string                 `json:"responseJSONSelector"`
	ResponseMapper       map[string]interface{} `json:"responseToImage"`
}
