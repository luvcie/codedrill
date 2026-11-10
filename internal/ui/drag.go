package ui

import (
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/drill"
)

const (
	// dragThreshold is how far the pointer must move before a press becomes a
	// drag. A list row is three cells tall, so two cells catches a move to the
	// next row without reacting to a shaky click.
	dragThreshold = 2
	// dragScrollDelay is how long the pointer must stay at an edge before the
	// menu pages. Repeating it lets a held drag walk through several pages.
	dragScrollDelay = 220 * time.Millisecond
)

// dragScrollMsg wakes a drag that is still held at the top or bottom edge.
// The terminal sends no motion events while the pointer is still, so a timer
// has to move the page instead.
type dragScrollMsg struct {
	press *menuPress
}

// menuPress is a left press that may still become a drag. It also keeps the
// click behaviour: the first click selects, the second click on the selected
// row acts.
type menuPress struct {
	index       int
	item        drillItem
	x, y        int
	wasSelected bool
	canDrag     bool
	row         menuRow

	dragging  bool
	target    int
	targetRow menuRow

	// pointerX and pointerY are the latest drag position. The scroll timer
	// reads them while no motion events arrive.
	pointerX, pointerY int
	scrollQueued       bool
}

// menuDelegate is the list delegate with the current search highlight and,
// while a row is being dragged, the drop target marker.
func (m *MainModel) menuDelegate() drillDelegate {
	d := newDrillDelegate(m.search.highlight())
	if m.press != nil && m.press.dragging && m.press.target >= 0 {
		d.drop = m.press.target
	}
	return d
}

// cancelPress drops a pending press or drag without changing the arrangement.
func (m *MainModel) cancelPress() {
	if m.press == nil {
		return
	}
	p := m.press
	m.press = nil
	// A drag may have paged away from the row it started on. Bring that row
	// back before dropping the delegate marker.
	if p.dragging {
		m.list.Select(p.index)
	}
	m.list.SetDelegate(m.menuDelegate())
}

