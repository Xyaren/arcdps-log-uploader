package model

type LogStatus int

const (
	Outstanding LogStatus = iota
	WaitingInQueue
	WaitingRateLimiting
	WaitingRateLimitingHard
	Uploading
	Done
	Error
)

type DetailedStatus int

const (
	True DetailedStatus = iota
	False
	ForcedFalse
)

type ArcLog struct {
	Checked      bool
	File         string
	Status       LogStatus
	ErrorMessage error
	Report       *DpsReportResponse
	Detailed     DetailedStatus
	Anonymized   bool
}
