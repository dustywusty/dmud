package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The auto-mapper builds a graph of rooms as the player explores. Rooms are
// keyed by their title (the server gives titles, not ids). Coordinates are
// inferred from the direction of each movement command and the room block that
// follows it. A pending queue lets a speedwalk (several moves at once) map
// correctly, since cur advances with each room block.

type coord struct{ x, y int }

type mroom struct {
	title string
	pos   coord
	exits map[string]bool // canonical directions that have an exit
}

type mapper struct {
	rooms    map[string]*mroom
	occupied map[coord]string // grid cell -> room title, for collision-free placement
	cur      string
	pending  []string // queued movement directions awaiting room blocks
}

func newMapper() *mapper {
	return &mapper{
		rooms:    map[string]*mroom{},
		occupied: map[coord]string{},
	}
}

// dirCanon maps movement tokens (and their aliases) to a canonical direction.
var dirCanon = map[string]string{
	"n": "north", "s": "south", "e": "east", "w": "west", "u": "up", "d": "down",
	"north": "north", "south": "south", "east": "east", "west": "west", "up": "up", "down": "down",
}

// dirDelta gives the grid offset for the four planar directions. Up/down have
// no 2D offset and are placed in the nearest free cell instead.
var dirDelta = map[string]coord{
	"north": {0, -1}, "south": {0, 1}, "east": {1, 0}, "west": {-1, 0},
}

// noteCommand queues a movement so the next room block can be linked.
func (mp *mapper) noteCommand(line string) {
	f := strings.Fields(line)
	if len(f) == 0 {
		return
	}
	if d, ok := dirCanon[strings.ToLower(f[0])]; ok {
		mp.pending = append(mp.pending, d)
	}
}

// arrive integrates a room block (title + exits). It returns true when the room
// was not previously mapped.
func (mp *mapper) arrive(title string, exits []string) bool {
	if title == "" {
		return false
	}

	// Re-describing the current room (look, or a failed move): drop any stale
	// pending direction and just refresh exits.
	if title == mp.cur {
		mp.pending = nil
		mp.setExits(title, exits)
		return false
	}

	var dir string
	if len(mp.pending) > 0 {
		dir = mp.pending[0]
		mp.pending = mp.pending[1:]
	}

	isNew := false
	if _, known := mp.rooms[title]; !known {
		pos := mp.place(dir)
		mp.rooms[title] = &mroom{title: title, pos: pos, exits: map[string]bool{}}
		mp.occupied[pos] = title
		isNew = true
	}
	mp.setExits(title, exits)
	mp.cur = title
	return isNew
}

func (mp *mapper) setExits(title string, exits []string) {
	r := mp.rooms[title]
	if r == nil {
		return
	}
	r.exits = map[string]bool{}
	for _, e := range exits {
		if d, ok := dirCanon[strings.ToLower(strings.TrimSpace(e))]; ok {
			r.exits[d] = true
		}
	}
}

// place decides where a newly seen room goes on the grid.
func (mp *mapper) place(dir string) coord {
	if mp.cur == "" {
		return coord{0, 0}
	}
	from := mp.rooms[mp.cur]
	if from != nil && dir != "" {
		if d, ok := dirDelta[dir]; ok {
			want := coord{from.pos.x + d.x, from.pos.y + d.y}
			if _, taken := mp.occupied[want]; !taken {
				return want
			}
			return mp.placeNear(want)
		}
		return mp.placeNear(from.pos) // up/down
	}
	return mp.placeNear(mp.curPos())
}

func (mp *mapper) curPos() coord {
	if r := mp.rooms[mp.cur]; r != nil {
		return r.pos
	}
	return coord{0, 0}
}

// placeNear returns c if free, otherwise the closest unoccupied cell.
func (mp *mapper) placeNear(c coord) coord {
	if _, taken := mp.occupied[c]; !taken {
		return c
	}
	for radius := 1; radius < 64; radius++ {
		for dx := -radius; dx <= radius; dx++ {
			for dy := -radius; dy <= radius; dy++ {
				p := coord{c.x + dx, c.y + dy}
				if _, taken := mp.occupied[p]; !taken {
					return p
				}
			}
		}
	}
	return c
}

// Room boxes are drawn "[X]" with connectors filling the gaps between them:
//
//	[@]───[ ]
//	 │
//	[◊]
//
// mapCellW/mapCellH are the screen distances between adjacent room centers.
const (
	mapCellW = 6
	mapCellH = 2
)

