#### HOW IT WORKS

The mouse_whisperer_talon_plugin folder goes in ~/.talon/user. This creates a networked connection to Talon mouse events (Getting the Tobii events was too hard).

`make bootstrap` sets this up for a Tobii Eye Tracker 5: it checks for Talon and
a connected tracker, then (on a yes) symlinks the folder to
`~/.talon/user/mouse_whisperer`, so `git pull` updates the plugin. Older copies
of the plugin elsewhere in `~/.talon/user` are moved to
`~/.talon/mouse_whisperer_backup_<timestamp>` so Talon doesn't load them twice.
Restart Talon afterwards and say "control mouse" to toggle gaze control.
`plugin/mouse/mouse.py` doesn't copy Talon's eye mouse: it swaps the `ctrl`
object Talon's built-in `eye_mouse` plugin moves the cursor with, so Talon's
own gaze filtering runs unchanged and each cursor move is sent to the app
(port 50055) instead. Restart Talon after updating from a version that
replaced `EyeMouse.on_gaze`.

#### LICENSE

The `edit/` actions, the mouse actions in `plugin/mouse/mouse.py` and
`HiddenCursor.cur` are adapted from [talonhub/community](https://github.com/talonhub/community),
MIT licensed — see `mouse_whisperer_talon_plugin/LICENSE-talonhub-community`.
