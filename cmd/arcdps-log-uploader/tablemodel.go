package main

import (
	"fmt"
	"github.com/lxn/walk"
	"path/filepath"
	"sort"
	"time"
)

type ArcLogModel struct {
	walk.TableModelBase
	walk.SorterBase
	sortColumn int
	sortOrder  walk.SortOrder
	items      []*ArcLog
}

// Called by the TableView to sort the model.
func (m *ArcLogModel) Sort(col int, order walk.SortOrder) error {
	m.sortColumn, m.sortOrder = col, order

	sortFuncs := []func(a, b *ArcLog) bool{
		func(a, b *ArcLog) bool {
			return a.file < b.file
		},
		func(a, b *ArcLog) bool {
			return a.status < b.status
		},
		func(a, b *ArcLog) bool {
			comparisonResult, oneIsMissing := modelUnavailable(a, b)
			if oneIsMissing {
				return comparisonResult
			}
			return a.report.Permalink < b.report.Permalink
		},
		func(a, b *ArcLog) bool {
			comparisonResult, oneIsMissing := modelUnavailable(a, b)
			if oneIsMissing {
				return comparisonResult
			}
			return time.Time(a.report.EncounterTime).Before(time.Time(b.report.EncounterTime))
		},
		func(a, b *ArcLog) bool {
			comparisonResult, oneIsMissing := modelUnavailable(a, b)
			if oneIsMissing {
				return comparisonResult
			}
			return a.report.Encounter.Duration < b.report.Encounter.Duration
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

func modelUnavailable(a *ArcLog, b *ArcLog) (bool, bool) {
	if a.report == nil && b.report == nil {
		return false, true
	} else if a.report == nil {
		return false, true
	} else if b.report == nil {
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

	valueFunc := []func(item *ArcLog) interface{}{
		func(item *ArcLog) interface{} {
			return filepath.Base(item.file)
		},
		func(item *ArcLog) interface{} {
			switch item.status {
			case Outstanding:
				return "Outstanding"
			case WaitingInQueue:
				return "Waiting (Queue)"
			case WaitingRateLimitingHard:
			case WaitingRateLimiting:
				return "Waiting (Rate Limit)"
			case Uploading:
				return "Uploading"
			case Done:
				return "Done"
			case Error:
				return fmt.Sprintf("Error (%v)", item.errorMessage)
			}
			return "Unknown"
		},
		func(item *ArcLog) interface{} {
			if item.report != nil {
				return item.report.Permalink
			}
			return ""
		},
		func(item *ArcLog) interface{} {
			if item.report != nil {
				return time.Time(item.report.EncounterTime)
			}
			return ""
		},
		func(item *ArcLog) interface{} {
			if item.report != nil {
				out := time.Time{}.Add(time.Duration(item.report.Encounter.Duration) * time.Second)
				return out.Format("04m 05s")
			}
			return ""
		},
	}
	return valueFunc[col](item)
}

// Called by the TableView to retrieve if a given row is checked.
func (m *ArcLogModel) Checked(row int) bool {
	return m.items[row].checked
}

// Called by the TableView when the user toggled the check box of a given row.
func (m *ArcLogModel) SetChecked(row int, checked bool) error {
	item := m.items[row]
	if checked {
		if item.status == Done {
			item.checked = checked
			refreshTextArea()
		}
	} else {
		item.checked = checked
	}
	return nil
}

func (m ArcLogModel) IndexOf(item *ArcLog) int {
	for i, v := range m.items {
		if v == item {
			return i
		}
	}
	return -1
}

func fileAlreadyInList(model *ArcLogModel, file string) (int, *ArcLog) {
	for i, item := range model.items {
		if item.file == file {
			return i, item
		}
	}
	return -1, nil
}
