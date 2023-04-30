package github

type CheckStatus struct {
	Name       string
	State      string
	Conclusion string
}

const (
	StatusTextNone            = "none"
	StatusTextSuccess         = "success"
	StatusTextFail            = "fail"
	StatusTextWorkflowRunning = "running-workflow"
)

type CanMerge struct {
	StatusText string
	Reason     string
	Checks     []*CheckStatus
}
