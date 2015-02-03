package main

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"crypto/hmac"
	cr "crypto/rand"
	"crypto/sha1"
	"cto"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hoisie/mustache"
	"github.com/hoisie/web"
	"github.com/scarlson/grow"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

var (
	// database vars
	mgoSession   *mgo.Session
	databaseName = "capturetheorangered"

	MAX_GAMES    = 9
	MAX_WIDTH    = 10
	MAX_HEIGHT   = 10
	GUEST        = true
	MAP          = true
	MARKET       = false
	Sprites      = []string{"Tophat", "Fishbowl", "Robinhood", "Wizard"}
	def          = UserMap{MapId: "0", Name: "Default", Owner: "CTO_Staff", Tiles: cto.Map2()}
	Maps         = []UserMap{def}
	games        = map[string]*cto.Level{}
	rooms        = map[string]*room{}
	users        = map[string]string{}
	usermaphash  = map[string]string{}
	mapeditcache = map[string]*UserMap{}
	ustut        = map[string]cto.TutLevel{}
	conf         = &Config{}
)

type MapJson struct {
	M    map[string]map[string]string `json:"m"`
	Name string                       `json:"name"`
}

type UserMap struct {
	Name  string
	MapId string
	Owner string
	Tiles map[string]map[string]string
}

func (self *UserMap) validate() error {
	if self.Name == "" {
		return errors.New("Invalid user")
	}
	tls := []string{}
	for _, m := range self.Tiles {
		for _, t := range m {
			tls = append(tls, t)
		}
	}
	if len(self.Tiles) > MAX_HEIGHT {
		return errors.New("Map height exceeds limit")
	}
	for _, r := range self.Tiles {
		if len(r) > MAX_WIDTH {
			return errors.New("Map width exceeds limit")
		}
	}
	if !Contains(tls, "o") {
		fmt.Println(tls, " does not contain O")
		return errors.New("Team Orangered spawn missing")
	}
	if !Contains(tls, "p") {
		return errors.New("Team Periwinkle spawn missing")
	}
	if !Contains(tls, "f") {
		return errors.New("Flag missing")
	}
	if !Contains(tls, "g") {
		return errors.New("Goal missing")
	}
	// only valid tiles
	// dimensions within limits
	// contains 1 of O, P, F, G
	return nil
}

func (self *UserMap) save() error {
	session := getSession()
	defer session.Close()
	usermaps := session.DB(databaseName).C("usermaps")
	_, err := usermaps.Upsert(bson.M{"mapid": self.MapId}, self)
	if err != nil {
		return err
	}
	return nil
}

func (self *UserMap) load(id string) error {
	session := getSession()
	defer session.Close()
	usermaps := session.DB(databaseName).C("usermaps")
	err := usermaps.Find(bson.M{"mapid": id}).One(&self)
	if err != nil {
		return err
	}
	return nil
}

func (self *UserMap) remove(id string) error {
	session := getSession()
	defer session.Close()
	usermaps := session.DB(databaseName).C("usermaps")
	err := usermaps.Remove(bson.M{"mapid": id})
	if err != nil {
		return err
	}
	return nil
}

type User struct {
	Name      string
	Karma     int
	Settings  *Settings
	Sprite    string
	Tileset   string
	Tilesets  []string
	Sprites   []string
	Maps      []string
	Elo       int
	Admin     bool
	Guest     bool
	Moderator bool
}

type Settings struct {
	Graphics   string
	Audio      int
	Music      int
	Forwards   int
	Forwards2  int
	Backwards  int
	Backwards2 int
	Jump       int
	Jump2      int
	Down       int
	Down2      int
	Attack     int
	Attack2    int
	Stat       int
	Tileset    string
}

func (self *User) save() error {
	session := getSession()
	defer session.Close()
	users := session.DB(databaseName).C("users")
	//self.Name = strings.ToLower(self.Name)
	_, err := users.Upsert(bson.M{"name": self.Name}, self)
	if err != nil {
		return err
	}
	return nil
}

func (self *User) load(name string) error {
	session := getSession()
	defer session.Close()
	users := session.DB(databaseName).C("users")
	//name = strings.ToLower(name)
	err := users.Find(bson.M{"name": name}).One(&self)
	if err != nil {
		return err
	}
	return nil
}

