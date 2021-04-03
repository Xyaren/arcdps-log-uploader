package main

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
	checked      bool
	file         string
	status       LogStatus
	errorMessage error
	report       *DpsReportResponse
	detailed     DetailedStatus
	anonymized   bool
}