func (m *MainModel) beginMenuPress(x, y int) tea.Cmd {
	idx, ok := m.menuItemAt(x, y)
	if !ok {
		return nil
	}
	item, ok := m.list.Items()[idx].(drillItem)
	if !ok {
		return nil
	}

	m.dismissNotice()
	// A press without a release can be replaced by a new press.
	m.cancelPress()

	wasSelected := idx == m.list.Index()
	if !wasSelected {
		m.list.Select(idx)
	}
	p := &menuPress{
		index: idx, item: item, x: x, y: y,
		pointerX: x, pointerY: y,
		wasSelected: wasSelected, target: -1,
	}
	if !item.isAdd && m.reorderableCategory(m.categoryFilter) {
		if row, ok := m.menuRowForIndex(idx); ok {
			p.canDrag = true
			p.row = row
		}
	}
	m.press = p
	return nil
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (m *MainModel) updateMenuDrag(msg tea.MouseMsg) tea.Cmd {
	p := m.press
	if p == nil {
		return nil
	}
	switch msg.Action {
	case tea.MouseActionMotion:
		m.menuDragMotion(p, msg.X, msg.Y)
		return m.scheduleDragScroll(p)
	case tea.MouseActionRelease:
		m.press = nil
		m.list.SetDelegate(m.menuDelegate())
		if p.dragging {
			if p.target >= 0 {
				m.dropMenuPress(p)
			} else {
				m.list.Select(p.index)
			}
			return nil
		}
		if !p.wasSelected {
			return nil
		}
		if idx, ok := m.menuItemAt(msg.X, msg.Y); !ok || idx != p.index {
			return nil
		}
		return m.activateSelected()
	}
	return nil
}

func (m *MainModel) menuDragMotion(p *menuPress, x, y int) {
	if !p.dragging {
		if !p.canDrag {
			return
		}
		if absInt(x-p.x) < dragThreshold && absInt(y-p.y) < dragThreshold {
			return
		}
		p.dragging = true
	}
	p.pointerX, p.pointerY = x, y
	m.updateDragTarget(p)
}

func (m *MainModel) updateDragTarget(p *menuPress) {
	if idx, row, ok := m.dragTarget(p, p.pointerX, p.pointerY); ok {
		p.target = idx
		p.targetRow = row
	} else {
		p.target = -1
		p.targetRow = menuRow{}
	}
	m.list.SetDelegate(m.menuDelegate())
}

// dragScrollDirection reports which way the pointer is asking the list to
// page. The bottom zone wins when a page has only one row, so the common
// bottom drag still pages forward.
func (m *MainModel) dragScrollDirection(y int) int {
	pageItems := m.list.Paginator.ItemsOnPage(len(m.list.VisibleItems()))
	if pageItems <= 0 {
		return 0
	}
	d := newDrillDelegate("")
	step := d.Height() + d.Spacing()
	first := m.menuItemTop() - m.menuShift()
	last := first + (pageItems-1)*step
	if y >= last {
		return 1
	}
	if y <= first+d.Height()-1 {
		return -1
	}
	return 0
}

func (m *MainModel) scheduleDragScroll(p *menuPress) tea.Cmd {
	if p == nil || m.press != p || !p.dragging || p.scrollQueued || m.state != StateMenu {
		return nil
	}
	direction := m.dragScrollDirection(p.pointerY)
	if direction == 0 ||
		(direction < 0 && m.list.Paginator.OnFirstPage()) ||
		(direction > 0 && m.list.Paginator.OnLastPage()) {
		return nil
	}
	p.scrollQueued = true
	return tea.Tick(dragScrollDelay, func(time.Time) tea.Msg {
		return dragScrollMsg{press: p}
	})
}

func (m *MainModel) updateDragScroll(msg dragScrollMsg) tea.Cmd {
	p := m.press
	if p == nil || p != msg.press || !p.dragging || m.state != StateMenu {
		return nil
	}
	p.scrollQueued = false
	direction := m.dragScrollDirection(p.pointerY)
	if direction == 0 {
		return nil
	}
	if direction < 0 {
		if m.list.Paginator.OnFirstPage() {
			return nil
		}
		m.list.PrevPage()
	} else {
		if m.list.Paginator.OnLastPage() {
			return nil
		}
		m.list.NextPage()
	}
	// SetDelegate recomputes the page from the selected row, so put the cursor
	// at the start of the page that was just reached.
	m.list.Select(m.list.Paginator.Page * m.list.Paginator.PerPage)
	m.updateDragTarget(p)
	return m.scheduleDragScroll(p)
}

func (m *MainModel) dragTarget(p *menuPress, x, y int) (int, menuRow, bool) {
	idx, ok := m.menuItemAt(x, y)
	if !ok {
		return 0, menuRow{}, false
	}
	rows := m.menuRows(m.categoryFilter)
	if idx < 0 || idx >= len(rows) {
		return 0, menuRow{}, false
	}
	row := rows[idx]
	blocks := m.menuBlocks(m.categoryFilter)

	if p.row.child >= 0 {
		// A drill inside a subcategory stays in that group.
		if row.add || !row.group || row.block != p.row.block {
			return 0, menuRow{}, false
		}
		to := row.child
		if to < 0 {
			to = 0
		}
		if to == p.row.child {
			return 0, menuRow{}, false
		}
		return idx, row, true
	}

	// A subcategory header or a loose drill moves as a block. Drop it on
	// another block header, another loose drill, or the end of the list.
	if row.add {
		if p.row.block >= len(blocks)-1 {
			return 0, menuRow{}, false
		}
		return idx, row, true
	}
	if row.group && row.child >= 0 {
		return 0, menuRow{}, false
	}
	if row.block == p.row.block {
		return 0, menuRow{}, false
	}
	return idx, row, true
}

func (m *MainModel) dropMenuPress(p *menuPress) {
	category := m.categoryFilter
	if !m.reorderableCategory(category) {
		return
	}
	blocks := m.menuBlocks(category)

	if p.row.child >= 0 {
		if p.row.block < 0 || p.row.block >= len(blocks) || !blocks[p.row.block].isGroup() {
			return
		}
		to := p.targetRow.child
		if to < 0 {
			to = 0
		}
		blocks[p.row.block].drills = moveMenuDrill(blocks[p.row.block].drills, p.row.child, to)
	} else {
		from := p.row.block
		to := p.targetRow.block
		if p.targetRow.add {
			to = len(blocks)
		}
		if from < 0 || from >= len(blocks) || to < 0 || to > len(blocks) {
			return
		}
		blocks = moveMenuBlock(blocks, from, to)
	}

	if err := m.saveMenuBlocks(category, blocks); err != nil {
		m.menuNotice = "Could not save arrangement: " + err.Error()
		m.list.Select(p.index)
		m.sizeList()
		return
	}
	m.refreshList()
	m.selectAfterDrag(p)
}

// keyboardMenuMove reorders the selected row by one block or one sibling. It
// follows the same valid moves as a mouse drop and saves through the same path.
func (m *MainModel) keyboardMenuMove(delta int) tea.Cmd {
	if !m.reorderableCategory(m.categoryFilter) {
		return nil
	}
	idx := m.list.Index()
	item, ok := m.list.Items()[idx].(drillItem)
	if !ok || item.isAdd {
		return nil
	}
	row, ok := m.menuRowForIndex(idx)
	if !ok {
		return nil
	}
	blocks := m.menuBlocks(m.categoryFilter)

	var target menuRow
	if row.child >= 0 {
		if row.block < 0 || row.block >= len(blocks) {
			return nil
		}
		child := row.child + delta
		if child < 0 || child >= len(blocks[row.block].drills) {
			return nil
		}
		target = menuRow{block: row.block, child: child, group: true}
	} else {
		block := row.block + delta
		if block < 0 || block >= len(blocks) {
			return nil
		}
		target = menuRow{block: block, child: -1, group: blocks[block].isGroup()}
	}

	m.dropMenuPress(&menuPress{index: idx, item: item, row: row, target: -1, targetRow: target})
	return nil
}

func (m *MainModel) selectAfterDrag(p *menuPress) {
	for i, item := range m.list.Items() {
		di, ok := item.(drillItem)
		if !ok || di.isAdd {
			continue
		}
		if p.row.child >= 0 || !p.row.group {
			if !di.isGroup() && di.drill.ID == p.item.drill.ID {
				m.list.Select(i)
				return
			}
			continue
		}
		if di.isGroup() && di.groupKey == p.item.groupKey {
			m.list.Select(i)
			return
		}
	}
}

func moveMenuBlock(blocks []menuBlock, from, target int) []menuBlock {
	if from < 0 || from >= len(blocks) || target < 0 || target > len(blocks) || from == target {
		return blocks
	}
	block := blocks[from]
	rest := append(slices.Clone(blocks[:from]), blocks[from+1:]...)
	if target > len(rest) {
		target = len(rest)
	}
	return slices.Insert(rest, target, block)
}

func moveMenuDrill(drills []drill.Drill, from, target int) []drill.Drill {
	if from < 0 || from >= len(drills) || target < 0 || target > len(drills) || from == target {
		return drills
	}
	d := drills[from]
	rest := append(slices.Clone(drills[:from]), drills[from+1:]...)
	if target > len(rest) {
		target = len(rest)
	}
	return slices.Insert(rest, target, d)
}
