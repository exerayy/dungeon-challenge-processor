package processor

import "time"

type floorStat struct {
	killedMonsters int64
	enterTime      time.Time
	clearTime      time.Duration
}

type Player struct {
	ID               int64
	Registered       bool
	InDungeon        bool
	CurrentFloor     int64
	Health           int64
	Dead             bool
	FloorsStat       map[int64]floorStat
	BossKilled       bool
	BossEnter        bool
	Disqualified     bool
	DungeonEnterTime time.Time
	BossEnterTime    time.Time
	BossKillTime     time.Duration
	TotalTime        time.Duration
}

func (p *Player) enterFloor(floor int64, enterTime time.Time) {
	p.CurrentFloor = floor
	_, alreadyEnter := p.FloorsStat[floor]
	if !alreadyEnter {
		p.FloorsStat[floor] = floorStat{
			enterTime: enterTime,
		}
	}
}

func (p *Player) clearCurFloor(eventTime time.Time) {
	fs, alreadyEnter := p.FloorsStat[p.CurrentFloor]
	if alreadyEnter {
		fs.clearTime = eventTime.Sub(fs.enterTime)
		p.FloorsStat[p.CurrentFloor] = fs
	}
}

func (p *Player) killMonster() int64 {
	fs, alreadyEnter := p.FloorsStat[p.CurrentFloor]
	if alreadyEnter {
		fs.killedMonsters++
		p.FloorsStat[p.CurrentFloor] = fs
		return fs.killedMonsters
	}

	return 0
}

func (p *Player) leftDungeon(leftTime time.Time) {
	if p.InDungeon {
		p.TotalTime = leftTime.Sub(p.DungeonEnterTime)
		p.InDungeon = false
	}
}
