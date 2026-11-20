package ui

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestFavoriteGutterAlignment(t *testing.T) {
	item := drillItem{drill: drill.Drill{ID: "rank05-vect2", Title: "vect2", Language: "cpp"}}
	for _, favorite := range []bool{false, true} {
		item.favorite = favorite
		title, description := item.Title(), item.Description()
		gutter := title[:strings.Index(title, "[CPP]")]
		if ansi.StringWidth(gutter) != favoriteGutterWidth || ansi.StringWidth(description[:strings.Index(description, "[")]) != favoriteGutterWidth {
			t.Fatalf("misaligned title/metadata: %q / %q", title, description)
		}
		if favorite {
			if !strings.Contains(gutter, "\U0001F7CA") || strings.Contains(gutter, "★") {
				t.Fatalf("wrong marker: %q", gutter)
			}
			left := ansi.StringWidth(strings.Split(gutter, favoriteMarker)[0])
			right := ansi.StringWidth(strings.Split(gutter, favoriteMarker)[1])
			if right != left {
				t.Fatalf("marker is not centered: %q", gutter)
			}
		} else if strings.TrimSpace(gutter) != "" {
			t.Fatalf("unfavorited gutter is not blank: %q", gutter)
		}
		var matched strings.Builder
		for _, index := range titleMatches(item, "vect2") {
			matched.WriteRune([]rune(title)[index])
		}
		if matched.String() != "vect2" {
			t.Fatalf("search highlighted %q", matched.String())
		}
	}
}

func TestFavoriteToggleAndRestart(t *testing.T) {
	m := newMenuTestModel(t)
	if !strings.Contains(m.menuKeys(), "[*] favorite") {
		t.Fatal("favorite key missing from help")
	}
	if err := storage.SaveConfig(storage.Config{Editor: "hx", Shell: "zsh"}); err != nil {
		t.Fatal(err)
	}
	selectDrill(t, m, "ft_strlen")
	m.Update(runeKey("*"))
	item := m.list.SelectedItem().(drillItem)
	if item.drill.ID != "ft_strlen" || !item.favorite || !strings.Contains(item.Title(), "🟊") {
		t.Fatalf("toggle lost selection or favorite marker: %+v", item)
	}
	m, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	if first := m.list.Items()[0].(drillItem); first.drill.ID != "ft_strlen" || !first.favorite {
		t.Fatalf("favorite did not survive restart: %+v", first)
	}
	cfg, err := storage.LoadConfig()
	if err != nil || !slices.Equal(cfg.Favorites, []string{"ft_strlen"}) || cfg.Editor != "hx" || cfg.Shell != "zsh" {
		t.Fatalf("saved favorite IDs or other settings changed: %+v, %v", cfg, err)
	}
	m.Update(runeKey("*"))
	if item := m.list.SelectedItem().(drillItem); item.drill.ID != "ft_strlen" || item.favorite || strings.Contains(item.Title(), "🟊") {
		t.Fatalf("unmark failed or lost selection: %+v", item)
	}
	m, err = NewMainModel()
	if err != nil || len(m.favorites) != 0 {
		t.Fatalf("unmark did not survive restart: %v", err)
	}
	m.list.Select(len(m.list.Items()) - 1)
	m.Update(runeKey("*"))
	if len(m.favorites) != 0 {
		t.Fatal("add row became a favorite")
	}
}