func (self *User) remove(name string) error {
	session := getSession()
	defer session.Close()
	users := session.DB(databaseName).C("users")
	err := users.Remove(bson.M{"name": name})
	if err != nil {
		return err
	}
	return nil
}

// config data gets loaded from local json file
type Config struct {
	RedditSecret string
	RedditId     string
	UserAgent    string
	RedirectURL  string
	CookieSecret string
}

// load process reads local json file to fill config struct
func (self *Config) load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &self)
	if err != nil {
		return err
	}
	return nil
}

type GameRecord struct {
	User   string
	Game   string
	GameID string
	Score  int
	Win    bool
}

func (self *GameRecord) save() error {
	session := getSession()
	defer session.Close()
	collection := session.DB(databaseName).C("gamerecords")
	_, err := collection.Upsert(bson.M{"GameID": self.GameID}, self)
	if err != nil {
		return err
	}
	return nil
}

func (self *GameRecord) load(GameID string) error {
	session := getSession()
	defer session.Close()
	collection := session.DB(databaseName).C("gamerecords")
	err := collection.Find(bson.M{"GameID": GameID}).One(&self)
	if err != nil {
		return err
	}
	return nil
}

func (self *GameRecord) remove(id string) error {
	session := getSession()
	defer session.Close()
	usermaps := session.DB(databaseName).C("gamerecords")
	err := usermaps.Remove(bson.M{"GameId": id})
	if err != nil {
		return err
	}
	return nil
}

func getSession() *mgo.Session {
	if mgoSession == nil {
		var err error
		mgoSession, err = mgo.Dial("localhost")
		safe := &mgo.Safe{W: 1, WTimeout: 20, FSync: true}
		mgoSession.SetSafe(safe)
		if err != nil {
			fmt.Printf("Session error!: %+v", err)
		}
	}
	return mgoSession.Clone()
}

type room struct {
	Sockets   map[*socket]bool
	Broadcast chan string
	Open      chan *socket
	Close     chan *socket
	Kill      chan bool
	Name      string
	GameId    string
}

func (r *room) run() {
	for {
		select {
		case s := <-r.Open:
			r.Sockets[s] = true
		case s := <-r.Close:
			delete(r.Sockets, s)
			close(s.Send)
		case m := <-r.Broadcast:
			for s := range r.Sockets {
				select {
				case s.Send <- m:
				default:
					delete(r.Sockets, s)
					close(s.Send)
					go s.Close()
				}
			}
		case <-r.Kill:
			for s := range r.Sockets {
				delete(r.Sockets, s)
				close(s.Send)
				go s.Close()
			}
			return
		}
	}
}

type msg struct {
	Key       string `json:"key"`
	Direction string `json:"direction"`
	Text      string `json:"text"`
}

type keypress struct {
	Key       string
	Direction string
}

type socket struct {
	Ws       *websocket.Conn
	Send     chan string
	Username string
	GameId   string
	Char     *cto.Char
}

func (s *socket) Read(output chan string) {
	defer s.Close()
	key := make(chan keypress)
	go s.keyHandler(key, s.Char)
	for {
		var unmarshal []byte
		err := websocket.Message.Receive(s.Ws, &unmarshal)
		if err != nil {
			fmt.Println(err)
			return
		}
		//fmt.Println("Unmarshal:",string(unmarshal))
		if string(unmarshal[2:7]) == "Audio" {
			// apply settings change to user...
			fmt.Println("SETTIN SOME SETTINGS")
			name, found := getSocketCookie(s.Ws, "user")
			if found {
				var settings = &Settings{}
				err = json.Unmarshal(unmarshal, &settings)
				fmt.Println("Settings:", settings)
				if err != nil {
					fmt.Println(err)
				} else {
					user, err := getUser(name)
					if err != nil {
						fmt.Println(err)
					} else {
						/*/ loop through settings message and set user settings
						  for k,v := range settings.__dict__... {
						      if v != "" {
						          user.Settings[k] = v
						      }
						  }
						  /*/
						user.Settings = settings
						user.save()
					}
				}
			}
			// getsocketuser, user.settings = unmarshal.settings, user.save
		}
		if string(unmarshal[2:5]) == "key" {
			var message = &msg{}
			err = json.Unmarshal(unmarshal, &message)
			if err != nil {
				fmt.Println(err)
			}
			if message.Key != "" {
				k := &keypress{Key: message.Key, Direction: message.Direction}
				//fmt.Println(*k)
				key <- *k
			}
		}
		//output <- string(unmarshal)
	}
}

