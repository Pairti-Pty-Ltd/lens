package edit

import "os"

// WriteFile replaces path with text. The bytes on disk are the buffer.
func WriteFile(path, text string) error {
	return os.WriteFile(path, []byte(text), 0o644)
}
