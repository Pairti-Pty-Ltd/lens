package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/gogpu/gogpu"
	"pairti-lense/internal/theme"
	"pairti-lense/internal/ui"
)

// Values used by NSMenuItem.keyEquivalentModifierMask.
const (
	menuShift      = 1 << 17
	menuControl    = 1 << 18
	menuCommandKey = 1 << 20
)

type menuCommand struct {
	group, title, key string
	mods              uintptr
	run               func()
	enabled, checked  func() bool
}

func (c menuCommand) invoke() {
	if c.enabled == nil || c.enabled() {
		c.run()
	}
}

func (s *shell) showingLineNumbers() bool {
	return s.buf != nil && ((s.editing && !s.hideRawNumbers) || (!s.editing && s.formattedNumbers))
}

func (s *shell) menuModal() bool {
	return (s.sheet != nil && s.sheet.Shown()) || (s.rename != nil && s.rename.Shown()) ||
		(s.palette != nil && s.palette.Shown()) || (s.mcpHelp != nil && s.mcpHelp.Shown()) ||
		(s.menu != nil && s.menu.Shown()) || (s.tabMenu != nil && s.tabMenu.Shown())
}

func (s *shell) menuCommands() []menuCommand {
	document := s.editableDocument
	split := func() bool { return s.pair != nil && s.pair.Split() }
	tabs := func() bool { return len(s.docs) > 1 }
	tool := func(group, title, name, key string, mods uintptr, checked func() bool) menuCommand {
		return menuCommand{group, title, key, mods, func() { s.onTool(name) }, func() bool { return s.toolAvailable(name) }, checked}
	}
	commands := []menuCommand{
		tool("Lens", "Open Documents in Raw", "Settings", ",", menuCommandKey, func() bool { return s.lib != nil && s.lib.OpenRaw }),
		{"View", "Formatted", "1", menuCommandKey, func() { s.pick("Formatted") }, document, func() bool { return document() && !s.editing }},
		{"View", "Raw", "2", menuCommandKey, func() { s.pick("Raw") }, document, func() bool { return document() && s.editing }},
		tool("View", "Show Line Numbers", "Line numbers", "", 0, s.showingLineNumbers),
		tool("View", "Show Sidebar", "Files", "s", menuCommandKey|menuControl, func() bool { return s.split != nil && s.split.Shown() }),
		tool("View", "Show Outline", "Outline", "", 0, func() bool { return s.railSplit != nil && s.railSplit.Shown() }),
		tool("View", "Split Pane", "Split", "", 0, split),
		tool("View", "Show Shell", "Shell", "", 0, func() bool { return s.console != nil && s.console.Open() }),
		{"View", "Zoom In", "=", menuCommandKey, func() { s.zoom(1) }, func() bool { return s.zoomIndex < theme.ZoomCount()-1 }, nil},
		{"View", "Zoom Out", "-", menuCommandKey, func() { s.zoom(-1) }, func() bool { return s.zoomIndex > 0 }, nil},
		tool("Go", "Back", "Back", "[", menuCommandKey, nil),
		tool("Go", "Forward", "Forward", "]", menuCommandKey, nil),
		{"Go", "Previous Tab", "\t", menuControl | menuShift, func() { s.cycleTab(-1) }, tabs, nil},
		{"Go", "Next Tab", "\t", menuControl, func() { s.cycleTab(1) }, tabs, nil},
		{"Go", "Focus Other Pane", "`", menuControl, func() { s.winCmd('w') }, split, nil},
		{"Go", "Project Files", "", 0, func() { s.goToSidebar(ui.SideFiles) }, nil, nil},
		{"Go", "History", "", 0, func() { s.goToSidebar(ui.SideHistory) }, nil, nil},
		{"Help", "Editor Keymap", "", 0, s.openKeymap, nil, nil},
	}
	files := s.fileMenuCommands()
	commands = append(files, commands...)
	commands = append(commands, s.reportMenuCommands()...)
	for i := range commands {
		enabled := commands[i].enabled
		commands[i].enabled = func() bool { return !s.menuModal() && (enabled == nil || enabled()) }
	}
	// Edit commands validate the focused field even while Rename is open.
	return slices.Insert(commands, len(files), s.editMenuCommands()...)
}

func (s *shell) goToSidebar(mode int) {
	if s.split != nil {
		s.split.SetShown(true)
	}
	s.setSide(mode)
	s.saveSession()
}

