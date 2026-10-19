package main

import (
	tea "charm.land/bubbletea/v2"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/charmbracelet/colorprofile"
	"io"
	"math"
	"os"
	"path/filepath"
	"sea-battle-2/internal/bot"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/play"
	"sea-battle-2/internal/visual"
	"strings"
	"time"
)

type tick time.Time
type model struct {
	state          visual.State
	width, height  int
	preferSD, blur bool
	last           time.Time
	notice         string
}

func pulse() tea.Cmd          { return tea.Tick(time.Second/24, func(t time.Time) tea.Msg { return tick(t) }) }
func (m model) Init() tea.Cmd { return pulse() }
func (m model) fits() bool    { return m.width >= 80 && m.height >= 24 }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
		m.state.SD = m.preferSD || v.Width < 160 || v.Height < 48
		m.last = time.Time{}
	case tea.BlurMsg:
		m.blur = true
		m.last = time.Time{}
	case tea.FocusMsg:
		m.blur = false
		m.last = time.Time{}
	case tick:
		now := time.Time(v)
		if !m.last.IsZero() && !m.state.Paused && !m.blur && m.fits() && !m.state.Picker && !m.state.Help {
			dt := math.Min(now.Sub(m.last).Seconds(), .12)
			if m.state.Slow {
				dt *= .25
			}
			m.state.T += dt
			s := visual.Scenes[m.state.Scene]
			if m.state.T > s.Duration {
				if m.state.Tour {
					m.change((m.state.Scene + 1) % len(visual.Scenes))
				} else {
					m.state.T = s.Duration
				}
			}
		}
		m.last = now
		return m, pulse()
	case tea.KeyPressMsg:
		key := v.String()
		if key == "ctrl+c" || key == "q" {
			return m, tea.Quit
		}
		if m.state.Help {
			if key == "?" || key == "esc" {
				m.state.Help = false
			}
			return m, nil
		}
		if m.state.Picker {
			switch key {
			case "tab", "esc":
				m.state.Picker = false
			case "up", "k":
				m.state.Selection = max(0, m.state.Selection-1)
			case "down", "j":
				m.state.Selection = min(len(visual.Scenes)-1, m.state.Selection+1)
			case "pgup":
				m.state.Selection = max(0, m.state.Selection-10)
			case "pgdown":
				m.state.Selection = min(len(visual.Scenes)-1, m.state.Selection+10)
			case "home":
				m.state.Selection = 0
			case "end":
				m.state.Selection = len(visual.Scenes) - 1
			case "enter":
				m.change(m.state.Selection)
				m.state.Picker = false
				m.state.Tour = false
			}
			return m, nil
		}
		kind := visual.Scenes[m.state.Scene].Kind
		tactical := kind == visual.Tactical || (kind == visual.Duel && m.state.T >= 12)
		switch key {
		case "tab":
			m.state.Picker = true
			m.state.Selection = m.state.Scene
		case "?":
			m.state.Help = true
		case "right", "n":
			m.change((m.state.Scene + 1) % len(visual.Scenes))
		case "left", "p":
			m.change((m.state.Scene + len(visual.Scenes) - 1) % len(visual.Scenes))
		case "space":
			m.state.Paused = !m.state.Paused
		case "f":
			m.state.Slow = !m.state.Slow
		case "r":
			m.state.T = 0
		case "t":
			m.state.Tour = !m.state.Tour
		case ".":
			m.state.Paused = true
			m.state.T = math.Min(m.state.T+.1, visual.Scenes[m.state.Scene].Duration)
		case ",":
			m.state.Paused = true
			m.state.T = math.Max(0, m.state.T-.1)
		case "v":
			current := m.state.Camera
			if current == "" {
				current = visual.Scenes[m.state.Scene].Camera
			}
			switch current {
			case "TOP":
				m.state.Camera = "SIDE"
			case "SIDE":
				m.state.Camera = "FRONT"
			default:
				m.state.Camera = "TOP"
			}
		case "w":
			if tactical {
				m.state.TargetY = max(0, m.state.TargetY-1)
			} else {
				current := m.state.Weather
				if current < 0 {
					current = visual.Scenes[m.state.Scene].Weather
				}
				m.state.Weather = (current + 1) % 4
			}
		case "s":
			if tactical {
				m.state.TargetY = min(9, m.state.TargetY+1)
			}
		case "a":
			if tactical {
				m.state.TargetX = max(0, m.state.TargetX-1)
			}
		case "d":
			if tactical {
				m.state.TargetX = min(9, m.state.TargetX+1)
			}
		case "x":
			m.state.Alt = !m.state.Alt
		case "b":
			i := 1
			if visual.Scenes[m.state.Scene].Team == 1 {
				i = 2
			}
			m.change(i)
			m.state.Tour = false
		case "enter":
			if tactical && visual.Scenes[m.state.Scene].ID != "deployment" {
				id := "exchange"
				if visual.Scenes[m.state.Scene].Team == 1 {
					id = "counterexchange"
				}
				miss := (m.state.TargetX+m.state.TargetY)%3 == 0
				m.change(visual.Find(id))
				m.state.ShotMiss = miss
				m.state.Paused = false
				m.state.Tour = false
			}
		case "f6":
			if err := saveCapture(m.state); err != nil {
				m.notice = "Capture failed: " + err.Error()
			} else {
				m.notice = "Saved cell snapshot in captures/"
			}
		}
	}
	return m, nil
}
func (m *model) change(i int) {
	m.state.Scene = i
	m.state.ShotMiss = false
	m.state.T = 0
	m.state.Camera = ""
	m.state.Weather = -1
	m.notice = ""
}
func (m model) View() tea.View {
	var content string
	if !m.fits() {
		content = fmt.Sprintf("SEA BATTLE 2\n\nNeed 80 columns × 24 rows. Current: %d × %d.\nEnlarge the terminal or reduce the font size.\nPlayback is suspended. Q exits.", m.width, m.height)
	} else {
		f := visual.Render(m.state)
		if m.notice != "" {
			msg := []rune(m.notice)
			for x := 10; x < f.W-2; x++ {
				c := &f.Cells[(f.H-2)*f.W+x]
				c.Ch = " "
				if x-10 < len(msg) {
					c.Ch = string(msg[x-10])
				}
			}
		}
		lines := strings.Split(f.ANSI(), "\n")
		left := strings.Repeat(" ", max(0, (m.width-f.W)/2))
		for i := range lines {
			lines[i] = left + lines[i]
		}
		content = strings.Repeat("\n", max(0, (m.height-f.H)/2)) + strings.Join(lines, "\n")
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.ReportFocus = true
	v.WindowTitle = "Sea Battle 2 — Operation: Unexpected Guests"
	return v
}
func saveCapture(s visual.State) error {
	if err := os.MkdirAll("captures", 0755); err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%s-%06.2f.json", visual.Scenes[s.Scene].ID, strings.ToLower(s.Size()), s.T)
	b, e := json.Marshal(visual.Render(s))
	if e != nil {
		return e
	}
	return os.WriteFile(filepath.Join("captures", name), b, 0644)
}
func run() error {
	lookdev := flag.Bool("lookdev", false, "open the scripted art review")
	difficulty := flag.String("difficulty", "medium", "solo bot: easy, medium or hard")
	solo := flag.String("solo", "", "single player faction: american or alien")
	host := flag.Bool("host", false, "host a direct battle as the American navy")
	join := flag.String("join", "", "join with a private sb2: invitation")
	joinFile := flag.String("join-file", "", "read an invitation from a file")
	port := flag.Int("port", 4747, "host TCP port (0 chooses an available port)")
	advertise := flag.String("advertise", "", "additional reachable host:port for invitations")
	noMap := flag.Bool("no-map", false, "disable automatic router mapping")

	scene := flag.String("scene", "arrival", "scene ID; see --list")
	size := flag.String("size", "hd", "hd or sd")
	at := flag.Float64("at", 0, "scene seconds for deterministic captures")
	dump := flag.String("dump", "", "json, ansi or text; no UI")
	list := flag.Bool("list", false, "list all scenes")
	camera := flag.String("camera", "", "top, side, front (optional)")
	weather := flag.Int("weather", -1, "-1 scene default; 0 day, 1 night, 2 squall, 3 spore tide")
	paused := flag.Bool("paused", false, "start paused")
	tour := flag.Bool("tour", true, "advance through scenes automatically")
	flag.Parse()
	if *port < 0 || *port > 65535 {
		return fmt.Errorf("port must be 0-65535")
	}
	if *advertise != "" {
		if err := peer.ValidAddress(*advertise); err != nil {
			return err
		}
	}
	if *joinFile != "" {
		if *join != "" {
			return fmt.Errorf("choose --join or --join-file")
		}
		f, err := os.Open(*joinFile)
		if err != nil {
			return err
		}
		b, err := io.ReadAll(io.LimitReader(f, peer.MaxInvitation+1))
		f.Close()
		if err != nil {
			return err
		}
		*join = string(b)
	}
	if !bot.Difficulty(*difficulty).Valid() {
		return fmt.Errorf("difficulty must be easy, medium or hard")
	}
	if *solo != "" && *solo != "american" && *solo != "alien" {
		return fmt.Errorf("solo must be american or alien")
	}
	if *solo != "" && (*host || *join != "" || *advertise != "") {
		return fmt.Errorf("choose solo or multiplayer")
	}
	if *host && *join != "" {
		return fmt.Errorf("choose host or join")
	}
	artMode := *lookdev
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "scene", "at", "dump", "list", "camera", "weather", "paused", "tour":
			artMode = true
		}
	})
	if artMode && (*host || *join != "" || *solo != "") {
		return fmt.Errorf("lookdev and a live battle are separate modes")
	}
	if *size != "hd" && *size != "sd" {
		return fmt.Errorf("size must be hd or sd")
	}
	if !artMode {
		if *join != "" {
			if _, err := peer.ParseInvitation(*join); err != nil {
				return err
			}
		}
		opts := play.Options{Solo: *solo, Host: *host, Join: *join, Port: *port, Advertise: *advertise, NoMap: *noMap, SD: *size == "sd"}
		flag.Visit(func(f *flag.Flag) {
			if f.Name == "difficulty" {
				opts.Difficulty = bot.Difficulty(*difficulty)
			}
			if f.Name == "size" {
				opts.DisplayExplicit = true
			}
		})
		if dir, err := os.UserConfigDir(); err == nil {
			opts.SettingsPath = filepath.Join(dir, "sea-battle-2", "settings.json")
		}
		m, cleanup := play.New(opts)
		defer cleanup()
		_, err := tea.NewProgram(m, tea.WithColorProfile(colorprofile.TrueColor)).Run()
		return err
	}

	if *list {
		for i, s := range visual.Scenes {
			fmt.Printf("%02d %-18s %-20s %s\n", i+1, s.ID, s.Category, s.Title)
		}
		return nil
	}
	idx := visual.Find(*scene)
	if idx < 0 {
		return fmt.Errorf("unknown scene %q; use --list", *scene)
	}
	if *size != "hd" && *size != "sd" {
		return fmt.Errorf("size must be hd or sd")
	}
	if *weather < -1 || *weather > 3 {
		return fmt.Errorf("weather must be -1..3")
	}
	cam := strings.ToUpper(*camera)
	if cam != "" && cam != "TOP" && cam != "SIDE" && cam != "FRONT" {
		return fmt.Errorf("camera must be top, side or front")
	}
	if math.IsNaN(*at) || math.IsInf(*at, 0) || *at < 0 {
		return fmt.Errorf("at must be finite and nonnegative")
	}
	st := visual.DefaultState()
	st.Scene = idx
	st.SD = *size == "sd"
	st.T = math.Min(*at, visual.Scenes[idx].Duration)
	st.Weather = *weather
	st.Camera = cam
	st.Paused = *paused
	st.Tour = *tour
	if *dump != "" {
		f := visual.Render(st)
		switch *dump {
		case "json":
			return json.NewEncoder(os.Stdout).Encode(f)
		case "text":
			fmt.Println(f.Plain())
		case "ansi":
			fmt.Print(f.ANSI())
		default:
			return fmt.Errorf("dump must be json, ansi or text")
		}
		return nil
	}
	_, err := tea.NewProgram(model{state: st, preferSD: st.SD}, tea.WithColorProfile(colorprofile.TrueColor)).Run()
	return err
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
