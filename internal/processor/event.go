package processor

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	eventTimeIdx = iota
	playerIDIdx
	eventIDIdx
	extraParamIdx
)

var (
	errEmptyLine         = fmt.Errorf("emtpy line")
	errInvalidTimeFormat = fmt.Errorf("invalid time format")
)

type Event struct {
	Time       time.Time
	PlayerID   int64
	EventID    int64
	ExtraParam string
}

func parseEvent(line string) (Event, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Event{}, errEmptyLine
	}

	parts := strings.Fields(line)
	if len(parts) < 3 {
		return Event{}, fmt.Errorf("not enough parameters in line: %s", line)
	}

	timeStr := parts[eventTimeIdx]
	eventTime, err := time.Parse(layout, timeStr[1:len(timeStr)-1])
	if err != nil {
		return Event{}, fmt.Errorf("%w: time parsing failed: %w", errInvalidTimeFormat, err)
	}

	playerID, err := strconv.Atoi(parts[playerIDIdx])
	if err != nil {
		return Event{}, fmt.Errorf("invalid EventID: %s", parts[playerIDIdx])
	}

	eventID, err := strconv.Atoi(parts[eventIDIdx])
	if err != nil {
		return Event{}, fmt.Errorf("invalid PlayerID: %s", parts[eventIDIdx])
	}

	extraParam := ""
	if len(parts) > 3 {
		extraParam = strings.Join(parts[extraParamIdx:], " ")
	}

	return Event{
		Time:       eventTime,
		EventID:    int64(eventID),
		PlayerID:   int64(playerID),
		ExtraParam: extraParam,
	}, nil
}
