package cto

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var DEBUG = false

func SetDebug(b bool) {
	DEBUG = b
}

const FRAME_RATE = 62.5 // frame refresh rate in MS
const TILE_SIZE = 64    // pixel HW of each tile
const CHAR_HEIGHT = 48  // pixel H of each char
const CHAR_WIDTH = 24   // pixel W of each char
const TEAM_MAX = 5      // max number of players on a team
const CAPTURE_MAX = 5   // max number of flag captures for a win
//const CAPTURE_MAX = 3   // max number of flag captures for a win
//const CAPTURE_MAX = 1   // max number of flag captures for a win

// PHYSICS!
const GRAVITY = -0.21875 * 4         // gravity, bitches
const FRICTION = 0.046875 * 3 * 10   // Xspeed gravity
const MOVE_SPEED = 0.046875 * 2 * 10 // base acceleration
const TERMINAL_AIR = -16 * 2         // terminal air velocity
const TERMINAL_GROUND = 8 * 2        // terminal ground velocity
const JUMP_SPEED = 18.5              // velocity for jumps -- should be negatives
const MOVER_LIMIT = TILE_SIZE * 3    // horizontal change limit for moving platforms
const MOVER_RATE = TILE_SIZE / 16    // rate of movement for moving platforms

type updates struct {
	Chars chan Char
	Tiles chan Tile
}

type Message struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type Score struct {
	Orangered  int
	Periwinkle int
}

type updjson struct {
	Chars       map[string]*Char `json:",omitempty"`
	Disconnects map[string]*Char `json:",omitempty"`
	Movers      []*Tile          `json:",omitempty"`
	Frame       int              `json:",omitempty"`
	Periwinkle  int              `json:",omitempty"`
	Orangered   int              `json:",omitempty"`
	Flag        *Object          `json:",omitempty"`
}

func (self *Level) Regenerate() {
	for {
		select {
		case <-self.Stahp:
			return
		default:
			time.Sleep(time.Second)
			for _, ch := range self.Chars {
				if ch.Attacks < 5 {
					ch.Attacks++
				}
			}
		}
	}
}

func (self *Level) Display() {
	for {
		select {
		case msg := <-self.Messages:
			a, _ := json.Marshal(msg)
			self.Output <- string(a)
		case upd := <-self.Updates:
			a, _ := json.Marshal(upd)
			self.Output <- string(a)
		case <-self.Stahp:
			return
		}
	}
}

func (self *Level) Logic() {
	if DEBUG {
		fmt.Printf("Game %d logic started.\n", self.Id)
	}
	for {
		select {
		case <-self.Stahp:
			return
		default:
			start := time.Now()
			if self.Started {
				self.Frame++
			}
			self.StepMovers()
			msg := &Message{}
			for _, ch := range self.Chars {
				i := self.Move(ch)
				if i != "" {
					msg.Type = "info"
					msg.Text = i
					self.Messages <- msg
				}
				// logic to disconnect idle players here
				// if player.lastaction - curtime >= 3m, disconnect them
				if time.Since(ch.LastAction) >= time.Duration(1)*time.Minute {
					self.Disconnect(ch.Name)
				}
			}
			fbool, flag := self.GetFlag()
			if !fbool {
				flag = nil
			}
			if self.OrangeredScore >= CAPTURE_MAX || self.PeriwinkleScore >= CAPTURE_MAX {
				go self.EndGame()
				return
			}
			u := &updjson{Chars: self.Chars, Disconnects: self.Disconnects, Movers: self.Movers, Frame: self.Frame, Periwinkle: self.PeriwinkleScore, Orangered: self.OrangeredScore, Flag: flag}
			self.Updates <- u
			for _, ch := range self.Chars {
				go self.IsDead(ch)
			}
			time.Sleep(FRAME_RATE*1000*time.Microsecond - time.Now().Sub(start))
		}
	}
}

type Object struct {
	Type  string
	Owner string
	Tile  *Tile
}

type Tile struct {
	Id           string
	X            float64
	Y            float64
	Type         string
	Destructible bool
	Solid        bool
	Width        float64
	Height       float64
	MoveOffset   int
	MoveDir      string
}

func (self *Tile) Destroy() {
	if self.Destructible {
		self.Solid = false
		self.Destructible = false
		self.Type = "Sky"
	}
}

func (self *Tile) Spawn(color string) {
	self.Solid = false
	self.Type = color
}

