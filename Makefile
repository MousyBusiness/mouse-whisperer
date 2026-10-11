# Entry points only - the implementation lives in mk/*.mk.
# Development commands (ps, kill, debug, generate, ...): make -f mk/dev.mk help

include mk/features.mk

.DEFAULT_GOAL := help

.PHONY: help bootstrap build run bootstrap-lite build-lite

help:
	@echo "Mouse Whisperer"
	@echo ""
	@echo "  make bootstrap    one-time setup: checks the machine, confirms installs"
	@echo "  make build        build everything: app, screend + helper, space-helper, overlay"
	@echo "  make run          start with the overlay panel"
	@echo ""
	@echo "  make bootstrap-lite | build-lite"
	@echo "                    lite profile: eye tracking + local OCR only (no AI, voice, companion)"
	@echo ""
	@echo "  Leave optional modules out: make bootstrap|build DISABLE=\"<modules>\""
	@echo "  (or persist DISABLE = <modules> in mk/local.mk). Modules: $(FEATURES)"

bootstrap:
	@DISABLE="$(DISABLED)" ./mk/bootstrap.sh

build:
	@$(MAKE) -f mk/build.mk all

# Lite profile (LITE_DISABLE in mk/features.mk): no LLM commands, no mic or
# whisper, no companion server. Plain ./run starts this build/app; make run
# (--dev) compiles from source with the DISABLE set instead.
bootstrap-lite:
	@DISABLE="$(LITE_DISABLE)" ./mk/bootstrap.sh

build-lite:
	@$(MAKE) -f mk/build.mk all DISABLE="$(LITE_DISABLE)"

run:
	@./run --dev
