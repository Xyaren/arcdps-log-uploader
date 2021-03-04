package main

import (
	"bytes"
	"context"
	"encoding/json"
	"golang.org/x/sync/semaphore"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
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

var sem = semaphore.NewWeighted(int64(10))
var restClient = http.Client{}

func UploadFile(path string) (*DpsReportResponse, error) {
	url := "https://dps.report/uploadContent?json=1&generator=ei"

	file, _ := os.Open(path)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", filepath.Base(file.Name()))
	_, _ = io.Copy(part, file)
	_ = writer.Close()
	_ = file.Close()

	req, err := http.NewRequest(http.MethodPost, url, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	if err != nil {
		log.Fatal(err)
	}

	_ = sem.Acquire(context.Background(), 1)
	defer sem.Release(1)

	res, getErr := restClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	responseBody, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	dpsReportResponse := DpsReportResponse{}
	jsonErr := json.Unmarshal(responseBody, &dpsReportResponse)
	if jsonErr != nil {
		log.Fatal(jsonErr, responseBody)
	}

	return &dpsReportResponse, nil

}
