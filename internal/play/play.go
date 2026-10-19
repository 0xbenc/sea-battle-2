// Package play binds the terminal controls to the authoritative direct session.
package play

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"sea-battle-2/internal/bot"
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/session"
	"sea-battle-2/internal/visual"
)

type Options struct {
	SettingsPath    string
	DisplayExplicit bool
	Difficulty      bot.Difficulty
	Solo            string
	Host            bool
	Join, Advertise string
	Port            int
	NoMap, SD       bool
}
type clockTick time.Time
type sent struct {
	generation uint64
	err        error
}
type saved struct{ err error }

// A transmission and its board result are one indivisible playback item.
type shotPlayback struct {
	game.Event
	After game.View
}
type factionDeploy struct {
	key    string
	serial uint64
}

type Model struct {
	inspectOwn, combatHelp                           bool
	resultCharts                                     bool
	resultStart                                      float64
	ownCursor                                        game.Coord
	placementUndo                                    []placementSnapshot
	placementHelp                                    bool
	factionPressed                                   string
	factionSerial                                    uint64
	menuReaction                                     visual.MenuReaction
	reactionStart                                    float64
	waterReadyAt                                     time.Time
	menuHover                                        string
	menuTime                                         float64
	stillMenu                                        bool
	menu                                             *visual.MenuDirector
	opts                                             Options
	session                                          *session.Controller
	running, host, solo, returnMenu                  bool
	port                                             int
	mode                                             string
	input, invite, notice, routerStatus              string
	width, height                                    int
	preferSD, offline, pending, invitation, quitting bool
	state                                            *game.View // newest authority; commands only
	visible                                          *game.View // state reached by playback
	fleet                                            []game.Placement
	cursor                                           game.Coord
	selected                                         int
	horizontal                                       bool
	t, clipTime                                      float64
	last                                             time.Time
	clips                                            []shotPlayback
}

func New(opts Options) (Model, func()) {
	prefs := readPreferences(opts.SettingsPath)
	if opts.Difficulty == "" {
		opts.Difficulty = prefs.Difficulty.Default()
	}
	if !opts.DisplayExplicit && !opts.SD {
		opts.SD = prefs.Compact
	}
	controller := session.New(context.Background())
	m := Model{menu: visual.NewMenuDirector(rand.Uint64()), opts: opts, session: controller, mode: "menu", stillMenu: prefs.StillMenu, preferSD: opts.SD, fleet: game.EmptyFleet()}
	if opts.Host || opts.Join != "" || opts.Solo != "" {
		m.mode = "connecting"
	}
	return m, controller.Close
}
func tick() tea.Cmd {
	return tea.Tick(time.Second/24, func(t time.Time) tea.Msg { return clockTick(t) })
}
func (m Model) Init() tea.Cmd {
	commands := []tea.Cmd{tick(), m.event()}
	if m.opts.Solo != "" {
		human := 0
		if m.opts.Solo == "alien" {
			human = 1
		}
		commands = append(commands, m.startSolo(human))
	}
	if m.opts.Host {
		commands = append(commands, m.startHost())
	}
	if m.opts.Join != "" {
		commands = append(commands, m.startJoin(m.opts.Join))
	}
	return tea.Batch(commands...)
}
func (m Model) startSolo(human int) tea.Cmd {
	m.session.Start(session.Config{Solo: true, Human: human, Difficulty: m.opts.Difficulty})
	return nil
}
func (m Model) startHost() tea.Cmd {
	m.session.Start(session.Config{Host: true, Port: m.opts.Port, Advertise: m.opts.Advertise, NoMap: m.opts.NoMap})
	return nil
}
func (m Model) startJoin(code string) tea.Cmd {
	m.session.Start(session.Config{Join: code})
	return nil
}
func (m Model) event() tea.Cmd {
	return func() tea.Msg {
		e, ok := m.session.Next()
		if !ok {
			return nil
		}
		return e
	}
}
func (m Model) send(command peer.Command) tea.Cmd {
	command.Match = m.state.Match
	command.Expected = len(m.state.Events)
	generation := m.session.Generation()
	execute := m.session.Prepare(generation, command)
	return func() tea.Msg { return sent{generation, execute()} }
}

