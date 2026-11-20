package ui

import (
	"slices"

	"github.com/luvcie/codedrill/internal/storage"
)

func (m *MainModel) isFavorite(id string) bool {
	return slices.Contains(m.favorites, id)
}

func (m *MainModel) toggleFavorite() {
	d, ok := m.selectedDrill()
	if !ok {
		return
	}
	cfg, err := storage.LoadConfig()
	if err == nil {
		if slices.Contains(cfg.Favorites, d.ID) {
			cfg.Favorites = slices.DeleteFunc(cfg.Favorites, func(id string) bool { return id == d.ID })
		} else {
			cfg.Favorites = append(cfg.Favorites, d.ID)
		}
		err = storage.SaveConfig(cfg)
	}
	if err != nil {
		m.menuNotice = "Could not save favorites: " + err.Error()
		m.sizeList()
		return
	}
	m.favorites = cfg.Favorites
	m.refreshList()
	for i, item := range m.list.Items() {
		if item.(drillItem).drill.ID == d.ID {
			m.list.Select(i)
			break
		}
	}
}
