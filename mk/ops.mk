# Matches all project processes regardless of which session spawned them:
# whisper servers, screend + helper, prod app, overlay app + its dart dev
# service, and dev-mode `go run` (parent and compiled child). Brackets stop
# the pattern matching the shell that carries it.
PROC_PATTERN := [b]uild/server --model|[b]uild/screend|[b]uild/app|[m]ouse_whisperer|[g]o run .*cmd/app|[g]o-build.*/exe/app

.PHONY: ps kill

# Show who is holding the app ports and any stale project processes.
# 51010/51011 whisper tiny/medium, 51012 overlay, 51013 screend.
ps:
	@echo "== Ports 51010-51013 (whisper tiny/medium, overlay, screend) =="
	@lsof -nP -iTCP:51010-51013 || echo "nothing on ports"
	@echo ""
	@echo "== Project processes =="
	@pgrep -fl '$(PROC_PATTERN)' || echo "none running"

# Terminate everything `make ps` finds, including orphans from dead sessions.
kill:
	@pkill -f '$(PROC_PATTERN)' && echo "terminated" || echo "nothing to kill"
	@sleep 1
	@pkill -9 -f '$(PROC_PATTERN)' 2>/dev/null && echo "sent SIGKILL to survivors" || true
