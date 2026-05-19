package processor

import (
	"context"
	"strings"
	"testing"

	"github.com/exerayy/dungeon-challenge-processor/internal/dungeoncfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGameState_ProcessEvents(t *testing.T) {
	tests := []struct {
		name           string
		cfg            dungeoncfg.ParsedCfg
		events         string
		expectedLogs   []string
		expectedReport []string
	}{
		{
			name: "success from example",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    2,
				Monsters:  2,
				OpenTime:  parseTestTime(t, "14:05:00"),
				CloseTime: parseTestTime(t, "16:05:00"),
			},
			events: `[14:00:00] 1 1
					[14:00:00] 2 1
					[14:10:00] 2 2
					[14:10:00] 3 2
					[14:11:00] 2 5
					[14:12:00] 3 3
					[14:14:00] 2 3
					[14:27:00] 2 11 60
					[14:29:00] 2 11 50
					[14:40:00] 1 2
					[14:41:00] 1 3
					[14:44:00] 1 11 50
					[14:45:00] 1 3
					[14:48:00] 1 4
					[14:48:00] 1 6
					[14:49:00] 1 11 25
					[14:49:02] 1 10 80
					[14:50:00] 1 11 65
					[14:59:00] 1 7
					[15:04:00] 1 8`,
			expectedLogs: []string{
				"[14:00:00] Player [1] registered",
				"[14:00:00] Player [2] registered",
				"[14:10:00] Player [2] entered the dungeon",
				"[14:10:00] Player [3] is disqualified",
				"[14:11:00] Player [2] makes imposible move [5]",
				"[14:14:00] Player [2] killed the monster",
				"[14:27:00] Player [2] recieved [60] of damage",
				"[14:29:00] Player [2] recieved [50] of damage",
				"[14:29:00] Player [2] is dead",
				"[14:40:00] Player [1] entered the dungeon",
				"[14:41:00] Player [1] killed the monster",
				"[14:44:00] Player [1] recieved [50] of damage",
				"[14:45:00] Player [1] killed the monster",
				"[14:48:00] Player [1] went to the next floor",
				"[14:48:00] Player [1] entered the boss's floor",
				"[14:49:00] Player [1] recieved [25] of damage",
				"[14:49:02] Player [1] has restored [80] of health",
				"[14:50:00] Player [1] recieved [65] of damage",
				"[14:59:00] Player [1] killed the boss",
				"[15:04:00] Player [1] left the dungeon",
			},
			expectedReport: []string{
				"[SUCCESS] 1 [00:24:00, 00:05:00, 00:11:00] HP:35",
				"[FAIL] 2 [00:19:00, 00:00:00, 00:00:00] HP:0",
				"[DISQUAL] 3 [00:00:00, 00:00:00, 00:00:00] HP:100",
			},
		},
		{
			name: "player dies from damage",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    1,
				Monsters:  1,
				OpenTime:  parseTestTime(t, "00:00:00"),
				CloseTime: parseTestTime(t, "23:59:59"),
			},
			events: `[10:00:00] 1 1
					 [10:01:00] 1 2
					 [10:02:00] 1 11 100`,
			expectedLogs: []string{
				"[10:00:00] Player [1] registered",
				"[10:01:00] Player [1] entered the dungeon",
				"[10:02:00] Player [1] recieved [100] of damage",
				"[10:02:00] Player [1] is dead",
			},
			expectedReport: []string{
				"[FAIL] 1 [00:01:00, 00:00:00, 00:00:00] HP:0",
			},
		},
		{
			name: "disqualified unregistered player",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    1,
				Monsters:  1,
				OpenTime:  parseTestTime(t, "00:00:00"),
				CloseTime: parseTestTime(t, "23:59:59"),
			},
			events: `[10:00:00] 1 2`,
			expectedLogs: []string{
				"[10:00:00] Player [1] is disqualified",
			},
			expectedReport: []string{
				"[DISQUAL] 1 [00:00:00, 00:00:00, 00:00:00] HP:100",
			},
		},
		{
			name: "dungeon closed before event",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    1,
				Monsters:  1,
				OpenTime:  parseTestTime(t, "10:00:00"),
				CloseTime: parseTestTime(t, "11:00:00"),
			},
			events: `[09:00:00] 1 1
				 	 [10:30:00] 1 2
					 [12:00:00] 1 3`,
			expectedLogs: []string{
				"[09:00:00] Player [1] registered",
				"[10:30:00] Player [1] entered the dungeon",
			},
			expectedReport: []string{
				"[FAIL] 1 [00:30:00, 00:00:00, 00:00:00] HP:100",
			},
		},
		{
			name: "cannot continue event",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    1,
				Monsters:  1,
				OpenTime:  parseTestTime(t, "00:00:00"),
				CloseTime: parseTestTime(t, "23:59:59"),
			},
			events: `[10:00:00] 1 1
					 [10:01:00] 1 2
					 [10:02:00] 1 9 ran out of potions`,
			expectedLogs: []string{
				"[10:00:00] Player [1] registered",
				"[10:01:00] Player [1] entered the dungeon",
				"[10:02:00] Player [1] cannot continue due to [ran out of potions]",
			},
			expectedReport: []string{
				"[DISQUAL] 1 [00:01:00, 00:00:00, 00:00:00] HP:100",
			},
		},
		{
			name: "health restore and damage with overflow protection",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    1,
				Monsters:  1,
				OpenTime:  parseTestTime(t, "00:00:00"),
				CloseTime: parseTestTime(t, "23:59:59"),
			},
			events: `[10:00:00] 1 1
					 [10:01:00] 1 2
					 [10:02:00] 1 11 50
					 [10:03:00] 1 10 100
					 [10:04:00] 1 11 30`,
			expectedLogs: []string{
				"[10:00:00] Player [1] registered",
				"[10:01:00] Player [1] entered the dungeon",
				"[10:02:00] Player [1] recieved [50] of damage",
				"[10:03:00] Player [1] has restored [100] of health",
				"[10:04:00] Player [1] recieved [30] of damage",
			},
			expectedReport: []string{
				"[FAIL] 1 [00:03:00, 00:00:00, 00:00:00] HP:70",
			},
		},
		{
			name: "multiple floors with backtracking",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    3,
				Monsters:  2,
				OpenTime:  parseTestTime(t, "00:00:00"),
				CloseTime: parseTestTime(t, "23:59:59"),
			},
			events: `[10:00:00] 1 1
					[10:01:00] 1 2
					[10:02:00] 1 3
					[10:03:00] 1 3
					[10:04:00] 1 4
					[10:05:00] 1 5
					[10:06:00] 1 3
					[10:07:00] 1 3
					[10:08:00] 1 4
					[10:09:00] 1 4
					[10:10:00] 1 6
					[10:11:00] 1 7
					[10:12:00] 1 8`,
			expectedLogs: []string{
				"[10:00:00] Player [1] registered",
				"[10:01:00] Player [1] entered the dungeon",
				"[10:02:00] Player [1] killed the monster",
				"[10:03:00] Player [1] killed the monster",
				"[10:04:00] Player [1] went to the next floor",
				"[10:05:00] Player [1] went to the previous floor",
				"[10:06:00] Player [1] makes imposible move [3]",
				"[10:07:00] Player [1] makes imposible move [3]",
				"[10:08:00] Player [1] went to the next floor",
				"[10:09:00] Player [1] went to the next floor",
				"[10:10:00] Player [1] makes imposible move [6]",
				"[10:11:00] Player [1] makes imposible move [7]",
				"[10:12:00] Player [1] left the dungeon",
			},
			expectedReport: []string{
				"[FAIL] 1 [00:11:00, 00:01:00, 00:00:00] HP:100",
			},
		},
		{
			name: "empty events file",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    1,
				Monsters:  1,
				OpenTime:  parseTestTime(t, "00:00:00"),
				CloseTime: parseTestTime(t, "23:59:59"),
			},
			events:         ``,
			expectedLogs:   []string{},
			expectedReport: []string{},
		},
		{
			name: "boss floor without clearing all floors",
			cfg: dungeoncfg.ParsedCfg{
				Floors:    2,
				Monsters:  2,
				OpenTime:  parseTestTime(t, "00:00:00"),
				CloseTime: parseTestTime(t, "23:59:59"),
			},
			events: `[10:00:00] 1 1
					[10:01:00] 1 2
					[10:02:00] 1 4
					[10:03:00] 1 6`,
			expectedLogs: []string{
				"[10:00:00] Player [1] registered",
				"[10:01:00] Player [1] entered the dungeon",
				"[10:02:00] Player [1] went to the next floor",
				"[10:03:00] Player [1] makes imposible move [6]",
			},
			expectedReport: []string{
				"[FAIL] 1 [00:02:00, 00:00:00, 00:00:00] HP:100",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			gs := NewGameState(tt.cfg)

			collectedLogs := make([]string, 0)
			logCallback := func(log string) {
				collectedLogs = append(collectedLogs, log)
			}

			reader := strings.NewReader(tt.events)
			err := gs.processEventsFromReader(ctx, reader, logCallback)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedLogs, collectedLogs)

			reports, err := gs.GetPlayersReportsSorted(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedReport, reports)
		})
	}
}