func (self *socket) keyHandler(k chan keypress, ch *cto.Char) {
	fmt.Printf("Socket keyhandler started for %v.\n", ch.Name)
	bstop := make(chan bool)
	fstop := make(chan bool)
	jstop := make(chan bool)
	//astop := make(chan bool)
	var bdown bool
	var fdown bool
	var jdown bool
	var adown bool
	for {
		if ch.Dead {
			bdown = false
			fdown = false
			jdown = false
			adown = false
		}
		select {
		//default:
		//	time.Sleep(100 * time.Microsecond)
		case m := <-k:
			if m.Key == "backwards" && !ch.Dead {
				if m.Direction == "down" && !bdown {
					bdown = true
					go ch.Backward(bstop)
				} else if m.Direction == "up" && bdown {
					bdown = false
					if ch.Backwards {
						bstop <- true
					}
				}
			}
			if m.Key == "forwards" && !ch.Dead {
				if m.Direction == "down" && !fdown {
					fdown = true
					go ch.Forward(fstop)
				} else if m.Direction == "up" && fdown {
					fdown = false
					if ch.Forwards {
						fstop <- true
					}
				}
			}
			if m.Key == "jump" && !ch.Dead {
				if m.Direction == "down" && !jdown {
					//fmt.Println("Jump down triggered")
					jdown = true
					go ch.Jump(jstop)
				} else if m.Direction == "up" && jdown {
					//fmt.Println("Jump up triggered")
					jdown = false
					jstop <- true
				}
			}
			if m.Key == "attack" && !ch.Dead {
				if m.Direction == "down" && !adown {
					adown = true
					go ch.Attack()
				} else if m.Direction == "up" && adown {
					adown = false
				}
			}
		}
	}
}

func (s *socket) Write() {
	for message := range s.Send {
		err := websocket.Message.Send(s.Ws, message)
		if err != nil {
			break
		}
	}
	s.Close()
}

func (s *socket) Close() {
	fmt.Println("Closing socket for", s.Username)
	go games[s.GameId].Disconnect(s.Username)
	s.Ws.Close()
}

/* ===========================================================================
                             HELPER FUNCS
=========================================================================== */

func ActiveGames() int {
	i := 0
	for _, g := range games {
		if !g.Ended {
			i++
		}
	}
	return i
}

func GetNewRoomId() string {
	id := randInt(0, 10)
	gid := strconv.Itoa(id)
	if _, exists := games[gid]; exists {
		// game exists, check if it's dead
		if !games[gid].Ended {
			// game is active, get a new ID!
			gid = GetNewRoomId()
		}
	}
	return gid
}

func GetNewMapId() string {
	h := randString(8)
	m := UserMap{}
	e := m.load(h)
	if e != nil { // load didn't find anything, use this hash
		er := fmt.Sprintf("%+v", e)
		fmt.Println("Error == 'not found'?", er == "not found")
		return h
	}
	return GetNewMapId()
}

func randString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	cr.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func SetGuest(b bool) {
	// TODO log that the var changed
	GUEST = b
}

func SetMarket(b bool) {
	// TODO log that the var changed
	MARKET = b
}

func Contains(sl []string, el string) bool {
	for _, i := range sl {
		if i == el {
			return true
		}
	}
	return false
}

// perform config load and initialize grow library with required oauth fields
func Init() {
	err := conf.load("./config.json")
	if err != nil {
		fmt.Println(err)
	}
	grow.Config(conf.UserAgent, "identity", conf.RedditId, conf.RedditSecret, conf.RedirectURL)

	// build all the games, socket channels, and start em
	/*/
		for i := 0; i < 9; i++ {
			kill := make(chan bool)
			rooms[i] = &room{
				Broadcast: make(chan string),
				Sockets:   make(map[*socket]bool),
				Open:      make(chan *socket),
				Close:     make(chan *socket),
				Name:      "Default",
				GameId:    i,
				Kill:      kill,
			}
	        m := UserMap{}
	        m.load("4")
	        if MAP {
	            games[i] = cto.NewGame(rooms[i].Broadcast, kill, m.Tiles)
	        } else {
	            games[i] = cto.NewGame(rooms[i].Broadcast, kill, cto.Map2())
			}
	        go rooms[i].run()
			games[i].Id = i
		}
	    /*/
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func getCurrentUser(ctx *web.Context) (*User, error) {
	usr, found := ctx.GetSecureCookie("user")
	u := &User{}
	if !found {
		//return u, fmt.Errorf("User not found: %v!", usr)
	}
	err := u.load(usr)
	if u.Name == "" && usr != "" {
		// Guest users shouldn't get saved
		u.Name = usr
		return u, nil
	}
	return u, err
}

func getUser(name string) (*User, error) {
	u := &User{}
	err := u.load(name)
	if u.Name == "" && name != "" {
		// Guest users shouldn't get saved
		u.Name = name
		return u, nil
	}
	return u, err

}

/* ===========================================================================
                             HTTP HANDLERS
=========================================================================== */

func loginGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, _ := getCurrentUser(ctx)
	if user.Name != "" {
		data["user"] = user
	}
	data["guest"] = GUEST
	if !GUEST {
		ctx.Redirect(302, "/reddit")
		return ""
	}
	return mustache.RenderFileInLayout("templates/login.html", "templates/base.html", data)
}

