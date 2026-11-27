package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
)

func TestPageCtrlFScrollsOneViewport(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, "line %d of the reading\n\n", i)
	}
	p := NewPage()
	buf := edit.New(b.String())
	p.UseBuffer(buf)
	p.SetBlocks(doc.Blocks([]byte(b.String())))
	sig := state.NewSignal(float32(0))
	p.OnScroll(func(y float32) { sig.Set(y) })
	sv := scrollview.New(ScrollInset(p),
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollYSignal(sig),
		scrollview.ScrollStep(12),
	)
	uitest.LayoutWidget(sv, 800, 400)
	sv.SetBounds(geometry.NewRect(0, 0, 800, 400))
	p.Reopen(1)
	p.SetFocused(true)

	ctrl := func(key event.Key) {
		t.Helper()
		ev := event.NewKeyEvent(event.KeyPress, key, 0, event.ModCtrl)
		if !p.Event(nil, ev) {
			t.Fatalf("ctrl-%c was not used", 'a'+(key-event.KeyA))
		}
	}
	viewH := sv.ViewportSize().Height
	content := sv.ContentSize().Height
	if viewH < 100 || content < viewH*3 {
		t.Fatalf("viewport %v content %v", viewH, content)
	}
	ctrl(event.KeyF)
	if sig.Get() <= 0 || sig.Get() >= viewH || buf.CaretLine() > 30 {
		t.Fatalf("ctrl-f scroll %v caret %d content %v", sig.Get(), buf.CaretLine(), content)
	}
	first := sig.Get()
	ctrl(event.KeyF)
	if sig.Get() <= first || sig.Get() >= content-viewH {
		t.Fatalf("second ctrl-f scroll %v", sig.Get())
	}
	ctrl(event.KeyB)
	if sig.Get() > first+1 {
		t.Fatalf("ctrl-b scroll %v, first page was %v", sig.Get(), first)
	}
	at := sig.Get()
	ctrl(event.KeyE)
	if sig.Get() <= at || sig.Get()-at > viewH/2 {
		t.Fatalf("ctrl-e moved %v", sig.Get()-at)
	}
	ctrl(event.KeyY)
	if sig.Get() > at+1 {
		t.Fatalf("ctrl-y scroll %v, was %v", sig.Get(), at)
	}
}

func TestPageMotionRevealsAndPlaces(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "paragraph %d sits on its own line in the reading\n\n", i)
	}
	p := NewPage()
	buf := edit.New(b.String())
	p.UseBuffer(buf)
	p.SetBlocks(doc.Blocks([]byte(b.String())))
	sig := state.NewSignal(float32(0))
	p.OnScroll(func(y float32) { sig.Set(y) })
	sv := scrollview.New(ScrollInset(p),
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollYSignal(sig),
	)
	uitest.LayoutWidget(sv, 800, 400)
	sv.SetBounds(geometry.NewRect(0, 0, 800, 400))
	p.Reopen(1)
	p.SetFocused(true)
	down := event.NewKeyEvent(event.KeyPress, event.KeyJ, 'j', 0)
	if !p.Event(nil, down) {
		t.Fatal("j was not used")
	}
	if sig.Get() != 0 {
		t.Fatalf("j scrolled a visible line to %v", sig.Get())
	}
	buf.GoLine(12)
	p.Reopen(12)
	z := event.NewKeyEvent(event.KeyPress, event.KeyZ, 'z', 0)
	tee := event.NewKeyEvent(event.KeyPress, event.KeyT, 't', 0)
	if !p.Event(nil, z) || !p.Event(nil, tee) {
		t.Fatal("zt was not used")
	}
	want := p.LineY(buf.CaretLine() + 1)
	if sig.Get() < want-1 || sig.Get() > want+1 {
		t.Fatalf("zt scroll %v, line at %v", sig.Get(), want)
	}
	if sig.Get() > sv.ContentSize().Height-sv.ViewportSize().Height {
		t.Fatalf("zt went to the end %v", sig.Get())
	}
}