func TestFavoritesSortWithinActiveFilter(t *testing.T) {
	m := &MainModel{
		drills: []drill.Drill{
			{ID: "a", Title: "Same", Category: "Algorithms", Language: "go", Difficulty: "easy"},
			{ID: "c", Title: "Same", Category: "42/rank 5", Language: "cpp", Difficulty: "medium"},
			{ID: "b", Category: "Algorithms", Language: "cpp", Difficulty: "hard"},
			{ID: "d", Category: "42/rank 5", Language: "cpp", Difficulty: "hard"},
			{ID: "e", Category: "LeetCode", Language: "go", Difficulty: "medium"},
		},
		history: &storage.HistoryStore{}, categories: []string{"ALL", "Algorithms", "42", "LeetCode"},
		favorites: []string{"c", "e", "deleted-id"},
	}
	for _, tc := range []struct {
		filter string
		want   []string
	}{
		{"ALL", []string{"e", "c", "a", "b", "d"}},
		{"Algorithms", []string{"a", "b"}},
		{"42", []string{"c", "d"}},
		{"42/rank 5", []string{"c", "d"}},
		{"cpp", []string{"c", "b", "d"}},
		{"medium", []string{"c", "e"}},
		{"hard", []string{"b", "d"}},
	} {
		t.Run(tc.filter, func(t *testing.T) {
			items := m.menuItemsFor(tc.filter)
			var ids []string
			for _, item := range items {
				di := item.(drillItem)
				if !di.isAdd && !di.isGroup() {
					ids = append(ids, di.drill.ID)
					if di.favorite != m.isFavorite(di.drill.ID) || strings.Contains(di.Title(), "🟊") != di.favorite {
						t.Fatalf("wrong favorite marker: %+v", di)
					}
				}
			}
			if !slices.Equal(ids, tc.want) {
				t.Fatalf("got %v, want %v", ids, tc.want)
			}
			if !items[len(items)-1].(drillItem).isAdd {
				t.Fatal("add row moved from the end")
			}
		})
	}
	m.collapsedGroups = map[string]bool{groupKey("42", "rank 5"): true}
	items := m.menuItemsFor("42")
	if len(items) != 3 || items[0].(drillItem).drill.ID != "c" || !items[1].(drillItem).isGroup() {
		t.Fatalf("collapsed group hid or duplicated favorite: %+v", items)
	}
	m.categoryFilter = "42"
	m.refreshList()
	m.list.Select(1)
	m.Update(runeKey("*"))
	if !slices.Equal(m.favorites, []string{"c", "e", "deleted-id"}) {
		t.Fatal("group header became a favorite")
	}
	// Renaming a drill does not change the favorite, or the search underline.
	m.drills[1].Title = "Renamed"
	m.refreshList()
	item := m.list.Items()[0].(drillItem)
	var matched strings.Builder
	for _, index := range titleMatches(item, "Renamed") {
		matched.WriteRune([]rune(item.Title())[index])
	}
	if !item.favorite || matched.String() != "Renamed" {
		t.Fatalf("renamed favorite lost marker or search match: %+v", item)
	}
}

func TestFavoriteSaveFailureKeepsMenuAndFile(t *testing.T) {
	m := newMenuTestModel(t)
	selectDrill(t, m, "ft_strlen")
	path, err := storage.GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("invalid JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	before := slices.Clone(m.list.Items())
	m.Update(runeKey("*"))
	if len(m.favorites) != 0 || !strings.Contains(m.menuNotice, "Could not save favorites") {
		t.Fatal("save failure was ignored")
	}
	if !slices.EqualFunc(before, m.list.Items(), func(a, b list.Item) bool { return a.(drillItem).Title() == b.(drillItem).Title() }) {
		t.Fatal("failed save changed the menu")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "invalid JSON" {
		t.Fatalf("failed save overwrote config: %q, %v", data, err)
	}
}

func TestFavoritesPreserveSavedOrderAndReordering(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := &MainModel{
		drills: []drill.Drill{
			{ID: "a", Category: "42/rank 5"},
			{ID: "b", Category: "42/rank 5"},
			{ID: "c", Category: "42/rank 5"},
			{ID: "d", Category: "42"},
		},
		history: &storage.HistoryStore{}, categories: []string{"ALL", "42"}, categoryFilter: "42",
		favorites: []string{"c", "d"},
		layout: storage.MenuLayout{Categories: map[string][]storage.LayoutBlock{
			"42": {{Drill: "d"}, {Group: "rank 5", Drills: []string{"c", "b", "a"}}},
		}},
	}
	m.refreshList()
	for _, want := range [][]string{{"d", "c", "", "b", "a", ""}, {"d", "c", "", "a", "b", ""}} {
		var ids []string
		rows, blocks := m.menuRows("42"), m.menuBlocks("42")
		for i, item := range m.list.Items() {
			di := item.(drillItem)
			ids = append(ids, di.drill.ID)
			if di.isAdd || di.isGroup() {
				continue
			}
			row := rows[i]
			d := blocks[row.block].drill
			if row.child >= 0 {
				d = blocks[row.block].drills[row.child]
			}
			if d.ID != di.drill.ID {
				t.Fatalf("display and drag row differ: %s, %s", di.drill.ID, d.ID)
			}
		}
		if !slices.Equal(ids, want) {
			t.Fatalf("order = %v, want %v", ids, want)
		}
		selectDrill(t, m, "b")
		m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	}
	blocks := layoutBlocksFor(t, "42")
	if len(blocks) != 2 || blocks[0].Drill != "d" || !slices.Equal(blocks[1].Drills, []string{"c", "a", "b"}) {
		t.Fatalf("reordering lost favorites or saved the wrong order: %+v", blocks)
	}
}
