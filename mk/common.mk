include mk/features.mk

UNAME_S := $(shell uname -s)
BUILD_DIR := build

# Same CGO environment ./run exports: gosseract needs leptonica/tesseract
# via homebrew pkg-config, portaudio needs the include/lib paths.
CGO_ENV := PKG_CONFIG_PATH="/opt/homebrew/lib/pkgconfig:$$PKG_CONFIG_PATH" \
           CGO_CPPFLAGS="-I/opt/homebrew/include" \
           CGO_LDFLAGS="-L/opt/homebrew/lib"

ifeq ($(UNAME_S),Darwin)
EXT_LDFLAGS := -framework Foundation -framework Metal -framework MetalKit
endif

$(BUILD_DIR):
	@install -d $(BUILD_DIR)

require-swiftc:
	@command -v swiftc >/dev/null || { echo "swiftc not found - install Xcode Command Line Tools"; exit 1; }

.PHONY: require-swiftc
