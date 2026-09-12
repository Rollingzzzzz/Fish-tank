// G5.3: Log tab — the shared tank log view (latest 200 lines, newest at the
// bottom, C3 cap). Fed by contract.EventLog pushes via Menu.PushEvent and by
// in-panel notices; the game can append through its own event stream (G6.1).
package ui

import "github.com/hajimehoshi/ebiten/v2"

// logCap is the tank-log limit from README C3.
const logCap = 200

type logTab struct {
	st *ScrollText
}

func newLogTab() *logTab { return &logTab{st: NewScrollText(logCap)} }

// append adds one log line (timestamping stays with the sender: Event.At).
func (l *logTab) append(text string) {
	if text == "" {
		return
	}
	l.st.AppendLine(KindLog, text)
}

func (l *logTab) wheel(dy int) { l.st.ScrollWheel(dy * 3) }

func (l *logTab) update(area Rect, mx, my int, pressed, released bool) {}

func (l *logTab) draw(dst *ebiten.Image, m *Menu, area Rect) {
	DrawText(dst, "TANK LOG", area.X+2, area.Y-14, 1, ColDim, 0.8)
	l.st.Draw(dst, area.X, area.Y, area.W, area.H-12, 2)
	DrawText(dst, "newest at the bottom", area.X+2, area.Y+area.H-10, 1, ColDim, 0.6)
}
