package ui

//goland:noinspection GoLinterLocal
import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"github.com/lxn/win"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
	log "github.com/sirupsen/logrus"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/model"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/utils"
)

func openLink(link *walk.LinkLabelLink) {
	utils.OpenBrowser(link.URL())
}

var logFilePattern = regexp.MustCompile(`(?m).+\.(evtc(\.zip)?|zevtc)$`)

var refreshTextArea func()

var changeCallback func(arcLog *model.ArcLog)
var latestVersion *selfupdate.Release

type Options struct {
	DetailedWvw bool
	Anonymous   bool
}

var options = new(Options)

//nolint:funlen
func StartUI() error {
	options.DetailedWvw = true

	var mainWindow *walk.MainWindow
	var tv *walk.TableView
	var prog *walk.ProgressBar
	var button *walk.PushButton
	var outputTextArea *walk.TextEdit
	var tableModel *ArcLogModel
	var db *walk.DataBinder
	var versionLinkLabel *walk.LinkLabel

	changeCallback = func(arcLog *model.ArcLog) {
		tableModel.PublishRowChanged(tableModel.IndexOf(arcLog))
		updateProgress(tableModel, prog)
		updateText(tableModel, outputTextArea)
	}

	isBrowsableAllowed := walk.NewMutableCondition()
	declarative.MustRegisterCondition("isBrowseAllowed", isBrowsableAllowed)

	isRetryAllowed := walk.NewMutableCondition()
	declarative.MustRegisterCondition("isRetryAllowed", isRetryAllowed)

	tableModel = new(ArcLogModel)

	refreshTextArea = func() {
		output := generateMessageText(tableModel.items)
		_ = outputTextArea.SetText(output)
	}

	go checkForUpdate(&versionLinkLabel)

	window := declarative.MainWindow{
		AssignTo: &mainWindow,
		Title:    "ArcDps Log Uploader & Formatter",
		MinSize:  declarative.Size{Width: 900, Height: 200},
		Size:     declarative.Size{Width: 1300, Height: 800},
		Layout:   declarative.Grid{Columns: 1},
		OnDropFiles: func(files []string) {
			onDrop(files, tableModel, prog)
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
										arcLog := tableModel.items[index]
										if arcLog.Status == model.Error {
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
									arcLog := tableModel.items[selectedIndexes[0]]
									go utils.OpenBrowser(arcLog.Report.Permalink)
								},
							},
						},
						OnItemActivated: func() {
							if tv.CurrentIndex() < 0 {
								return
							}
							currentItem := tableModel.items[tv.CurrentIndex()]
							if currentItem.Report != nil {
								utils.OpenBrowser(currentItem.Report.Permalink)
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
							item := tableModel.items[style.Row()]

							if item.Checked {
								if style.Row()%2 == 0 {
									style.BackgroundColor = walk.RGB(159, 215, 255)
								} else {
									style.BackgroundColor = walk.RGB(143, 199, 239)
								}
							}
						},
						Model: tableModel,
						OnSelectedIndexesChanged: func() {
							fmt.Printf("SelectedIndexes: %v\n", tv.SelectedIndexes())
							_ = isBrowsableAllowed.SetSatisfied(checkBrowsable(tv, tableModel))
							_ = isRetryAllowed.SetSatisfied(shouldRetryBeAllowed(tv, tableModel))
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
							go utils.CopyToClipboard(outputTextArea.Text())
						},
						MinSize: declarative.Size{Width: 150},
					},
				},
			},
			declarative.Composite{
				Layout: declarative.HBox{MarginsZero: true, Spacing: 2},
				Name:   "Footer",
				Children: []declarative.Widget{
					declarative.LinkLabel{Text: "New Releases, Issue Tracker and Source Code on " +
						"<a href=\"https://github.com/Xyaren/arcdps-log-uploader\">Github</a>",
						OnLinkActivated: openLink,
					},
					declarative.HSpacer{},
					declarative.Composite{
						StretchFactor: 5,
						Layout:        declarative.HBox{MarginsZero: true, Spacing: 2},
						Children: []declarative.Widget{
							declarative.LinkLabel{
								Font: declarative.Font{
									Bold:      true,
									Underline: true,
								},
								GraphicsEffects: []walk.WidgetGraphicsEffect{},
								Visible:         false,
								AssignTo:        &versionLinkLabel,
								OnLinkActivated: func(link *walk.LinkLabelLink) {
									versionLinkLabel.SetVisible(false)
									defer versionLinkLabel.SetVisible(true)

									answer := walk.MsgBox(mainWindow, "Update",
										"Do you want to update now?\nThis will restart the application after the update.",
										walk.MsgBoxYesNo|walk.MsgBoxIconQuestion|walk.MsgBoxTaskModal)
									log.Debugf("Clicked: %v", answer)
									if win.LOWORD(uint32(answer)) == walk.DlgCmdYes {
										mainWindow.SetEnabled(false) // prevent any input
										go func() {
											utils.DoUpdate(latestVersion)
											err := utils.ForkExec()
											if err != nil {
												panic(err)
											}
											defer syscall.Exit(0)
										}()
										walk.MsgBox(mainWindow, "Update in Progress", "Updating now...\n"+
											"The application will restart itself",
											walk.DlgCmdOK|walk.MsgBoxTaskModal|walk.MsgBoxIconInformation)
									}
								},
								Text: "An new version is available: v%v - <a>Click here to update!</a>",
							},
						},
					},
					declarative.HSpacer{},
					declarative.Composite{
						StretchFactor: 1,
						Layout:        declarative.HBox{MarginsZero: true, Spacing: 2},
						Children: []declarative.Widget{
							declarative.Label{Text: "© Xyaren", Enabled: false},
							declarative.Label{Text: " - ", Enabled: false},
							declarative.Label{Text: utils.Version(), Enabled: false},
						},
					},
					// declarative.HSpacer{StretchFactor: 2},

				},
			},
		},
	}
	var err error
	_, err = window.Run()
	return err
}