func logoutGet(ctx *web.Context) {
	// remove the user cookie
	ctx.SetSecureCookie("user", "", -1)
	ctx.Redirect(302, "/")
}

func marketGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, _ := getCurrentUser(ctx)
	if user.Name != "" {
		data["user"] = user
	}
	return mustache.RenderFileInLayout("templates/market.html", "templates/base.html", data)
}

func indexGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, _ := getCurrentUser(ctx)
	if user.Name != "" {
		data["user"] = user
	}
	return mustache.RenderFileInLayout("templates/index.html", "templates/base.html", data)
}

func aboutGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, _ := getCurrentUser(ctx)
	if user.Name != "" {
		data["user"] = user
	}
	return mustache.RenderFileInLayout("templates/about.html", "templates/base.html", data)
}

func statsGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, _ := getCurrentUser(ctx)
	if user.Name != "" {
		data["user"] = user
	}
	return mustache.RenderFileInLayout("templates/stats.html", "templates/base.html", data)
}

func flushGet(ctx *web.Context) {
	data := make(map[string]interface{})
	user, _ := getCurrentUser(ctx)
	if user.Name != "" {
		data["user"] = user
		if user.Name == "Kamoi" && user.Admin == false {
			user.Admin = true
			user.save()
		}
	}
	if !user.Admin {
		ctx.Redirect(302, "/")
	}
}

func adminGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, _ := getCurrentUser(ctx)
	if user.Name != "" {
		data["user"] = user
		if user.Name == "Kamoi" && user.Admin == false {
			user.Admin = true
			user.save()
		}
	}
	if !user.Admin {
		ctx.Redirect(302, "/")
	}
	data["maxgames"] = MAX_GAMES
	data["maxwidth"] = MAX_WIDTH
	data["maxheight"] = MAX_HEIGHT
	data["guest"] = GUEST
	data["map"] = MAP
	data["market"] = MARKET
	return mustache.RenderFileInLayout("templates/admin.html", "templates/base.html", data)
}

func adminPost(ctx *web.Context) {
	user, _ := getCurrentUser(ctx)
	if !user.Admin {
		ctx.Redirect(302, "/")
		return
	}
	mg := ctx.Params["maxgames"]
	mw := ctx.Params["maxwidth"]
	mh := ctx.Params["maxheight"]
	gu := ctx.Params["guest"]
	ma := ctx.Params["map"]
	mk := ctx.Params["market"]
	if mg != "" {
		mg, e := strconv.Atoi(mg)
		if e == nil {
			MAX_GAMES = mg
		}
	}
	if mw != "" {
		mw, e := strconv.Atoi(mw)
		if e == nil {
			MAX_WIDTH = mw
		}
	}
	if mh != "" {
		mh, e := strconv.Atoi(mh)
		if e == nil {
			MAX_HEIGHT = mh
		}
	}
	if gu != "" {
		if gu == "true" {
			GUEST = true
		} else {
			GUEST = false
		}
	}
	if ma != "" {
		if ma == "true" {
			MAP = true
		} else {
			MAP = false
		}
	}
	if mk != "" {
		if mk == "true" {
			MARKET = true
		} else {
			MARKET = false
		}
	}
	ctx.Redirect(302, "/admin")
}

