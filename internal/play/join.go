package play

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sea-battle-2/internal/peer"
	"strings"
	"time"
)

type clipboardRead struct {
	serial  uint64
	content string
}
type clipboardLimit struct{ bytes.Buffer }

func (b *clipboardLimit) Write(p []byte) (int, error) {
	if b.Len()+len(p) > peer.MaxInvitation*2 {
		return 0, errors.New("clipboard too large")
	}
	return b.Buffer.Write(p)
}

// Read only on Join, bounded in time and size. SSH uses the terminal clipboard
// instead of reading a potentially unrelated remote desktop's clipboard.
func nativeClipboard(serial uint64) tea.Cmd {
	return func() tea.Msg {
		result := clipboardRead{serial: serial}
		if os.Getenv("SSH_CONNECTION") != "" {
			return result
		}
		var commands [][]string
		switch runtime.GOOS {
		case "darwin":
			commands = [][]string{{"pbpaste"}}
		case "windows":
			commands = [][]string{{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new(); Get-Clipboard -Raw"}}
		case "linux":
			if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSL_INTEROP") != "" {
				commands = [][]string{{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new(); Get-Clipboard -Raw"}}
			} else {
				if os.Getenv("WAYLAND_DISPLAY") != "" {
					commands = append(commands, []string{"wl-paste", "--no-newline"})
				}
				if os.Getenv("DISPLAY") != "" {
					commands = append(commands, []string{"xclip", "-selection", "clipboard", "-o"}, []string{"xsel", "--clipboard", "--output"})
				}
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		for _, args := range commands {
			path, err := exec.LookPath(args[0])
			if err != nil {
				continue
			}
			cmd := exec.CommandContext(ctx, path, args[1:]...)
			var output clipboardLimit
			cmd.Stdout = &output
			cmd.Stderr = io.Discard
			cmd.WaitDelay = 100 * time.Millisecond
			if cmd.Run() == nil {
				result.content = output.String()
				return result
			}
		}
		return result
	}
}

func (m *Model) openJoin(rejoin bool) tea.Cmd {
	m.mode = "join"
	if rejoin {
		m.mode = "rejoin"
	}
	m.input = ""
	m.notice = "Please paste an invitation."
	m.clipboardSerial++
	m.clipboardPending = true
	return tea.Batch(tea.ReadClipboard, nativeClipboard(m.clipboardSerial))
}

func (m *Model) acceptInvitation(content string, pasted bool) tea.Cmd {
	if m.mode != "join" && m.mode != "rejoin" {
		return nil
	}
	code := cleanInput(content)
	if !pasted && !strings.HasPrefix(code, "sb2:") {
		return nil
	}
	if _, err := peer.ParseInvitation(code); err != nil {
		if pasted {
			m.input = ""
			m.notice = "That invitation looks incomplete. Please paste the whole thing."
			if strings.Contains(err.Error(), "expired") {
				m.notice = "That invitation has expired. Ask your friend for a new one."
			}
		}
		return nil
	}
	m.clipboardPending = false
	m.input = code
	if m.mode == "rejoin" {
		if err := m.session.UpdateInvitation(m.session.Generation(), code); err != nil {
			m.input = ""
			m.notice = "Please paste an invitation from the same host session."
			return nil
		}
		m.mode = "game"
		m.notice = "Reconnecting…"
		return nil
	}
	m.mode = "connecting"
	m.joinAttempt = true
	m.notice = ""
	return m.startJoin(code)
}