func cloneView(v game.View) game.View {
	v.Own = append([]game.Placement(nil), v.Own...)
	v.Events = append([]game.Event(nil), v.Events...)
	return v
}
func (m *Model) finishClip() {
	snapshot := m.clips[0].After
	m.visible = &snapshot
	m.clips = m.clips[1:]
	m.clipTime = 0
	// Non-shot updates (readiness/rematch votes) may arrive during a clip.
	if len(m.clips) == 0 && m.state != nil {
		snapshot := cloneView(*m.state)
		m.visible = &snapshot
		if snapshot.Phase == "finished" {
			m.resultStart = m.t
			m.resultCharts = false
		}
	}
}

const handoffDuration = 2.0

func (m Model) handoff() bool {
	return m.solo && m.state != nil && len(m.clips) > 0 && m.clips[0].Player != m.state.Player
}
func (m Model) clipDuration() float64 {
	duration := 3.5
	if !m.clips[0].Hit {
		duration = 2
	}
	if m.clips[0].Sunk {
		duration = 6
	}
	if m.handoff() {
		duration += handoffDuration
	}
	return duration
}
func (m Model) fits() bool { return m.width >= 80 && m.height >= 24 }
func (m Model) sd() bool   { return m.preferSD || m.width < 160 || m.height < 48 }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case factionDeploy:
		if m.mode == "faction" && v.serial == m.factionSerial && m.factionPressed == v.key {
			return m.Update(tea.KeyPressMsg(tea.Key{Code: []rune(v.key)[0]}))
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.menuHover = ""
		m.width, m.height = v.Width, v.Height
	case tea.MouseMsg:
		if (m.mode != "menu" && m.mode != "faction" && !m.placementScreen() && !m.combatScreen()) || m.quitting || !m.fits() {
			return m, nil
		}
		mouse := v.Mouse()
		w, h := 160, 48
		if m.sd() {
			w, h = 80, 24
		}
		x, y := mouse.X-(m.width-w)/2, mouse.Y-(m.height-h)/2
		if m.placementScreen() {
			return m.placementMouse(v, x, y)
		}
		if m.combatScreen() {
			return m.combatMouse(v, x, y)
		}
		m.menuHover = visual.MenuControlAt(m.sd(), x, y)
		if m.mode == "faction" {
			m.menuHover = visual.FactionControlAt(m.sd(), x, y)
		}
		if _, clicked := v.(tea.MouseClickMsg); clicked && mouse.Button == tea.MouseLeft {
			if m.menuHover != "" {
				if m.mode == "faction" && (m.menuHover == "a" || m.menuHover == "b") {
					m.factionSerial++
					m.factionPressed = m.menuHover
					message := factionDeploy{m.menuHover, m.factionSerial}
					return m, tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return message })
				}
				code := []rune(m.menuHover)[0]
				if m.menuHover == "esc" {
					code = tea.KeyEscape
				}
				return m.Update(tea.KeyPressMsg(tea.Key{Code: code}))
			}
			if m.mode == "faction" {
				return m, nil
			}
			fx := visual.MenuReactionAt(m.sd(), m.menu.Snapshot(), x, y)
			if fx.Kind != "" {
				now := time.Now()
				if fx.Kind == "ripple" {
					if now.Before(m.waterReadyAt) {
						return m, nil
					}
					m.waterReadyAt = now.Add(2 * time.Second)
				}
				m.menuReaction, m.reactionStart = fx, m.t
			}
		}
		return m, nil
	case clockTick:
		now := time.Time(v)
		if !m.last.IsZero() && m.fits() {
			dt := min(.12, now.Sub(m.last).Seconds())
			m.t += dt
			if !m.stillMenu && (m.mode != "game" || m.invitation || m.quitting) {
				m.menuTime += dt
				m.menu.Advance(dt)
			}
			if len(m.clips) > 0 && !m.invitation {
				m.clipTime += dt
				duration := m.clipDuration()
				if m.clipTime >= duration {
					m.finishClip()
				}
			}
		}
		m.last = now
		return m, tick()
	case session.Event:
		if v.Generation != m.session.Generation() {
			return m, m.event()
		}
		if v.State == session.Starting {
			m.state = nil
			m.visible = nil
			m.clips = nil
			m.pending = false
			m.running = false
			m.host = false
			m.solo = false
			m.invitation = false
			m.invite = ""
			m.routerStatus = ""
			m.notice = ""
			m.mode = "connecting"
		}
		if v.State == session.Connecting && v.Message.Type == "" {
			m.running = true
			m.host = v.Host
			m.solo = v.Solo
			m.port = v.Port
			m.mode = "game"
			m.offline = true
			m.invitation = v.Host
		}
		if v.Invitation != "" {
			if m.invite != "" && m.invite != v.Invitation {
				m.notice = "Reconnect routes updated. I then C copies the current invitation."
			}
			m.invite = v.Invitation
		}
		if v.Router != nil {
			m.routerStatus = v.Router.Notice
			if v.Router.State == "active" {
				m.routerStatus = v.Router.Protocol + " mapping ready. Internet reachability is not verified."
			}
		}
		if v.Message.Type != "" {
			next, _ := m.Update(v.Message)
			m = next.(Model)
		}
		if v.State == session.Failed {
			m.offline = true
			m.pending = false
			m.notice = v.Error
			if m.state == nil {
				m.mode = "menu"
				m.running = false
			}
		}
		if v.State == session.Stopped {
			if m.mode != "faction" {
				m.mode = "menu"
			}
			m.running = false
		}
		return m, m.event()
	case peer.Message:
		switch v.Type {
		case "state":
			previous := m.state
			if v.View.Phase == "finished" && (previous == nil || previous.Phase != "finished") {
				m.resultStart = m.t
				m.resultCharts = false
			}
			// Each live state is an immutable post-shot snapshot. Reconnect/catch-up
			// replaces presentation atomically rather than inventing intermediate boards.
			delta := 0
			if previous != nil {
				delta = len(v.View.Events) - len(previous.Events)
			}
			live := previous != nil && !m.offline && previous.Match == v.View.Match
			if live && delta == 1 {
				if m.visible == nil {
					old := cloneView(*previous)
					m.visible = &old
				}
				e := v.View.Events[len(v.View.Events)-1]
				m.clips = append(m.clips, shotPlayback{Event: e, After: cloneView(*v.View)})
				m.combatHelp = false
			} else if !live || delta != 0 {
				m.clips = nil
				m.clipTime = 0
				snapshot := cloneView(*v.View)
				m.visible = &snapshot
			} else if len(m.clips) == 0 {
				snapshot := cloneView(*v.View)
				m.visible = &snapshot
			}

			if previous == nil || previous.Match != v.View.Match {
				m.fleet = game.EmptyFleet()
				m.clips = nil
				m.clipTime = 0
				m.selected = 0
				m.cursor = game.Coord{}
				m.horizontal = false
				m.placementUndo = nil
				m.placementHelp = false
				m.inspectOwn, m.combatHelp = false, false
				m.resultCharts = false
				m.resultStart = m.t
				m.ownCursor = game.Coord{}
			}
			if previous != nil && previous.Phase == "placement" && v.View.Phase == "battle" {
				m.cursor = game.Coord{}
				m.selected = -1
				m.placementUndo = nil
				m.placementHelp = false
				m.notice = ""
			}
			if m.offline || m.pending {
				m.notice = ""
			}
			snapshot := cloneView(*v.View)
			m.state = &snapshot
			m.offline = false
			m.pending = false
			guestArrived := m.state.Connected[1] && (previous == nil || !previous.Connected[1])
			if m.state.Phase == "battle" || guestArrived {
				m.invitation = false
			}
		case "error":
			m.notice = v.Error
			m.pending = false
		case "offline":
			m.offline = true
			m.pending = false
			m.notice = "Link interrupted. Retrying automatically; J updates guest routes."
		case "connecting":
			m.offline = true
		case "fatal":
			m.offline = true
			m.notice = v.Error
			return m, nil
		}
		return m, nil
	case sent:
		if v.generation != m.session.Generation() {
			return m, nil
		}
		if v.err != nil {
			m.pending = false
			m.notice = v.err.Error()
		}
	case saved:
		if v.err != nil {
			m.notice = v.err.Error()
		} else {
			m.notice = "Invitation saved to invitation.txt (private file)."
		}
	case tea.PasteMsg:
		if m.mode == "join" || m.mode == "rejoin" {
			m.input = cleanInput(v.Content)
		}
	case tea.KeyPressMsg:
		m.factionPressed = ""
		m.factionSerial++
		m.menuHover = ""
		k := v.String()
		if k == "ctrl+c" {
			return m, tea.Quit
		}
		if m.mode == "join" || m.mode == "rejoin" {
			switch k {
			case "esc":
				if m.mode == "rejoin" {
					m.mode = "game"
				} else {
					m.mode = "menu"
				}
			case "enter":
				if m.mode == "rejoin" {
					if err := m.session.UpdateInvitation(m.session.Generation(), m.input); err != nil {
						m.notice = err.Error()
					} else {
						m.mode = "game"
						m.notice = "Reconnect routes updated; retrying the same session."
					}
					return m, nil
				}
				m.mode = "connecting"
				return m, m.startJoin(m.input)
			case "backspace":
				if len(m.input) > 0 {
					m.input = m.input[:len(m.input)-1]
				}
			default:
				if v.Key().Text != "" {
					m.input = cleanInput(m.input + v.Key().Text)
				}
			}
			return m, nil
		}
		if m.combatScreen() && m.combatHelp {
			if k == "esc" || k == "?" {
				m.combatHelp = false
			}
			if k != "q" {
				return m, nil
			}
		}
		if m.combatScreen() && k == "esc" && !m.solo {
			if m.state.Phase == "finished" {
				m.session.Stop()
				m.mode = "menu"
				m.running = false
				return m, nil
			}
			m.quitting, m.returnMenu = true, true
			return m, nil
		}
		if m.placementScreen() && m.placementHelp {
			if k == "esc" || k == "?" {
				m.placementHelp = false
			}
			if k != "q" {
				return m, nil
			}
		}
		if m.placementScreen() && k == "?" {
			m.placementHelp = true
			return m, nil
		}
		if m.placementEditable() && k == "esc" && m.placementDirty() {
			m.selectPlacementShip(m.selected)
			m.selected = -1
			return m, nil
		}
		if m.placementScreen() && k == "esc" && !m.solo {
			m.quitting, m.returnMenu = true, true
			return m, nil
		}
		if m.quitting {
			if k == "y" || k == "enter" {
				if m.returnMenu {
					m.session.Stop()
					m.quitting = false
					m.returnMenu = false
					m.mode = "menu"
					m.running = false
					return m, nil
				}
				return m, tea.Quit
			}
			if k == "n" || k == "esc" || k == "q" {
				m.quitting = false
				m.returnMenu = false
			}
			return m, nil
		}
		if k == "q" {
			if !m.running {
				return m, tea.Quit
			}
			m.quitting = true
			m.returnMenu = false
			return m, nil
		}
		if k == "esc" && (m.mode == "connecting" || (m.mode == "game" && m.state == nil)) {
			m.session.Stop()
			m.mode = "menu"
			m.notice = "Connection attempt cancelled."
			return m, nil
		}
		if k == "esc" && m.mode == "faction" {
			m.mode = "menu"
			return m, nil
		}
		if k == "esc" && m.solo && m.mode == "game" {
			if m.visible != nil && m.visible.Phase == "finished" {
				m.session.Stop()
				m.mode = "menu"
				m.running = false
				return m, nil
			}
			m.quitting = true
			m.returnMenu = true
			return m, nil
		}
		if !m.fits() {
			return m, nil
		}
		if m.mode == "help" || m.mode == "settings" {
			if k == "esc" || (m.mode == "help" && k == "?") {
				m.mode, m.notice = "menu", ""
				return m, nil
			}
			if m.mode == "settings" {
				switch k {
				case "d":
					m.preferSD = !m.preferSD
				case "m":
					m.stillMenu = !m.stillMenu
				case "1":
					m.opts.Difficulty = bot.Easy
				case "2":
					m.opts.Difficulty = bot.Medium
				case "3":
					m.opts.Difficulty = bot.Hard
				default:
					return m, nil
				}
				m.savePreferences()
			}
			return m, nil
		}
		if m.mode == "menu" {
			switch k {
			case "?":
				m.mode, m.notice = "help", ""
			case "t":
				m.mode, m.notice = "settings", ""
			case "s":
				m.mode = "faction"
				m.notice = ""
			case "h":
				m.mode = "connecting"
				return m, m.startHost()
			case "j":
				m.mode = "join"
				m.notice = ""
			}
			return m, nil
		}
		if m.mode == "faction" {
			switch k {
			case "1":
				m.opts.Difficulty = bot.Easy
			case "2":
				m.opts.Difficulty = bot.Medium
			case "3":
				m.opts.Difficulty = bot.Hard
			}
			if k == "a" || k == "b" {
				human := 0
				if k == "b" {
					human = 1
				}
				m.mode = "connecting"
				return m, m.startSolo(human)
			}
			return m, nil
		}
		if k == "j" && !m.solo && m.offline && !m.host && m.running {
			m.mode = "rejoin"
			m.input = ""
			m.notice = "Paste an updated invitation from the SAME host session."
			return m, nil
		}
		if k == "i" && m.host {
			m.invitation = !m.invitation
			return m, nil
		}
		if m.invitation {
			switch k {
			case "esc", "enter":
				m.invitation = false
			case "c":
				m.notice = "Clipboard requested (OSC52). S saves a file if unsupported."
				return m, tea.SetClipboard(m.invite)
			case "s":
				code := m.invite
				return m, func() tea.Msg { return saved{saveInvitation(code)} }
			}
			return m, nil
		}
		if m.state == nil {
			return m, nil
		}
		if len(m.clips) > 0 {
			if (k == "space" || k == "enter") && !(m.handoff() && m.clipTime < handoffDuration) {
				m.finishClip()
			}
			return m, nil
		}
		if m.combatScreen() {
			if k == "?" {
				m.combatHelp = true
				return m, nil
			}
			if k == "tab" {
				if m.visible != nil && m.visible.Phase == "finished" {
					m.resultCharts = !m.resultCharts
					m.notice = ""
					return m, nil
				}
				m.inspectOwn = !m.inspectOwn
				m.notice = ""
				return m, nil
			}
			if m.inspectOwn {
				if k == "enter" {
					m.inspectOwn = false
					return m, nil
				}
				next := moveAim(m.ownCursor, k)
				if next != m.ownCursor {
					m.ownCursor = next
					m.notice = ""
					return m, nil
				}
			} else if !m.pending {
				next := moveAim(m.cursor, k)
				if next != m.cursor {
					m.cursor = next
					m.notice = ""
				}
			}
		} else {
			m.cursor = moveAim(m.cursor, k)
		}

		p := m.state.Player
		if m.state.Phase == "placement" && !m.state.Ready[p] && !m.pending {
			if k == "up" || k == "down" || k == "left" || k == "right" || k == "w" || k == "a" || k == "s" || k == "d" {
				m.notice = ""
			}
			cmd := m.placementKey(k)
			return m, cmd
		} else if m.state.Phase == "battle" && k == "enter" && !m.pending && !m.offline {
			if !m.state.Connected[0] || !m.state.Connected[1] {
				m.notice = "Both commanders must be connected."
			} else if m.state.Turn != p {
				m.notice = "Wait for your turn."
			} else if m.state.Outgoing[m.cursor.Index()] != game.Unknown {
				m.notice = "That square has already been fired on."
			} else {
				m.pending = true
				return m, m.send(peer.Command{Type: "fire", At: m.cursor})
			}
		} else if m.state.Phase == "finished" && k == "r" && !m.pending && !m.offline && !m.state.Rematch[p] {
			m.pending = true
			return m, m.send(peer.Command{Type: "rematch"})
		}
	}
	return m, nil
}
func cleanInput(s string) string {
	s = strings.Map(func(r rune) rune {
		if r > 127 || unicode.IsSpace(r) || unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	if len(s) > peer.MaxInvitation {
		s = s[:peer.MaxInvitation]
	}
	return s
}
func saveInvitation(code string) error {
	// O_EXCL avoids following a pre-existing symlink or overwriting someone else's file.
	f, err := os.OpenFile("invitation.txt", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("Cannot save: %w (remove the old invitation.txt first)", err)
	}
	_, err = f.WriteString(code + "\n")
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func (m Model) View() tea.View {
	var f *visual.Frame
	switch {
	case m.quitting:
		detail := "Disconnect from the battle?"
		if m.solo {
			detail = "Leave this battle? Progress will be lost."
		}
		if m.host {
			detail = "Closing the host ends this session for both players."
		}
		f = m.front("LEAVE COMMAND?", []string{detail, "", "Y / ENTER leave    N / ESC return"})
	case m.mode == "menu":
		reaction := m.menuReaction
		reaction.Age = m.t - m.reactionStart
		f = visual.RenderMenuWithHover(m.sd(), m.menuTime, m.notice, m.menuHover, m.menu.Snapshot(), reaction)
	case m.mode == "help":
		f = m.front("FIELD MANUAL", helpLines)
	case m.mode == "settings":
		f = m.front("SETTINGS", m.settingsLines())
	case m.mode == "faction":
		f = visual.RenderFactionMenu(m.sd(), m.menuTime, string(m.opts.Difficulty), m.menuHover, m.factionPressed, m.menu.Snapshot())
	case m.mode == "join" || m.mode == "rejoin":
		tail := m.input
		if len(tail) > 52 {
			tail = "..." + tail[len(tail)-49:]
		}
		f = m.front("JOIN THE BROOD", []string{"Paste the complete sb2: invitation.", "Wrapped invitations are accepted.", "", tail, "", fmt.Sprintf("%d characters", len(m.input)), "", "ENTER connect    BACKSPACE edit    ESC back", m.notice})
	case m.invitation && m.host:
		f = m.front("HOST INVITATION / KEEP PRIVATE", []string{
			"C copy invitation    S save invitation.txt    ENTER deploy", "", m.routerStatus, "",
			fmt.Sprintf("Private invitation: %d characters", len(m.invite)), m.invite[:min(50, len(m.invite))] + "...", "Use C or S for the COMPLETE invitation.", "", "TCP " + strconv.Itoa(m.port) + " / host must remain running",
			"Manual: --host --advertise PUBLIC_IP:PORT (forward TCP).",
			"If upstream NAT blocks both hosts, use reachable IPv6 or LAN.", "", m.notice})
	case m.state == nil:
		f = m.front("ESTABLISHING COMMAND", []string{"Connecting directly to the host...", "", "No relay fallback is used.", "If this persists: check the host is still running, allow its TCP", "port through the firewall, or try the other player hosting.", "For upstream NAT/CGNAT: use reachable IPv6 or the same LAN.", "", "ESC back    Q quit", m.notice})
	default:
		f = visual.RenderBattle(m.battlePresentation())
	}
	content := ""
	if !m.fits() {
		content = fmt.Sprintf("SEA BATTLE 2\nNeed 80 x 24. Current: %d x %d.\nEnlarge the terminal. Ctrl+C quits.", m.width, m.height)
	} else {
		lines := strings.Split(f.ANSI(), "\n")
		pad := strings.Repeat(" ", max(0, (m.width-f.W)/2))
		for i := range lines {
			lines[i] = pad + lines[i]
		}
		content = strings.Repeat("\n", max(0, (m.height-f.H)/2)) + strings.Join(lines, "\n")
	}
	v := tea.NewView(content)
	v.AltScreen = true
	if (m.mode == "menu" || m.mode == "faction" || m.placementScreen() || m.combatScreen()) && !m.quitting && m.fits() {
		v.MouseMode = tea.MouseModeAllMotion
	}
	v.WindowTitle = "Sea Battle 2 / Live command"
	return v
}

func (m Model) front(title string, lines []string) *visual.Frame {
	return visual.RenderFront(m.sd(), m.menuTime, title, lines, m.menu.Snapshot())
}
