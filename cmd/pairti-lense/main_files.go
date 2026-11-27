package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func copyText(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip")
	default:
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func revealPath(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", "-R", path).Run()
	case "windows":
		return exec.Command("explorer", "/select,", path).Run()
	default:
		return exec.Command("xdg-open", path).Run()
	}
}

func moveToBin(path string) error {
	bin, args, err := binCommand(path)
	if err != nil {
		return err
	}
	return exec.Command(bin, args...).Run()
}

func binCommand(path string) (string, []string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil, fmt.Errorf("missing path")
	}
	switch runtime.GOOS {
	case "darwin":
		return "osascript", []string{"-e", "tell application \"Finder\" to delete POSIX file " + appleString(path)}, nil
	case "windows":
		return "powershell", []string{"-NoProfile", "-Command", "Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.FileIO.FileSystem]::DeleteFile('" + strings.ReplaceAll(path, "'", "''") + "', 'OnlyErrorDialogs', 'SendToRecycleBin')"}, nil
	default:
		if _, err := exec.LookPath("gio"); err == nil {
			return "gio", []string{"trash", path}, nil
		}
		if _, err := exec.LookPath("trash"); err == nil {
			return "trash", []string{path}, nil
		}
		return "", nil, fmt.Errorf("move to bin is not available")
	}
}

func appleString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