func (self *Tile) Goal() {
	self.Solid = false
	self.Type = "Goal"
}

func (self *Tile) Flag() {
	self.Solid = false
	self.Type = "Flag"
}

func (self *Tile) Dirt() {
	self.Solid = true
	self.Type = "Dirt"
}

func (self *Tile) Upvote() {
	self.Solid = true
	self.Type = "Upvote"
}

func (self *Tile) Mover() {
	self.Solid = true
	self.MoveDir = "Left"
	self.Type = "Mover"
}

func (self *Tile) NPC() {
	self.Solid = false
	self.Type = "NPC"
}

type Level struct {
	Id              string
	Height          int
	Width           int
	Tiles           []*Tile
	Movers          []*Tile
	Tramps          []*Tile
	Objects         []*Object
	Chars           map[string]*Char
	Disconnects     map[string]*Char
	IdCount         int
	Started         bool
	Ended           bool
	PeriwinkleScore int
	OrangeredScore  int
	Stahp           chan bool
	Output          chan string
	KillRoom        chan bool
	Updates         chan *updjson
	Messages        chan *Message
	Frame           int
	StartTime       time.Time
}

func (self *Level) GetChar(name string) *Char {
	for _, ch := range self.Chars {
		if ch.Name == name {
			return ch
		}
	}
	return &Char{}
}

func (self *Level) IsFull() bool {
	if self.CountTeam("Periwinkle") >= TEAM_MAX && self.CountTeam("Orangered") >= TEAM_MAX {
		return true
	}
	return false
}

func (self *Level) PlayerCount() string {
	i := strconv.Itoa(len(self.Chars))
	return i
}

func (self *Level) ElapsedTime() string {
	ms := int(float64(self.Frame) * FRAME_RATE)
	s := ms / 1000
	m := s / 60
	if s <= 0 {
		return ""
	}
	if s <= 100 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm %00ds", m, s%60)
}

