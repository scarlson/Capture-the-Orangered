package cto

import (
	"encoding/json"
	"fmt"
	"time"
)

var Tutorials = make(map[int]TutLevel)

type TutLevel interface {
	Load(TutMap)
	Won() bool
	Init(chan bool, chan string)
	Setup()
	Display()
	Logic()
	Regenerate()
	GetRandomSpawn(string) *Tile
	AddPlayer(string, float64, float64, float64, float64, string)
	GetObjects() []*Object
	SetObjects([]*Object)
	GetChars() map[string]*Char
	SetChars(map[string]*Char)
	GetPlayer() *Char
	SetPlayer(*Char)
	GetId() int
	SetId(int)
	GetTiles() []*Tile
	SetTiles([]*Tile)
	GetTramps() []*Tile
	SetTramps([]*Tile)
	GetMovers() []*Tile
	SetMovers([]*Tile)
	GetDescription() string
	SetDescription(string)
	GetName() string
	SetName(string)
}

type TutMap map[int]map[int]string

type LvlBase struct {
	player      *Char
	id          int
	name        string
	chars       map[string]*Char
	objects     []*Object
	tiles       []*Tile
	tramps      []*Tile
	movers      []*Tile
	description string
	started     bool
	updates     chan *updjson
	messages    chan *Message
	stahp       chan bool
	output      chan string
}

func (lvl *LvlBase) Init(stahp chan bool, output chan string) {
	lvl.stahp = stahp
	lvl.output = output
	go lvl.Display()
	go lvl.Logic()
	go lvl.Regenerate()
}

func (lvl *LvlBase) Setup() {
	lvl.player = &Char{}
	lvl.chars = make(map[string]*Char)
	lvl.objects = make([]*Object, 0)
	lvl.tiles = make([]*Tile, 0)
	lvl.tramps = make([]*Tile, 0)
	lvl.movers = make([]*Tile, 0)
	lvl.updates = make(chan *updjson)
	lvl.messages = make(chan *Message)
}

func (lvl *LvlBase) GetPlayer() *Char              { return lvl.player }
func (lvl *LvlBase) SetPlayer(ch *Char)            { lvl.player = ch }
func (lvl *LvlBase) GetChars() map[string]*Char    { return lvl.chars }
func (lvl *LvlBase) SetChars(chs map[string]*Char) { lvl.chars = chs }
func (lvl *LvlBase) GetObjects() []*Object         { return lvl.objects }
func (lvl *LvlBase) SetObjects(obj []*Object)      { lvl.objects = obj }
func (lvl *LvlBase) GetId() int                    { return lvl.id }
func (lvl *LvlBase) SetId(i int)                   { lvl.id = i }
func (lvl *LvlBase) GetTiles() []*Tile             { return lvl.tiles }
func (lvl *LvlBase) SetTiles(tls []*Tile)          { lvl.tiles = tls }
func (lvl *LvlBase) GetTramps() []*Tile            { return lvl.tramps }
func (lvl *LvlBase) SetTramps(tra []*Tile)         { lvl.tramps = tra }
func (lvl *LvlBase) GetMovers() []*Tile            { return lvl.movers }
func (lvl *LvlBase) SetMovers(mvs []*Tile)         { lvl.movers = mvs }
func (lvl *LvlBase) GetDescription() string        { return lvl.description }
func (lvl *LvlBase) SetDescription(desc string)    { lvl.description = desc }
func (lvl *LvlBase) GetName() string               { return lvl.name }
func (lvl *LvlBase) SetName(name string)           { lvl.name = name }

