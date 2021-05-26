package ui

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/lxn/walk"
	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/model"
)

type ArcLogModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      []*model.ArcLog
}

var (
	checkmark = string([]byte{0xE2, 0x9C, 0x94})
	cross     = string([]byte{0xE2, 0x9C, 0x96})
)

// Called by the TableView to sort the model.
func (m *ArcLogModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order

	sortFuncs := []func(a, b *model.ArcLog) bool{
		func(a, b *model.ArcLog) bool {
			return a.File < b.File
		},
		func(a, b *model.ArcLog) bool {
			return a.Status < b.Status
		},
		func(a, b *model.ArcLog) bool {
			comparisonResult, oneIsMissing := modelUnavailable(a, b)
			if oneIsMissing {
				return comparisonResult
			}
			return time.Time(a.Report.EncounterTime).Before(time.Time(b.Report.EncounterTime))
		},
		func(a, b *model.ArcLog) bool {
			comparisonResult, oneIsMissing := modelUnavailable(a, b)
			if oneIsMissing {
				return comparisonResult
			}
			return a.Report.Encounter.Duration < b.Report.Encounter.Duration
		},
		func(a, b *model.ArcLog) bool {
			return a.Detailed < b.Detailed
		},
		func(a, b *model.ArcLog) bool {
			return a.Anonymized && b.Anonymized
		},
		func(a, b *model.ArcLog) bool {
			comparisonResult, oneIsMissing := modelUnavailable(a, b)
			if oneIsMissing {
				return comparisonResult
			}
			return a.Report.Permalink < b.Report.Permalink
		},
	}

	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]

		c := func(ls bool) bool {
			if m.sortOrder == walk.SortAscending {
				return ls
			}
			return !ls
		}
		funcs := sortFuncs[m.sortColumn]
		if funcs != nil {
			return c(funcs(a, b))
		}
		panic(fmt.Sprintf("sort function missing for column %v", m.sortColumn))
	})

	return m.SorterBase.Sort(col, order)
}

func modelUnavailable(a, b *model.ArcLog) (result, oneOrMoreIsMissing bool) {
	switch {
	case a.Report == nil && b.Report == nil:
		return false, true
	case a.Report == nil:
		return false, true
	case b.Report == nil:
		return true, true
	}
	return false, false
}

func (m *ArcLogModel) RefreshTable() {
	// Notify TableView and other interested parties about the reset.
	m.PublishRowsReset()
}

// Called by the TableView from SetModel and every time the model publishes a
// RowsReset event.
func (m *ArcLogModel) RowCount() int {
	return len(m.items)
}

// Called by the TableView when it needs the text to display for a given cell.
func (m *ArcLogModel) Value(row, col int) interface{} {
	item := m.items[row]

	valueFunc := []func(item *model.ArcLog) interface{}{
		func(item *model.ArcLog) interface{} {
			return filepath.Base(item.File)
		},
		func(item *model.ArcLog) interface{} {
			switch item.Status {
			case model.Outstanding:
				return "Outstanding"
			case model.WaitingInQueue:
				return "Waiting (Queue)"
			case model.WaitingRateLimitingHard:
				return "Waiting (Rate Limited)"
			case model.WaitingRateLimiting:
				return "Waiting (Rate Limit)"
			case model.Uploading:
				return "Uploading"
			case model.Done:
				return "Done"
			case model.Error:
				return fmt.Sprintf("Error (%v)", item.ErrorMessage)
			}
			return "Unknown"
		},
		func(item *model.ArcLog) interface{} {
			if item.Report != nil {
				return time.Time(item.Report.EncounterTime)
			}
			return ""
		},
		func(item *model.ArcLog) interface{} {
			if item.Report != nil {
				out := time.Time{}.Add(time.Duration(item.Report.Encounter.Duration) * time.Second)
				return out.Format("04m 05s")
			}
			return ""
		},
		func(item *model.ArcLog) interface{} {
			switch item.Detailed {
			case model.True:
				return checkmark
			case model.False:
				return cross
			case model.ForcedFalse:
				return "Forced Off"
			}
			return ""
		},
		func(item *model.ArcLog) interface{} {
			if item.Anonymized {
				return checkmark
			}
			return cross
		},
		func(item *model.ArcLog) interface{} {
			if item.Report != nil {
				return item.Report.Permalink
			}
			return ""
		},
	}
	return valueFunc[col](item)
}

// Called by the TableView to retrieve if a given row is checked.
func (m *ArcLogModel) Checked(row int) bool {
	return m.items[row].Checked
}

// Called by the TableView when the user toggled the check box of a given row.
func (m *ArcLogModel) SetChecked(row int, checked bool) error {
	item := m.items[row]
	if checked {
		if item.Status == model.Done {
			item.Checked = checked
		}
	} else {
		item.Checked = checked
	}
	reprocessOutput()
	return nil
}

func (m *ArcLogModel) IndexOf(item *model.ArcLog) int {
	for i, v := range m.items {
		if v == item {
			return i
		}
	}
	return -1
}

func fileAlreadyInList(m *ArcLogModel, file string) (int, *model.ArcLog) {
	for i, item := range m.items {
		if item.File == file {
			return i, item
		}
	}
	return -1, nil
}