func mapGet(ctx *web.Context) string {
	if !MAP {
		ctx.Redirect(302, "/cto/")
		return ""
	}
	data := make(map[string]interface{})
	user, err := getCurrentUser(ctx)
	if err != nil {
		ctx.Redirect(302, "/login")
		return ""
	}
	if user.Name != "" {
		data["user"] = user
	}
	data["maxwidth"] = MAX_WIDTH
	data["maxheight"] = MAX_HEIGHT
	h := GetNewMapId()
	usermaphash[user.Name] = h
	return mustache.RenderFileInLayout("templates/map.html", "templates/base.html", data)
}

func mapPost(ctx *web.Context) {
	if !MAP {
		ctx.Redirect(302, "/cto/")
		return
	}
	user, err := getCurrentUser(ctx)
	m := ctx.Params["m"]
	maptiles := &MapJson{}
	fmt.Println("M: ", m)
	err = json.Unmarshal([]byte(m), maptiles)
	if err != nil {
		fmt.Println("Unmarshal error: ", err)
		return
	}
	umap := UserMap{}
	umap.MapId = usermaphash[user.Name]
	umap.Name = ctx.Params["name"]
	umap.Owner = user.Name
	umap.Tiles = make(map[string]map[string]string)
	for i, s := range maptiles.M {
		r := make(map[string]string)
		for j, t := range s {
			r[j] = t
		}
		umap.Tiles[i] = r
	}
	v := umap.validate()
	if v == nil {
		e := umap.save()
		if e != nil {
			fmt.Println("Save Error:", e)
		}
	} else {
		// oh!
		fmt.Println("Map failed to validate: ", v)
		ctx.Redirect(302, "/tools/map")
		return
	}
	if !Contains(user.Maps, umap.MapId) {
		user.Maps = append(user.Maps, umap.MapId)
		user.save()
	}
	ctx.Redirect(302, "/newcto")
}

func accountGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, err := getCurrentUser(ctx)
	if err != nil {
		ctx.Redirect(302, "/login")
		return ""
	}
	if user.Name != "" {
		data["user"] = user
	}
	data["sprites"] = Sprites
	return mustache.RenderFileInLayout("templates/account.html", "templates/base.html", data)
}

func accountPost(ctx *web.Context) {
	user, _ := getCurrentUser(ctx)
	sp := ctx.Params["sprite"]
	if sp != "" {
		if Contains(Sprites, sp) || Contains(user.Sprites, sp) {
			user.Sprite = sp
		}
	}
	user.save()
}

func ctoGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, _ := getCurrentUser(ctx)
	if user.Name != "" {
		data["user"] = user
	}
	g := []*cto.Level{}
	for _, ga := range games {
		g = append(g, ga)
	}
	data["games"] = g
	return mustache.RenderFileInLayout("templates/cto.html", "templates/base.html", data)
}

// pass http handlers to grow library for oauth redirect
func handleAuthorize(ctx *web.Context) {
	grow.Authorize(ctx.ResponseWriter, ctx.Request)
}

func guestLoginGet(ctx *web.Context) {
	name := fmt.Sprintf("Guest%d", randInt(0, 10000))
	ctx.SetSecureCookie("user", name, 36000)
	ctx.Redirect(302, "/")
}

// callback from reddit, process oauth response
func handleOAuth2Callback(ctx *web.Context) {
	user, err := grow.Authorized(ctx.ResponseWriter, ctx.Request)
	u := &User{}
	u.Name = user.Name
	u.Karma = int(user.Comment_karma) + int(user.Link_karma)
	u.save()
	//fmt.Println(user.Name)
	_ = err
	ctx.SetSecureCookie("user", user.Name, 3600)
	ctx.Redirect(302, "/")
}

func tutSocketHandler(ws *websocket.Conn) {
	//fmt.Println("Tutorial Socket opened.")
	name, found := getSocketCookie(ws, "user")
	if found {
		tut := ustut[name]
		stahp := make(chan bool)
		s := &socket{ws, make(chan string, 256), name, "0", tut.GetPlayer()}
		tut.Init(stahp, s.Send)
		go s.Write()
		s.Read(s.Send)
	}
}

func getCookieSig(key string, val []byte, timestamp string) string {
	hm := hmac.New(sha1.New, []byte(key))

	hm.Write(val)
	hm.Write([]byte(timestamp))

	hex := fmt.Sprintf("%02x", hm.Sum(nil))
	return hex
}

