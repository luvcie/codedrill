package ui

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromastyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	glamourconfig "github.com/charmbracelet/glamour/ansi"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const chromaStyleName = "charm"

func briefingStyle(t Theme) glamourconfig.StyleConfig {
	cfg := glamourstyles.DarkStyleConfig
	if t.Light {
		cfg = glamourstyles.LightStyleConfig
	}
	text := string(t.Text)
	muted := string(t.Muted)
	accent := string(t.Accent)
	border := string(t.Border)
	bold := true

	cfg.Document.Color = &text
	cfg.Paragraph.Color = &text
	cfg.Text.Color = &text
	cfg.BlockQuote.Color = &muted
	cfg.List.Color = &text
	cfg.Item.Color = &muted
	cfg.Enumeration.Color = &muted
	cfg.Task.Color = &text
	cfg.Strikethrough.Color = &text
	cfg.Emph.Color = &text
	cfg.Strong.Color = &text
	cfg.HorizontalRule.Color = &border
	cfg.Link.Color = &accent
	cfg.LinkText.Color = &accent
	cfg.Image.Color = &accent
	cfg.ImageText.Color = &muted
	cfg.Table.Color = &text
	cfg.DefinitionDescription.Color = &text

	cfg.Heading.Color = &text
	cfg.Heading.Bold = &bold
	for _, h := range []*glamourconfig.StyleBlock{&cfg.H1, &cfg.H2, &cfg.H3, &cfg.H4, &cfg.H5, &cfg.H6} {
		h.Prefix = ""
		h.Suffix = ""
		h.Color = &text
		h.BackgroundColor = nil
		h.Bold = &bold
	}

	cfg.Code.Color = &text
	cfg.Code.BackgroundColor = &border

	cfg.CodeBlock.Color = &text
	cfg.CodeBlock.Chroma = codeChroma(t)
	return cfg
}

func codeChroma(t Theme) *glamourconfig.Chroma {
	text := colourPrim(t.Text)
	accent := colourPrim(t.Accent)
	success := colourPrim(t.Success)
	danger := colourPrim(t.Danger)
	muted := colourPrim(t.Muted)

	comment := muted
	comment.Italic = boolRef(true)
	emph := text
	emph.Italic = boolRef(true)
	strong := text
	strong.Bold = boolRef(true)

	return &glamourconfig.Chroma{
		Text:    text,
		Error:   danger,
		Comment: comment,
		CommentPreproc: accent,

		Keyword:          accent,
		KeywordReserved:  accent,
		KeywordNamespace: accent,
		KeywordType:      accent,

		Operator:    text,
		Punctuation: text,

		Name:          text,
		NameBuiltin:   text,
		NameTag:       text,
		NameAttribute: text,
		NameClass:     text,
		NameConstant:  text,
		NameDecorator: text,
		NameException: text,
		NameFunction:  text,
		NameOther:     text,

		Literal:             text,
		LiteralNumber:       text,
		LiteralDate:         text,
		LiteralString:       success,
		LiteralStringEscape: success,

		GenericDeleted:    danger,
		GenericEmph:       emph,
		GenericInserted:   success,
		GenericStrong:     strong,
		GenericSubheading: muted,
	}
}

func colourPrim(c lipgloss.Color) glamourconfig.StylePrimitive {
	s := string(c)
	return glamourconfig.StylePrimitive{Color: &s}
}

func boolRef(b bool) *bool { return &b }

func registerChroma(c *glamourconfig.Chroma) {
	entry := func(p glamourconfig.StylePrimitive) string {
		var parts []string
		if p.Color != nil {
			parts = append(parts, *p.Color)
		}
		if p.BackgroundColor != nil {
			parts = append(parts, "bg:"+*p.BackgroundColor)
		}
		if p.Italic != nil && *p.Italic {
			parts = append(parts, "italic")
		}
		if p.Bold != nil && *p.Bold {
			parts = append(parts, "bold")
		}
		if p.Underline != nil && *p.Underline {
			parts = append(parts, "underline")
		}
		return strings.Join(parts, " ")
	}
	chromastyles.Register(chroma.MustNewStyle(chromaStyleName, chroma.StyleEntries{
		chroma.Text:                entry(c.Text),
		chroma.Error:               entry(c.Error),
		chroma.Comment:             entry(c.Comment),
		chroma.CommentPreproc:      entry(c.CommentPreproc),
		chroma.Keyword:             entry(c.Keyword),
		chroma.KeywordReserved:     entry(c.KeywordReserved),
		chroma.KeywordNamespace:    entry(c.KeywordNamespace),
		chroma.KeywordType:         entry(c.KeywordType),
		chroma.Operator:            entry(c.Operator),
		chroma.Punctuation:         entry(c.Punctuation),
		chroma.Name:                entry(c.Name),
		chroma.NameBuiltin:         entry(c.NameBuiltin),
		chroma.NameTag:             entry(c.NameTag),
		chroma.NameAttribute:       entry(c.NameAttribute),
		chroma.NameClass:           entry(c.NameClass),
		chroma.NameConstant:        entry(c.NameConstant),
		chroma.NameDecorator:       entry(c.NameDecorator),
		chroma.NameException:       entry(c.NameException),
		chroma.NameFunction:        entry(c.NameFunction),
		chroma.NameOther:           entry(c.NameOther),
		chroma.Literal:             entry(c.Literal),
		chroma.LiteralNumber:       entry(c.LiteralNumber),
		chroma.LiteralDate:         entry(c.LiteralDate),
		chroma.LiteralString:       entry(c.LiteralString),
		chroma.LiteralStringEscape: entry(c.LiteralStringEscape),
		chroma.GenericDeleted:      entry(c.GenericDeleted),
		chroma.GenericEmph:         entry(c.GenericEmph),
		chroma.GenericInserted:     entry(c.GenericInserted),
		chroma.GenericStrong:       entry(c.GenericStrong),
		chroma.GenericSubheading:   entry(c.GenericSubheading),
		chroma.Background:          entry(c.Background),
	}))
}

func RenderMarkdown(content string, width int) string {
	cfg := briefingStyle(CurrentTheme())
	registerChroma(cfg.CodeBlock.Chroma)
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(cfg),
		glamour.WithWordWrap(max(width-10, 40)),
	glamour.WithChromaFormatter("terminal16m"),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return strings.Trim(out, "\n")
}

func dropTitleHeading(desc string) string {
	lines := strings.Split(desc, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimLeft(strings.Join(lines[i+1:], "\n"), "\n")
		}
		return desc
	}
	return desc
}

func CleanSummary(desc string) string {
	inCode := false
	for _, line := range strings.Split(desc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if !inCode && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return shorten(trimmed, 60)
		}
	}
	return shorten(strings.TrimSpace(desc), 60)
}

func shorten(s string, width int) string {
	if ansi.StringWidth(s) <= width {
		return s
	}
	return strings.TrimRight(ansi.Truncate(s, width-3, ""), " ") + "..."
}
