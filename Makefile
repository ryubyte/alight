.PHONY: build run test clean install app install-app release

APP_NAME    := AgLight
APP_BUNDLE  := $(APP_NAME).app
BUILD_DIR   := build
ICONSET_DIR := $(BUILD_DIR)/AppIcon.iconset
VERSION     := $(shell plutil -extract CFBundleShortVersionString raw Info.plist)
export MACOSX_DEPLOYMENT_TARGET = 12.0

build:
	go build -ldflags="-s -w" -o aglight .

run: build
	./aglight

test:
	go test ./... -v

clean:
	rm -f aglight
	rm -rf $(APP_BUNDLE)
	rm -rf $(BUILD_DIR)

install: build
	cp aglight /usr/local/bin/

# ── macOS .app bundle (local dev) ──────────────────────────

app: $(APP_BUNDLE)

$(APP_BUNDLE): $(BUILD_DIR)/aglight $(BUILD_DIR)/AppIcon.icns Info.plist
	@mkdir -p $(APP_BUNDLE)/Contents/MacOS
	@mkdir -p $(APP_BUNDLE)/Contents/Resources
	cp $(BUILD_DIR)/aglight $(APP_BUNDLE)/Contents/MacOS/aglight
	cp $(BUILD_DIR)/AppIcon.icns $(APP_BUNDLE)/Contents/Resources/AppIcon.icns
	cp Info.plist $(APP_BUNDLE)/Contents/Info.plist
	@printf 'APPL????' > $(APP_BUNDLE)/Contents/PkgInfo

$(BUILD_DIR)/aglight:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/aglight .

$(BUILD_DIR)/AppIcon.icns: $(ICONSET_DIR)/icon_512x512@2x.png
	iconutil -c icns -o $(BUILD_DIR)/AppIcon.icns $(ICONSET_DIR)

$(ICONSET_DIR)/icon_512x512@2x.png: $(BUILD_DIR)/icon_master.png
	@mkdir -p $(ICONSET_DIR)
	sips -z 16 16     $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_16x16.png -s format png >/dev/null
	sips -z 32 32     $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_16x16@2x.png -s format png >/dev/null
	sips -z 32 32     $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_32x32.png -s format png >/dev/null
	sips -z 64 64     $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_32x32@2x.png -s format png >/dev/null
	sips -z 128 128   $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_128x128.png -s format png >/dev/null
	sips -z 256 256   $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_128x128@2x.png -s format png >/dev/null
	sips -z 256 256   $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_256x256.png -s format png >/dev/null
	sips -z 512 512   $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_256x256@2x.png -s format png >/dev/null
	sips -z 512 512   $(BUILD_DIR)/icon_master.png --out $(ICONSET_DIR)/icon_512x512.png -s format png >/dev/null
	cp $(BUILD_DIR)/icon_master.png $(ICONSET_DIR)/icon_512x512@2x.png

$(BUILD_DIR)/icon_master.png:
	@mkdir -p $(BUILD_DIR)
	go run ./cmd/icongen -o $(BUILD_DIR)/icon_master.png

install-app: app
	cp -R $(APP_BUNDLE) /Applications/

# ── Release ────────────────────────────────────────────────
# Usage:
#   make release             # bump patch  (0.1.0 → 0.1.1)
#   make release BUMP=minor  # bump minor  (0.1.0 → 0.2.0)
#   make release BUMP=major  # bump major  (0.1.0 → 1.0.0)

BUMP ?= patch

release:
	@if [ -n "$$(git status --porcelain)" ]; then echo "error: working tree not clean"; exit 1; fi
	@NEW_VERSION=$$(python3 -c "\
v = '$(VERSION)'.split('.'); \
i = {'major':0,'minor':1,'patch':2}['$(BUMP)']; \
v[i] = str(int(v[i])+1); \
[v.__setitem__(j,'0') for j in range(i+1,3)]; \
print('.'.join(v))"); \
	echo "Release $(VERSION) → $$NEW_VERSION"; \
	plutil -replace CFBundleShortVersionString -string $$NEW_VERSION Info.plist; \
	plutil -replace CFBundleVersion -string $$NEW_VERSION Info.plist; \
	git add Info.plist; \
	git commit -m "release: v$$NEW_VERSION"; \
	git tag "v$$NEW_VERSION"; \
	git push origin master "v$$NEW_VERSION"; \
	echo "Released v$$NEW_VERSION — CI will build binaries and create GitHub Release"
