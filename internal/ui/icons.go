package ui

import "github.com/gogpu/ui/icon"

func stroke(name string, ops ...icon.PathOp) icon.IconData {
	return icon.IconData{Name: name, ViewBox: 24, Ops: ops}
}

var (
	iconSide = stroke("side",
		icon.Move(4, 4), icon.Line(20, 4), icon.Line(20, 20), icon.Line(4, 20), icon.ClosePath(),
		icon.Move(9, 4), icon.Line(9, 20),
	)
	iconForward = stroke("forward",
		icon.Move(4, 12), icon.Line(20, 12),
		icon.Move(20, 12), icon.Line(14, 6),
		icon.Move(20, 12), icon.Line(14, 18),
	)
	iconLink = stroke("link",
		icon.Move(9, 15), icon.Line(15, 9),
		icon.Move(8, 10), icon.Line(8, 16), icon.Line(14, 16),
		icon.Move(10, 8), icon.Line(16, 8), icon.Line(16, 14),
	)
	iconImage = stroke("image",
		icon.Move(4, 5), icon.Line(20, 5), icon.Line(20, 19), icon.Line(4, 19), icon.ClosePath(),
		icon.Move(4, 15), icon.Line(9, 10), icon.Line(13, 14), icon.Line(16, 12), icon.Line(20, 15),
	)
	iconTable = stroke("table",
		icon.Move(4, 4), icon.Line(20, 4), icon.Line(20, 20), icon.Line(4, 20), icon.ClosePath(),
		icon.Move(4, 10), icon.Line(20, 10),
		icon.Move(12, 4), icon.Line(12, 20),
	)
	iconCode = stroke("code",
		icon.Move(9, 7), icon.Line(4, 12), icon.Line(9, 17),
		icon.Move(15, 7), icon.Line(20, 12), icon.Line(15, 17),
	)
	iconTag = stroke("tag",
		icon.Move(4, 12), icon.Line(12, 4), icon.Line(20, 4), icon.Line(20, 12), icon.Line(12, 20), icon.ClosePath(),
		icon.Move(15, 8), icon.Line(16, 9),
	)
	iconExport = stroke("export",
		icon.Move(12, 4), icon.Line(12, 15),
		icon.Move(12, 4), icon.Line(8, 8),
		icon.Move(12, 4), icon.Line(16, 8),
		icon.Move(5, 14), icon.Line(5, 20), icon.Line(19, 20), icon.Line(19, 14),
	)
	iconRail = stroke("rail",
		icon.Move(4, 4), icon.Line(20, 4), icon.Line(20, 20), icon.Line(4, 20), icon.ClosePath(),
		icon.Move(15, 4), icon.Line(15, 20),
	)
	iconList = stroke("list",
		icon.Move(5, 7), icon.Line(19, 7),
		icon.Move(5, 12), icon.Line(19, 12),
		icon.Move(5, 17), icon.Line(19, 17),
	)
	iconDoc = stroke("doc",
		icon.Move(7, 3), icon.Line(14, 3), icon.Line(18, 7), icon.Line(18, 21), icon.Line(7, 21), icon.ClosePath(),
		icon.Move(14, 3), icon.Line(14, 7), icon.Line(18, 7),
	)
	iconCopy = stroke("copy",
		icon.Move(8, 8), icon.Line(18, 8), icon.Line(18, 20), icon.Line(8, 20), icon.ClosePath(),
		icon.Move(6, 5), icon.Line(14, 5), icon.Line(14, 8),
	)
	iconClip = stroke("clip",
		icon.Move(9, 7), icon.Line(15, 7), icon.Line(15, 16), icon.Line(12, 19), icon.Line(9, 16), icon.Line(9, 10),
	)
	iconSplit = stroke("split",
		icon.Move(4, 4), icon.Line(20, 4), icon.Line(20, 20), icon.Line(4, 20), icon.ClosePath(),
		icon.Move(12, 4), icon.Line(12, 20),
	)
	iconShell = stroke("shell",
		icon.Move(4, 7), icon.Line(10, 12), icon.Line(4, 17),
		icon.Move(12, 17), icon.Line(20, 17),
	)
	iconEye = stroke("eye",
		icon.Move(3, 12), icon.Line(6, 8), icon.Line(12, 6), icon.Line(18, 8), icon.Line(21, 12),
		icon.Move(21, 12), icon.Line(18, 16), icon.Line(12, 18), icon.Line(6, 16), icon.Line(3, 12),
		icon.Move(10, 12), icon.Line(12, 10), icon.Line(14, 12), icon.Line(12, 14), icon.ClosePath(),
	)
	iconDocs = stroke("docs",
		icon.Move(4, 5), icon.Line(20, 5),
		icon.Move(7, 10), icon.Line(17, 10),
		icon.Move(10, 15), icon.Line(14, 15), icon.Line(14, 19), icon.Line(10, 19), icon.ClosePath(),
	)
	iconIgnore = stroke("ignore",
		icon.Move(12, 4), icon.Line(18, 7), icon.Line(20, 12), icon.Line(18, 17), icon.Line(12, 20),
		icon.Move(12, 20), icon.Line(6, 17), icon.Line(4, 12), icon.Line(6, 7), icon.Line(12, 4),
		icon.Move(7, 7), icon.Line(17, 17),
	)
	iconTitle = stroke("title",
		icon.Move(4, 8), icon.Line(15, 8),
		icon.Move(4, 13), icon.Line(20, 13),
		icon.Move(4, 17), icon.Line(12, 17),
	)
	iconFile = stroke("file",
		icon.Move(7, 3), icon.Line(14, 3), icon.Line(18, 7), icon.Line(18, 21), icon.Line(7, 21), icon.ClosePath(),
		icon.Move(14, 3), icon.Line(14, 7), icon.Line(18, 7),
		icon.Move(10, 14), icon.Line(15, 14),
	)
)

// DocumentTools is the Zettlr-shaped toolbar. Files, Open, and Remove do
// something. The rest are named placeholders.
func DocumentTools() []Tool {
	return []Tool{
		{Name: "Files", Icon: iconSide},
		{Name: "Search", Icon: icon.Search},
		{Name: "Open", Icon: icon.FolderClosed, Gap: true},
		{Name: "Remove", Icon: icon.Minus},
		{Name: "New", Icon: icon.Plus, Gap: true},
		{Name: "Tag", Icon: iconTag},
		{Name: "Settings", Icon: icon.Gear},
		{Name: "Line numbers", Mark: "1‒3"},
		{Name: "Back", Icon: icon.ArrowBack, Gap: true},
		{Name: "Forward", Icon: iconForward},
		{Name: "Code", Icon: iconCode},
		{Name: "Link", Icon: iconLink},
		{Name: "Image", Icon: iconImage},
		{Name: "Table", Icon: iconTable},
		{Name: "Footnote", Mark: "fn"},
		{Name: "Split", Icon: iconSplit, Gap: true},
		{Name: "Shell", Icon: iconShell},
	}
}

// ExportTool is the export button. It sits on the right, beside the side panel.
func ExportTool() Tool {
	return Tool{Name: "Export", Icon: iconExport}
}

// RailTool is the right sidebar button. It sits at the far right of the menu bar.
func RailTool() Tool {
	return Tool{Name: "Outline", Icon: iconRail}
}