func getSocketCookie(ws *websocket.Conn, name string) (string, bool) {
	cookie, _ := ws.Request().Cookie(name)
	parts := strings.SplitN(cookie.Value, "|", 3)

	val := parts[0]
	timestamp := parts[1]
	sig := parts[2]

	if getCookieSig(conf.CookieSecret, []byte(val), timestamp) != sig {
		return "", false
	}

	ts, _ := strconv.ParseInt(timestamp, 0, 64)

	if time.Now().Unix()-31*86400 > ts {
		return "", false
	}

	buf := bytes.NewBufferString(val)
	encoder := base64.NewDecoder(base64.StdEncoding, buf)
	res, _ := ioutil.ReadAll(encoder)
	return string(res), true
}

func socketHandler(ws *websocket.Conn) {
	name, _ := getSocketCookie(ws, "user")
	gameid := users[name]
	c := &cto.Char{}
	for _, ch := range games[gameid].Chars {
		if strings.ToLower(ch.Name) == strings.ToLower(name) {
			c = ch
		}
	}
	s := &socket{ws, make(chan string, 256), name, gameid, c}
	defer func() { rooms[s.GameId].Close <- s }()
	rooms[s.GameId].Open <- s
	go s.Write()
	s.Read(rooms[s.GameId].Broadcast)
}

func newGamePost(ctx *web.Context) {
	if ActiveGames() >= MAX_GAMES {
		ctx.Redirect(302, "/cto/")
		return
	}
	mapid := ctx.Params["mapid"]
	fmt.Println("Map ID:", mapid)
	gid := GetNewRoomId()
	kill := make(chan bool)
	rooms[gid] = &room{
		Broadcast: make(chan string),
		Sockets:   make(map[*socket]bool),
		Open:      make(chan *socket),
		Close:     make(chan *socket),
		Name:      "Default",
		GameId:    gid,
		Kill:      kill,
	}
	m := UserMap{}
	m.load(mapid)
	e := m.validate()
	if e != nil {
		m.load("0")
	}
	if MAP {
		games[gid] = cto.NewGame(rooms[gid].Broadcast, kill, m.Tiles)
	} else {
		games[gid] = cto.NewGame(rooms[gid].Broadcast, kill, cto.Map2())
	}
	go rooms[gid].run()
	games[gid].Id = gid
	red := fmt.Sprintf("/cto/%v", gid)
	ctx.Redirect(302, red)
}

func newGameGet(ctx *web.Context) string {
	data := make(map[string]interface{})
	user, err := getCurrentUser(ctx)
	if err != nil {
		ctx.Redirect(302, "/login")
	}
	data["user"] = user
	if MAP {
		um := make([]UserMap, len(user.Maps))
		for _, m := range user.Maps {
			u := UserMap{}
			u.load(m)
			um = append(um, u)
		}
		data["usermaps"] = um
		fmt.Println("User Maps:", um)
		fmt.Println("User.Maps:", user.Maps)
	}
	def.save()
	data["maps"] = Maps
	return mustache.RenderFileInLayout("templates/newgame.html", "templates/base.html", data)
}

func tutorialGet(ctx *web.Context) string {
	// present user with a list of tutorials if he's already completed some, otherwise go to tut0
	data := make(map[string]interface{})
	user, err := getCurrentUser(ctx)
	if err != nil {
		fmt.Printf("Tutorial Error: %+v\n", err)
		ctx.Redirect(302, "/login")
	}
	data["user"] = user
	tuts := []cto.TutLevel{}
	for _, tut := range cto.Tutorials {
		tuts = append(tuts, tut)
	}
	data["tutorials"] = tuts
	return mustache.RenderFileInLayout("templates/tutorials.html", "templates/base.html", data)
}

func tutorialHandler(ctx *web.Context, tid string) string {
	tutid, err := strconv.Atoi(tid)
	if err != nil { //
		ctx.Redirect(302, "/cto/tutorial/")
		return ""
	}
	data := make(map[string]interface{})
	user, err := getCurrentUser(ctx)
	if err != nil {
		ctx.Redirect(302, "/login")
	}
	data["user"] = user
	tutorial := cto.GetTutorial(tutid - 1)
	ustut[user.Name] = tutorial
	tl := tutorial.GetRandomSpawn("Orangered")
	tutorial.AddPlayer(user.Name, tl.X, tl.Y+2, cto.CHAR_WIDTH, cto.CHAR_HEIGHT, "Orangered")
	data["level"] = tutorial
	data["tiles"] = tutorial.GetTiles()
	var a = make([]*cto.Char, 0)
	for _, ch := range tutorial.GetChars() {
		a = append(a, ch)
	}
	data["sprites"] = a
	data["me"] = user.Name
	if tutid >= len(cto.Tutorials) {
		data["next"] = "/cto/tutorial/"
	} else {
		data["next"] = tutid + 1
	}
	return mustache.RenderFile("templates/tutorial.html", data)
}

