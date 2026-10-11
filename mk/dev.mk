# Development commands: make -f mk/dev.mk <target>

include mk/build.mk   # app, helper, screend, space-helper (individual builds)
include mk/ops.mk     # ps, kill
include mk/proto.mk   # generate, clean

.DEFAULT_GOAL := help

.PHONY: help debug overlay

help:
	@echo "Development commands (make -f mk/dev.mk <target>)"
	@echo ""
	@echo "  debug         run with rolling CSV logs under /tmp/mousewhisperer"
	@echo "  ps            show project processes and ports"
	@echo "  kill          stop everything, including orphans"
	@echo "  generate      regenerate protobuf bindings (clean deletes gen/ first)"
	@echo "  overlay       run the Flutter overlay standalone"
	@echo "  app | screend | helper | space-helper | overlay-app"
	@echo "                build a single artifact"
	@echo "  check-features  vet each module off, vet+test the lite profile and all off"

debug:
	@./run --dev --debug

overlay:
	@(cd _/overlay/ && flutter run -d macos)
