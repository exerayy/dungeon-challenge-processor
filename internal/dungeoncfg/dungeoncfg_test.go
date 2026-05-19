package dungeoncfg

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name        string
		jsonContent string
		expected    ParsedCfg
		expectError bool
	}{
		{
			name: "valid config with all fields",
			jsonContent: `{
				"Floors": 2,
				"Monsters": 3,
				"OpenAt": "14:05:00",
				"Duration": 2
			}`,
			expected: ParsedCfg{
				Floors:    2,
				Monsters:  3,
				OpenTime:  parseTestTime(t, "14:05:00"),
				CloseTime: parseTestTime(t, "16:05:00"),
			},
		},
		{
			name: "config with single floor",
			jsonContent: `{
				"Floors": 1,
				"Monsters": 5,
				"OpenAt": "00:00:00",
				"Duration": 23
			}`,
			expected: ParsedCfg{
				Floors:    1,
				Monsters:  5,
				OpenTime:  parseTestTime(t, "00:00:00"),
				CloseTime: parseTestTime(t, "00:00:00").Add(23 * time.Hour),
			},
		},
		{
			name: "config with zero monsters - valid",
			jsonContent: `{
				"Floors": 3,
				"Monsters": 0,
				"OpenAt": "12:00:00",
				"Duration": 5
			}`,
			expected: ParsedCfg{
				Floors:    3,
				Monsters:  0,
				OpenTime:  parseTestTime(t, "12:00:00"),
				CloseTime: parseTestTime(t, "17:00:00"),
			},
		},
		{
			name: "config with zero duration",
			jsonContent: `{
				"Floors": 3,
				"Monsters": 10,
				"OpenAt": "12:00:00",
				"Duration": 0
			}`,
			expected: ParsedCfg{
				Floors:    3,
				Monsters:  10,
				OpenTime:  parseTestTime(t, "12:00:00"),
				CloseTime: parseTestTime(t, "12:00:00"),
			},
		},
		{
			name: "config with large numbers",
			jsonContent: `{
				"Floors": 1000,
				"Monsters": 1000,
				"OpenAt": "10:00:00",
				"Duration": 10
			}`,
			expected: ParsedCfg{
				Floors:    1000,
				Monsters:  1000,
				OpenTime:  parseTestTime(t, "10:00:00"),
				CloseTime: parseTestTime(t, "10:00:00").Add(10 * time.Hour),
			},
		},
		{
			name: "invalid JSON syntax",
			jsonContent: `{
				"Floors": 2,
				"Monsters": 3,
				invalid
			}`,
			expectError: true,
		},
		{
			name: "missing required field - OpenAt",
			jsonContent: `{
				"Floors": 2,
				"Monsters": 3,
				"Duration": 2
			}`,
			expectError: true,
		},
		{
			name: "invalid time format",
			jsonContent: `{
				"Floors": 2,
				"Monsters": 3,
				"OpenAt": "invalid-time",
				"Duration": 2
			}`,
			expectError: true,
		},
		{
			name:        "empty JSON object",
			jsonContent: `{}`,
			expectError: true,
		},
		{
			name: "incorrect field types - Floors as string",
			jsonContent: `{
				"Floors": "two",
				"Monsters": 3,
				"OpenAt": "14:00:00",
				"Duration": 2
			}`,
			expectError: true,
		},
		{
			name: "negative floors",
			jsonContent: `{
				"Floors": -1,
				"Monsters": 3,
				"OpenAt": "14:00:00",
				"Duration": 2
			}`,
			expectError: true,
		},
		{
			name: "zero floors",
			jsonContent: `{
				"Floors": 0,
				"Monsters": 3,
				"OpenAt": "14:00:00",
				"Duration": 2
			}`,
			expectError: true,
		},
		{
			name: "negative monsters",
			jsonContent: `{
				"Floors": 2,
				"Monsters": -5,
				"OpenAt": "14:00:00",
				"Duration": 2
			}`,
			expectError: true,
		},
		{
			name: "negative duration",
			jsonContent: `{
				"Floors": 2,
				"Monsters": 3,
				"OpenAt": "14:00:00",
				"Duration": -1
			}`,
			expectError: true,
		},
		{
			name: "invalid time format - wrong separator",
			jsonContent: `{
				"Floors": 2,
				"Monsters": 3,
				"OpenAt": "14-05-00",
				"Duration": 2
			}`,
			expectError: true,
		},
		{
			name: "invalid time format - out of range hours",
			jsonContent: `{
				"Floors": 2,
				"Monsters": 3,
				"OpenAt": "25:00:00",
				"Duration": 2
			}`,
			expectError: true,
		},
		{
			name: "time with leading zeros",
			jsonContent: `{
				"Floors": 2,
				"Monsters": 3,
				"OpenAt": "09:05:03",
				"Duration": 1
			}`,
			expected: ParsedCfg{
				Floors:    2,
				Monsters:  3,
				OpenTime:  parseTestTime(t, "09:05:03"),
				CloseTime: parseTestTime(t, "10:05:03"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.json")

			err := os.WriteFile(configPath, []byte(tt.jsonContent), 0644)
			require.NoError(t, err, "Failed to write test config file")

			cfg, err := ParseConfig(configPath)

			if tt.expectError {
				require.Error(t, err, "Expected error but got nil")
				return
			}

			require.NoError(t, err, "Unexpected error: %v", err)
			assert.Equal(t, tt.expected.Floors, cfg.Floors, "Floors mismatch")
			assert.Equal(t, tt.expected.Monsters, cfg.Monsters, "Monsters mismatch")

			assert.True(t, tt.expected.OpenTime.Equal(cfg.OpenTime),
				"OpenTime mismatch: expected %v, got %v",
				tt.expected.OpenTime.Format(layout),
				cfg.OpenTime.Format(layout))

			assert.True(t, tt.expected.CloseTime.Equal(cfg.CloseTime),
				"CloseTime mismatch: expected %v, got %v",
				tt.expected.CloseTime.Format(layout),
				cfg.CloseTime.Format(layout))
		})
	}
}

func parseTestTime(t *testing.T, timeStr string) time.Time {
	t.Helper()
	parsed, err := time.Parse(layout, timeStr)
	require.NoError(t, err, "Failed to parse time in test: %s", timeStr)
	return parsed
}
