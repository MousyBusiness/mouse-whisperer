from typing import Optional, cast
import math
import os
import queue
import threading
import time
import warnings

from talon import actions, app, canvas, cron, ctrl, ui, metrics, skia
from talon.api import ffi, lib
from talon.types import Point2d, Point3d

from talon.track import tobii

from talon import Module, clip, imgui, noise
from talon_plugins import eye_zoom_mouse, eye_mouse

key = actions.key
self = actions.self
scroll_amount = 0
click_job = None
scroll_job = None
gaze_job = None
cancel_scroll_on_pop = True
control_mouse_forced = False

default_cursor = {
    "AppStarting": r"%SystemRoot%\Cursors\aero_working.ani",
    "Arrow": r"%SystemRoot%\Cursors\aero_arrow.cur",
    "Hand": r"%SystemRoot%\Cursors\aero_link.cur",
    "Help": r"%SystemRoot%\Cursors\aero_helpsel.cur",
    "No": r"%SystemRoot%\Cursors\aero_unavail.cur",
    "NWPen": r"%SystemRoot%\Cursors\aero_pen.cur",
    "Person": r"%SystemRoot%\Cursors\aero_person.cur",
    "Pin": r"%SystemRoot%\Cursors\aero_pin.cur",
    "SizeAll": r"%SystemRoot%\Cursors\aero_move.cur",
    "SizeNESW": r"%SystemRoot%\Cursors\aero_nesw.cur",
    "SizeNS": r"%SystemRoot%\Cursors\aero_ns.cur",
    "SizeNWSE": r"%SystemRoot%\Cursors\aero_nwse.cur",
    "SizeWE": r"%SystemRoot%\Cursors\aero_ew.cur",
    "UpArrow": r"%SystemRoot%\Cursors\aero_up.cur",
    "Wait": r"%SystemRoot%\Cursors\aero_busy.ani",
    "Crosshair": "",
    "IBeam": "",
}

# todo figure out why notepad++ still shows the cursor sometimes.
hidden_cursor = os.path.join(
    os.path.dirname(os.path.realpath(__file__)), r"Resources\HiddenCursor.cur"
)

mod = Module()
mod.list(
    "mouse_button", desc="List of mouse button words to mouse_click index parameter"
)
mod.tag(
    "mouse_cursor_commands_enable", desc="Tag enables hide/show mouse cursor commands"
)
setting_mouse_enable_pop_click = mod.setting(
    "mouse_enable_pop_click",
    type=int,
    default=0,
    desc="Enable pop to click when control mouse is enabled.",
)
setting_mouse_enable_pop_stops_scroll = mod.setting(
    "mouse_enable_pop_stops_scroll",
    type=int,
    default=0,
    desc="When enabled, pop stops continuous scroll modes (wheel upper/downer/gaze)",
)
setting_mouse_wake_hides_cursor = mod.setting(
    "mouse_wake_hides_cursor",
    type=int,
    default=0,
    desc="When enabled, mouse wake will hide the cursor. mouse_wake enables zoom mouse.",
)
setting_mouse_hide_mouse_gui = mod.setting(
    "mouse_hide_mouse_gui",
    type=int,
    default=0,
    desc="When enabled, the 'Scroll Mouse' GUI will not be shown.",
)
setting_mouse_continuous_scroll_amount = mod.setting(
    "mouse_continuous_scroll_amount",
    type=int,
    default=80,
    desc="The default amount used when scrolling continuously",
)
setting_mouse_wheel_down_amount = mod.setting(
    "mouse_wheel_down_amount",
    type=int,
    default=120,
    desc="The amount to scroll up/down (equivalent to mouse wheel on Windows by default)",
)
setting_mouse_wheel_horizontal_amount = mod.setting(
    "mouse_wheel_horizontal_amount",
    type=int,
    default=40,
    desc="The amount to scroll left/right",
)

continuous_scoll_mode = ""


@imgui.open(x=700, y=0)
def gui_wheel(gui: imgui.GUI):
    gui.text(f"Scroll mode: {continuous_scoll_mode}")
    gui.line()
    if gui.button("Wheel Stop [stop scrolling]"):
        actions.user.mouse_scroll_stop()


