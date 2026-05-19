package processor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/exerayy/dungeon-challenge-processor/internal/dungeoncfg"
)

const (
	layout                = "15:04:05"
	maxHealthPlayer       = 100
	minFloor        int64 = 1

	// eventMessagesCapacity - начальная ёмкость слайса для сообщений получаемых с одного события
	// Превышение безопасно: слайс автоматически расширится при добавлении элементов.
	eventMessagesCapacity = 2

	reportStateSuccess = "SUCCESS"
	reportStateFail    = "FAIL"
	reportStateDisqual = "DISQUAL"
)

const (
	eventRegister = iota + 1
	eventEnterTheDungeon
	eventKillMonster
	eventWentToNextFloor
	eventWentToPreviousFloor
	eventEnterBossFloor
	eventKillBoss
	eventLeftDungeon
	eventCannotContinue
	eventRestoreHealth
	eventRecieveDamage
)

type GameState struct {
	Cfg           dungeoncfg.ParsedCfg
	Players       map[int64]*Player
	DungeonClosed bool
}

func NewGameState(cfg dungeoncfg.ParsedCfg) *GameState {
	return &GameState{
		Cfg:     cfg,
		Players: make(map[int64]*Player),
	}
}

func (gs *GameState) getPlayer(id int64) *Player {
	if player, exists := gs.Players[id]; exists {
		return player
	}

	player := &Player{
		ID:         id,
		Health:     maxHealthPlayer,
		FloorsStat: make(map[int64]floorStat, gs.Cfg.Floors-1),
	}
	gs.Players[id] = player

	return player
}

// ProcessEvents читает файл событий и обрабатывает их последовательно.
// Для каждого события вызывает onEventLog с логами действий игроков.
func (gs *GameState) ProcessEvents(ctx context.Context, filename string, onEventLog func(string)) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open events file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	return gs.processEventsFromReader(ctx, file, onEventLog)
}

func (gs *GameState) processEventsFromReader(ctx context.Context, reader io.Reader, onEventLog func(string)) error {
	scanner := bufio.NewScanner(reader)
	lineNum := 0

	var event Event
	var err error
	for scanner.Scan() {
		if err = ctx.Err(); err != nil {
			return err
		}

		lineNum++
		line := scanner.Text()

		event, err = parseEvent(line)
		if err != nil {
			if errors.Is(err, errEmptyLine) {
				continue
			}
			return fmt.Errorf("failed to parse event, line %d: %w", lineNum, err)
		}

		eventLogs := gs.processEvent(event)
		if gs.DungeonClosed {
			break
		}
		for _, log := range eventLogs {
			onEventLog(log)
		}
	}

	// Если подземелье не успело закрыться после всех событий, то принудительно закрываем его
	// total_time для игроков не успевших выйти будет считаться на основе последнего события
	if !gs.DungeonClosed {
		gs.closeDungeon(event.Time)
	}

	if err = scanner.Err(); err != nil {
		return fmt.Errorf("failed to read events file: %w", err)
	}

	return nil
}

