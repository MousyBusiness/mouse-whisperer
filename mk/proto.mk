.PHONY: generate clean

# clean deletes committed gen/ first - a failed generate needs
# `git checkout -- gen/` to restore it.
clean:
	@rm -rf gen/mouse_whisperer
	@rm -rf _/overlay/lib/gen/mouse_whisperer _/companion/lib/gen/mouse_whisperer

generate: clean
	@buf lint
	@buf generate
