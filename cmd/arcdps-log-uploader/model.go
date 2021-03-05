package main

type LogStatus int

const (
	Outstanding LogStatus = iota
	Uploaded
	Error
)

type ArcLog struct {
	checked      bool
	file         string
	report       *DpsReportResponse
	status       LogStatus
	errorMessage error
}
