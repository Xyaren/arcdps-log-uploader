package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Encounter struct {
	Duration int `json:"duration"`
}

type DpsReportResponse struct {
	Id            string    `json:"id"`
	Error         string    `json:"error"`
	Permalink     string    `json:"permalink"`
	Encounter     Encounter `json:"encounter"`
	EncounterTime jsonTime  `json:"encounterTime"`
}

var client = NewRateLimitedClient(rate.NewLimiter(rate.Every(10*time.Second), 45))
var rateLimitedUntil *time.Time = nil

var uploadQueue = make(chan QueueEntry, 10000)
var wg sync.WaitGroup

type QueueEntry struct {
	arcLog   *ArcLog
	callback func(*DpsReportResponse, error)
	onChange func()
}

//func createRateLimiter() *rate.Limiter {
//	//var interval time.Duration = time.Duration((1/5)*1000) * time.Millisecond
//	return
//}

func startWorkerGroup() {
	// start the worker
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go worker(uploadQueue)
	}
}

func closeQueue() {
	close(uploadQueue)
}

func worker(jobChan <-chan QueueEntry) {
	for next := range jobChan {
		report, err := uploadFile(next.arcLog.file, func(status LogStatus) {
			next.arcLog.status = status
			next.onChange()
		})
		next.callback(report, err)
	}
}

func uploadFile(path string, callback func(status LogStatus)) (*DpsReportResponse, error) {
	filename := filepath.Base(path)
	logger := log.WithField("filename", filename)

	logger.Info("Uploading File ", path)

	responseBody, err := doRequest(callback, path, logger)
	if err != nil {
		return nil, err
	}

	dpsReportResponse := DpsReportResponse{}
	jsonErr := json.Unmarshal(responseBody, &dpsReportResponse)
	if jsonErr != nil {
		logger.Errorf("Could not unmarshal json response due to: %s \n Response: \n %s", jsonErr, string(responseBody))
		return nil, fmt.Errorf("could not read dps.report response %v", jsonErr)
	}

	return &dpsReportResponse, nil

}

func doRequest(callback func(status LogStatus), path string, logger *log.Entry) ([]byte, error) {
	//sem.Acquire(context.Background(), 1)
	//defer sem.Release(1) // release semaphore later

	return doRequestInternal(callback, path, logger)
}

func doRequestInternal(callback func(status LogStatus), path string, logger *log.Entry) ([]byte, error) {
	req, err := buildRequest(path)
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
		return doRequestInternal(callback, path, logger)
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

func buildRequest(path string) (*http.Request, error) {
	url := "https://dps.report/uploadContent?json=1&generator=ei"

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

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", writer.FormDataContentType())
	return req, nil
}