func (self *Level) StepMovers() {
	for _, t := range self.Movers {
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

func (self *Level) StartGame() {
	self.Started = true
	self.StartTime = time.Now()
	m := &Message{Text: "start", Type: "status"}
	self.Messages <- m
	time.Sleep(3 * time.Second)
	msg := &Message{}
	m.Type = "warn"
	m.Text = "Game starting in"
	self.Messages <- m
	go self.RespawnFlag()
	for i := 0; i < 3; i++ {
		msg.Type = "subwarn"
		msg.Text = strconv.Itoa(3 - i)
		self.Messages <- msg
		time.Sleep(time.Second)
	}
	if DEBUG {
		fmt.Printf("Starting new game %d.\n", self.Id)
	}
	self.PeriwinkleScore = 0
	self.OrangeredScore = 0
	for _, ch := range self.Chars {
		ch.Dead = false
		ch.Jumping = false
		ch.Jumped = false
		ch.Jumpme = false
		ch.Forwards = false
		ch.Backwards = false
		ch.Moving = false
		ch.Xspeed = 0
		ch.Yspeed = 0
		ch.Attacks = 5
		tl := self.GetRandomSpawn(ch.Team)
		ch.X = tl.X
		ch.Y = tl.Y + 2
	}
	m = &Message{Text: "started", Type: "status"}
	self.Messages <- m
}

func (self *Level) EndGame() {
	msg := &Message{}
	msg.Type = "gg"
	s := &Score{Orangered: self.OrangeredScore, Periwinkle: self.PeriwinkleScore}
	a, _ := json.Marshal(s)
	msg.Text = string(a)
	self.Messages <- msg
	u := &updjson{Chars: self.Chars, Disconnects: self.Disconnects, Movers: self.Movers, Frame: self.Frame, Periwinkle: self.PeriwinkleScore, Orangered: self.OrangeredScore}
	self.Updates <- u
	m := &Message{Text: "end", Type: "status"}
	self.Messages <- m
	time.Sleep(3 * time.Second)
	self.Ended = true
	if DEBUG {
		fmt.Printf("Ending game %d.\n", self.Id)
	}
	self.DestroyFlag()
	self.Started = false
	self.Stahp <- true
	self.KillRoom <- true
}

func (self *Level) GiveFlag(ch *Char) {
	for _, it := range self.Objects {
		if it.Type == "flag" {
			m := &Message{Text: ch.Name, Type: "flag"}
			self.Messages <- m
			ch.Objects = append(ch.Objects, it)
			it.Owner = ch.Name
			it.Tile = &Tile{}
			return
		}
	}
}

func (self *Level) DeleteChar(ch *Char) {
	m := &Message{Text: ch.Name, Type: "delete"}
	self.Messages <- m
	delete(self.Chars, ch.Name)
}

func (self *Level) Disconnect(name string) {
	for _, ch := range self.Chars {
		if strings.ToLower(ch.Name) == strings.ToLower(name) {
			ch.BlockDc = false
		}
	}
	time.Sleep(5 * time.Second)
	for _, ch := range self.Chars {
		if strings.ToLower(ch.Name) == strings.ToLower(name) && ch.BlockDc {
			return
		}
	}
	m := &Message{Text: name, Type: "disconnect"}
	self.Messages <- m
	fmt.Println("Disconnecting ", name)
	for _, ch := range self.Chars {
		if strings.ToLower(ch.Name) == strings.ToLower(name) && !ch.BlockDc {
			if ch.HasFlag() {
				self.RespawnFlag()
			}
			ch.Dead = true
			self.Disconnects[ch.Name] = ch
			self.DeleteChar(ch)
			fmt.Println(name, "disconnected!")
			return
		}
	}
}

func (self *Level) Attackables(sp *Char) (bool, *Char) {
	if !sp.Dead {
		for _, ch := range self.Chars {
			if ch.Team != sp.Team {
				if sp.Facing == "Right" {
					if sp.X < ch.X && sp.X+4*ch.Width > ch.X {
						if sp.Y >= ch.Y-ch.Height/2 && sp.Y <= ch.Y+ch.Height/2 {
							return true, ch
						}
					}
				} else if sp.Facing == "Left" {
					if sp.X > ch.X && sp.X < ch.X+4*ch.Width {
						if sp.Y >= ch.Y-ch.Height/2 && sp.Y <= ch.Y+ch.Height/2 {
							return true, ch
						}
					}

				}
			}
		}
	}
	return false, &Char{}
}

func (self *Level) FlagExists() bool {
	for _, it := range self.Objects {
		if it.Type == "flag" {
			return true
		}
	}
	return false
}

func (self *Level) DestroyFlag() {
	for ix, it := range self.Objects {
		if it.Type == "flag" {
			self.Objects = append(self.Objects[:ix], self.Objects[ix+1:]...)
		}
	}
	for _, ch := range self.Chars {
		for ix, it := range ch.Objects {
			if it.Type == "flag" {
				ch.Objects = append(ch.Objects[:ix], ch.Objects[ix+1:]...)
			}
		}
	}
}

func (self *Level) CountTeam(team string) int {
	i := 0
	for _, ch := range self.Chars {
		if ch.Team == team {
			i++
		}
	}
	return i
}

func (self *Level) RespawnFlag() {
	self.DestroyFlag()
	time.Sleep(3 * time.Second)
	self.SpawnFlag()
}

func (self *Level) SpawnFlag() {
	if self.FlagExists() {
		return
	}
	flagtiles := []*Tile{}
	for _, tl := range self.Tiles {
		if tl.Type == "Flag" {
			flagtiles = append(flagtiles, tl)
		}
	}
	tl := flagtiles[randInt(0, len(flagtiles))]
	flag := &Object{Tile: tl, Type: "flag"}
	m := &Message{Text: tl.Id, Type: "flag"}
	self.Messages <- m
	self.Objects = append(self.Objects, flag)
}

func (self *Level) GetFlag() (bool, *Object) {
	for _, it := range self.Objects {
		if it.Type == "flag" {
			return true, it
		}
	}
	return false, &Object{}
}

func (self *Level) WhoHasFlag() string {
	for _, it := range self.Objects {
		if it.Type == "flag" {
			return it.Owner
		}
	}
	return "level"
}

func (self *Level) OnGoal(sp *Char) (bool, *Tile) {
	for _, tl := range self.Tiles {
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

func (self *Level) OnFlag(sp *Char) bool {
	for _, it := range self.Objects {
		if it.Type == "flag" && it.Owner == "" {
			if sp.X > it.Tile.X-sp.Width && sp.X < it.Tile.X+TILE_SIZE {
				if sp.Y <= it.Tile.Y+sp.Height && sp.Y >= it.Tile.Y {
					return true
				}
			}
		}
	}
	return false
}

func (self *Level) GetRandomSpawn(team string) *Tile {
	spawntiles := []*Tile{}
	for _, tl := range self.Tiles {
		if tl.Type == team {
			spawntiles = append(spawntiles, tl)
		}
	}
	return spawntiles[randInt(0, len(spawntiles))]
}

func (self *Level) Respawn(ch *Char) {
	ch.BlockDc = true
	if !ch.Respawning && ch.Dead {
		ch.Respawning = true
		time.Sleep(3 * time.Second)
		tl := self.GetRandomSpawn(ch.Team)
		ch.X = tl.X + TILE_SIZE/2 - CHAR_WIDTH/2
		ch.Y = tl.Y + 2
		ch.Dead = false
		ch.Respawning = false
		ch.Jumping = false
		ch.Jumped = false
		ch.Jumpme = false
		ch.Forwards = false
		ch.Backwards = false
		ch.Moving = false
		ch.Xspeed = 0
		ch.Yspeed = 0
	}
}

func (self *Level) IsDead(sp *Char) {
	if sp.Dead {
		if sp.HasFlag() {
			sp.RemoveItems()
			self.RespawnFlag()
		}
		go self.Respawn(sp)
	}
}

func (self *Level) Move(ch *Char) string {
	info := ""
	if !ch.Dead {
		ch.Attacked = false
		if ch.Forwards {
			//ch.Forwards = false
			if ch.Xspeed < TERMINAL_GROUND {
				ch.Xspeed += MOVE_SPEED
			} else {
				ch.Xspeed = TERMINAL_GROUND
			}
		}
		if ch.Backwards {
			//ch.Backwards = false
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
			self.Messages <- &Message{Text: ch.Name, Type: "death"}
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
						sp.RemoveItems()
						self.GiveFlag(ch)
						info = fmt.Sprintf("%v stole the flag from %v!", ch.Name, sp.Name)
					}
					go sp.Die()
					self.Messages <- &Message{Text: ch.Name, Type: "death"}
				}
			}
		}

		ch.X += ch.Xspeed // move char based on X rate of change
		ch.Y += ch.Yspeed // move char based on Y rate of change

		if ch.Yspeed > 0 {
			ia, ta := self.TileAbove(ch)
			if ia {
				if ta.Y-(ch.Y+ch.Height) <= ch.Yspeed {
					ch.Yspeed = -2
					ch.Y = ta.Y - ch.Height - 1
				}
			}
		}

		if ch.Xspeed < 0 {
			il, tl := self.TileLeft(ch)
			if il {
				// if X distance < x.Speed, ch.X = tl.X
				if ch.X-(tl.X+TILE_SIZE) <= 0-ch.Xspeed {
					ch.Xspeed = 0
					ch.X = tl.X + TILE_SIZE + 1
				}
			}
		}

		if ch.Xspeed > 0 {
			ir, tr := self.TileRight(ch)
			if ir {
				if tr.X-(ch.X+ch.Width) <= ch.Xspeed {
					ch.Xspeed = 0
					ch.X = tr.X - ch.Width - 1
				}
			}
		}

		// logic for detecting falling
		falling := true
		if ch.Yspeed <= GRAVITY {
			osg, osgtile := self.TileBelow(ch)
			if osg {
				falling = false
				ch.Yspeed = 0
				//ch.Y = float64(osgtile.Y + osgtile.Height + 2)
				ch.Y = float64(osgtile.Y + osgtile.Height)
			}
			if self.OnTrampoline(ch) {
				ch.Jumping = false
				// check if char is jumping, and prevent them from being able to slow jump
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
			og, ogt := self.OnGoal(ch)
			if og {
				if ch.Team == "Periwinkle" {
					self.PeriwinkleScore++
					_, flag := self.GetFlag()
					flag.Owner = ""
					flag.Tile = ogt
					ch.RemoveItems()
				} else if ch.Team == "Orangered" {
					self.OrangeredScore++
					_, flag := self.GetFlag()
					flag.Owner = ""
					flag.Tile = ogt
					ch.RemoveItems()
				}
				info = fmt.Sprintf("%v scored for Team %v!", ch.Name, ch.Team)
				go self.RespawnFlag()
				// scored!
			}
		}
	}
	//ch.Moving = false
	return info
}

func (self *Level) OnMover(sp *Char) (bool, *Tile) {
	for _, t := range self.Movers {
		if t.Solid {
			if sp.X > (t.X-sp.Width) && t.X+t.Width > sp.X {
				if sp.Y <= t.Y+t.Height+2 && sp.Y >= t.Y+t.Height+sp.Yspeed {
					return true, t
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *Level) OnTrampoline(sp *Char) bool {
	for _, t := range self.Tramps {
		if sp.X > t.X-sp.Width && sp.X < t.X+t.Width {
			if sp.Y <= t.Y+t.Height+2 && sp.Y >= t.Y+t.Height/4 {
				return true
			}
		}
	}
	return false
}

func (self *Level) TileRight(sp *Char) (bool, *Tile) {
	for _, t := range self.Tiles {
		if t.Solid {
			if sp.X+sp.Width+sp.Xspeed > t.X && sp.X < t.X {
				//if sp.Y <= t.Y+t.Height/2 && sp.Y >= t.Y-sp.Height+8 {
				if sp.Yspeed <= 0 { // falling
					if sp.Y <= t.Y+t.Height+sp.Yspeed-2 && sp.Y+sp.Height-4 >= t.Y {
						return true, t
					}
				} else { // jumping
					if sp.Y <= t.Y+t.Height-sp.Yspeed && sp.Y+sp.Height-2-sp.Yspeed >= t.Y {
						return true, t
					}
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *Level) TileLeft(sp *Char) (bool, *Tile) {
	for _, t := range self.Tiles {
		if t.Solid {
			if sp.X > t.X && sp.X < t.X+t.Width-sp.Xspeed {
				//if sp.Y <= t.Y+t.Height/2 && sp.Y >= t.Y-sp.Height+8 {
				if sp.Yspeed <= 0 { // falling
					if sp.Y <= t.Y+t.Height+sp.Yspeed-2 && sp.Y+sp.Height-4 >= t.Y {
						return true, t
					}
				} else { // jumping
					if sp.Y <= t.Y+t.Height-sp.Yspeed && sp.Y+sp.Height-2-sp.Yspeed >= t.Y {
						return true, t
					}
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *Level) TileAbove(sp *Char) (bool, *Tile) {
	for _, t := range self.Tiles {
		if t.Solid {
			if sp.X+sp.Width-2 > t.X && t.X+t.Width-2 > sp.X {
				if sp.Y <= t.Y && sp.Y+sp.Height+sp.Yspeed >= t.Y {
					return true, t
				}
			}
		}
	}
	return false, &Tile{}
}

func (self *Level) TileBelow(sp *Char) (bool, *Tile) {
	for _, t := range self.Tiles {
		if t.Solid {
			if sp.X > (t.X-sp.Width) && t.X+t.Width > sp.X {
				if sp.Y <= t.Y+t.Height+2 && sp.Y >= t.Y+t.Height+sp.Yspeed {
					return true, t
				}
			}
		}
	}
	return false, &Tile{}
}

func (lvl *Level) AddChar(name string, x float64, y float64, width float64, height float64, team string, sprite string) {
	for _, ch := range lvl.Chars { // check if the char is already playing, shouldn't happen
		if strings.ToLower(ch.Name) == strings.ToLower(name) {
			ch.BlockDc = true
			fmt.Println("AddChar: Char exists, returning")
			return
		}
	}
	for _, ch := range lvl.Disconnects { // check to see if the player is currently disconnected and readd them
		if strings.ToLower(ch.Name) == strings.ToLower(name) { // TODO check for a full game, and reset team
			ch.LastAction = time.Now()
			ch.BlockDc = true
			ch.Team = team
			fmt.Println("AddChar: Char in disconnects, moving to active and returning")
			lvl.Chars[name] = ch
			delete(lvl.Disconnects, ch.Name)
			return
		}
	}
	ch := &Char{Name: name, X: x, Y: y, Height: height, Width: width, Yspeed: 0.0, Xspeed: 0.0, Moving: false, Team: team, Sprite: sprite}
	ch.Attacks = 5
	ch.LastAction = time.Now()
	lvl.Chars[name] = ch
	if DEBUG && !lvl.Started {
		fmt.Printf("AddChar: %+v\n", ch)
		go lvl.StartGame()
	} else {
		if !lvl.Started && lvl.CountTeam("Periwinkle") > 0 && lvl.CountTeam("Orangered") > 0 {
			go lvl.StartGame()
		}
	}
}

type Char struct {
	Name       string
	X          float64
	Y          float64
	Height     float64
	Width      float64
	Yspeed     float64
	Xspeed     float64
	Attacks    int
	Dead       bool
	Moving     bool
	Jumped     bool
	Jumping    bool
	Jumpme     bool
	Forwards   bool
	Backwards  bool
	Attacking  bool
	Attacked   bool
	Respawning bool
	BlockDc    bool
	Facing     string
	Team       string
	Objects    []*Object
	Sprite     string
	LastAction time.Time
}

func (self *Char) RemoveItems() {
	self.Objects = []*Object{}
}

func (self *Char) HasFlag() bool {
	for _, it := range self.Objects {
		if it.Type == "flag" {
			return true
		}
	}
	return false
}

func (self *Char) Attack() {
	if !self.HasFlag() {
		self.Attacking = true
	}
}

func (self *Char) Forward(stahp chan bool) {
	for {
		if self.Dead {
			self.Moving = false
			self.Forwards = false
			return
		}
		select {
		case <-stahp:
			self.BlockDc = true
			self.Moving = false
			self.Forwards = false
			return
		default:
			self.LastAction = time.Now()
			self.Forwards = true
			self.Moving = true
			self.Facing = "Right"
			time.Sleep(FRAME_RATE * 100 * time.Microsecond)
		}
	}
}

func (self *Char) Backward(stahp chan bool) {
	for {
		if self.Dead {
			self.Backwards = false
			self.Moving = false
			return
		}
		select {
		case <-stahp:
			self.BlockDc = true
			self.Backwards = false
			self.Moving = false
			return
		default:
			self.LastAction = time.Now()
			self.Backwards = true
			self.Moving = true
			self.Facing = "Left"
			time.Sleep(FRAME_RATE * 100 * time.Microsecond)
		}
	}
}

func (self *Char) Jump(stahp chan bool) {
	for {
		if self.Dead {
			self.Jumping = false
			self.Jumped = false
			self.Jumpme = false
			return
		}
		select {
		case <-stahp:
			self.BlockDc = true
			self.Jumpme = false
			if self.Jumping { // only allow slow down if player jumped off solid ground
				self.Jumped = true
			}
			self.Jumping = false
			return
		default:
			self.LastAction = time.Now()
			self.Jumpme = true
			time.Sleep(FRAME_RATE * 100 * time.Microsecond)
		}
	}
}

func (self *Char) Die() {
	//self.RemoveItems()
	self.Jumping = false
	self.Jumped = false
	self.Jumpme = false
	self.Forwards = false
	self.Backwards = false
	self.Moving = false
	self.Xspeed = 0
	self.Yspeed = 0
	self.Dead = true
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func (self *Level) LoadMap(m map[string]map[string]string) {
	for i, r := range m {
		i, _ := strconv.Atoi(i)
		for j, t := range r {
			j, _ := strconv.Atoi(j)
			if t != "s" {
				tl := &Tile{X: float64(j * TILE_SIZE), Y: float64(i * TILE_SIZE), Height: TILE_SIZE, Width: TILE_SIZE}
				tl.Id = fmt.Sprintf("%dx%d", int(tl.X), int(tl.Y))
				if t == "f" {
					tl.Flag()
				}
				if t == "g" {
					tl.Goal()
				}
				if t == "p" {
					tl.Spawn("Periwinkle")
				}
				if t == "o" {
					tl.Spawn("Orangered")
				}
				if t == "d" {
					tl.Dirt()
				}
				if t == "u" {
					tl.Upvote()
					self.Tramps = append(self.Tramps, tl)
				}
				if t == "m" {
					tl.Mover()
					self.Movers = append(self.Movers, tl)
				}
				self.Tiles = append(self.Tiles, tl)
			}
		}
	}
}

func NewGame(output chan string, killroom chan bool, m map[string]map[string]string) *Level {
	rand.Seed(time.Now().UTC().UnixNano())
	//m := Map2()
	chars := make(map[string]*Char)
	dcs := make(map[string]*Char)
	updates := make(chan *updjson)
	messages := make(chan *Message)
	height := len(m) + 1
	width := len(m["0"])
	lvl := &Level{Height: TILE_SIZE * height, Width: TILE_SIZE * width, Chars: chars}
	lvl.LoadMap(m)
	lvl.Stahp = make(chan bool)
	lvl.Output = output
	lvl.Updates = updates
	lvl.Messages = messages
	lvl.Disconnects = dcs
	lvl.KillRoom = killroom
	go lvl.Display()
	go lvl.Logic()
	go lvl.Regenerate()
	fmt.Printf("New game %d made.\n", lvl.Id)
	return lvl
}

func main() {
	fmt.Println("GO GO CTO!")
}
