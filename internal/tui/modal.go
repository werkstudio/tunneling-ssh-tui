package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpItem struct {
	key  string
	desc string
}

type helpSection struct {
	title string
	items []helpItem
}

var (
	modalBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cAccent).
			Padding(1, 2)

	modalSectionStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(cAccent)

	modalKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cAccent).
			Width(16)

	modalDescStyle = lipgloss.NewStyle().
			Foreground(cText)

	modalFooterStyle = lipgloss.NewStyle().
				Foreground(cDim)
)

func renderSection(s helpSection) string {
	var lines []string
	lines = append(lines, modalSectionStyle.Render(s.title))
	for _, item := range s.items {
		line := modalKeyStyle.Render(item.key) + " " + modalDescStyle.Render(item.desc)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func viewHelpModal(w, h int) string {
	sections := []helpSection{
		{
			title: "Navigasi",
			items: []helpItem{
				{key: "↑↓ / jk", desc: "Pindah kursor"},
				{key: "PgUp/PgDn / JK", desc: "Gulir log"},
				{key: "/", desc: "Cari / filter tunnel"},
			},
		},
		{
			title: "Kontrol Tunnel",
			items: []helpItem{
				{key: "Space / Enter", desc: "Start / Stop tunnel"},
				{key: "r", desc: "Restart tunnel"},
				{key: "a", desc: "Start semua tunnel"},
				{key: "x", desc: "Stop semua tunnel"},
			},
		},
		{
			title: "Aksi Cepat",
			items: []helpItem{
				{key: "o", desc: "Buka di browser"},
				{key: "c", desc: "Salin URL ke clipboard"},
				{key: "n", desc: "Tunnel baru"},
				{key: "e", desc: "Edit tunnel"},
				{key: "d", desc: "Hapus tunnel"},
			},
		},
		{
			title: "Aplikasi",
			items: []helpItem{
				{key: "?", desc: "Toggle bantuan"},
				{key: "Esc / Enter", desc: "Tutup dialog bantuan"},
				{key: "q / Ctrl+C", desc: "Keluar aplikasi"},
			},
		},
	}

	header := titleStyle.Render("sshtui") + " " + lipgloss.NewStyle().Bold(true).Foreground(cText).Render("Bantuan & Shortcut")

	var body string
	if w >= 70 {
		// 2-column layout
		col1 := lipgloss.JoinVertical(lipgloss.Left,
			renderSection(sections[0]), // Navigasi
			"",
			renderSection(sections[3]), // Aplikasi
		)
		col2 := lipgloss.JoinVertical(lipgloss.Left,
			renderSection(sections[1]), // Kontrol Tunnel
			"",
			renderSection(sections[2]), // Aksi Cepat
		)
		body = lipgloss.JoinHorizontal(lipgloss.Top, col1, "    ", col2)
	} else {
		// Stacked 1-column layout for narrower terminals
		var secRenders []string
		for i, sec := range sections {
			secRenders = append(secRenders, renderSection(sec))
			if i < len(sections)-1 {
				secRenders = append(secRenders, "")
			}
		}
		body = strings.Join(secRenders, "\n")
	}

	footer := modalFooterStyle.Render("Tekan Esc, ?, atau Enter untuk menutup")
	content := lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer)

	modal := modalBoxStyle.Render(content)

	if w <= 0 || h <= 0 {
		return modal
	}

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, modal)
}
