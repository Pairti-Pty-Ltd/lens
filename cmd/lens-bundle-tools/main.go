// Command lens-bundle-tools stages the signed app's pinned report tools.
package main

import (
	"context"
	"log"
	"os"

	"pairti-lense/internal/report"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: lens-bundle-tools Lens.app/Contents")
	}
	if err := report.StageMacTools(context.Background(), os.Args[1]); err != nil {
		log.Fatal(err)
	}
}