func gameHandler(ctx *web.Context, gameid string) string {
	data := make(map[string]interface{})
	user, err := getCurrentUser(ctx)
	if err != nil {
		ctx.Redirect(302, "/login")
		return ""
	}
	if user.Name != "" {
		data["user"] = user
		users[user.Name] = gameid
	}
	if _, exists := games[gameid]; !exists { // game isn't in the map, send em to the directory
		ctx.Redirect(302, "/cto/")
		return ""
	}
	if games[gameid].Ended { // game exists but isn't running, send em to game setup
		ctx.Redirect(302, "/newcto")
		return ""
	}
	if games[gameid].IsFull() { // game is full, abort
		ctx.Redirect(302, "/cto/")
		return ""
	} else { // game exists, is running, has room -- join up!
		pect := games[gameid].CountTeam("Periwinkle")
		orct := games[gameid].CountTeam("Orangered")
		t := ""
		if pect < orct {
			t = "Periwinkle"
		} else {
			t = "Orangered"
		}
		tl := games[gameid].GetRandomSpawn(t)
		if user.Sprite != "" {
			games[gameid].AddChar(user.Name, tl.X+cto.TILE_SIZE/2-cto.CHAR_WIDTH/2, tl.Y+2, cto.CHAR_WIDTH, cto.CHAR_HEIGHT, t, user.Sprite)
		} else {
			games[gameid].AddChar(user.Name, tl.X+cto.TILE_SIZE/2-cto.CHAR_WIDTH/2, tl.Y+2, cto.CHAR_WIDTH, cto.CHAR_HEIGHT, t, "Tophat")
		}
		data["tiles"] = games[gameid].Tiles
		data["level"] = games[gameid]
		data["sprites"] = games[gameid].Chars
		var a = make([]*cto.Char, 0)
		for key := range games[gameid].Chars {
			a = append(a, games[gameid].Chars[key])
		}
		data["sprites"] = a
		data["me"] = user.Name
		flool, flag := games[gameid].GetFlag()
		if flool {
			data["Flag"] = flag
		}
		data["Periwinkle"] = games[gameid].PeriwinkleScore
		data["Orangered"] = games[gameid].OrangeredScore
		data["Gameid"] = gameid
		data["CapMax"] = cto.CAPTURE_MAX
		data["settings"] = user.Settings
		fmt.Println("User settings:", user.Settings)
		return mustache.RenderFile("templates/game.html", data)
	}
}

func main() {
	Init()
	cto.LoadTutorials()
	fmt.Printf("config = %+v\n", conf)
	web.Config.CookieSecret = conf.CookieSecret
	web.Get("/", indexGet)
	web.Get("/about", aboutGet)
	web.Get("/flush", flushGet)
	web.Get("/admin", adminGet)
	web.Post("/admin", adminPost)
	web.Get("/market", marketGet)
	web.Get("/stats", statsGet)
	web.Get("/login", loginGet)
	web.Get("/logout", logoutGet)
	web.Get("/account", accountGet)
	web.Post("/account", accountPost)
	web.Get("/tools/map", mapGet)
	web.Post("/tools/map", mapPost)
	web.Get("/cto/", ctoGet)
	web.Get("/reddit", handleAuthorize)
	if GUEST {
		web.Get("/guest", guestLoginGet)
	}
	web.Get("/auth", handleOAuth2Callback)
	web.Get("/newcto", newGameGet)
	web.Post("/newcto", newGamePost)
	web.Get("/cto/tutorial/", tutorialGet)
	web.Get("/cto/tutorial/([0-9])", tutorialHandler)
	web.Get("/cto/([0-9])", gameHandler)
	web.Websocket("/socket", websocket.Handler(socketHandler))
	web.Websocket("/tutsocket", websocket.Handler(tutSocketHandler))
	web.Run("0.0.0.0:10000")
}
