package types

type CheckStatus struct {
	Name       string
	State      string
	Conclusion string
}

const (
	StatusTextNone            = "none"
	StatusTextSuccess         = "success"
	StatusTextFail            = "fail"
	StatusTextMerged          = "merged"
	StatusTextWorkflowRunning = "running-workflow"
)

type CanMerge struct {
	StatusText string
	Reason     string
	Checks     []*CheckStatus
}
