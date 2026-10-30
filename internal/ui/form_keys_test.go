package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
)

// cmdTimeout bounds one command run by runCmd. A cursor blink waits the
// cursor's blink speed, which is about half a second, so the bound is well
// above that.
const cmdTimeout = 2 * time.Second

// quitsApp reports whether cmd, or any command in its batch, ends the program.
// An interrupt ends it too: the program returns when it gets one.
func quitsApp(t *testing.T, cmd tea.Cmd) bool {
	t.Helper()
	if cmd == nil {
		return false
	}
	msg, ok := runCmd(cmd)
	if !ok {
		t.Errorf("command did not return within %v, so it could not be checked for quitting", cmdTimeout)
		return false
	}
	switch msg := msg.(type) {
	case tea.QuitMsg, tea.InterruptMsg:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if quitsApp(t, c) {
				return true
			}
		}
	}
	return false
}

// runCmd runs cmd and returns the message it produces. ok is false when cmd has
// not returned after cmdTimeout. The goroutine running it is left behind.
func runCmd(cmd tea.Cmd) (msg tea.Msg, ok bool) {
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		return msg, true
	case <-time.After(cmdTimeout):
		return nil, false
	}
}

// press sends msg to the model and settles the commands it returns. It reports
// whether the app quit.
func press(t *testing.T, m *MainModel, msg tea.Msg) bool {
	t.Helper()
	_, cmd := m.Update(msg)
	return settle(t, m, cmd)
}

// settle runs cmd and feeds the messages it produces back into m, the way the
// Bubble Tea loop does. huh moves between fields this way. It reports whether
// the app quit.
//
// Cursor blink messages are dropped instead of fed back. Fed back, each one
// starts the next blink, so settling would never end. A command that does not
// return in time is reported and skipped.
func settle(t *testing.T, m *MainModel, cmd tea.Cmd) bool {
	t.Helper()
	if cmd == nil {
		return false
	}
	msg, ok := runCmd(cmd)
	if !ok {
		t.Errorf("command did not return within %v and was skipped", cmdTimeout)
		return false
	}
	switch msg := msg.(type) {
	case nil, cursor.BlinkMsg:
		return false
	case tea.QuitMsg, tea.InterruptMsg:
		return true
	case tea.BatchMsg:
		quit := false
		for _, c := range msg {
			quit = settle(t, m, c) || quit
		}
		return quit
	}
	_, next := m.Update(msg)
	return settle(t, m, next)
}

// focusedField returns the key of the field that has focus in the form.
func focusedField(t *testing.T, m *MainModel) string {
	t.Helper()
	if m.form == nil {
		t.Fatal("form is nil")
	}
	return m.form.GetFocusedField().GetKey()
}

// drillFiles counts the drill files saved under the test's config directory.
func drillFiles(t *testing.T) int {
	t.Helper()
	dir, err := drill.GetUserDrillsDir()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

// assertCancelled checks that the form was closed without saving and that the
// model is back on the menu.
// cmd is the command the cancelling key returned.
func assertCancelled(t *testing.T, m *MainModel, cmd tea.Cmd, before int) {
	t.Helper()
	if m.state != StateMenu {
		t.Errorf("state = %v after cancel, want menu", m.state)
	}
	if m.form != nil {
		t.Errorf("form is still set after cancel")
	}
	if quitsApp(t, cmd) {
		t.Errorf("cancel returned a command that quits the app")
	}
	if got := drillFiles(t); got != before {
		t.Errorf("drill files = %d after cancel, want %d (nothing saved)", got, before)
	}
	if m.menuNotice != "Cancelled." {
		t.Errorf("menu notice = %q, want %q", m.menuNotice, "Cancelled.")
	}
}

func TestFormEscCancelsOnFirstField(t *testing.T) {
	m := openAddDrillForm(t)
	before := drillFiles(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assertCancelled(t, m, cmd, before)
}

func TestFormEscCancelsAfterTyping(t *testing.T) {
	m := openAddDrillForm(t)
	before := drillFiles(t)
	press(t, m, runeKey("Abandoned"))
	press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.form == nil {
		t.Fatalf("expected form to still be open after moving to the next field")
	}
	if got := focusedField(t, m); got != "category" {
		t.Fatalf("focused field %q after enter, want %q", got, "category")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assertCancelled(t, m, cmd, before)
}

func TestFormEscCancelsOnLaterField(t *testing.T) {
	m := openAddDrillForm(t)
	before := drillFiles(t)
	for range 3 {
		press(t, m, tea.KeyMsg{Type: tea.KeyTab})
	}
	if got := focusedField(t, m); got != "language" {
		t.Fatalf("focused field %q after three tabs, want %q", got, "language")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assertCancelled(t, m, cmd, before)
}

func TestFormCtrlCStillCancels(t *testing.T) {
	m := openAddDrillForm(t)
	before := drillFiles(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assertCancelled(t, m, cmd, before)
}

func TestFormReopensFreshAfterCancel(t *testing.T) {
	m := openAddDrillForm(t)
	m.Update(runeKey("Abandoned"))
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	m.list.Select(len(m.list.Items()) - 1)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateAddDrill || m.form == nil {
		t.Fatalf("enter on add item: state %v, want add drill form", m.state)
	}
	if got := m.form.GetString("title"); got != "" {
		t.Errorf("title = %q on reopen, want empty", got)
	}
	if strings.Contains(ansi.Strip(m.View()), "Abandoned") {
		t.Errorf("reopened form still shows the cancelled title")
	}
}

func TestFormShowsCancelKey(t *testing.T) {
	m := openAddDrillForm(t)
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "[Esc] cancel") {
		t.Errorf("add drill form does not show the cancel key:\n%s", view)
	}
}

func TestFormCancelChoiceOnConfirmCancels(t *testing.T) {
	m := openAddDrillForm(t)
	before := drillFiles(t)
	press(t, m, runeKey("Abandoned"))
	// Tab moves to the next field. Eight moves reach the final confirm from
	// the title, since the new category group is hidden for "Custom".
	for range 8 {
		press(t, m, tea.KeyMsg{Type: tea.KeyTab})
	}
	if m.form == nil || m.form.State != huh.StateNormal {
		t.Fatalf("form closed before the final confirm")
	}
	if !strings.Contains(ansi.Strip(m.View()), "Save and create this drill?") {
		t.Fatalf("form is not on the final confirm:\n%s", ansi.Strip(m.View()))
	}
	_, cmd := m.Update(runeKey("n"))
	if settle(t, m, cmd) {
		t.Errorf("cancel choice quit the app")
	}
	assertCancelled(t, m, cmd, before)
}
