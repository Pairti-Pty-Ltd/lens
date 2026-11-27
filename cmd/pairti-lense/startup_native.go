//go:build !js

// Pairti Lens build 0: several project folders, one rendered Markdown file.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/gogpu/gg/gpu"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/desktop"
	"github.com/gogpu/ui/theme/material3"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/agent"
	"pairti-lense/internal/diagram"
	"pairti-lense/internal/diagram/mermaid"
	"pairti-lense/internal/doc"
	"pairti-lense/internal/index"
	"pairti-lense/internal/library"
	"pairti-lense/internal/report"
	"pairti-lense/internal/session"
	"pairti-lense/internal/theme"
	"pairti-lense/internal/ui"
	"pairti-lense/internal/watch"
)

func main() {
	diagram.Register(mermaid.New())
	if _, err := ui.RegisterShellFace(); err != nil {
		log.Printf("shell font: %v", err)
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Printf("config dir: %v", err)
		configDir = os.TempDir()
	}
	libPath := filepath.Join(configDir, "pairti-lense", "library.json")
	lib, err := library.Load(libPath)
	if err != nil {
		log.Printf("library: %v", err)
		quarantine(libPath, err)
		lib = &library.Library{}
	}
	themePath := filepath.Join(configDir, "pairti-lense", "theme.json")
	loaded, err := theme.Load(themePath)
	if err != nil {
		log.Printf("theme: %v", err)
		quarantine(themePath, err)
		loaded = theme.Pairti()
	}
	savedZoom, err := theme.ReadZoom(themePath)
	if err != nil {
		log.Printf("zoom: %v", err)
		savedZoom = ""
	}
	base := loaded
	zoomIndex := theme.ZoomIndex("100%")
	if savedZoom != "" {
		base = theme.WithPairtiType(loaded)
		zoomIndex = theme.ZoomIndex(savedZoom)
	} else {
		zoomIndex = theme.NearestZoom(loaded.Body)
		base = theme.WithPairtiType(loaded)
	}
	k, fit := theme.ZoomScale(zoomIndex)
	if fit {
		k = 1
	}
	look := base.Scale(k)

	s := &shell{
		lib:         lib,
		libPath:     libPath,
		look:        look,
		base:        base,
		zoomIndex:   zoomIndex,
		themePath:   themePath,
		cache:       map[string][]doc.DirEntry{},
		expanded:    map[string]bool{},
		active:      -1,
		splitAt:     -1,
		histAt:      -1,
		sessionPath: filepath.Join(configDir, "pairti-lense", "session.json"),
		idx:         index.New(),
	}
	s.rescan()
	if st, err := session.Load(s.sessionPath); err != nil {
		log.Printf("session: %v", err)
		quarantine(s.sessionPath, err)
	} else {
		s.boot = st
		if st.RestoreTree {
			s.expanded = map[string]bool{}
			for _, p := range st.Expanded {
				p = strings.TrimSpace(p)
				if p != "" {
					s.expanded[p] = true
				}
			}
		}
	}
	s.console = ui.NewConsole()
	s.initWidgets()
	m3 := material3.NewDark(widget.Hex(s.look.Accent))
	gogpuApp := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle("Lens").
		WithAppName("Lens").
		WithSize(1200, 800))
	fw, err := watch.New(gogpuApp.RequestRedraw)
	if err != nil {
		log.Printf("watch: %v", err)
	} else {
		s.watch = fw
		s.syncWatch()
	}
	s.wake = gogpuApp.RequestRedraw
	report.WatchTransfer(s.wake)
	go func() {
		if _, err := report.Bin(); err != nil {
			log.Printf("typst: %v", err)
		}
	}()
	s.agent = agent.NewHub(gogpuApp.RequestRedraw)
	if err := s.listenMCP(filepath.Join(configDir, "pairti-lense", "mcp.json")); err != nil {
		log.Printf("mcp: %v", err)
		s.mcpErr = err.Error()
	}
	s.noteMCP()
	uiApp := app.New(
		app.WithWindowProvider(gogpuApp),
		app.WithPlatformProvider(gogpuApp),
		app.WithEventSource(s.menuEventSource(gogpuApp.EventSource())),
		app.WithTheme(m3.AsTheme()),
	)
	s.menuContext = uiApp.Window().Context()
	s.clipboard = gogpuApp
	uiApp.SetRoot(ui.NewPulse(s.build(), s.tick))

	s.files = fileActions{pick: gogpuApp.ShowOpenFileDialog, newPath: gogpuApp.ShowSaveFileDialog, reveal: revealPath, confirmClose: confirmFileClose}
	if err := installFinderOpen(func(path string) {
		s.finderOpens = append(s.finderOpens, path)
		s.redraw()
	}); err != nil {
		log.Printf("Finder open: %v", err)
	}
	s.installMenus(gogpuApp)
	gogpuApp.OnUpdate(s.updateNativeCommands)
	tray := startTray(gogpuApp, func() {
		s.mcpAsk.Store(true)
		if s.wake != nil {
			s.wake()
		}
	}, func() {
		s.quitAsked.Store(true)
		gogpuApp.RequestRedraw()
	})
	defer tray.Remove()
	defer s.closeDiff()
	defer func() {
		if s.console != nil {
			s.console.Close()
		}
		if s.mcpSrv != nil {
			_ = s.mcpSrv.Close()
		}
	}()

	if err := desktop.Run(gogpuApp, uiApp); err != nil {
		log.Fatal(err)
	}
	_ = s.lib.Save(s.libPath)
	s.saveSession()
}
