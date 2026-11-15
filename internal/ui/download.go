package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/download"
)

type audioDownload struct {
	mu                sync.Mutex
	name              string
	progress          download.Progress
	completed, tracks int
	frame             int
	cancel            context.CancelFunc
}

type downloadTickMsg struct{ job *audioDownload }
type downloadFinishedMsg struct {
	job  *audioDownload
	err  error
	next tea.Cmd
}

func downloadTick(job *audioDownload) tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg { return downloadTickMsg{job} })
}

func (m *MainModel) startDownload(name string, tracks int, run func(context.Context, download.Report, func()) error, next tea.Cmd) tea.Cmd {
	if m.download != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	job := &audioDownload{name: name, tracks: tracks, cancel: cancel}
	m.download = job
	m.downloadNotice = ""
	m.typingNotice = ""
	m.musicNotice = ""
	worker := func() tea.Msg {
		defer cancel()
		err := run(ctx, func(p download.Progress) {
			job.mu.Lock()
			job.progress = p
			job.mu.Unlock()
		}, func() {
			job.mu.Lock()
			job.completed++
			job.mu.Unlock()
		})
		return downloadFinishedMsg{job: job, err: err, next: next}
	}
	return tea.Batch(worker, downloadTick(job))
}

func byteSize(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1<<20 {
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
}

func (m *MainModel) downloadStatus(w int) string {
	job := m.download
	if job == nil {
		return truncateBlock("  "+m.downloadNotice, w)
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	p := job.progress
	label := byteSize(p.Bytes)
	width := max(4, min(24, w-6))
	var bar string
	if p.Total > 0 {
		percent := min(100, int(p.Bytes*100/p.Total))
		filled := width * percent / 100
		bar = "[" + strings.Repeat("━", filled) + strings.Repeat("─", width-filled) + "]"
		label += fmt.Sprintf(" / %s · %d%%", byteSize(p.Total), percent)
	} else {
		at := job.frame % width
		bar = "[" + strings.Repeat("─", at) + "━" + strings.Repeat("─", width-at-1) + "]"
	}
	if job.tracks > 1 {
		label += fmt.Sprintf(" · %d/%d tracks completed", job.completed, job.tracks)
	}
	return truncateBlock("  "+bar+" "+job.name+"\n  "+label, w)
}