func (lvl *LvlBase) Load(z TutMap) {
	for i, m := range z {
		for j, v := range m {
			if v != "Sky" {
				tl := &Tile{X: float64(j * TILE_SIZE), Y: float64(i * TILE_SIZE), Height: TILE_SIZE, Width: TILE_SIZE}
				tl.Id = fmt.Sprintf("%dx%d", int(tl.X), int(tl.Y))
				if v == "Flag" {
					tl.Flag()
				}
				if v == "Goal" {
					tl.Goal()
				}
				if v == "Periwinkle" {
					tl.Spawn("Periwinkle")
				}
				if v == "Orangered" {
					tl.Spawn("Orangered")
				}
				if v == "Dirt" {
					tl.Dirt()
				}
				if v == "Upvote" {
					tl.Upvote()
					tr := lvl.GetTramps()
					tr = append(tr, tl)
					lvl.SetTramps(tr)
				}
				if v == "Mover" {
					tl.Mover()
					mv := lvl.GetMovers()
					mv = append(mv, tl)
					lvl.SetMovers(mv)
				}
				if v == "NPC" { //todo fix vals
					lvl.AddChar("NPC", tl.X, tl.Y+2, CHAR_WIDTH, CHAR_HEIGHT, "Periwinkle")
				}
				tls := lvl.GetTiles()
				tls = append(tls, tl)
				lvl.SetTiles(tls)
				go lvl.RespawnFlag()
			}
		}
	}
}

func (self *LvlBase) GetRandomSpawn(team string) *Tile {
	spawntiles := []*Tile{}
	for _, tl := range self.GetTiles() {
		if tl.Type == team {
			spawntiles = append(spawntiles, tl)
		}
	}
	if len(spawntiles) == 0 {
		return &Tile{}
	}
	return spawntiles[randInt(0, len(spawntiles))]
}

func (self *LvlBase) Respawn(ch *Char) {
	if !ch.Respawning {
		ch.Respawning = true
		time.Sleep(3 * time.Second)
		tl := self.GetRandomSpawn(ch.Team)
		if tl.Type != "" {
			ch.X = tl.X
			ch.Y = tl.Y + 2
			ch.Dead = false
			ch.Respawning = false
		}
	}
}

func (self *LvlBase) IsDead(sp *Char) {
	if sp.Dead {
		if sp.HasFlag() {
			self.DestroyFlag()
			self.SpawnFlag()
		}
		go self.Respawn(sp)
	}
}

func (self *LvlBase) FlagExists() bool {
	for _, it := range self.GetObjects() {
		if it.Type == "flag" {
			return true
		}
	}
	return false
}

func (self *LvlBase) OnMover(sp *Char) (bool, *Tile) {
	for _, t := range self.GetMovers() {
		if sp.X > (t.X-TILE_SIZE) && t.X+TILE_SIZE > sp.X {
			if sp.Y <= t.Y+TILE_SIZE+2 && sp.Y >= t.Y+TILE_SIZE/2 {
				return true, t
			}
		}
	}
	return false, &Tile{}
}

func (self *LvlBase) OnTrampoline(sp *Char) bool {
	for _, t := range self.GetTramps() {
		if sp.X > (t.X-TILE_SIZE) && t.X+TILE_SIZE > sp.X {
			if sp.Y <= t.Y+TILE_SIZE+2 && sp.Y >= t.Y+TILE_SIZE/2 {
				return true
			}
		}
	}
	return false
}

