package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func generateMessageText(entries []*ArcLog) string {
	var result []ArcLog
	for _, e := range entries {
		if e.report != nil {
			if e.checked {
				result = append(result, *e)
			}
		}
	}

	if len(result) < 1 {
		return ""
	}

	sort.SliceStable(result, func(i, j int) bool {
		return time.Time(result[i].report.EncounterTime).Before(time.Time(result[j].report.EncounterTime))
	})

	var lines []string

	for _, entry := range result {
		output := ""
		output += "`" + time.Time(entry.report.EncounterTime).Format("15:04") + "`"
		output += " "

		out := time.Time{}.Add(time.Duration(entry.report.Encounter.Duration) * time.Second)
		output += "`" + out.Format("04m 05s") + "`"
		output += " "
		output += "<"
		output += entry.report.Permalink
		output += ">"
		lines = append(lines, output)
	}

	var messages []string
	var currentMessage = ""
	for _, line := range lines {
		var currentMessagePlusThisLine = currentMessage + "\r\n" + line
		if len(currentMessagePlusThisLine) > (2000 - 40) {
			messages = append(messages, currentMessage)
			currentMessage = "\r\n" + line
		} else {
			currentMessage = currentMessagePlusThisLine
		}
	}
	if len(currentMessage) > 0 {
		messages = append(messages, currentMessage)
	}

	headline := "**Training " + time.Time(result[0].report.EncounterTime).Format("02.01.2006") + "**"

	for i, message := range messages {
		paging := ""
		if len(messages) > 1 {
			paging = fmt.Sprintf(" (%d/%d)", i+1, len(messages))
		}
		messages[i] = headline + paging + message
	}

	return strings.Join(messages, "\r\n\r\n--------\r\n\r\n")
}
