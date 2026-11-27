package report

import (
	"fmt"
	"sync"
)

var (
	transferMu    sync.Mutex
	transferMsg   string
	transferDirty bool
	transferWake  func()
)

// SetTransfer replaces the status-bar line for a background download.
// An empty string clears it. It is safe to call from a goroutine.
func SetTransfer(msg string) {
	transferMu.Lock()
	if transferMsg == msg {
		transferMu.Unlock()
		return
	}
	transferMsg = msg
	transferDirty = true
	wake := transferWake
	transferMu.Unlock()
	if wake != nil {
		wake()
	}
}

// WatchTransfer registers the redraw used when a transfer line changes.
func WatchTransfer(wake func()) {
	transferMu.Lock()
	transferWake = wake
	transferMu.Unlock()
}

// Transfer is the current download line. changed is true once per update.
func Transfer() (string, bool) {
	transferMu.Lock()
	defer transferMu.Unlock()
	if !transferDirty {
		return transferMsg, false
	}
	transferDirty = false
	return transferMsg, true
}

func transferLine(name string, got, total int64) string {
	if total > 0 {
		return fmt.Sprintf("Downloading %s · %s / %s", name, sizeText(got), sizeText(total))
	}
	if got > 0 {
		return fmt.Sprintf("Downloading %s · %s", name, sizeText(got))
	}
	return "Downloading " + name
}

func sizeText(n int64) string {
	if n < 0 {
		n = 0
	}
	const kb = 1024
	const mb = 1024 * 1024
	if n < mb {
		return fmt.Sprintf("%d KB", (n+kb-1)/kb)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
}
