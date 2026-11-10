package ui

import (
	"slices"
	"strings"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

// menuBlock is one reorderable row block in a category. A group block is a
// subcategory header with its drills; a drill block is one loose drill.
type menuBlock struct {
	group  string
	drills []drill.Drill
	drill  drill.Drill
}

func (b menuBlock) isGroup() bool { return b.group != "" }

// reorderableCategory reports whether the menu can persist a row order for
// category. Views without a real category tab, such as ALL, a language filter
// or a full subcategory filter, stay flat, so dragging is off there.
func (m *MainModel) reorderableCategory(category string) bool {
	return category != "" &&
		!strings.EqualFold(category, "ALL") &&
		strings.EqualFold(category, drill.ParentCategory(category)) &&
		slices.ContainsFunc(m.categories, func(c string) bool { return strings.EqualFold(c, category) })
}

func (m *MainModel) savedBlocks(category string) []storage.LayoutBlock {
	for key, blocks := range m.layout.Categories {
		if strings.EqualFold(key, category) {
			return blocks
		}
	}
	return nil
}

// menuBlocks is the ordered block list for a category. Saved order comes
// first; groups, drills and group children that are new since the last drag
// are appended in the default order.
func (m *MainModel) menuBlocks(category string) []menuBlock {
	rows := drill.Filter(m.drills, category)

	type groupData struct {
		name   string
		drills []drill.Drill
	}
	groups := make(map[string]*groupData)
	loose := make(map[string]drill.Drill)
	var looseOrder []string
	for _, d := range rows {
		sub := drill.SubcategoryOf(d)
		if sub != "" && strings.EqualFold(drill.ParentCategoryOf(d), category) {
			key := strings.ToUpper(sub)
			data := groups[key]
			if data == nil {
				data = &groupData{name: sub}
				groups[key] = data
			}
			data.drills = append(data.drills, d)
			continue
		}
		if _, seen := loose[d.ID]; !seen {
			looseOrder = append(looseOrder, d.ID)
		}
		loose[d.ID] = d
	}

	seenGroups := make(map[string]bool)
	seenLoose := make(map[string]bool)
	var blocks []menuBlock
	for _, block := range m.savedBlocks(category) {
		if group := strings.TrimSpace(block.Group); group != "" {
			key := strings.ToUpper(group)
			data := groups[key]
			if data == nil || seenGroups[key] {
				continue
			}
			data.drills = orderDrills(data.drills, block.Drills)
			blocks = append(blocks, menuBlock{group: data.name, drills: data.drills})
			seenGroups[key] = true
			continue
		}
		id := strings.TrimSpace(block.Drill)
		if id == "" || seenLoose[id] {
			continue
		}
		d, ok := loose[id]
		if !ok {
			continue
		}
		blocks = append(blocks, menuBlock{drill: d})
		seenLoose[id] = true
	}

	var unseenGroups []string
	for key := range groups {
		if !seenGroups[key] {
			unseenGroups = append(unseenGroups, key)
		}
	}
	slices.SortStableFunc(unseenGroups, func(a, b string) int {
		return strings.Compare(strings.ToUpper(groups[a].name), strings.ToUpper(groups[b].name))
	})
	for _, key := range unseenGroups {
		data := groups[key]
		blocks = append(blocks, menuBlock{group: data.name, drills: data.drills})
	}
	for _, id := range looseOrder {
		if !seenLoose[id] {
			blocks = append(blocks, menuBlock{drill: loose[id]})
		}
	}
	return blocks
}

func orderDrills(drills []drill.Drill, ids []string) []drill.Drill {
	byID := make(map[string]drill.Drill, len(drills))
	for _, d := range drills {
		byID[d.ID] = d
	}
	seen := make(map[string]bool, len(drills))
	ordered := make([]drill.Drill, 0, len(drills))
	for _, id := range ids {
		if d, ok := byID[id]; ok && !seen[id] {
			ordered = append(ordered, d)
			seen[id] = true
		}
	}
	for _, d := range drills {
		if !seen[d.ID] {
			ordered = append(ordered, d)
		}
	}
	return ordered
}

func (m *MainModel) saveMenuBlocks(category string, blocks []menuBlock) error {
	next := storage.MenuLayout{Categories: make(map[string][]storage.LayoutBlock, len(m.layout.Categories)+1)}
	for key, value := range m.layout.Categories {
		next.Categories[key] = value
	}
	key := category
	for existing := range m.layout.Categories {
		if strings.EqualFold(existing, category) {
			key = existing
			break
		}
	}
	next.Categories[key] = blocksToLayout(blocks)
	if err := storage.SaveLayout(next); err != nil {
		return err
	}
	m.layout = next
	return nil
}

func blocksToLayout(blocks []menuBlock) []storage.LayoutBlock {
	out := make([]storage.LayoutBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.isGroup() {
			ids := make([]string, len(block.drills))
			for i, d := range block.drills {
				ids[i] = d.ID
			}
			out = append(out, storage.LayoutBlock{Group: block.group, Drills: ids})
			continue
		}
		out = append(out, storage.LayoutBlock{Drill: block.drill.ID})
	}
	return out
}

// menuRow says where a rendered list row sits in the block tree. child is -1
// for a group header, a loose drill or the add row.
type menuRow struct {
	block int
	child int
	group bool
	add   bool
}

func (m *MainModel) menuRows(category string) []menuRow {
	blocks := m.menuBlocks(category)
	var rows []menuRow
	for bi, block := range blocks {
		if block.isGroup() {
			rows = append(rows, menuRow{block: bi, child: -1, group: true})
			if m.groupCollapsed(groupKey(category, block.group)) {
				continue
			}
			for ci := range block.drills {
				rows = append(rows, menuRow{block: bi, child: ci, group: true})
			}
			continue
		}
		rows = append(rows, menuRow{block: bi, child: -1})
	}
	return append(rows, menuRow{block: len(blocks), child: -1, add: true})
}

func (m *MainModel) menuRowForIndex(index int) (menuRow, bool) {
	rows := m.menuRows(m.categoryFilter)
	if index < 0 || index >= len(rows) {
		return menuRow{}, false
	}
	return rows[index], true
}