func (s *shell) cycleTab(step int) {
	if n := len(s.docs); n > 1 {
		s.show((s.focusedIndex() + step + n) % n)
	}
}

func (s *shell) openKeymap() {
	exe, err := os.Executable()
	if err == nil {
		path := filepath.Join(filepath.Dir(exe), "..", "Resources", "editor-keymap.md")
		if !isFile(path) {
			path = filepath.Join("doc", "editor-keymap.md") // go run from the checkout
		}
		err = exec.Command("open", "-a", "TextEdit", path).Run()
	}
	if err != nil {
		s.line3.Set("Editor keymap: " + err.Error())
		s.writeFooter()
	}
}

// Keep all tabs open when a writer cancels closing an unsaved document.
func (s *shell) canCloseWindow() bool {
	for i := range s.docs {
		if !s.canCloseFile(i) {
			return false
		}
	}
	return true
}

// GoGPU runs OnUpdate on the platform event thread, before OnDraw dispatches
// the UI frame to the render thread. All AppKit menu work belongs here.
func (s *shell) updateNativeCommands(_ float64) {
	if !s.menuModal() && len(s.finderOpens) > 0 {
		paths := s.finderOpens
		s.finderOpens = nil
		for _, path := range paths {
			s.openPath(path)
		}
	}
	if s.quitAsked.Swap(false) && s.quit != nil {
		s.quit()
	}
	if s.refreshMenu != nil {
		s.refreshMenu()
	}
}

func (s *shell) installMenus(app *gogpu.App) {
	s.quit = func() {
		if s.canCloseWindow() {
			app.Quit()
		}
	}
	if runtime.GOOS != "darwin" {
		return
	}
	commands := s.menuCommands()
	app.SetMenu(s.buildNativeMenu(commands))
	var refresh func()
	s.menusDirty = false
	s.refreshMenu = func() {
		if app.PrimaryWindow() == nil {
			return
		}
		if s.menusDirty {
			commands = s.menuCommands()
			app.SetMenu(s.buildNativeMenu(commands))
			refresh = nil
			s.menusDirty = false
		}
		if refresh == nil {
			app.PrimaryWindow().SetOnClose(s.canCloseWindow)
			var err error
			refresh, err = bindMenuItems(commands)
			if err != nil {
				log.Printf("menu state: %v", err)
				return
			}
		}
		refresh()
	}
}

// A path component is a native menu title. NUL cannot occur in a file path.
func menuGroup(titles ...string) string { return strings.Join(titles, "\x00") }

func (s *shell) invalidateMenus() {
	s.menusDirty = true
	// A change discovered during a demand-driven draw needs another main
	// loop turn to install the native menu, even when there is no more input.
	s.redraw()
}

func (s *shell) buildNativeMenu(commands []menuCommand) *gogpu.Menu {
	menu := gogpu.NewMenu()
	menu.AddItem(gogpu.MenuItem{Title: "About Lens", Role: gogpu.RoleAbout}).AddSeparator()
	for _, command := range commands {
		if command.group == "Lens" {
			menu.AddItem(gogpu.MenuItem{Title: command.title, Role: gogpu.RolePreferences, Action: func() {
				command.invoke()
				s.redraw()
			}})
		}
	}
	menu.AddSeparator()
	for _, item := range []gogpu.MenuItem{
		{Title: "Services", Role: gogpu.RoleServices}, gogpu.NewSeparator(),
		{Title: "Hide Lens", Role: gogpu.RoleHide}, {Title: "Hide Others", Role: gogpu.RoleHideOthers},
		{Title: "Show All", Role: gogpu.RoleShowAll}, gogpu.NewSeparator(),
		{Title: "Quit Lens", Role: gogpu.RoleQuit, Action: s.quit},
	} {
		menu.AddItem(item)
	}
	groups := map[string]*gogpu.Menu{"": menu}
	for _, command := range commands {
		if command.group == "Lens" {
			continue
		}
		parent := menu
		parts := strings.Split(command.group, "\x00")
		for i, title := range parts {
			path := menuGroup(parts[:i+1]...)
			group := groups[path]
			if group == nil {
				group = gogpu.NewMenuWithTitle(title)
				parent.AddItem(gogpu.MenuItem{Title: title, Submenu: group})
				groups[path] = group
			}
			parent = group
		}
		parent.AddItem(gogpu.MenuItem{Title: command.title, Action: func() {
			command.invoke()
			s.redraw()
		}})
	}
	return menu
}
