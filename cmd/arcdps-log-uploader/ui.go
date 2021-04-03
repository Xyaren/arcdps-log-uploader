package main

//goland:noinspection GoLinterLocal
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	log "github.com/sirupsen/logrus"
)

func openLink(link *walk.LinkLabelLink) {
	openBrowser(link.URL())
}

var logFilePattern = regexp.MustCompile(`(?m).+\.(evtc(\.zip)?|zevtc)$`)

var refreshTextArea func()

var changeCallback func(arcLog *ArcLog)

type Options struct {
	DetailedWvw bool
	Anonymous   bool
}

var options = new(Options)

//nolint:funlen
func startUI() error {
	options.DetailedWvw = true

	var tv *walk.TableView
	var prog *walk.ProgressBar
	var button *walk.PushButton
	var outputTextArea *walk.TextEdit
	var model *ArcLogModel
	var db *walk.DataBinder

	changeCallback = func(arcLog *ArcLog) {
		model.PublishRowChanged(model.IndexOf(arcLog))
		updateProgress(model, prog)
		updateText(model, outputTextArea)
	}

	isBrowsableAllowed := walk.NewMutableCondition()
	declarative.MustRegisterCondition("isBrowseAllowed", isBrowsableAllowed)

	isRetryAllowed := walk.NewMutableCondition()
	declarative.MustRegisterCondition("isRetryAllowed", isRetryAllowed)

	model = new(ArcLogModel)

	refreshTextArea = func() {
		output := generateMessageText(model.items)
		_ = outputTextArea.SetText(output)
	}
	var err error

	window := declarative.MainWindow{
		Title:   "Arcdps Log Uploader & Formatter",
		MinSize: declarative.Size{Width: 900, Height: 200},
		Size:    declarative.Size{Width: 1300, Height: 800},
		Layout:  declarative.Grid{Columns: 1},
		OnDropFiles: func(files []string) {
			onDrop(files, model, prog)
		},
		Icon: 2,
		Children: []declarative.Widget{
			declarative.Composite{
				Layout:             declarative.VBox{MarginsZero: true, Spacing: 2},
				StretchFactor:      0,
				AlwaysConsumeSpace: false,
				Name:               "Header",
				Children: []declarative.Widget{
					declarative.LinkLabel{Text: "1. Drop the arcdps log files into this window.- " +
						"2. Wait until the logs are uploaded to <a href=\"https://dps.report/\">dps.report</a> - " +
						"3. Optional: Deselect logs if desired. - " +
						"4. Copy the Text from the right panel into discord.", OnLinkActivated: openLink},
					declarative.Label{Text: "If you are using Windows 10, I highly recommend enabling log compression in arcdps options."},
					declarative.Label{Text: "Due to rate limiting, bulk uploading 40 or more logs at once can take quite a while."},
				},
			},
			declarative.Composite{
				Layout:             declarative.HBox{MarginsZero: true, Spacing: 20},
				StretchFactor:      0,
				AlwaysConsumeSpace: false,
				Name:               "Options",
				DataBinder: declarative.DataBinder{
					AssignTo:       &db,
					Name:           "state",
					DataSource:     options,
					ErrorPresenter: declarative.ToolTipErrorPresenter{},
					AutoSubmit:     true,
				},
				Children: []declarative.Widget{
					declarative.CheckBox{
						Name:        "DetailedLogs",
						Text:        "Use Detailed WvW Logs if possible.",
						ToolTipText: "Detailed WvW is currently not possible for large log files. They will fallback to non-detailed upload.",
						Checked:     declarative.Bind("DetailedWvw"),
					},
					declarative.CheckBox{
						Name:        "AnonymousLogs",
						Text:        "Enable anonymized Reports",
						ToolTipText: "Replace player names in report.",
						Checked:     declarative.Bind("Anonymous"),
					},
				},
			},
			declarative.HSplitter{
				StretchFactor: 150,
				Children: []declarative.Widget{
					declarative.TableView{
						Name:             "tv",
						StretchFactor:    18,
						AssignTo:         &tv,
						AlternatingRowBG: true,
						CheckBoxes:       true,
						ColumnsOrderable: true,
						MultiSelection:   true,
						ContextMenuItems: []declarative.MenuItem{
							declarative.Action{
								Text:    "Retry",
								Enabled: declarative.Bind("isRetryAllowed"),
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
							declarative.Action{
								Text:    "Open Log in Browser",
								Enabled: declarative.Bind("tv.SelectedCount == 1 && isBrowseAllowed"),
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
						Columns: []declarative.TableViewColumn{
							{Title: "File", Width: 150},
							{Title: "Status", Width: 85},
							{Title: "Date", Format: "2006-01-02 15:04:05", Width: 120},
							{Title: "Duration", Width: 60},
							{Title: "Detailed", Width: 50},
							{Title: "Anonymized", Width: 70},
							{Title: "Link", Width: 260},
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
					declarative.TextEdit{
						StretchFactor: 10,
						AssignTo:      &outputTextArea,
						Text:          "",
						ReadOnly:      true,
						HScroll:       true,
						VScroll:       true,
					},
				},
			},

			declarative.Composite{
				Layout: declarative.HBox{MarginsZero: true},

				StretchFactor: 1,
				Children: []declarative.Widget{
					declarative.ProgressBar{
						AssignTo: &prog,
					},
					declarative.PushButton{
						AssignTo: &button,
						Text:     "Copy to Clipboard",
						OnClicked: func() {
							go copyToClipboard(outputTextArea.Text())
						},
						MinSize: declarative.Size{Width: 150},
					},
				},
			},
			declarative.Composite{
				Layout:             declarative.HBox{MarginsZero: true, Spacing: 2},
				StretchFactor:      -1,
				AlwaysConsumeSpace: false,
				Name:               "Footer",
				Children: []declarative.Widget{
					declarative.LinkLabel{Text: "New Releases, Issue Tracker and Source Code at " +
						"<a href=\"https://github.com/Xyaren/arcdps-log-uploader\">" +
						"https://github.com/Xyaren/arcdps-log-uploader" +
						"</a>",
						OnLinkActivated: openLink,
					},
					declarative.HSpacer{StretchFactor: 2},
					declarative.Label{Text: "© Xyaren", Enabled: false},
				},
			},
		},
	}

	_, err = window.Run()
	return err
}

func shouldRetryBeAllowed(tv *walk.TableView, model *ArcLogModel) bool {
	if len(tv.SelectedIndexes()) == 0 {
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

func onDrop(files []string, model *ArcLogModel, prog *walk.ProgressBar) {
	for _, file := range files {
		// handle folder
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			foundFiles, _ := onFolderDrop(file)
			if len(foundFiles) > 0 {
				onDrop(foundFiles, model, prog)
			}
		}

		filename := strings.ToLower(filepath.Base(file))
		if logFilePattern.MatchString(filename) {
			// handle if item already exists in list
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
		} else {
			log.Debugf("%v does not match the arc log file patern", filename)
		}
	}
	updateProgress(model, prog)
}

func onFolderDrop(file string) ([]string, error) {
	var folderFiles []string
	err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
		if path != file {
			folderFiles = append(folderFiles, path)
		}
		return nil
	})
	return folderFiles, err
}

func updateText(model *ArcLogModel, area *walk.TextEdit) {
	output := generateMessageText(model.items)
	_ = area.SetText(output)
}

func queueUpload(newElem *ArcLog) {
	uploadOptions := getCurrentOptions()
	newElem.anonymized = uploadOptions.anonymous
	if uploadOptions.detailedWvw {
		newElem.detailed = True
	} else {
		newElem.detailed = False
	}

	onDone := func(report *DpsReportResponse, err error) {
		if err != nil {
			newElem.status = Error
			newElem.errorMessage = err
		} else {
			newElem.status = Done
			newElem.report = report
			newElem.checked = true
		}
		changeCallback(newElem)
	}

	entry := QueueEntry{
		arcLog:  newElem,
		options: &uploadOptions,
		onDone:  onDone,
		onChange: func() {
			changeCallback(newElem)
		},
	}

	newElem.status = WaitingInQueue
	changeCallback(newElem)

	// queue entry
	uploadQueue <- entry
}

func getCurrentOptions() UploadOptions {
	uploadOptions := UploadOptions{
		detailedWvw: options.DetailedWvw,
		anonymous:   options.Anonymous,
	}
	return uploadOptions
}

var progressBarLock sync.Mutex

func updateProgress(model *ArcLogModel, progressBar *walk.ProgressBar) {
	progressBarLock.Lock()
	progressBar.SetRange(0, len(model.items))

	var count = 0
	for _, v := range model.items {
		if v.status == Done || v.status == Error {
			// Append desired values to slice
			count++
		}
	}
	progressBar.SetValue(count)
	progressBarLock.Unlock()
}
