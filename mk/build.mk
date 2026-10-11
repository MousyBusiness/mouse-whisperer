include mk/common.mk

# `all`, not `build`: a target named build would collide with the build/
# directory prerequisite and make would flag circular dependencies.
.PHONY: all app helper screend space-helper overlay-app check-features

# screend and space-helper only serve the screend and workspace features;
# the overlay is skipped (with a note) on machines without Flutter
all: app $(if $(call enabled,screend),screend) $(if $(call enabled,workspace),space-helper) overlay-app

app: | $(BUILD_DIR)
	@echo "Building app$(if $(DISABLED), (without: $(DISABLED)))"
ifeq ($(UNAME_S),Darwin)
	@$(CGO_ENV) go build -tags '$(GO_TAGS)' $(BUILD_FLAGS) -ldflags "-extldflags '$(EXT_LDFLAGS)'" -o $(BUILD_DIR)/app ./cmd/app
else
	@go build -tags '$(GO_TAGS)' $(BUILD_FLAGS) -o $(BUILD_DIR)/main ./cmd/app
endif
	@# ./run reads this to skip services for compiled-out features
	@echo "$(DISABLED)" > $(BUILD_DIR)/features

helper: require-swiftc | $(BUILD_DIR)
	@echo "Building screend-helper"
	@swiftc -O -o $(BUILD_DIR)/screend-helper native/screend-helper/*.swift

screend: helper | $(BUILD_DIR)
	@echo "Building screend"
	@go build $(BUILD_FLAGS) -o $(BUILD_DIR)/screend ./cmd/screend

# Release build of the Flutter overlay, copied to build/overlay.app for
# ./run to launch (--dev keeps `flutter run` for hot reload)
overlay-app: | $(BUILD_DIR)
ifeq ($(shell command -v flutter),)
	@echo "Skipping overlay (flutter not installed)"
else
	@echo "Building overlay"
	@cd _/overlay && flutter build macos --release
	@rm -rf $(BUILD_DIR)/overlay.app
	@cp -R _/overlay/build/macos/Build/Products/Release/mouse_whisperer.app $(BUILD_DIR)/overlay.app
endif

space-helper: require-swiftc | $(BUILD_DIR)
	@echo "Building space-helper"
	@swiftc -O -o $(BUILD_DIR)/space-helper native/space-helper/main.swift

# Compile and vet the full build plus every feature off (alone and all
# together), so a stub can't silently drift from the code it stands in for
check-features:
	@echo "Checking feature build combinations"
	@$(CGO_ENV) go vet ./...
	@for f in $(FEATURES); do \
		echo "  no$$f"; \
		$(CGO_ENV) go vet -tags no$$f ./... || exit 1; \
	done
	@for tags in '$(addprefix no,$(LITE_DISABLE))' '$(addprefix no,$(FEATURES))'; do \
		echo "  $$tags"; \
		$(CGO_ENV) go vet -tags "$$tags" ./... || exit 1; \
		out=$$($(CGO_ENV) go test -tags "$$tags" ./... 2>&1) || { echo "$$out"; exit 1; }; \
	done
	@echo "ok"
