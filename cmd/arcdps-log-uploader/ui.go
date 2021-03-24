package main

import (
	"fmt"
	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	log "github.com/sirupsen/logrus"
	"sync"
)

func openLink(link *walk.LinkLabelLink) {
	openBrowser(link.URL())
}

var refreshTextArea func() = nil

var changeCallback func(arcLog *ArcLog) = nil

func startUi() error {
	var tv *walk.TableView
	var prog *walk.ProgressBar
	var button *walk.PushButton
	var outputTextArea *walk.TextEdit = nil
	var model *ArcLogModel = nil

	changeCallback = func(arcLog *ArcLog) {
		model.PublishRowChanged(model.IndexOf(arcLog))
		updateProgress(model, prog)
		updateText(model, outputTextArea)
	}

	isBrowsableAllowed := walk.NewMutableCondition()
	MustRegisterCondition("isBrowseAllowed", isBrowsableAllowed)

	isRetryAllowed := walk.NewMutableCondition()
	MustRegisterCondition("isRetryAllowed", isRetryAllowed)

	model = new(ArcLogModel)

	refreshTextArea = func() {
		output := generateMessageText(model.items)
		_ = outputTextArea.SetText(output)
	}
	var err error

	window := MainWindow{
		Title:   "Arcdps Log Uploader & Formatter",
		MinSize: Size{Width: 900, Height: 200},
		Size:    Size{Width: 1120, Height: 800},
		Layout:  Grid{Columns: 1},
		OnDropFiles: func(files []string) {
			onFilesDrop(files, model, prog, outputTextArea)
		},
		Icon: 2,
		Children: []Widget{
			Composite{
				Layout:             VBox{MarginsZero: true, Spacing: 2},
				StretchFactor:      0,
				AlwaysConsumeSpace: false,
				Name:               "Header",
				Children: []Widget{
					LinkLabel{Text: `1. Drop the arcdps log files into this window. - 2. Wait until the logs are uploaded to <a href="https://dps.report/">dps.report</a> - 3. Optional: Deselect logs if desired. - 4. Copy the Text from the right panel into discord.`, OnLinkActivated: openLink},
					Label{Text: "If you are using Windows 10, I highly recommend enabling log compression in arcdps options."},
					Label{Text: "Due to rate limiting, bulk uploading 40 or more logs at once can take quite a while."},
				},
			},
			//TextEdit{AssignTo: &filesTextEdit},
			HSplitter{
				StretchFactor: 100,
				Children: []Widget{
					TableView{
						Name:             "tv",
						StretchFactor:    17,
						AssignTo:         &tv,
						AlternatingRowBG: true,
						CheckBoxes:       true,
						ColumnsOrderable: true,
						MultiSelection:   true,
						ContextMenuItems: []MenuItem{
							Action{
								Text:    "Retry",
								Enabled: Bind("isRetryAllowed"),
								OnTriggered: func() {
									selectedIndexes := tv.SelectedIndexes()
									for _, index := range selectedIndexes {
										arcLog := model.items[index]
										if arcLog.status == Error {
											log.Debugf("Reqeue requested: %v", arcLog)
											go queueUpload(arcLog)
										}
									}
								},
							},
							Action{
								Text:    "Open Log in Browser",
								Enabled: Bind("tv.SelectedCount == 1 && isBrowseAllowed"),
								OnTriggered: func() {
									selectedIndexes := tv.SelectedIndexes()
									arcLog := model.items[selectedIndexes[0]]
									go openBrowser(arcLog.report.Permalink)
								},
							},
						},
						OnItemActivated: func() {
							if tv.CurrentIndex() < 0 {
								return
							}
							currentItem := model.items[tv.CurrentIndex()]
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
							fmt.Printf("SelectedIndexes: %v\n", tv.SelectedIndexes())
							_ = isBrowsableAllowed.SetSatisfied(checkBrowsable(tv, model))
							_ = isRetryAllowed.SetSatisfied(shouldRetryBeAllowed(tv, model))
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
						OnLinkActivated: openLink,
					},
					HSpacer{StretchFactor: 2},
					Label{Text: "© Xyaren", Enabled: false},
				},
			},
		},
	}

	_, err = window.Run()
	return err
}

func shouldRetryBeAllowed(tv *walk.TableView, model *ArcLogModel) bool {
	if len(tv.SelectedIndexes()) <= 0 {
		return false
	}
	indexes := tv.SelectedIndexes()
	for _, index := range indexes {
		if model.items[index].status == Error {
			return true
		}
	}
	return false
}

func checkBrowsable(tv *walk.TableView, model *ArcLogModel) bool {
	if len(tv.SelectedIndexes()) == 1 {
		arcLog := model.items[tv.SelectedIndexes()[0]]
		if arcLog.status == Done && arcLog.report != nil && len(arcLog.report.Permalink) > 0 {
			return true
		}
	}
	return false
}

func onFilesDrop(files []string, model *ArcLogModel, prog *walk.ProgressBar, outputTextArea *walk.TextEdit) {
	for _, file := range files {

		//handle if item already exists in list
		possibleIndex, existingItem := fileAlreadyInList(model, file)
		if possibleIndex >= 0 {
			if existingItem.report == nil {
				go queueUpload(existingItem)
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

		go queueUpload(newElem)
	}
	updateProgress(model, prog)
	//model.Sort(model.sortColumn,model.sortOrder)
}

func updateText(model *ArcLogModel, area *walk.TextEdit) {
	output := generateMessageText(model.items)
	_ = area.SetText(output)
}

func queueUpload(newElem *ArcLog) {
	entry := QueueEntry{
		arcLog: newElem,
		callback: func(report *DpsReportResponse, err error) {
			if err != nil {
				newElem.status = Error
				newElem.errorMessage = err
			} else {
				newElem.status = Done
				newElem.report = report
				newElem.checked = true
			}
			changeCallback(newElem)
		},
		onChange: func() {
			changeCallback(newElem)
		},
	}

	newElem.status = WaitingInQueue
	changeCallback(newElem)

	// queue entry
	uploadQueue <- entry
}

var progressBarLock sync.Mutex

func updateProgress(model *ArcLogModel, progressBar *walk.ProgressBar) {
	progressBarLock.Lock()
	progressBar.SetRange(0, len(model.items))

	var count = 0
	for _, v := range model.items {
		if v.status == Done || v.status == Error {
			// Append desired values to slice
			count = count + 1
		}
	}
	progressBar.SetValue(count)
	progressBarLock.Unlock()
}