func (self *LvlBase) TileRight(sp *Char) (bool, *Tile) {
	for _, t := range self.GetTiles() {
		if t.Solid {
			if sp.X > t.X-sp.Width-4 && sp.X < t.X {
				if sp.Y <= t.Y+t.Height/2 && sp.Y >= t.Y-sp.Height+8 {
					return true, t
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *LvlBase) TileLeft(sp *Char) (bool, *Tile) {
	for _, t := range self.GetTiles() {
		if t.Solid {
			if sp.X > t.X && sp.X < t.X+t.Width+4 {
				if sp.Y <= t.Y+t.Height/2 && sp.Y >= t.Y-sp.Height+8 {
					return true, t
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *LvlBase) TileAbove(sp *Char) (bool, *Tile) {
	for _, t := range self.GetTiles() {
		if t.Solid {
			if sp.X > t.X-sp.Width && t.X+t.Width > sp.X {
				if sp.Y <= t.Y && sp.Y >= t.Y-sp.Height-2 {
					return true, t
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *LvlBase) TileBelow(sp *Char) (bool, *Tile) {
	for _, t := range self.GetTiles() {
		if t.Solid {
			if sp.X > (t.X-sp.Width) && t.X+t.Width > sp.X {
				if sp.Y <= t.Y+sp.Height+2 && sp.Y >= t.Y+t.Height-t.Height/2 {
					return true, t
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *LvlBase) StepMovers() {
	for _, t := range self.GetMovers() {
		if t.MoveDir == "Left" && t.MoveOffset <= -MOVER_LIMIT {
			t.MoveDir = "Right"
		}
		if t.MoveDir == "Right" && t.MoveOffset >= MOVER_LIMIT {
			t.MoveDir = "Left"
		}
		if t.MoveDir == "Right" {
			t.MoveOffset += MOVER_RATE
			t.X += MOVER_RATE
		}
		if t.MoveDir == "Left" {
			t.MoveOffset -= MOVER_RATE
			t.X -= MOVER_RATE
		}
	}
}

func (self *LvlBase) Attackables(sp *Char) (bool, *Char) {
	if !sp.Dead {
		for _, ch := range self.GetChars() {
			if ch.Team != sp.Team {
				if sp.Facing == "Right" {
					if sp.X < ch.X && sp.X+(2*TILE_SIZE) > ch.X {
						if sp.Y >= ch.Y-TILE_SIZE/2 && sp.Y <= ch.Y+TILE_SIZE/2 {
							return true, ch
						}
					}
				} else if sp.Facing == "Left" {
					if sp.X > ch.X && sp.X < ch.X+2*TILE_SIZE {
						if sp.Y >= ch.Y-TILE_SIZE/2 && sp.Y <= ch.Y+TILE_SIZE/2 {
							return true, ch
						}
					}

				}
			}
		}
	}
	return false, &Char{}
}

func (self *LvlBase) Move(ch *Char) string {
	info := ""
	if !ch.Dead {
		ch.Attacked = false
		if ch.Forwards {
			if ch.Xspeed < TERMINAL_GROUND {
				ch.Xspeed += MOVE_SPEED
			} else {
				ch.Xspeed = TERMINAL_GROUND
			}
		}
		if ch.Backwards {
			if ch.Xspeed > -TERMINAL_GROUND {
				ch.Xspeed -= MOVE_SPEED
			} else {
				ch.Xspeed = -TERMINAL_GROUND
			}
		}
		if ch.Jumpme {
			osg, _ := self.TileBelow(ch)
			if osg {
				ch.Yspeed = JUMP_SPEED
				ch.Jumping = true
			}
		}
		if ch.Jumped {
			if ch.Yspeed > JUMP_SPEED/2 {
				ch.Yspeed = JUMP_SPEED / 2
			}
			ch.Jumped = false
		}
		mvbool, mv := self.OnMover(ch)
		if mvbool {
			if mv.MoveDir == "Left" {
				ch.X -= MOVER_RATE
			}
			if mv.MoveDir == "Right" {
				ch.X += MOVER_RATE
			}
		}
		if ch.Xspeed != 0 {
			if ch.Xspeed < 0 && !ch.Moving { // we're moving left!
				if ch.Xspeed > -FRICTION {
					ch.Xspeed = 0.0
				} else {
					ch.Xspeed += FRICTION
				}
			} else if ch.Xspeed > 0 && !ch.Moving { // we're moving right!
				if ch.Xspeed < FRICTION {
					ch.Xspeed = 0.0
				} else {
					ch.Xspeed -= FRICTION
				}
			}
		}
		if int(ch.Y) < -10 {
			go ch.Die()
			self.messages <- &Message{Type: "death", Text: ch.Name}
			info = fmt.Sprintf("%v cratered!", ch.Name)
		}
		if ch.Attacking {
			ch.Attacking = false
			if ch.Attacks > 0 {
				ch.Attacks--
				ch.Attacked = true
				at, sp := self.Attackables(ch)
				if at {
					if sp.HasFlag() {
						self.GiveFlag(ch)
						info = fmt.Sprintf("%v stole the flag from %v!", ch.Name, sp.Name)
					}
					sp.RemoveItems()
					go sp.Die()
					self.messages <- &Message{Type: "death", Text: ch.Name}
				}
			}
		}

		ch.X += ch.Xspeed // move char based on X rate of change
		ch.Y += ch.Yspeed // move char based on Y rate of change

		if ch.Yspeed > 0 {
			ta, _ := self.TileAbove(ch)
			if ta {
				ch.Yspeed = -2
			}
		}

		if ch.Xspeed < 0 {
			tl, _ := self.TileLeft(ch)
			if tl {
				ch.Xspeed = 2
			}
		}

		if ch.Xspeed > 0 {
			tr, _ := self.TileRight(ch)
			if tr {
				ch.Xspeed = -2
			}
		}

		// logic for detecting falling
		falling := true
		if ch.Yspeed <= GRAVITY {
			osg, osgtile := self.TileBelow(ch)
			if osg {
				falling = false
				ch.Yspeed = 0
				ch.Y = float64(osgtile.Y + osgtile.Height + 2)
			}
			if self.OnTrampoline(ch) {
				ch.Yspeed = JUMP_SPEED * 2
			}
		}
		if falling {
			if ch.Yspeed > TERMINAL_AIR {
				ch.Yspeed += GRAVITY
			}
		}

		if self.OnFlag(ch) {
			self.GiveFlag(ch)
		}
		if ch.HasFlag() {
			og, _ := self.OnGoal(ch)
			if og {
				// ASLKDFJLKSDFJLKDSJF
			}
		}
	}
	return info
}

func (self *LvlBase) AddPlayer(name string, x float64, y float64, width float64, height float64, team string) {
	ch := &Char{Name: name, X: x, Y: y, Height: height, Width: width, Yspeed: 0.0, Xspeed: 0.0, Moving: false, Team: team}
	self.player = ch
	chs := self.GetChars()
	if len(chs) == 0 {
		chs = make(map[string]*Char)
	}
	ch.Attacks = 5
	chs[name] = ch
	self.SetChars(chs)
}

func (self *LvlBase) AddChar(name string, x float64, y float64, width float64, height float64, team string) {
	ch := &Char{Name: name, X: x, Y: y, Height: height, Width: width, Yspeed: 0.0, Xspeed: 0.0, Moving: false, Team: team}
	chs := self.GetChars()
	if len(chs) == 0 {
		chs = make(map[string]*Char)
	}
	chs[name] = ch
	self.SetChars(chs)
}

func (self *LvlBase) RespawnFlag() {
	self.DestroyFlag()
	time.Sleep(3 * time.Second)
	self.SpawnFlag()
}

func (self *LvlBase) DestroyFlag() {
	objects := self.GetObjects()
	for ix, it := range objects {
		if it.Type == "flag" {
			objects = append(objects[:ix], objects[ix+1:]...)
			self.SetObjects(objects)
		}
	}
	for _, ch := range self.GetChars() {
		for ix, it := range ch.Objects {
			if it.Type == "flag" {
				ch.Objects = append(ch.Objects[:ix], ch.Objects[ix+1:]...)
			}
		}
	}
}

func (self *LvlBase) SpawnFlag() {
	if self.FlagExists() {
		return
	}
	flagtiles := []*Tile{}
	for _, tl := range self.GetTiles() {
		if tl.Type == "Flag" {
			flagtiles = append(flagtiles, tl)
		}
	}
	if len(flagtiles) == 0 {
		return
	}
	tl := flagtiles[randInt(0, len(flagtiles))]
	flag := &Object{Tile: tl, Type: "flag"}
	obj := self.GetObjects()
	obj = append(obj, flag)
	self.SetObjects(obj)
}

func (self *LvlBase) GetFlag() (bool, *Object) {
	for _, it := range self.GetObjects() {
		if it.Type == "flag" {
			return true, it
		}
	}
	return false, &Object{}
}

func (self *LvlBase) OnGoal(sp *Char) (bool, *Tile) {
	for _, tl := range self.GetTiles() {
		if tl.Type == "Goal" {
			if sp.X > tl.X-sp.Width && sp.X < tl.X+tl.Width {
				if sp.Y <= tl.Y+tl.Height && sp.Y >= tl.Y {
					return true, tl
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *LvlBase) OnFlag(sp *Char) bool {
	for _, it := range self.GetObjects() {
		if it.Type == "flag" && it.Owner == "" {
			if sp.X > it.Tile.X-TILE_SIZE && it.Tile.X+TILE_SIZE > sp.X {
				if sp.Y <= it.Tile.Y+TILE_SIZE && sp.Y >= it.Tile.Y {
					return true
				}
			}
		}
	}
	return false
}

func (self *LvlBase) GiveFlag(ch *Char) {
	for _, it := range self.GetObjects() {
		if it.Type == "flag" {
			ch.Objects = append(ch.Objects, it)
			it.Owner = ch.Name
			return
		}
	}
}

func (self *LvlBase) Won() bool {
	flool, _ := self.GetFlag()
	goal, _ := self.OnGoal(self.GetPlayer())
	if flool && self.GetPlayer().HasFlag() && goal {
		return true
	}
	if !flool && goal {
		return true
	}
	return false
}

func LoadTutorials() {
	Tutorials[0] = GetTutorial(0)
	Tutorials[1] = GetTutorial(1)
	Tutorials[2] = GetTutorial(2)
	Tutorials[3] = GetTutorial(3)
	Tutorials[4] = GetTutorial(4)
}

func GetTutorial(which int) TutLevel {
	switch which {
	case 0:
		level0 := &LvlBase{}
		level0.Setup()
		level0.SetId(1)
		level0.Load(map0)
		level0.SetName("1 - Basic Movement")
		level0.SetDescription("Use <A> and <D> to move left and right, get to the goal to advance.")
		return level0
	case 1:
		level1 := &LvlBase{}
		level1.Setup()
		level1.SetId(2)
		level1.Load(map1)
		level1.SetName("2 - Jumping")
		level1.SetDescription("Tap <SPACE> or <W> to jump, hold to jump higher, get to the goal to advance.")
		return level1
	case 2:
		level2 := &LvlBase{}
		level2.Setup()
		level2.SetId(3)
		level2.Load(map2)
		level2.SetName("3 - Capturing")
		level2.SetDescription("Grab the Orangered, then capture it to advance.")
		return level2
	case 3:
		level3 := &LvlBase{}
		level3.Setup()
		level3.SetId(4)
		level3.Load(map3)
		level3.SetName("4 - Attacking")
		level3.SetDescription("Use <SHIFT> to attack the Periwinkle and steal his Orangered, then capture it to advance.")
		return level3
	case 4:
		level4 := &LvlBase{}
		level4.Setup()
		level4.SetId(5)
		level4.Load(map4)
		level4.SetName("5 - Special Tiles")
		level4.SetDescription("Upvotes will launch you, time the jump from the mover right and capture the Orangered to advance.")
		return level4
	}
	return &LvlBase{}
}

func (self *LvlBase) Regenerate() {
	for {
		select {
		case <-self.stahp:
			return
		default:
			time.Sleep(time.Second)
			for _, ch := range self.GetChars() {
				if ch.Attacks < 5 {
					ch.Attacks++
				}
			}
		}
	}
}

func (self *LvlBase) Display() {
	for {
		select {
		case msg := <-self.messages:
			a, _ := json.Marshal(msg)
			self.output <- string(a)
		case upd := <-self.updates:
			a, _ := json.Marshal(upd)
			self.output <- string(a)
		case <-self.stahp:
			return
		}
	}
}

func (self *LvlBase) Logic() {
	ch := self.GetPlayer()
	if self.started {
		return
	}
	self.started = true
	for {
		select {
		case <-self.stahp:
			return
		default:
			self.StepMovers()
			for _, sp := range self.GetChars() {
				self.Move(sp)
			}
			if int(ch.Y) < -10 {
				go ch.Die()
			}
			if ch.Attacking {
				ch.Attacking = false
				at, sp := self.Attackables(ch)
				if at {
					if sp.HasFlag() {
						self.GiveFlag(ch)
					}
					sp.RemoveItems()
					go sp.Die()
				}
			}
			if self.OnFlag(ch) {
				self.GiveFlag(ch)
			}
			fbool, flag := self.GetFlag()
			if !fbool {
				flag = nil
			}
			if self.Won() {
				m := &Message{Type: "gg", Text: "You've won!"}
				self.messages <- m
				return
			}
			u := &updjson{Chars: self.GetChars(), Movers: self.GetMovers(), Flag: flag}
			self.updates <- u
			for _, ch := range self.GetChars() {
				go self.IsDead(ch)
			}
			time.Sleep(FRAME_RATE * 1000 * time.Microsecond)
		}
	}
}

var map0 = TutMap{
	1: map[int]string{0: s, 1: o, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: g, 13: s, 14: s},
	0: map[int]string{0: d, 1: d, 2: d, 3: d, 4: d, 5: d, 6: d, 7: d, 8: d, 9: d, 10: d, 11: d, 12: d, 13: d, 14: d},
}

var map1 = TutMap{
	10: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: g, 14: s},
	9:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: d, 13: d, 14: d},
	8:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s},
	7:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s},
	6:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: d, 9: d, 10: d, 11: s, 12: s, 13: s, 14: s},
	5:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s},
	4:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s},
	3:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: d, 5: d, 6: d, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s},
	2:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s},
	1:  map[int]string{0: s, 1: o, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s},
	0:  map[int]string{0: d, 1: d, 2: d, 3: d, 4: d, 5: d, 6: d, 7: d, 8: d, 9: d, 10: d, 11: d, 12: d, 13: d, 14: d},
}

var map2 = TutMap{
	14: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	13: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: f, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: g, 19: s},
	12: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: d, 10: d, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: d, 18: d, 19: s},
	11: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	10: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	9:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: d, 8: d, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: d, 16: d, 17: s, 18: s, 19: s},
	8:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	7:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	6:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: d, 6: d, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: d, 14: d, 15: s, 16: s, 17: s, 18: s, 19: s},
	5:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	4:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	3:  map[int]string{0: s, 1: s, 2: s, 3: d, 4: d, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: d, 12: d, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	2:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	1:  map[int]string{0: s, 1: o, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	0:  map[int]string{0: d, 1: d, 2: d, 3: d, 4: d, 5: d, 6: d, 7: d, 8: d, 9: d, 10: d, 11: d, 12: d, 13: d, 14: d, 15: d, 16: d, 17: d, 18: d, 19: d},
}

var map3 = TutMap{
	14: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: z, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	13: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: f, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: g, 19: s},
	12: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: d, 10: d, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: d, 18: d, 19: s},
	11: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	10: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	9:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: d, 8: d, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: d, 16: d, 17: s, 18: s, 19: s},
	8:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	7:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	6:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: d, 6: d, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: d, 14: d, 15: s, 16: s, 17: s, 18: s, 19: s},
	5:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	4:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	3:  map[int]string{0: s, 1: s, 2: s, 3: d, 4: d, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: d, 12: d, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	2:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	1:  map[int]string{0: s, 1: o, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	0:  map[int]string{0: d, 1: d, 2: d, 3: d, 4: d, 5: d, 6: d, 7: d, 8: d, 9: d, 10: d, 11: d, 12: d, 13: d, 14: d, 15: d, 16: d, 17: d, 18: d, 19: d},
}

var map4 = TutMap{
	14: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: f},
	13: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: d, 19: d},
	12: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: m, 7: m, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	11: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	10: map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	9:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	8:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	7:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	6:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	5:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	4:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	3:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	2:  map[int]string{0: s, 1: s, 2: s, 3: s, 4: s, 5: s, 6: s, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	1:  map[int]string{0: s, 1: o, 2: s, 3: s, 4: s, 5: s, 6: g, 7: s, 8: s, 9: s, 10: s, 11: s, 12: s, 13: s, 14: s, 15: s, 16: s, 17: s, 18: s, 19: s},
	0:  map[int]string{0: d, 1: d, 2: d, 3: d, 4: u, 5: d, 6: d, 7: d, 8: d, 9: d, 10: d, 11: d, 12: d, 13: d, 14: d, 15: d, 16: d, 17: d, 18: d, 19: d},
}