@mod.action_class
class Actions:
    def mouse_show_cursor():
        """Shows the cursor"""
        show_cursor_helper(True)

    def mouse_hide_cursor():
        """Hides the cursor"""
        show_cursor_helper(False)

    def mouse_wake():
        """Enable control mouse, zoom mouse, and disables cursor"""
        actions.tracking.control_zoom_toggle(True)

        if setting_mouse_wake_hides_cursor.get() >= 1:
            show_cursor_helper(False)

    def mouse_sleep():
        """Disables control mouse, zoom mouse, and re-enables cursor"""
        actions.tracking.control_zoom_toggle(False)
        actions.tracking.control_toggle(False)
        actions.tracking.control1_toggle(False)

        show_cursor_helper(True)
        stop_scroll()

        # todo: fixme temporary fix for drag command
        button_down = len(list(ctrl.mouse_buttons_down())) > 0
        if button_down:
            ctrl.mouse_click(button=0, up=True)

    def mouse_scroll_down(amount: float = 1):
        """Scrolls down"""
        mouse_scroll(amount * setting_mouse_wheel_down_amount.get())()

    def mouse_scroll_down_continuous():
        """Scrolls down continuously"""
        global continuous_scoll_mode
        continuous_scoll_mode = "scroll down continuous"
        mouse_scroll(setting_mouse_continuous_scroll_amount.get())()

        if scroll_job is None:
            start_scroll()

        if setting_mouse_hide_mouse_gui.get() == 0:
            gui_wheel.show()

    def mouse_scroll_up(amount: float = 1):
        """Scrolls up"""
        mouse_scroll(-amount * setting_mouse_wheel_down_amount.get())()

    def mouse_scroll_up_continuous():
        """Scrolls up continuously"""
        global continuous_scoll_mode
        continuous_scoll_mode = "scroll up continuous"
        mouse_scroll(-setting_mouse_continuous_scroll_amount.get())()

        if scroll_job is None:
            start_scroll()
        if setting_mouse_hide_mouse_gui.get() == 0:
            gui_wheel.show()

    def mouse_scroll_left(amount: float = 1):
        """Scrolls left"""
        actions.mouse_scroll(0, -amount * setting_mouse_wheel_horizontal_amount.get())

    def mouse_scroll_right(amount: float = 1):
        """Scrolls right"""
        actions.mouse_scroll(0, amount * setting_mouse_wheel_horizontal_amount.get())

    def mouse_scroll_stop():
        """Stops scrolling"""
        stop_scroll()

    def mouse_gaze_scroll():
        """Starts gaze scroll"""
        global continuous_scoll_mode
        continuous_scoll_mode = "gaze scroll"

        start_cursor_scrolling()
        if setting_mouse_hide_mouse_gui.get() == 0:
            gui_wheel.show()

        # enable 'control mouse' if eye tracker is present and not enabled already
        global control_mouse_forced
        if not actions.tracking.control_enabled():
            actions.tracking.control_toggle(True)
            control_mouse_forced = True

    def copy_mouse_position():
        """Copy the current mouse position coordinates"""
        position = ctrl.mouse_pos()
        clip.set_text(repr(position))

    def mouse_move_center_active_window():
        """move the mouse cursor to the center of the currently active window"""
        rect = ui.active_window().rect
        ctrl.mouse_move(rect.left + (rect.width / 2), rect.top + (rect.height / 2))


def show_cursor_helper(show):
    """Show/hide the cursor"""
    if app.platform == "windows":
        import ctypes
        import winreg

        import win32con

        try:
            Registrykey = winreg.OpenKey(
                winreg.HKEY_CURRENT_USER, r"Control Panel\Cursors", 0, winreg.KEY_WRITE
            )

            for value_name, value in default_cursor.items():
                if show:
                    winreg.SetValueEx(
                        Registrykey, value_name, 0, winreg.REG_EXPAND_SZ, value
                    )
                else:
                    winreg.SetValueEx(
                        Registrykey, value_name, 0, winreg.REG_EXPAND_SZ, hidden_cursor
                    )

            winreg.CloseKey(Registrykey)

            ctypes.windll.user32.SystemParametersInfoA(
                win32con.SPI_SETCURSORS, 0, None, 0
            )

        except OSError:
            print(f"Unable to show_cursor({str(show)})")
    else:
        ctrl.cursor_visible(show)


def on_pop(active):
    if setting_mouse_enable_pop_stops_scroll.get() >= 1 and (gaze_job or scroll_job):
        stop_scroll()
    elif not actions.tracking.control_zoom_enabled():
        if setting_mouse_enable_pop_click.get() >= 1:
            ctrl.mouse_click(button=0, hold=16000)


noise.register("pop", on_pop)


def mouse_scroll(amount):
    def scroll():
        global scroll_amount
        if continuous_scoll_mode:
            if (scroll_amount >= 0) == (amount >= 0):
                scroll_amount += amount
            else:
                scroll_amount = amount
        actions.mouse_scroll(y=int(amount))

    return scroll


def scroll_continuous_helper():
    global scroll_amount
    # print("scroll_continuous_helper")
    if scroll_amount and (eye_zoom_mouse.zoom_mouse.state == eye_zoom_mouse.STATE_IDLE):
        actions.mouse_scroll(by_lines=False, y=int(scroll_amount / 10))


def start_scroll():
    global scroll_job
    scroll_job = cron.interval("60ms", scroll_continuous_helper)