func checkForUpdate(versionLinkLabel **walk.LinkLabel) {
	var currentIsLatest bool
	latestVersion, currentIsLatest = utils.CheckUpdate()

	if !currentIsLatest {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// wait
		for {
			if versionLinkLabel != nil || ctx.Err() != nil {
				break
			}
		}
		if versionLinkLabel != nil {
			_ = (*versionLinkLabel).SetText(
				strings.ReplaceAll((*versionLinkLabel).Text(),
					"%v",
					latestVersion.Version.String()))
			(*versionLinkLabel).SetVisible(true)
		}
	}
}

func shouldRetryBeAllowed(tv *walk.TableView, m *ArcLogModel) bool {
	if len(tv.SelectedIndexes()) == 0 {
		return false
	}
	indexes := tv.SelectedIndexes()
	for _, index := range indexes {
		if m.items[index].Status == model.Error {
			return true
		}
	}
	return false
}

func checkBrowsable(tv *walk.TableView, m *ArcLogModel) bool {
	if len(tv.SelectedIndexes()) == 1 {
		arcLog := m.items[tv.SelectedIndexes()[0]]
		if arcLog.Status == model.Done && arcLog.Report != nil && len(arcLog.Report.Permalink) > 0 {
			return true
		}
	}
	return false
}

func onDrop(files []string, m *ArcLogModel, prog *walk.ProgressBar) {
	for _, file := range files {
		// handle folder
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			foundFiles, _ := onFolderDrop(file)
			if len(foundFiles) > 0 {
				onDrop(foundFiles, m, prog)
			}
		}

		filename := strings.ToLower(filepath.Base(file))
		if logFilePattern.MatchString(filename) {
			// handle if item already exists in list
			possibleIndex, existingItem := fileAlreadyInList(m, file)
			if possibleIndex >= 0 {
				if existingItem.Report == nil {
					go queueUpload(existingItem)
				}
				continue
			}

			// create new
			newElem := new(model.ArcLog)
			newElem.Status = model.Outstanding
			newElem.File = file
			m.items = append(m.items, newElem)
			var index = len(m.items) - 1
			m.PublishRowsInserted(index, index)

			go queueUpload(newElem)
		} else {
			log.Debugf("%v does not match the arc log file patern", filename)
		}
	}
	updateProgress(m, prog)
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

func updateText(m *ArcLogModel, area *walk.TextEdit) {
	output := generateMessageText(m.items)
	_ = area.SetText(output)
}

func queueUpload(newElem *model.ArcLog) {
	uploadOptions := getCurrentOptions()
	newElem.Anonymized = uploadOptions.Anonymous
	if uploadOptions.DetailedWvw {
		newElem.Detailed = model.True
	} else {
		newElem.Detailed = model.False
	}

	onDone := func(report *model.DpsReportResponse, err error) {
		if err != nil {
			newElem.Status = model.Error
			newElem.ErrorMessage = err
		} else {
			newElem.Status = model.Done
			newElem.Report = report
			newElem.Checked = true
		}
		changeCallback(newElem)
	}

	entry := model.QueueEntry{
		ArcLog:  newElem,
		Options: &uploadOptions,
		OnDone:  onDone,
		OnChange: func() {
			changeCallback(newElem)
		},
	}

	newElem.Status = model.WaitingInQueue
	changeCallback(newElem)

	// queue entry
	model.UploadQueue <- entry
}

func getCurrentOptions() model.UploadOptions {
	uploadOptions := model.UploadOptions{
		DetailedWvw: options.DetailedWvw,
		Anonymous:   options.Anonymous,
	}
	return uploadOptions
}

var progressBarLock sync.Mutex

func updateProgress(m *ArcLogModel, progressBar *walk.ProgressBar) {
	progressBarLock.Lock()
	progressBar.SetRange(0, len(m.items))

	var count = 0
	for _, v := range m.items {
		if v.Status == model.Done || v.Status == model.Error {
			// Append desired values to slice
			count++
		}
	}
	progressBar.SetValue(count)
	progressBarLock.Unlock()
}