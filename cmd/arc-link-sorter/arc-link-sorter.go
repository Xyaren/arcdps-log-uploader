package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"log"
	"math/rand"
	"sync"
	"time"
)

type ArcLog struct {
	checked bool
	file    string
	report  *DpsReportResponse
}

func refresh() {

}

var refreshTextArea func() = nil

func main() {
	var logTable *walk.TableView
	var prog *walk.ProgressBar
	var button *walk.PushButton
	var outputTextArea *walk.TextEdit = nil
	var model *ArcLogModel = nil

	rand.Seed(time.Now().UnixNano())
	model = new(ArcLogModel)
	model.RefreshTable()

	refreshTextArea = func() {
		output := generateMessageText(model.items)
		_ = outputTextArea.SetText(output)
	}

	var err error
	_, err = MainWindow{
		Title:   "ArcDPS Log to Discord Formatter",
		MinSize: Size{Width: 900, Height: 800},
		Size:    Size{Width: 1100, Height: 800},
		Layout:  VBox{},
		OnDropFiles: func(files []string) {
			onFilesDrop(files, model, prog, outputTextArea)
		},
		Children: []Widget{
			//TextEdit{AssignTo: &filesTextEdit},
			HSplitter{
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
							{Title: "Include", Width: 50},
							{Title: "File", Width: 140},
							{Title: "Link", Width: 260},
							{Title: "Date", Format: "2006-01-02 15:04:05", Width: 125},
							{Title: "Duration", Width: 70},
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
			HSplitter{
				Children: []Widget{
					PushButton{
						AssignTo: &button,
						Text:     "Copy to Clipboard",
						OnClicked: func() {
							go copyToClipboard(outputTextArea.Text())
						},
						MaxSize: Size{Width: 50},
					},
					ProgressBar{
						AssignTo: &prog,
					},
				},
			},
		},
	}.Run()
	if err != nil {
		panic(err)
	}
}

func onFilesDrop(files []string, model *ArcLogModel, prog *walk.ProgressBar, outputTextArea *walk.TextEdit) {
	for _, file := range files {

		//handle if item already exists in list
		possibleIndex, existingItem := fileAlreadyInList(model, file)
		if possibleIndex >= 0 {
			if existingItem.report == nil {
				go upload(file, existingItem, func() {
					existingItem.checked = true
					model.PublishRowChanged(possibleIndex)
					updateProgress(model, prog)
					updateText(model, outputTextArea)
				})
			}
			continue
		}

		// reate new
		newElem := new(ArcLog)
		newElem.file = file
		model.items = append(model.items, newElem)
		var index = len(model.items) - 1
		//trigger download
		go upload(file, newElem, func() {
			newElem.checked = true
			model.PublishRowChanged(index)
			updateProgress(model, prog)
			updateText(model, outputTextArea)
		})

		updateProgress(model, prog)
	}
	model.RefreshTable()
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
		if v.report != nil {
			// Append desired values to slice
			count = count + 1
		}
	}
	prog.SetValue(count)
	progressBarLock.Unlock()
}

func upload(file string, newElem *ArcLog, callback func()) {
	defer callback()

	uploadFile, err2 := UploadFile(file)
	if err2 != nil {
		panic(err2)
	}
	newElem.report = uploadFile
}

func copyToClipboard(text string) {
	if err := walk.Clipboard().SetText(text); err != nil {
		log.Print("Copy: ", err)
	}
}