def gaze_scroll():
    # print("gaze_scroll")
    if (
        eye_zoom_mouse.zoom_mouse.state == eye_zoom_mouse.STATE_IDLE
    ):  # or eye_zoom_mouse.zoom_mouse.state == eye_zoom_mouse.STATE_SLEEP:
        x, y = ctrl.mouse_pos()

        # the rect for the window containing the mouse
        rect = None

        # on windows, check the active_window first since ui.windows() is not z-ordered
        if app.platform == "windows" and ui.active_window().rect.contains(x, y):
            rect = ui.active_window().rect
        else:
            windows = ui.windows()
            for w in windows:
                if w.rect.contains(x, y):
                    rect = w.rect
                    break

        if rect is None:
            # print("no window found!")
            return

        midpoint = rect.y + rect.height / 2
        amount = int(((y - midpoint) / (rect.height / 10)) ** 3)
        actions.mouse_scroll(by_lines=False, y=amount)

    # print(f"gaze_scroll: {midpoint} {rect.height} {amount}")


def stop_scroll():
    global scroll_amount, scroll_job, gaze_job, continuous_scoll_mode
    scroll_amount = 0
    if scroll_job:
        cron.cancel(scroll_job)

    if gaze_job:
        cron.cancel(gaze_job)

    global control_mouse_forced
    if control_mouse_forced:
        actions.tracking.control_toggle(False)
        control_mouse_forced = False

    scroll_job = None
    gaze_job = None
    gui_wheel.hide()

    continuous_scoll_mode = ""


def start_cursor_scrolling():
    global scroll_job, gaze_job
    stop_scroll()
    gaze_job = cron.interval("60ms", gaze_scroll)


import socket
import sys
import threading
import types

GAZE_ADDR = ("127.0.0.1", 50055)

# Talon re-imports this module on every edit and when it restarts its
# plugins, but the server thread outlives the module. Shared state survives
# the reload so the new module can close the old listener and client (ending
# the old thread) instead of failing to bind, while its own on_gaze would
# send to nobody.
_gaze = sys.modules.setdefault("_mouse_whisperer_gaze", types.ModuleType("_mouse_whisperer_gaze"))
_gaze.generation = getattr(_gaze, "generation", 0) + 1
for _old in (getattr(_gaze, "sock", None), getattr(_gaze, "conn", None)):
    if _old is not None:
        try:
            # shutdown, not just close: close alone doesn't wake a blocked
            # accept() on macOS
            _old.shutdown(socket.SHUT_RDWR)
        except OSError:
            pass
        _old.close()
_gaze.sock = _gaze.conn = None

conn = None


def process(generation):
    # SO_REUSEADDR: the app's connection to a previous Talon run leaves the
    # port in TIME_WAIT, which otherwise fails the bind with EADDRINUSE
    while True:
        if _gaze.generation != generation:
            return
        sock = socket.socket()
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            sock.bind(GAZE_ADDR)
            break
        except OSError as e:
            sock.close()
            print("Gaze server: port {} busy ({}), retrying".format(GAZE_ADDR[1], e))
            time.sleep(1)
    sock.listen(1)
    _gaze.sock = sock

    global conn
    while True:
        print("Waiting on connection")
        try:
            conn, addr = sock.accept()
        except OSError:
            print("Gaze server: listener closed")
            return
        _gaze.conn = conn
        print("Client connected ", addr)


thread = threading.Thread(target=process, args=(_gaze.generation,))
thread.daemon = True
thread.start()


class GazeCtrl:
    """Stands in for talon's ctrl inside talon_plugins.eye_mouse, so Talon's
    own gaze filtering runs unchanged but its cursor moves go to the
    mouse-whisperer app (which snaps to on-screen elements) instead of the
    OS. Everything not overridden falls through to the real ctrl."""

    def __init__(self, real):
        self._real = real
        self._last = None

    def __getattr__(self, name):
        return getattr(self._real, name)

    def mouse_move(self, x, y, dx=0, dy=0):
        global conn
        self._last = (x, y)
        if conn is None:
            return
        try:
            conn.send("x: {}, y: {}, dx: {}, dy: {}\n".format(x, y, dx, dy).encode())
        except OSError:
            # client went away; the accept loop hands over the next one
            conn = None

    def mouse_pos(self):
        # Report where gaze last put the cursor: eye_mouse pauses itself when
        # the real pointer drifts from that (it assumes the user grabbed the
        # mouse), and the app's own cursor moves would trip that constantly
        if self._last is not None:
            return self._last
        return self._real.mouse_pos()


# On a reload eye_mouse.ctrl is the previous module's GazeCtrl: unwrap it
eye_mouse.ctrl = GazeCtrl(getattr(eye_mouse.ctrl, "_real", eye_mouse.ctrl))