func (gs *GameState) processEvent(event Event) []string {
	player := gs.getPlayer(event.PlayerID)

	if player.Disqualified {
		return nil
	}

	if event.Time.After(gs.Cfg.CloseTime) {
		gs.closeDungeon(gs.Cfg.CloseTime)
		return nil
	}

	messages := make([]string, 0, eventMessagesCapacity)

	switch event.EventID {
	case eventRegister:
		if !player.Registered {
			player.Registered = true
			messages = append(messages, "registered")
		}

	case eventEnterTheDungeon:
		if !player.Registered {
			player.Disqualified = true
			messages = append(messages, "is disqualified")
			break
		}

		if player.InDungeon ||
			player.Dead {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.InDungeon = true
		player.DungeonEnterTime = event.Time
		player.enterFloor(minFloor, event.Time)

		messages = append(messages, "entered the dungeon")

	case eventKillMonster:
		if !player.InDungeon ||
			player.Dead ||
			player.CurrentFloor > gs.Cfg.Floors-1 ||
			player.FloorsStat[player.CurrentFloor].killedMonsters >= gs.Cfg.Monsters {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		killedMonsters := player.killMonster()

		if killedMonsters >= gs.Cfg.Monsters {
			player.clearCurFloor(event.Time)
		}

		messages = append(messages, "killed the monster")

	case eventWentToNextFloor:
		if !player.InDungeon ||
			player.Dead ||
			player.CurrentFloor >= gs.Cfg.Floors {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.enterFloor(player.CurrentFloor+1, event.Time)
		messages = append(messages, "went to the next floor")

	case eventWentToPreviousFloor:
		if !player.InDungeon ||
			player.Dead ||
			player.CurrentFloor <= minFloor {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.enterFloor(player.CurrentFloor-1, event.Time)
		player.BossEnter = false
		messages = append(messages, "went to the previous floor")

	case eventEnterBossFloor:
		if !player.InDungeon ||
			player.Dead ||
			player.CurrentFloor != gs.Cfg.Floors ||
			player.BossEnter {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		allFloorsClear := true
		for floor := minFloor; floor <= gs.Cfg.Floors-1; floor++ {
			if player.FloorsStat[floor].killedMonsters < gs.Cfg.Monsters {
				allFloorsClear = false
				break
			}
		}

		if !allFloorsClear {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.BossEnter = true
		player.BossEnterTime = event.Time
		messages = append(messages, "entered the boss's floor")

	case eventKillBoss:
		if !player.InDungeon ||
			player.Dead ||
			!player.BossEnter ||
			player.BossKilled ||
			player.CurrentFloor != gs.Cfg.Floors {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.BossKilled = true
		player.BossKillTime = event.Time.Sub(player.BossEnterTime)
		messages = append(messages, "killed the boss")

	case eventLeftDungeon:
		if !player.InDungeon ||
			player.Dead {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.leftDungeon(event.Time)
		messages = append(messages, "left the dungeon")

	case eventCannotContinue:
		if !player.InDungeon {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.Disqualified = true
		player.leftDungeon(event.Time)

		messages = append(messages, fmt.Sprintf("cannot continue due to [%s]", event.ExtraParam))

	case eventRestoreHealth:
		if !player.InDungeon ||
			player.Dead {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		heal, err := strconv.Atoi(event.ExtraParam)
		if err != nil || heal < 0 {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.Health += int64(heal)
		if player.Health > maxHealthPlayer {
			player.Health = maxHealthPlayer
		}

		messages = append(messages, fmt.Sprintf("has restored [%d] of health", heal))

	case eventRecieveDamage:
		if !player.InDungeon ||
			player.Dead {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		damage, err := strconv.Atoi(event.ExtraParam)
		if err != nil || damage < 0 {
			messages = append(messages, fmt.Sprintf("makes imposible move [%d]", event.EventID))
			break
		}

		player.Health -= int64(damage)
		messages = append(messages, fmt.Sprintf("recieved [%d] of damage", damage))

		if player.Health <= 0 {
			player.Health = 0
			player.Dead = true
			player.leftDungeon(event.Time)
			messages = append(messages, "is dead")
		}

	default:
		messages = append(messages, fmt.Sprintf("unknown event id [%d]", event.EventID))
	}

	logs := make([]string, len(messages))
	for i, msg := range messages {
		logs[i] = newLog(event.Time, event.PlayerID, msg)
	}

	return logs
}

func newLog(eventTime time.Time, playerID int64, message string) string {
	return fmt.Sprintf("[%s] Player [%d] %s",
		eventTime.Format(layout),
		playerID,
		message,
	)
}

// closeDungeon - закрывает подземелье, если есть игроки не успевшие завершить игру,
// то они принудительно выходят и считается их total_time в подземелье
// totalTime = closeTime - enterDungeonTime
func (gs *GameState) closeDungeon(closeTime time.Time) {
	gs.DungeonClosed = true

	for _, p := range gs.Players {
		p.leftDungeon(closeTime)
	}
}

// GetPlayersReportsSorted - возвращает отчёт на каждого игрока
// Отсортировано по возрастанию PlayerID
func (gs *GameState) GetPlayersReportsSorted(ctx context.Context) ([]string, error) {
	ids := make([]int64, 0, len(gs.Players))
	for id := range gs.Players {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	report := make([]string, 0, len(gs.Players))

	for _, id := range ids {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		player := gs.Players[id]
		state := reportStateSuccess
		if player.Disqualified || !player.Registered {
			state = reportStateDisqual
		} else if player.Dead || !player.BossKilled {
			state = reportStateFail
		}

		var avgClearTime time.Duration
		if gs.Cfg.Floors > 1 {
			var totalClearTime time.Duration
			for _, floor := range player.FloorsStat {
				totalClearTime += floor.clearTime
			}
			avgClearTime = totalClearTime / time.Duration(gs.Cfg.Floors-1)
		}

		msg := fmt.Sprintf("[%s] %d [%s, %s, %s] HP:%d",
			state,
			id,
			formatDuration(player.TotalTime),
			formatDuration(avgClearTime),
			formatDuration(player.BossKillTime),
			player.Health,
		)

		report = append(report, msg)
	}

	return report, nil
}

// formatDuration форматирует time.Duration в string HH:MM:SS
func formatDuration(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	hours := totalSeconds / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}