// render draws the map into a w×h block centered on the current room, using
// boxed rooms and line connectors. A connector is only drawn when both adjacent
// rooms agree on the exit; the current room's unexplored exits get a stub tick.
func (mp *mapper) render(w, h int) string {
	if w < 5 || h < 1 {
		return ""
	}
	if mp.cur == "" && len(mp.rooms) == 0 {
		return dimStyle.Render("(unmapped — move to explore)")
	}

	grid := make([][]rune, h)
	for i := range grid {
		grid[i] = make([]rune, w)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}
	set := func(x, y int, r rune) {
		if x >= 0 && x < w && y >= 0 && y < h {
			grid[y][x] = r
		}
	}
	put := func(x, y int, s string) {
		for i, r := range []rune(s) {
			set(x+i, y, r)
		}
	}
	roomAt := func(c coord) *mroom {
		if title, ok := mp.occupied[c]; ok {
			return mp.rooms[title]
		}
		return nil
	}

	center := mp.curPos()
	cx, cy := w/2, h/2
	colOf := func(gx int) int { return cx + (gx-center.x)*mapCellW } // box center
	rowOf := func(gy int) int { return cy + (gy-center.y)*mapCellH }

	for _, r := range mp.rooms {
		bx, by := colOf(r.pos.x), rowOf(r.pos.y)
		// Connectors, drawn only when both rooms agree (east/south cover each edge).
		if r.exits["east"] {
			if e := roomAt(coord{r.pos.x + 1, r.pos.y}); e != nil && e.exits["west"] {
				put(bx+2, by, "───") // fills the gap between "]" and the next "["
			}
		}
		if r.exits["south"] {
			if s := roomAt(coord{r.pos.x, r.pos.y + 1}); s != nil && s.exits["north"] {
				set(bx, by+1, '│')
			}
		}
		// The room box: "[X]" where X marks current / stairwell rooms.
		vertical := r.exits["up"] || r.exits["down"]
		glyph := ' '
		switch {
		case r.title == mp.cur && vertical:
			glyph = '◈'
		case r.title == mp.cur:
			glyph = '@'
		case vertical:
			glyph = '◊'
		}
		set(bx-1, by, '[')
		set(bx, by, glyph)
		set(bx+1, by, ']')
	}

	// Current room's exits, like a compass on the map: unexplored directions get
	// a stub tick just outside the box (explored ones already have a connector).
	if cur := mp.rooms[mp.cur]; cur != nil {
		bx, by := colOf(cur.pos.x), rowOf(cur.pos.y)
		stub := func(x, y int, r rune) {
			if x >= 0 && x < w && y >= 0 && y < h && grid[y][x] == ' ' {
				grid[y][x] = r
			}
		}
		if cur.exits["north"] {
			stub(bx, by-1, '╵')
		}
		if cur.exits["south"] {
			stub(bx, by+1, '╷')
		}
		if cur.exits["east"] {
			stub(bx+2, by, '╶')
		}
		if cur.exits["west"] {
			stub(bx-2, by, '╴')
		}
	}

	var b strings.Builder
	for i, row := range grid {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(string(row))
	}
	return styleMap(b.String())
}

// styleMap colors the plain map grid: current room, other rooms, connectors.
func styleMap(s string) string {
	s = strings.ReplaceAll(s, "◈", roomNameStyle.Render("◈"))
	s = strings.ReplaceAll(s, "◊", exitStyle.Render("◊"))
	s = strings.ReplaceAll(s, "@", roomNameStyle.Render("@"))
	s = strings.ReplaceAll(s, "─", dimStyle.Render("─"))
	s = strings.ReplaceAll(s, "│", dimStyle.Render("│"))
	// Unexplored current-room exit stubs (green, like the compass).
	for _, tick := range []string{"╵", "╷", "╶", "╴"} {
		s = strings.ReplaceAll(s, tick, exitStyle.Render(tick))
	}
	return s
}

// --- persistence ---

type mapData struct {
	Cur   string     `json:"cur"`
	Rooms []roomData `json:"rooms"`
}

type roomData struct {
	Title string   `json:"title"`
	X     int      `json:"x"`
	Y     int      `json:"y"`
	Exits []string `json:"exits"`
}

func (mp *mapper) snapshot() mapData {
	d := mapData{Cur: mp.cur}
	for _, r := range mp.rooms {
		var exits []string
		for dir := range r.exits {
			exits = append(exits, dir)
		}
		sort.Strings(exits)
		d.Rooms = append(d.Rooms, roomData{Title: r.title, X: r.pos.x, Y: r.pos.y, Exits: exits})
	}
	sort.Slice(d.Rooms, func(i, j int) bool { return d.Rooms[i].Title < d.Rooms[j].Title })
	return d
}

func (mp *mapper) restore(d mapData) {
	for _, rd := range d.Rooms {
		r := &mroom{title: rd.Title, pos: coord{rd.X, rd.Y}, exits: map[string]bool{}}
		for _, e := range rd.Exits {
			r.exits[e] = true
		}
		mp.rooms[rd.Title] = r
		mp.occupied[r.pos] = rd.Title
	}
	mp.cur = d.Cur
}

// mapFilePath returns the per-server map file, e.g.
// ~/.config/dmud-client/maps/localhost_8080.json.
func mapFilePath(addr string) string {
	dir := configDir()
	if dir == "" {
		return ""
	}
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, addr)
	return filepath.Join(dir, "maps", safe+".json")
}

func loadMapData(path string) mapData {
	if path == "" {
		return mapData{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return mapData{}
	}
	var d mapData
	if err := json.Unmarshal(data, &d); err != nil {
		return mapData{}
	}
	return d
}

func saveMapData(path string, d mapData) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
