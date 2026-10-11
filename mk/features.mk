# Optional modules - all built by default. DISABLE (on the make command line,
# or persisted in the untracked mk/local.mk) lists the ones to leave out.
# Each name maps to a no<name> Go build tag, see internal/features.
FEATURES := voice eye ocr screend companion ai workspace

# Lite profile (make bootstrap-lite / build-lite): eye tracking with OpenCV
# snap-to-element and local OCR only
LITE_DISABLE := voice screend companion ai workspace

-include mk/local.mk
DISABLE ?=

UNKNOWN_FEATURES := $(filter-out $(FEATURES),$(DISABLE))
ifneq ($(UNKNOWN_FEATURES),)
$(error unknown DISABLE feature(s): $(UNKNOWN_FEATURES) - valid: $(FEATURES))
endif

DISABLED := $(sort $(DISABLE))
GO_TAGS := $(addprefix no,$(DISABLED))

# $(call enabled,<feature>) expands to 1 when the feature is built in
enabled = $(if $(filter $(1),$(DISABLED)),,1)
