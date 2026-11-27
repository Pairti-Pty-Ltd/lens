package ui

// WheelScroll turns a wheel delta into a sidebar scroll change.
// natural is macOS natural scrolling: the list moves with the gesture.
// A positive result reveals rows below.
func WheelScroll(deltaY float32, natural bool) float32 {
	const speed = 12
	if natural {
		return deltaY * speed
	}
	return -deltaY * speed
}

// RowIndex is the row under localY. top is the list's top in the same
// coordinate space as localY. scroll is how far the list is scrolled.
func RowIndex(localY, top, scroll, rowH float32, n int) int {
	if rowH <= 0 || n <= 0 {
		return -1
	}
	y := localY - top + scroll
	if y < 0 {
		return -1
	}
	i := int(y / rowH)
	if i >= n {
		return -1
	}
	return i
}

// SidebarWidth keeps the sidebar at a pixel width across window resizes.
// It shrinks only when the window no longer fits both panes.
func SidebarWidth(current, total float32) float32 {
	const minPane = 160
	const divider = 6
	if total <= minPane*2+divider {
		if total < 1 {
			return 0
		}
		return total * 0.4
	}
	if current < minPane {
		current = minPane
	}
	maxW := total - minPane - divider
	if current > maxW {
		return maxW
	}
	return current
}
