# Pairti Lens. The default target prints this list.
APP_NAME := Lens
BINARY := pairti-lense
BUNDLE_ID := pairti.lens
VERSION := 0.1.0
BUILD := build
LIBRARY ?= $(CURDIR)
APP := $(BUILD)/$(APP_NAME).app
DEST := $(HOME)/Applications/$(APP_NAME).app
ICON_PNG := assets/AppIcon.png
ICONSET := $(BUILD)/AppIcon.iconset
ICON_ICNS := $(BUILD)/AppIcon.icns

.DEFAULT_GOAL := help

.PHONY: help binary app clean wasm wasm-serve

help:
	@printf '\n  Pairti Lens\n\n'
	@printf '  %-10s %s\n' help 'show this list'
	@printf '  %-10s %s\n' binary 'build the executable into $(BUILD)/$(BINARY)'
	@printf '  %-10s %s\n' wasm 'build the browser editor into $(BUILD)/wasm/'
	@printf '  %-10s %s\n' wasm-serve 'serve the Git library on localhost:8081 (LIBRARY defaults to this checkout)'
	@printf '  %-10s %s\n' app 'build $(APP_NAME).app, sign it, and install it for Spotlight'
	@printf '  %-10s %s\n' clean 'remove $(BUILD)/'
	@printf '  Spotlight name: $(APP_NAME)\n'
	@printf '  Installed at:   $(DEST)\n\n'

binary:
	@mkdir -p $(BUILD)
	CGO_ENABLED=0 go build -o $(BUILD)/$(BINARY) ./cmd/pairti-lense

wasm:
	@mkdir -p $(BUILD)/wasm
	GOOS=js GOARCH=wasm CGO_ENABLED=0 go build -o $(BUILD)/wasm/lens.wasm ./cmd/pairti-lense
	cp -f "$$(go env GOROOT)/lib/wasm/wasm_exec.js" cmd/pairti-lense/web/index.html $(BUILD)/wasm/

wasm-serve: export LIBRARY := $(LIBRARY)
wasm-serve: wasm
	CGO_ENABLED=0 go build -o $(BUILD)/lens-library ./cmd/lens-library
	$(BUILD)/lens-library -repo "$$LIBRARY" -assets $(BUILD)/wasm

$(ICON_ICNS): $(ICON_PNG)
	@mkdir -p $(ICONSET)
	sips -z 16 16 $(ICON_PNG) --out $(ICONSET)/icon_16x16.png >/dev/null
	sips -z 32 32 $(ICON_PNG) --out $(ICONSET)/icon_16x16@2x.png >/dev/null
	sips -z 32 32 $(ICON_PNG) --out $(ICONSET)/icon_32x32.png >/dev/null
	sips -z 64 64 $(ICON_PNG) --out $(ICONSET)/icon_32x32@2x.png >/dev/null
	sips -z 128 128 $(ICON_PNG) --out $(ICONSET)/icon_128x128.png >/dev/null
	sips -z 256 256 $(ICON_PNG) --out $(ICONSET)/icon_128x128@2x.png >/dev/null
	sips -z 256 256 $(ICON_PNG) --out $(ICONSET)/icon_256x256.png >/dev/null
	sips -z 512 512 $(ICON_PNG) --out $(ICONSET)/icon_256x256@2x.png >/dev/null
	sips -z 512 512 $(ICON_PNG) --out $(ICONSET)/icon_512x512.png >/dev/null
	sips -z 1024 1024 $(ICON_PNG) --out $(ICONSET)/icon_512x512@2x.png >/dev/null
	iconutil -c icns $(ICONSET) -o $(ICON_ICNS)

app: binary $(ICON_ICNS)
	@mkdir -p "$(APP)/Contents/MacOS" "$(APP)/Contents/Resources"
	cp $(BUILD)/$(BINARY) "$(APP)/Contents/MacOS/$(BINARY)"
	cp $(ICON_ICNS) "$(APP)/Contents/Resources/AppIcon.icns"
	cp -f doc/editor-keymap.md "$(APP)/Contents/Resources/editor-keymap.md"
	@printf '%s\n' \
		'<?xml version="1.0" encoding="UTF-8"?>' \
		'<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">' \
		'<plist version="1.0"><dict>' \
		'<key>CFBundleName</key><string>$(APP_NAME)</string>' \
		'<key>CFBundleDisplayName</key><string>$(APP_NAME)</string>' \
		'<key>CFBundleIdentifier</key><string>$(BUNDLE_ID)</string>' \
		'<key>CFBundleExecutable</key><string>$(BINARY)</string>' \
		'<key>CFBundlePackageType</key><string>APPL</string>' \
		'<key>CFBundleShortVersionString</key><string>$(VERSION)</string>' \
		'<key>CFBundleVersion</key><string>1</string>' \
		'<key>CFBundleIconFile</key><string>AppIcon</string>' \
		'<key>CFBundleDocumentTypes</key><array><dict><key>CFBundleTypeName</key><string>Markdown document</string><key>CFBundleTypeRole</key><string>Editor</string><key>LSHandlerRank</key><string>Alternate</string><key>LSItemContentTypes</key><array><string>net.daringfireball.markdown</string></array></dict></array>' \
		'<key>UTImportedTypeDeclarations</key><array><dict><key>UTTypeIdentifier</key><string>net.daringfireball.markdown</string><key>UTTypeConformsTo</key><array><string>public.plain-text</string></array><key>UTTypeTagSpecification</key><dict><key>public.filename-extension</key><array><string>md</string><string>markdown</string></array></dict></dict></array>' \
		'<key>CFBundleInfoDictionaryVersion</key><string>6.0</string>' \
		'<key>LSMinimumSystemVersion</key><string>12.0</string>' \
		'<key>NSHighResolutionCapable</key><true/>' \
		'</dict></plist>' \
		> "$(APP)/Contents/Info.plist"
	plutil -lint "$(APP)/Contents/Info.plist"
	codesign --force --sign - "$(APP)"
	@mkdir -p "$(HOME)/Applications"
	rm -rf "$(DEST)"
	mv -f "$(APP)" "$(DEST)"
	codesign --force --sign - "$(DEST)"
	@mdimport "$(DEST)" >/dev/null 2>&1 || true
	@printf '\n  Installed %s\n  Spotlight: %s\n\n' "$(DEST)" "$(APP_NAME)"

clean:
	rm -rf $(BUILD)
