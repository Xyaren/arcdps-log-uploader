package main

import (
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

	sort.SliceStable(m.items, func(i, j int) bool {
		a, b := m.items[i], m.items[j]

		c := func(ls bool) bool {
			if m.sortOrder == walk.SortAscending {
				return ls
			}

			return !ls
		}

		switch m.sortColumn {
		case 0:
			return c(a.checked && b.checked)
		case 1:
			return c(a.file < b.file)
		case 2:
			comparisonResult, oneIsMissing := modelUnavailable(a, b, c)
			if oneIsMissing {
				return comparisonResult
			}
			return c(a.report.Permalink < b.report.Permalink)
		case 3:
			comparisonResult, oneIsMissing := modelUnavailable(a, b, c)
			if oneIsMissing {
				return comparisonResult
			}
			return c(time.Time(a.report.EncounterTime).Before(time.Time(b.report.EncounterTime)))
		case 4:
			comparisonResult, oneIsMissing := modelUnavailable(a, b, c)
			if oneIsMissing {
				return comparisonResult
			}
			return c(a.report.Encounter.Duration < b.report.Encounter.Duration)
		}

		panic("unreachable")
	})

	return m.SorterBase.Sort(col, order)
}

func modelUnavailable(a *ArcLog, b *ArcLog, c func(ls bool) bool) (bool, bool) {
	if a.report == nil && b.report == nil {
		return c(false), true
	} else if a.report == nil {
		return c(false), true
	} else if b.report == nil {
		return c(true), true
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

	switch col + 1 {
	case 1:
		return item.checked
	case 2:
		return filepath.Base(item.file)
	case 3:
		if item.report != nil {
			return item.report.Permalink
		}
		return ""
	case 4:
		if item.report != nil {
			return time.Time(item.report.EncounterTime)
		}
		return ""
	case 5:
		if item.report != nil {
			out := time.Time{}.Add(time.Duration(item.report.Encounter.Duration) * time.Second)
			return out.Format("04m 05s")
		}
		return ""
	}

	panic("unexpected col")
}

// Called by the TableView to retrieve if a given row is checked.
func (m *ArcLogModel) Checked(row int) bool {
	return m.items[row].checked
}

// Called by the TableView when the user toggled the check box of a given row.
func (m *ArcLogModel) SetChecked(row int, checked bool) error {
	m.items[row].checked = checked
	refreshTextArea()
	return nil
}
