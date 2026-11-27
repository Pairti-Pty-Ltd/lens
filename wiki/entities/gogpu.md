---
title: GoGPU
type: entity
sources: [doc:///concept.md]
related: [entities/pairti-lens.md, topics/ui-layout-and-surfaces.md, topics/macos-native-integration.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# GoGPU

Lens uses GoGPU for the native window, events, and widget rendering. The desktop and browser builds share the Go widget layout. Consult the versions pinned in [go.mod](../../go.mod) and a Lens widget with the same interaction before changing layout, painting, focus, or input.

On macOS, AppKit menu work belongs on the platform event thread. In the browser, the shared widgets paint to Canvas2D. Measure from input to the presented result when investigating responsiveness. [macOS integration](../topics/macos-native-integration.md) · [Browser rendering](../topics/browser-wasm-editor.md)
