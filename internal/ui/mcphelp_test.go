package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestRenameFieldSitsInsideTheCard(t *testing.T) {
	r := NewRename(nil)
	r.Show("Report")
	// Layout runs before the window bounds exist, which is what the cover does.
	r.Layout(nil, geometry.Constraints{MaxWidth: 1200, MaxHeight: 800})
	r.SetBounds(geometry.NewRect(0, 0, 1200, 800))
	uitest.DrawWidget(r)
	card := r.card()
	field := r.Field().(interface{ Bounds() geometry.Rect }).Bounds()
	if field.Min.X < card.Min.X+16 || field.Max.X > card.Max.X-16 {
		t.Fatalf("field %v card %v", field, card)
	}
	if field.Min.Y < card.Min.Y+32 || field.Max.Y > card.Max.Y-8 {
		t.Fatalf("field %v card %v", field, card)
	}
}

func TestMCPHelpShowsTheAddressOnce(t *testing.T) {
	const (
		url   = "http://127.0.0.1:47328/mcp"
		token = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	)
	m := NewMCPHelp()
	m.Show(url, token, "")
	m.SetBounds(geometry.NewRect(0, 0, 800, 600))
	var b strings.Builder
	for _, tx := range uitest.DrawWidget(m).StyledTexts {
		b.WriteString(tx.Text)
		b.WriteByte('\n')
	}
	text := b.String()
	if strings.Count(text, url) != 1 || strings.Count(text, token) != 1 {
		t.Fatalf("%s", text)
	}
	if strings.Contains(text, "Bearer "+token+"\n"+token) {
		t.Fatal("token repeated")
	}
}
