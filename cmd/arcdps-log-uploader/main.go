//go:generate goversioninfo ./_res/versioninfo.json

package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"os"
	"sync"
	"time"
)

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

var refreshTextArea func() = nil

func main() {
	setupLogging()
	log.Info("Starting")

	var logTable *walk.TableView
	var prog *walk.ProgressBar
	var button *walk.PushButton
	var outputTextArea *walk.TextEdit = nil
	var model *ArcLogModel = nil

	rand.Seed(time.Now().UnixNano())
	model = new(ArcLogModel)

	refreshTextArea = func() {
		output := generateMessageText(model.items)
		_ = outputTextArea.SetText(output)
	}
	var err error

	_, err = MainWindow{
		Title:   "ArcDPS Log to Discord Formatter",
		MinSize: Size{Width: 900, Height: 200},
		Size:    Size{Width: 1120, Height: 800},
		Layout:  Grid{Columns: 1},
		OnDropFiles: func(files []string) {
			onFilesDrop(files, model, prog, outputTextArea)
		},
		Icon: 2,
		Children: []Widget{
			Composite{
				Layout:             HBox{MarginsZero: true, Spacing: 2},
				StretchFactor:      0,
				AlwaysConsumeSpace: false,
				Name:               "Header",
				Children: []Widget{
					Label{Text: "1. Drop the arcdps log files into this window. - 2. Wait until the logs are uploaded. - 3. Optional: Deselect logs if desired. - 4. Copy the Text from the right panel into discord."},
				},
			},
			//TextEdit{AssignTo: &filesTextEdit},
			HSplitter{
				StretchFactor: 100,
				Children: []Widget{
					TableView{
						StretchFactor:    17,
						AssignTo:         &logTable,
						AlternatingRowBG: true,
						CheckBoxes:       true,
						ColumnsOrderable: true,
						MultiSelection:   true,

						OnItemActivated: func() {
							if logTable.CurrentIndex() < 0 {
								return
							}
							currentItem := model.items[logTable.CurrentIndex()]
							if currentItem.report != nil {
								openBrowser(currentItem.report.Permalink)
							}
						},
						Columns: []TableViewColumn{
							{Title: "File", Width: 150},
							{Title: "Status", Width: 85},
							{Title: "Link", Width: 260},
							{Title: "Date", Format: "2006-01-02 15:04:05", Width: 120},
							{Title: "Duration", Width: 60},
						},
						StyleCell: func(style *walk.CellStyle) {
							item := model.items[style.Row()]

							if item.checked {
								if style.Row()%2 == 0 {
									style.BackgroundColor = walk.RGB(159, 215, 255)
								} else {
									style.BackgroundColor = walk.RGB(143, 199, 239)
								}
							}
						},
						Model: model,
						OnSelectedIndexesChanged: func() {
							fmt.Printf("SelectedIndexes: %v\n", logTable.SelectedIndexes())
						},
					},

					TextEdit{
						StretchFactor: 10,
						AssignTo:      &outputTextArea,
						ReadOnly:      true,
						HScroll:       true,
						VScroll:       true,
					},
				},
			},

			Composite{
				Layout: HBox{MarginsZero: true},

				StretchFactor: 1,
				Children: []Widget{
					ProgressBar{
						AssignTo: &prog,
					},
					PushButton{
						AssignTo: &button,
						Text:     "Copy to Clipboard",
						OnClicked: func() {
							go copyToClipboard(outputTextArea.Text())
						},
						MinSize: Size{Width: 150},
					},
				},
			},
			Composite{
				Layout:             HBox{MarginsZero: true, Spacing: 2},
				StretchFactor:      -1,
				AlwaysConsumeSpace: false,
				Name:               "Footer",
				Children: []Widget{
					LinkLabel{Text: `New Releases, Issue Tracker and Source Code at <a href="https://github.com/Xyaren/arcdps-log-uploader">https://github.com/Xyaren/arcdps-log-uploader</a>`,
						OnLinkActivated: func(link *walk.LinkLabelLink) {
							openBrowser(link.URL())
						},
					},
					HSpacer{StretchFactor: 2},
					Label{Text: "© Xyaren", Enabled: false},
				},
			},
		},
	}.Run()
	if err != nil {
		panic(err)
	}
	log.Info("Bye")
}

func setupLogging() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true, PadLevelText: true})
}

func onFilesDrop(files []string, model *ArcLogModel, prog *walk.ProgressBar, outputTextArea *walk.TextEdit) {
	for _, file := range files {

		//handle if item already exists in list
		possibleIndex, existingItem := fileAlreadyInList(model, file)
		if possibleIndex >= 0 {
			if existingItem.report == nil {
				go upload(file, existingItem, func() {
					model.PublishRowChanged(model.IndexOf(existingItem))
					updateProgress(model, prog)
					updateText(model, outputTextArea)
				})
			}
			continue
		}

		// create new
		newElem := new(ArcLog)
		newElem.status = Outstanding
		newElem.file = file
		model.items = append(model.items, newElem)
		var index = len(model.items) - 1
		model.PublishRowsInserted(index, index)

		go upload(file, newElem, func() {
			of := model.IndexOf(newElem)
			model.PublishRowChanged(of)
			updateProgress(model, prog)
			updateText(model, outputTextArea)
		})
	}
	updateProgress(model, prog)
	//model.Sort(model.sortColumn,model.sortOrder)
}

func updateText(model *ArcLogModel, area *walk.TextEdit) {
	output := generateMessageText(model.items)
	_ = area.SetText(output)
}

func fileAlreadyInList(model *ArcLogModel, file string) (int, *ArcLog) {
	for i, item := range model.items {
		if item.file == file {
			return i, item
		}
	}
	return -1, nil
}

var progressBarLock sync.Mutex

func updateProgress(model *ArcLogModel, prog *walk.ProgressBar) {
	progressBarLock.Lock()
	prog.SetRange(0, len(model.items))

	var count = 0
	for _, v := range model.items {
		if v.status != Outstanding {
			// Append desired values to slice
			count = count + 1
		}
	}
	prog.SetValue(count)
	progressBarLock.Unlock()
}

func upload(file string, newElem *ArcLog, callback func()) {

	uploadFile, err2 := UploadFile(file)
	if err2 != nil {
		newElem.status = Error
		newElem.errorMessage = err2
	} else {
		newElem.status = Uploaded
		newElem.report = uploadFile
		newElem.checked = true
	}

	callback()
}

func copyToClipboard(text string) {
	if err := walk.Clipboard().SetText(text); err != nil {
		log.Print("Copy: ", err)
	}
}
