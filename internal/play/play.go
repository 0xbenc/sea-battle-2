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
	"sea-battle-2/internal/game"
	"sea-battle-2/internal/peer"
	"sea-battle-2/internal/session"
	"sea-battle-2/internal/visual"
)

type Options struct {
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
type Model struct {
	opts                                             Options
	session                                          *session.Controller
	running, host                                    bool
	port                                             int
	mode                                             string
	input, invite, notice, routerStatus              string
	width, height                                    int
	preferSD, offline, pending, invitation, quitting bool
	state                                            *game.View
	fleet                                            []game.Placement
	cursor                                           game.Coord
	selected                                         int
	horizontal                                       bool
	t, clipTime                                      float64
	last                                             time.Time
	clips                                            []game.Event
}

func New(opts Options) (Model, func()) {
	controller := session.New(context.Background())
	m := Model{opts: opts, session: controller, mode: "menu", preferSD: opts.SD, fleet: game.EmptyFleet()}
	if opts.Host || opts.Join != "" {
		m.mode = "connecting"
	}
	return m, controller.Close
}
func tick() tea.Cmd {
	return tea.Tick(time.Second/24, func(t time.Time) tea.Msg { return clockTick(t) })
}
func (m Model) Init() tea.Cmd {
	commands := []tea.Cmd{tick(), m.event()}
	if m.opts.Host {
		commands = append(commands, m.startHost())
	}
	if m.opts.Join != "" {
		commands = append(commands, m.startJoin(m.opts.Join))
	}
	return tea.Batch(commands...)
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
func (m Model) fits() bool { return m.width >= 80 && m.height >= 24 }
func (m Model) sd() bool   { return m.preferSD || m.width < 160 || m.height < 48 }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case clockTick:
		now := time.Time(v)
		if !m.last.IsZero() && m.fits() {
			dt := min(.12, now.Sub(m.last).Seconds())
			m.t += dt
			if len(m.clips) > 0 && !m.invitation {
				m.clipTime += dt
				duration := 3.5
				if m.clips[0].Sunk {
					duration = 6
				}
				if m.clipTime >= duration {
					m.clips = m.clips[1:]
					m.clipTime = 0
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
			m.clips = nil
			m.pending = false
			m.running = false
			m.host = false
			m.invitation = false
			m.invite = ""
			m.routerStatus = ""
			m.notice = ""
			m.mode = "connecting"
		}
		if v.State == session.Connecting && v.Message.Type == "" {
			m.running = true
			m.host = v.Host
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
			m.mode = "menu"
			m.running = false
		}
		return m, m.event()
	case peer.Message:
		switch v.Type {
		case "state":
			previous := m.state
			if previous != nil && !m.offline && previous.Match == v.View.Match {
				for _, e := range v.View.Events {
					if e.Seq > len(previous.Events) {
						m.clips = append(m.clips, e)
					}
				}
			}
			if previous == nil || previous.Match != v.View.Match {
				m.fleet = game.EmptyFleet()
				m.clips = nil
				m.clipTime = 0
				m.selected = 0
			}
			if m.offline || m.pending {
				m.notice = ""
			}
			m.state = v.View
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
		if m.quitting {
			if k == "y" || k == "enter" {
				return m, tea.Quit
			}
			if k == "n" || k == "esc" || k == "q" {
				m.quitting = false
			}
			return m, nil
		}
		if k == "q" {
			if !m.running {
				return m, tea.Quit
			}
			m.quitting = true
			return m, nil
		}
		if k == "esc" && (m.mode == "connecting" || (m.mode == "game" && m.state == nil)) {
			m.session.Stop()
			m.mode = "menu"
			m.notice = "Connection attempt cancelled."
			return m, nil
		}
		if k == "f2" {
			m.preferSD = !m.preferSD
			return m, nil
		}
		if !m.fits() {
			return m, nil
		}
		if m.mode == "menu" {
			switch k {
			case "h":
				m.mode = "connecting"
				return m, m.startHost()
			case "j":
				m.mode = "join"
				m.notice = ""
			}
			return m, nil
		}
		if k == "j" && m.offline && !m.host && m.running {
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
			if k == "space" || k == "enter" {
				m.clips = m.clips[1:]
				m.clipTime = 0
			}
			return m, nil
		}
		switch k {
		case "up", "w":
			m.cursor.Y = max(0, m.cursor.Y-1)
		case "down", "s":
			m.cursor.Y = min(9, m.cursor.Y+1)
		case "left", "a":
			m.cursor.X = max(0, m.cursor.X-1)
		case "right", "d":
			m.cursor.X = min(9, m.cursor.X+1)
		}
		p := m.state.Player
		if m.state.Phase == "placement" && !m.state.Ready[p] && !m.pending {
			switch k {
			case "1", "2", "3", "4", "5":
				m.selected = int(k[0] - '1')
			case "x":
				m.horizontal = !m.horizontal
			case "r":
				m.fleet = game.RandomFleet(rand.Uint64())
				m.notice = "Fleet arranged. SPACE locks it in."
			case "backspace", "delete":
				m.fleet[m.selected] = game.EmptyFleet()[0]
			case "enter":
				p := game.Placement{Coord: m.cursor, Horizontal: m.horizontal}
				if game.CanPlace(m.fleet, m.selected, p) {
					m.fleet[m.selected] = p
					m.selected = (m.selected + 1) % 5
					m.notice = "Ship placed."
				} else {
					m.notice = "Hull overlaps another ship or crosses the chart edge."
				}
			case "space":
				if err := game.ValidateFleet(m.fleet); err != nil {
					m.notice = err.Error()
				} else if !m.offline {
					m.pending = true
					return m, m.send(peer.Command{Type: "ready", Fleet: append([]game.Placement(nil), m.fleet...)})
				}
			}
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
		} else if m.state.Phase == "finished" && k == "r" && !m.pending && !m.offline {
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
		if m.host {
			detail = "Closing the host ends this session for both players."
		}
		f = visual.RenderFront(m.sd(), m.t, "LEAVE COMMAND?", []string{detail, "", "Y / ENTER leave    N / ESC return"})
	case m.mode == "menu":
		f = visual.RenderFront(m.sd(), m.t, "SEA BATTLE 2 / OPERATION PEST TIDE", []string{"", "H  Host a battle / American navy", "J  Join by invitation / The brood", "", "One commander hosts. The other pastes their invitation.", "No accounts. No shared servers.", "", "Lookdev remains available with --lookdev.", "", "Q quit    F2 SD / HD", "", m.notice})
	case m.mode == "join" || m.mode == "rejoin":
		tail := m.input
		if len(tail) > 52 {
			tail = "..." + tail[len(tail)-49:]
		}
		f = visual.RenderFront(m.sd(), m.t, "JOIN THE BROOD", []string{"Paste the complete sb2: invitation.", "Wrapped invitations are accepted.", "", tail, "", fmt.Sprintf("%d characters", len(m.input)), "", "ENTER connect    BACKSPACE edit    ESC back", m.notice})
	case m.invitation && m.host:
		f = visual.RenderFront(m.sd(), m.t, "HOST INVITATION / KEEP PRIVATE", []string{
			"C copy invitation    S save invitation.txt    ENTER deploy", "", m.routerStatus, "",
			fmt.Sprintf("Private invitation: %d characters", len(m.invite)), m.invite[:min(50, len(m.invite))] + "...", "Use C or S for the COMPLETE invitation.", "", "TCP " + strconv.Itoa(m.port) + " / host must remain running",
			"Manual: --host --advertise PUBLIC_IP:PORT (forward TCP).",
			"If upstream NAT blocks both hosts, use reachable IPv6 or LAN.", "", m.notice})
	case m.state == nil:
		f = visual.RenderFront(m.sd(), m.t, "ESTABLISHING COMMAND", []string{"Connecting directly to the host...", "", "No relay fallback is used.", "If this persists: check the host is still running, allow its TCP", "port through the firewall, or try the other player hosting.", "For upstream NAT/CGNAT: use reachable IPv6 or the same LAN.", "", "ESC back    Q quit", m.notice})
	default:
		v := *m.state
		if m.offline {
			v.Connected = [2]bool{}
		}
		b := visual.BattlePresentation{State: v, Fleet: m.fleet, Cursor: m.cursor, Selected: m.selected, Horizontal: m.horizontal, SD: m.sd(), T: m.t, Notice: m.notice, Pending: m.pending}
		if len(m.clips) > 0 {
			b.Clip = &m.clips[0]
			b.ClipTime = m.clipTime
		}
		f = visual.RenderBattle(b)
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
	v.WindowTitle = "Sea Battle 2 / Live command"
	return v
}
