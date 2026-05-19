package processor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    Event
		expectError bool
		error       error
	}{
		{
			name:        "empty line should return errEmptyLine",
			input:       "",
			expectError: true,
			error:       errEmptyLine,
		},
		{
			name:        "whitespace only line should return errEmptyLine",
			input:       "   ",
			expectError: true,
			error:       errEmptyLine,
		},
		{
			name:  "valid event without extra param - register",
			input: "[14:00:00] 1 1",
			expected: Event{
				Time:     parseTestTime(t, "14:00:00"),
				PlayerID: 1,
				EventID:  1,
			},
		},
		{
			name:  "valid event without extra param - enter dungeon",
			input: "[14:10:00] 2 2",
			expected: Event{
				Time:     parseTestTime(t, "14:10:00"),
				PlayerID: 2,
				EventID:  2,
			},
		},
		{
			name:  "valid event with extra param - damage",
			input: "[14:27:00] 2 11 60",
			expected: Event{
				Time:       parseTestTime(t, "14:27:00"),
				PlayerID:   2,
				EventID:    11,
				ExtraParam: "60",
			},
		},
		{
			name:  "valid event with extra param - restore health",
			input: "[14:49:02] 1 10 80",
			expected: Event{
				Time:       parseTestTime(t, "14:49:02"),
				PlayerID:   1,
				EventID:    10,
				ExtraParam: "80",
			},
		},
		{
			name:  "multi-word extra param - cannot continue",
			input: "[15:00:00] 1 9 out of health potions",
			expected: Event{
				Time:       parseTestTime(t, "15:00:00"),
				PlayerID:   1,
				EventID:    9,
				ExtraParam: "out of health potions",
			},
		},
		{
			name:        "insufficient parameters",
			input:       "[14:00:00] 1",
			expectError: true,
		},
		{
			name:        "invalid time format - wrong brackets",
			input:       "14:00:00 1 1",
			expectError: true,
			error:       errInvalidTimeFormat,
		},
		{
			name:        "invalid time format - wrong time",
			input:       "[25:00:00] 1 1",
			expectError: true,
			error:       errInvalidTimeFormat,
		},
		{
			name:        "invalid player ID - not a number",
			input:       "[14:00:00] abc 1",
			expectError: true,
		},
		{
			name:        "invalid event ID - not a number",
			input:       "[14:00:00] 1 abc",
			expectError: true,
		},
		{
			name:  "extra spaces between fields",
			input: "[14:00:00]   1    2",
			expected: Event{
				Time:     parseTestTime(t, "14:00:00"),
				PlayerID: 1,
				EventID:  2,
			},
		},
		{
			name:  "trailing and leading spaces",
			input: "  [14:00:00] 1 2  ",
			expected: Event{
				Time:     parseTestTime(t, "14:00:00"),
				PlayerID: 1,
				EventID:  2,
			},
		},
		{
			name:  "event with large player ID",
			input: "[14:00:00] 999999 11 100",
			expected: Event{
				Time:       parseTestTime(t, "14:00:00"),
				PlayerID:   999999,
				EventID:    11,
				ExtraParam: "100",
			},
		},
		{
			name:  "event with negative damage",
			input: "[14:00:00] 1 11 -50",
			expected: Event{
				Time:       parseTestTime(t, "14:00:00"),
				PlayerID:   1,
				EventID:    11,
				ExtraParam: "-50",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := parseEvent(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				if tt.error != nil {
					assert.ErrorIs(t, err, tt.error)
				}
				return
			}

			require.NoError(t, err)
			assert.True(t, tt.expected.Time.Equal(event.Time), "Time mismatch: expected %v, got %v", tt.expected.Time, event.Time)
			assert.Equal(t, tt.expected.PlayerID, event.PlayerID)
			assert.Equal(t, tt.expected.EventID, event.EventID)
			assert.Equal(t, tt.expected.ExtraParam, event.ExtraParam)
		})
	}
}

func parseTestTime(t *testing.T, timeStr string) time.Time {
	t.Helper()
	parsed, err := time.Parse(layout, timeStr)
	require.NoError(t, err, "Failed to parse time in test helper")
	return parsed
}
