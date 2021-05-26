package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xyaren/arcdps-log-uploader/cmd/arcdps-log-uploader/model"
)

type ProcessedArcLog struct {
	arcLog        *model.ArcLog
	encounterTime time.Time
}

func generateMessageText(entries []*model.ArcLog, title string) string {
	var dates = make(map[time.Time]struct{}) // make a "set"
	var result []ProcessedArcLog
	for _, arcLog := range entries {
		if arcLog.Report != nil {
			if arcLog.Checked {
				encounterTime := time.Time((*arcLog).Report.EncounterTime)
				arcLog := ProcessedArcLog{arcLog, encounterTime}
				result = append(result, arcLog)
				dateOnly := time.Date(encounterTime.Year(), encounterTime.Month(), encounterTime.Day(), 0, 0, 0, 0, time.Local)
				dates[dateOnly] = struct{}{}
			}
		}
	}

	if len(result) < 1 {
		return ""
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].encounterTime.Before(result[j].encounterTime)
	})

	var lines []string

	for _, entry := range result {
		output := ""
		if len(dates) > 1 {
			output += "`" + entry.encounterTime.Format("02.01.2006") + "`"
			output += " "
		}
		output += "`" + entry.encounterTime.Format("15:04") + "`"
		output += " "

		out := time.Time{}.Add(time.Duration(entry.arcLog.Report.Encounter.Duration) * time.Second)
		output += "`" + out.Format("04m 05s") + "`"
		output += " "
		output += "<"
		output += entry.arcLog.Report.Permalink
		output += ">"
		lines = append(lines, output)
	}

	firstDateTime := result[0].encounterTime
	headline := headline(title, firstDateTime, len(dates) > 1)

	var messages []string
	var currentMessage = ""
	for _, line := range lines {
		var currentMessagePlusThisLine = currentMessage + "\r\n" + line
		if len(currentMessagePlusThisLine) > (2000 - len(headline) - 10) {
			messages = append(messages, currentMessage)
			currentMessage = "\r\n" + line
		} else {
			currentMessage = currentMessagePlusThisLine
		}
	}
	if len(currentMessage) > 0 {
		messages = append(messages, currentMessage)
	}

	for i, message := range messages {
		paging := ""
		if len(messages) > 1 {
			paging = fmt.Sprintf(" (%d/%d)", i+1, len(messages))
		}
		messages[i] = headline + paging + message
	}

	return strings.Join(messages, "\r\n\r\n--------\r\n\r\n")
}

func headline(title string, time time.Time, multipleDays bool) string {
	var elements = make([]string, 0)
	trimmedTitle := strings.TrimSpace(title)
	if len(trimmedTitle) > 0 {
		elements = append(elements, trimmedTitle)
	}

	if !multipleDays {
		elements = append(elements, time.Format("02.01.2006"))
	}

	return "**" + strings.Join(elements, " ") + "**"
}
