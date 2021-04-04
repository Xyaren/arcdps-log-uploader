package model

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/utils"
	"golang.org/x/time/rate"
)

type Encounter struct {
	Duration int `json:"duration"`
}

type DpsReportResponse struct {
	ID            string         `json:"id"`
	Error         string         `json:"error"`
	Permalink     string         `json:"permalink"`
	Encounter     Encounter      `json:"encounter"`
	EncounterTime utils.JSONTime `json:"encounterTime"`
}

var client = utils.NewRateLimitedClient(rate.NewLimiter(rate.Every(10*time.Second), 45))
var rateLimitedUntil *time.Time

var UploadQueue = make(chan QueueEntry, 1000)
var wg sync.WaitGroup

type UploadOptions struct {
	DetailedWvw bool
	Anonymous   bool
}

type QueueEntry struct {
	ArcLog   *ArcLog
	Options  *UploadOptions
	OnDone   func(*DpsReportResponse, error)
	OnChange func()
}

func StartWorkerGroup() {
	// start the worker
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go worker(UploadQueue)
	}
}

func CloseQueue() {
	close(UploadQueue)
}

func worker(jobChan <-chan QueueEntry) {
	for job := range jobChan {
		options := job.Options
		file := job.ArcLog.File

		report, err := uploadFile(file, options, func(status LogStatus) {
			job.ArcLog.Status = status
			job.OnChange()
		})
		if job.ArcLog.Detailed == True && !options.DetailedWvw {
			job.ArcLog.Detailed = ForcedFalse
		}
		job.OnDone(report, err)
	}
}

func uploadFile(path string, options *UploadOptions, callback func(status LogStatus)) (*DpsReportResponse, error) {
	filename := filepath.Base(path)
	logger := log.WithField("filename", filename)

	logger.Info("Uploading File ", path)

	responseBody, err := doRequest(callback, path, options, logger)
	if err != nil {
		return nil, err
	}

	dpsReportResponse := DpsReportResponse{}
	jsonErr := json.Unmarshal(responseBody, &dpsReportResponse)
	if jsonErr != nil {
		logger.Errorf("Could not unmarshal json response due to: %s \n Response: \n %s", jsonErr, string(responseBody))
		return nil, fmt.Errorf("could not read dps.report response %w", jsonErr)
	}

	return &dpsReportResponse, nil
}

func doRequest(callback func(status LogStatus), path string, options *UploadOptions, logger *log.Entry) ([]byte, error) {
	return doRequestInternal(callback, path, options, logger)
}

func doRequestInternal(callback func(status LogStatus), path string, options *UploadOptions, logger *log.Entry) ([]byte, error) {
	req, err := buildRequest(path, options)
	if err != nil {
		return nil, err
	}

	callback(WaitingRateLimiting)

	waitUntilUnbanned()

	res, err := client.Do(req, func() { callback(Uploading) })
	if err != nil {
		logger.Fatal(err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode == 429 {
		retryAfter, _ := strconv.Atoi(res.Header.Get("Retry-After"))
		logger.Warnf("Request Rate Limited. Trying again in %v", retryAfter)
		callback(WaitingRateLimitingHard)

		timeToUnban := time.Duration(retryAfter+2) * time.Second
		freeTime := time.Now().Add(timeToUnban)
		rateLimitedUntil = &freeTime
		time.Sleep(timeToUnban)
		return doRequestInternal(callback, path, options, logger)
	}
	if res.StatusCode == 500 {
		if options.DetailedWvw {
			logger.Warnf("Upload failed due to server error. Trying again without detailed wvw")
			options.DetailedWvw = false
			return doRequestInternal(callback, path, options, logger)
		}
	}
	if res.StatusCode != 200 {
		logger.Errorf("dps.report responded with status %v (%v). Header: %v", res.StatusCode, res.Status, res.Header)
		return nil, fmt.Errorf("upload failed: %v", res.Status)
	}
	responseBody, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		logger.Fatal(readErr)
	}
	return responseBody, nil
}

func waitUntilUnbanned() {
	if rateLimitedUntil != nil {
		if rateLimitedUntil.After(time.Now()) {
			sub := time.Until(*rateLimitedUntil)
			log.Debugf("Waiting to be unblocked (in %v)", sub)
			<-time.After(sub)
		}
	}
}

func buildRequest(path string, options *UploadOptions) (*http.Request, error) {
	requestURL, urlErr := buildURL(options)
	if urlErr != nil {
		return nil, urlErr
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(file.Name()))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, requestURL.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	return req, nil
}

func buildURL(options *UploadOptions) (*url.URL, error) {
	u, err := url.Parse("https://dps.report/uploadContent?json=1&generator=ei")
	if err != nil {
		log.Fatal(err)
	}
	q := u.Query()

	if options.DetailedWvw {
		q.Set("detailedwvw", "true")
	} else {
		q.Set("detailedwvw", "false")
	}

	if options.Anonymous {
		q.Set("anonymous", "true")
	} else {
		q.Set("anonymous", "false")
	}
	u.RawQuery = q.Encode()
	return u, err
}
