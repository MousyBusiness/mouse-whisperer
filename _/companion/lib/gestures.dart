import 'dart:async';

import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:mouse_companion/simulation.dart';
import 'package:mouse_companion/gen/mouse_whisperer/companion/v1/companion.pb.dart'
    as pb;
import 'package:uuid/uuid.dart';

const _uuid = Uuid();

class Gestures extends StatefulWidget {
  const Gestures({super.key, required this.onMouseEvent, required this.child});
  final void Function(pb.MouseEvent event, String eventGroupId) onMouseEvent;
  final Widget child;

  @override
  _GesturesState createState() => _GesturesState();
}

class _GesturesState extends State<Gestures> {
  final GlobalKey<RawGestureDetectorState> _gestureDetectorKey =
      GlobalKey<RawGestureDetectorState>();

  // Track active pointers by their ID
  final Map<int, Offset> _activePointers = {};
  int get _pointerCount => _activePointers.length;

  // Block new gestures until all pointers are lifted
  bool _waitingForAllPointersUp = false;

  bool _isBlocked() => _waitingForAllPointersUp;

  final _dragStartBehavior = DragStartBehavior.start;
  late final _verticalDrag = _Controller(
    _gestureDetectorKey,
    "vertical",
    _emitMouseEvent,
    () => _pointerCount,
    _isBlocked,
    Axis.vertical,
  );
  late final _horizontalDrag = _Controller(
    _gestureDetectorKey,
    "horizontal",
    _emitMouseEvent,
    () => _pointerCount,
    _isBlocked,
    Axis.horizontal,
  );
  late final _panDrag = _Controller(
    _gestureDetectorKey,
    "pan",
    _emitMouseEvent,
    () => _pointerCount,
    _isBlocked,
    null,
    dwellDuration: const Duration(milliseconds: 700),
  );

  void _emitMouseEvent(pb.MouseEvent event, String eventGroupId) {
    if (!event.hasPointerCount()) {
      event.pointerCount = _pointerCount;
    }
    widget.onMouseEvent(event, eventGroupId);
  }

  void _emitClickEvent(int pointerCount) {
    HapticFeedback.lightImpact();

    _emitMouseEvent(
      pb.MouseEvent(
        type: pb.MouseEvent_EventType.EVENT_TYPE_CLICK,
        pointerCount: pointerCount,
        pointerSequence: pointerCount, // Use count as sequence for now
      ),
      _uuid.v4(),
    );
  }

  void _emitDoubleClickEvent() {
    HapticFeedback.mediumImpact();

    _emitMouseEvent(
      pb.MouseEvent(
        type: pb.MouseEvent_EventType.EVENT_TYPE_CLICK,
        pointerCount: 1,
        pointerSequence: 2,
      ),
      _uuid.v4(),
    );
  }

  void _handlePointerDown(PointerDownEvent event) {
    _activePointers[event.pointer] = event.position;
    // debugPrint('pointer down: ${event.pointer} total: $_pointerCount');
  }

  void _handlePointerUp(PointerUpEvent event) {
    _activePointers.remove(event.pointer);
    final wasWaiting = _waitingForAllPointersUp;
    if (_pointerCount == 0) {
      _waitingForAllPointersUp = false;
      if (wasWaiting) {
        _emitMouseEvent(
          pb.MouseEvent(type: pb.MouseEvent_EventType.EVENT_TYPE_FINISH),
          _uuid.v4(),
        );
      }
    } else {
      _waitingForAllPointersUp = true;
    }
    // debugPrint('pointer up: ${event.pointer} total: $_pointerCount blocked: $_waitingForAllPointersUp');
  }

  void _handlePointerCancel(PointerCancelEvent event) {
    _activePointers.remove(event.pointer);
    final wasWaiting = _waitingForAllPointersUp;
    if (_pointerCount == 0) {
      _waitingForAllPointersUp = false;
      if (wasWaiting) {
        _emitMouseEvent(
          pb.MouseEvent(type: pb.MouseEvent_EventType.EVENT_TYPE_FINISH),
          _uuid.v4(),
        );
      }
    } else {
      _waitingForAllPointersUp = true;
    }
    // debugPrint('pointer cancel: ${event.pointer} total: $_pointerCount blocked: $_waitingForAllPointersUp');
  }

  @override
  Widget build(BuildContext context) {
    return Listener(
      onPointerDown: _handlePointerDown,
      onPointerUp: _handlePointerUp,
      onPointerCancel: _handlePointerCancel,
      child: RawGestureDetector(
        key: _gestureDetectorKey,
        gestures: {
          ConcurrentTapGestureRecognizer:
              GestureRecognizerFactoryWithHandlers<
                ConcurrentTapGestureRecognizer
              >(
                () => ConcurrentTapGestureRecognizer(
                  supportedDevices: {
                    PointerDeviceKind.touch,
                    PointerDeviceKind.stylus,
                  },
                ),
                (ConcurrentTapGestureRecognizer instance) {
                  instance.onClick = (int count) {
                    _emitClickEvent(count);
                  };
                  instance.onDoubleClick = () {
                    _emitDoubleClickEvent();
                  };
                  instance.supportedDevices = {
                    PointerDeviceKind.touch,
                    PointerDeviceKind.stylus,
                  };
                  instance.gestureSettings = const DeviceGestureSettings(
                    touchSlop: kTouchSlop,
                  );
                },
              ),
          SingleTouchPanGestureRecognizer:
              GestureRecognizerFactoryWithHandlers<
                SingleTouchPanGestureRecognizer
              >(
                () => SingleTouchPanGestureRecognizer(
                  pointerCountGetter: () => _pointerCount,
                  supportedDevices: {
                    PointerDeviceKind.touch,
                    PointerDeviceKind.stylus,
                  },
                ),
                (SingleTouchPanGestureRecognizer instance) {
                  instance
                    ..onDown = _panDrag._handleDragDown
                    ..onStart = _panDrag._handleDragStart
                    ..onUpdate = _panDrag._handleDragUpdate
                    ..onEnd = _panDrag._handleDragEnd
                    ..onCancel = _panDrag._handleDragCancel
                    ..minFlingDistance = kTouchSlop
                    ..minFlingVelocity = kMinFlingVelocity
                    ..maxFlingVelocity = kMaxFlingVelocity
                    ..velocityTrackerBuilder = _panDrag._velocityTracker
                    ..dragStartBehavior = _dragStartBehavior
                    ..multitouchDragStrategy =
                        MultitouchDragStrategy.latestPointer
                    ..supportedDevices = {
                      PointerDeviceKind.touch,
                      PointerDeviceKind.stylus,
                    }
                    ..gestureSettings = const DeviceGestureSettings(
                      touchSlop: kTouchSlop / 3,
                    );
                },
              ),
          MultitouchVerticalDragGestureRecognizer:
              GestureRecognizerFactoryWithHandlers<
                MultitouchVerticalDragGestureRecognizer
              >(
                () => MultitouchVerticalDragGestureRecognizer(
                  pointerCountGetter: () => _pointerCount,
                  supportedDevices: {PointerDeviceKind.touch},
                ),
                (MultitouchVerticalDragGestureRecognizer instance) {
                  instance
                    ..onDown = _verticalDrag._handleDragDown
                    ..onStart = _verticalDrag._handleDragStart
                    ..onUpdate = _verticalDrag._handleDragUpdate
                    ..onEnd = _verticalDrag._handleDragEnd
                    ..onCancel = _verticalDrag._handleDragCancel
                    ..minFlingDistance = kTouchSlop * 3
                    ..minFlingVelocity = kMinFlingVelocity
                    ..maxFlingVelocity = kMaxFlingVelocity
                    ..velocityTrackerBuilder = _verticalDrag._velocityTracker
                    ..dragStartBehavior = _dragStartBehavior
                    ..multitouchDragStrategy =
                        MultitouchDragStrategy.latestPointer;
                },
              ),
          MultitouchHorizontalDragGestureRecognizer:
              GestureRecognizerFactoryWithHandlers<
                MultitouchHorizontalDragGestureRecognizer
              >(
                () => MultitouchHorizontalDragGestureRecognizer(
                  pointerCountGetter: () => _pointerCount,
                  supportedDevices: {PointerDeviceKind.touch},
                ),
                (MultitouchHorizontalDragGestureRecognizer instance) {
                  instance
                    ..onDown = _horizontalDrag._handleDragDown
                    ..onStart = _horizontalDrag._handleDragStart
                    ..onUpdate = _horizontalDrag._handleDragUpdate
                    ..onEnd = _horizontalDrag._handleDragEnd
                    ..onCancel = _horizontalDrag._handleDragCancel
                    ..minFlingDistance = kTouchSlop * 3
                    ..minFlingVelocity = kMinFlingVelocity
                    ..maxFlingVelocity = kMaxFlingVelocity
                    ..velocityTrackerBuilder = _horizontalDrag._velocityTracker
                    ..dragStartBehavior = _dragStartBehavior
                    ..multitouchDragStrategy =
                        MultitouchDragStrategy.latestPointer;
                },
              ),
        },
        child: widget.child,
      ),
    );
  }
}

class _Controller {
  _Controller(
    this.key,
    this.debugLabel,
    this.onMouseEvent,
    this.getPointerCount,
    this.isBlocked,
    this.axis, {
    this.dwellDuration,
  });
  final GlobalKey<RawGestureDetectorState> key;
  final String debugLabel;
  final void Function(pb.MouseEvent, String eventGroupId) onMouseEvent;
  final int Function() getPointerCount;
  final bool Function() isBlocked;
  final Duration? dwellDuration; // If set, enables dwell-and-drag detection
  final Axis? axis;
  _Drag? _drag;
  _Hold? _hold;
  Offset? _lastPosition;
  int _dragPointerCount = 1; // Captured at drag start for use during fling
  String _eventGroupId =
      ''; // Generated at drag start, used for all events in this gesture
  bool _suppressed = false; // Track if this gesture was suppressed
  Timer? _dwellTimer; // Timer for dwell detection
  bool _dwellCompleted =
      false; // True if dwell timer fired (user held still long enough)

  final _velocitySimulation = VelocitySimulation();
  VelocityTracker _velocityTracker(PointerEvent event) {
    return _velocitySimulation.generateTracker(event);
  }

  _Position get position => _position!;

  _Position? _position;

  void _handleDragDown(DragDownDetails details) {
    assert(_drag == null);
    assert(_hold == null);

    // Suppress new gestures during cooldown (e.g., when transitioning from 2-finger to 1-finger)
    if (isBlocked()) {
      _suppressed = true;
      // debugPrint('$debugLabel drag down SUPPRESSED (waiting for all pointers up)');
      return;
    }
    _suppressed = false;

    _velocitySimulation.reset();
    _lastPosition = details.globalPosition;
    _dragPointerCount = getPointerCount(); // Reset pointer count at drag down
    _eventGroupId = _uuid.v4(); // Generate new event group ID for this gesture
    _position ??= _Position(details);
    _hold = position.hold(_disposeHold);

    // Start dwell detection if enabled
    if (dwellDuration != null) {
      _dwellCompleted = false;
      _dwellTimer?.cancel();
      _dwellTimer = Timer(dwellDuration!, () {
        // Discard the long press if the user has moved too far
        if (_hold != null) {
          final distance = (details.globalPosition - _lastPosition!).distance;
          final threshold = kTouchSlop / 3;
          if (distance > threshold) {
            _dwellTimer?.cancel();
            return;
          }
        } else {
          return;
        }
        _dwellCompleted = true;
        // debugPrint(
        //   '$debugLabel DWELL COMPLETED - waiting for drag start, eventGroupId: $_eventGroupId, pointerCount: $_dragPointerCount',
        // );
        HapticFeedback.heavyImpact();
        onMouseEvent(
          pb.MouseEvent(
            type: pb.MouseEvent_EventType.EVENT_TYPE_LONG_PRESS,
            // pointerCount: _dragPointerCount, - we don't know the pointer count at this point
          ),
          _eventGroupId,
        );
      });
    }

    // debugPrint('$debugLabel drag down: $details pointers: $_dragPointerCount eventGroupId: $_eventGroupId');
  }

  void _handleDragStart(DragStartDetails details) {
    // Skip if this gesture was suppressed by cooldown
    if (_suppressed) {
      // debugPrint('$debugLabel drag start SUPPRESSED (waiting for all pointers up)');
      return;
    }

    // Cancel dwell timer - drag is starting
    _dwellTimer?.cancel();

    // If dwell completed before drag started, emit long press (mouse button down)
    // if (_dwellCompleted) {
    //   debugPrint('$debugLabel DWELL DRAG START - emitting long press');
    // }

    // It's possible for _hold to become null between _handleDragDown and
    // _handleDragStart, for example if some user code calls jumpTo or otherwise
    // triggers a new activity to begin.
    assert(_drag == null);
    _lastPosition = details.globalPosition;
    _dragPointerCount =
        getPointerCount(); // Capture pointer count at drag start
    _drag = position.drag(details, _disposeDrag);
    assert(_drag != null);
    // _hold might be non-null if the scroll position is currently animating.
    if (_hold != null) {
      _disposeHold();
    }

    // debugPrint(
    //   '$debugLabel drag start: $details pointers: $_dragPointerCount isDwellDrag: $_dwellCompleted eventGroupId: $_eventGroupId',
    // );
  }

  void _handleDragUpdate(DragUpdateDetails details) {
    // Skip if this gesture was suppressed by cooldown
    if (_suppressed) return;

    // _drag might be null if the drag activity ended and called _disposeDrag.
    assert(_hold == null || _drag == null);
    _drag?.update(details);

    // Record velocity sample for average calculation
    _velocitySimulation.recordVelocitySample();

    final delta = details.delta;
    final deltaX = delta.dx;
    final deltaY = delta.dy;

    // Skip zero-delta events (they carry no useful information)
    if (deltaX == 0 && deltaY == 0) return;

    // Calculate velocity-based multiplier (1.0 at low velocity, up to 1.25 at high velocity)
    final velocityMagnitude = _velocitySimulation.velocity.distance;
    const minVelocity = 100.0;
    const maxVelocity = 3000.0;
    const minMultiplier = 0.0;
    const maxMultiplier = 1;
    final velocityMultiplier =
        (minMultiplier +
                (maxMultiplier - minMultiplier) *
                    ((velocityMagnitude - minVelocity) /
                        (maxVelocity - minVelocity)))
            .clamp(minMultiplier, maxMultiplier);

    final curve = Curves.easeInQuad;
    var dx = deltaX + curve.transform(velocityMultiplier.toDouble()) * deltaX;
    var dy = deltaY + curve.transform(velocityMultiplier.toDouble()) * deltaY;

    // Zero out non-relevant axis for multitouch gestures
    if (axis == Axis.vertical) {
      dx = 0.0;
    } else if (axis == Axis.horizontal) {
      dy = 0.0;
    }

    onMouseEvent(
      pb.MouseEvent(
        type: pb.MouseEvent_EventType.EVENT_TYPE_DRAG,
        pointerCount: _dragPointerCount,
        deltaX: dx,
        deltaY: dy,
      ),
      _eventGroupId,
    );
  }

  void _handleDragEnd(DragEndDetails details) {
    // Clean up dwell timer
    _dwellTimer?.cancel();
    final wasDwellDrag = _dwellCompleted;
    _dwellCompleted = false;

    // Skip if this gesture was suppressed by cooldown
    if (_suppressed) {
      _suppressed = false;
      // debugPrint('$debugLabel drag end SUPPRESSED (waiting for all pointers up)');
      return;
    }

    _velocitySimulation.recordVelocitySample();

    final dragStartPosition = _drag?.start?.globalPosition;

    // _drag might be null if the drag activity ended and called _disposeDrag.
    assert(_hold == null || _drag == null);
    _drag?.end(details);
    assert(_drag == null);

    // debugPrint(
    //   '$debugLabel drag end: $details pointers: $_dragPointerCount isDwellDrag: $wasDwellDrag eventGroupId: $_eventGroupId',
    // );

    // If this was a dwell drag, emit finish event (mouse button up) and skip fling
    if (wasDwellDrag) {
      onMouseEvent(
        pb.MouseEvent(type: pb.MouseEvent_EventType.EVENT_TYPE_FINISH),
        _eventGroupId,
      );
      _dragPointerCount = 1; // Reset for next gesture
      _lastPosition = null;
      return;
    }

    if (axis == null) {
      return;
    }

    final pointerCount = _dragPointerCount; // Capture for closure
    final eventGroupId = _eventGroupId; // Capture for closure
    final flingAxis = axis; // Capture for closure
    final dragDistance = dragStartPosition != null
        ? (details.globalPosition - dragStartPosition).distance
        : 0.0;
    final n = Notifier(
      onUpdate: (offset) {
        // Calculate delta from last position
        final delta = _lastPosition != null
            ? offset - _lastPosition!
            : Offset.zero;
        _lastPosition = offset;

        var deltaX = delta.dx;
        var deltaY = delta.dy;

        // Zero out non-relevant axis for multitouch gestures
        if (flingAxis == Axis.vertical) {
          deltaX = 0.0;
        } else if (flingAxis == Axis.horizontal) {
          deltaY = 0.0;
        }

        // Skip zero-delta fling events
        if (deltaX == 0 && deltaY == 0) return;

        onMouseEvent(
          pb.MouseEvent(
            type: pb.MouseEvent_EventType.EVENT_TYPE_FLING,
            pointerCount: pointerCount,
            deltaX: deltaX,
            deltaY: deltaY,
          ),
          eventGroupId,
        );
      },
      onEnd: () {
        _lastPosition = null;
        // debugPrint('$debugLabel continueTrackpadPanMomentum velocity end');
      },
    );

    // Skip fling simulation if drag distance or velocity is too low
    final minDistance = 20;
    if (dragDistance < minDistance) {
      return;
    }

    debugPrint("---");
    final spikeThreshold = pointerCount <= 2 ? 10.0 : 1.0;
    _velocitySimulation.continueTrackpadPanMomentum(
      details: details,
      notifier: n,
      axis: axis!,
      distance: dragDistance,
      spikeThreshold: spikeThreshold,
    );
    _dragPointerCount = 1; // Reset for next gesture
  }

  void _handleDragCancel() {
    // Clean up dwell timer
    _dwellTimer?.cancel();
    _dwellCompleted = false;

    // Reset suppression flag
    _suppressed = false;

    if (key.currentContext == null) {
      // The cancel was caused by the GestureDetector getting disposed, which
      // means we will get disposed momentarily as well and shouldn't do
      // any work.
      return;
    }
    // _hold might be null if the drag started.
    // _drag might be null if the drag activity ended and called _disposeDrag.
    assert(_hold == null || _drag == null);
    _hold?.cancel();
    _drag?.cancel();
    assert(_hold == null);
    assert(_drag == null);
    _lastPosition = null;
    _dragPointerCount = 1; // Reset for next gesture
    // debugPrint('$debugLabel drag cancel');
  }

  void _disposeHold() {
    _hold = null;
  }

  void _disposeDrag() {
    _drag = null;
  }
}

class _Drag implements Drag {
  _Drag(this.start, this.down, this.dispose);

  DragDownDetails? down;
  DragStartDetails? start;
  VoidCallback? dispose;
  @override
  void update(DragUpdateDetails details) {}

  @override
  void end(DragEndDetails details) {
    dispose?.call();
  }

  @override
  void cancel() {
    dispose?.call();
  }
}

class _Position {
  _Position(this.down);

  DragDownDetails? down;

  _Hold hold(VoidCallback hold) {
    return _Hold(hold);
  }

  _Drag? drag(DragStartDetails start, void Function() dispose) {
    return _Drag(start, down, dispose);
  }
}

class _Hold {
  _Hold(this.disposeHold);
  VoidCallback disposeHold;
  void cancel() {
    disposeHold();
  }
}

/// A [VerticalDragGestureRecognizer] that only accepts gestures when
/// multiple pointers are active. Single-finger drags are rejected,
/// allowing other recognizers (like [PanGestureRecognizer]) to win.
class MultitouchVerticalDragGestureRecognizer
    extends VerticalDragGestureRecognizer {
  MultitouchVerticalDragGestureRecognizer({
    required this.pointerCountGetter,
    super.supportedDevices,
  });

  final int Function() pointerCountGetter;
  final Set<int> _rejectedPointers = {};
  bool _gestureAccepted = false;

  // Track movement to determine dominant direction
  Offset _cumulativeDelta = Offset.zero;

  @override
  void addAllowedPointer(PointerDownEvent event) {
    // Always track, but mark for rejection if single-touch
    super.addAllowedPointer(event);
  }

  @override
  void handleEvent(PointerEvent event) {
    // Track movement delta for direction determination
    if (event is PointerMoveEvent) {
      _cumulativeDelta += event.delta;
    } else if (event is PointerDownEvent) {
      _cumulativeDelta = Offset.zero;
    }

    // Once gesture is accepted, continue processing regardless of pointer count
    if (_gestureAccepted) {
      super.handleEvent(event);
      return;
    }
    // If single-touch and not yet accepted, reject
    if (pointerCountGetter() < 2) {
      if (!_rejectedPointers.contains(event.pointer)) {
        _rejectedPointers.add(event.pointer);
        resolve(GestureDisposition.rejected);
      }
      return;
    }
    _rejectedPointers.remove(event.pointer);
    super.handleEvent(event);
  }

  @override
  void rejectGesture(int pointer) {
    _rejectedPointers.remove(pointer);
    _gestureAccepted = false;
    _cumulativeDelta = Offset.zero;
    super.rejectGesture(pointer);
  }

  @override
  void acceptGesture(int pointer) {
    _rejectedPointers.remove(pointer);
    if (pointerCountGetter() < 2) {
      resolve(GestureDisposition.rejected);
      return;
    }

    // Only accept if vertical movement is dominant
    final dy = _cumulativeDelta.dy.abs();
    final dx = _cumulativeDelta.dx.abs();
    // Require horizontal to be clearly dominant (30% more) before rejecting vertical
    if (dx > dy * 1.3) {
      // Horizontal is clearly dominant - reject this vertical gesture
      resolve(GestureDisposition.rejected);
      return;
    }

    _gestureAccepted = true;
    super.acceptGesture(pointer);
  }

  @override
  void didStopTrackingLastPointer(int pointer) {
    _gestureAccepted = false;
    _cumulativeDelta = Offset.zero;
    super.didStopTrackingLastPointer(pointer);
  }
}

/// A [HorizontalDragGestureRecognizer] that only accepts gestures when
/// multiple pointers are active. Single-finger drags are rejected,
/// allowing other recognizers (like [PanGestureRecognizer]) to win.
class MultitouchHorizontalDragGestureRecognizer
    extends HorizontalDragGestureRecognizer {
  MultitouchHorizontalDragGestureRecognizer({
    required this.pointerCountGetter,
    super.supportedDevices,
  });

  final int Function() pointerCountGetter;
  final Set<int> _rejectedPointers = {};
  bool _gestureAccepted = false;

  // Track movement to determine dominant direction
  Offset _cumulativeDelta = Offset.zero;

  @override
  void addAllowedPointer(PointerDownEvent event) {
    // Always track, but mark for rejection if single-touch
    super.addAllowedPointer(event);
  }

  @override
  void handleEvent(PointerEvent event) {
    // Track movement delta for direction determination
    if (event is PointerMoveEvent) {
      _cumulativeDelta += event.delta;
    } else if (event is PointerDownEvent) {
      _cumulativeDelta = Offset.zero;
    }

    // Once gesture is accepted, continue processing regardless of pointer count
    if (_gestureAccepted) {
      super.handleEvent(event);
      return;
    }
    // If single-touch and not yet accepted, reject
    if (pointerCountGetter() < 2) {
      if (!_rejectedPointers.contains(event.pointer)) {
        _rejectedPointers.add(event.pointer);
        resolve(GestureDisposition.rejected);
      }
      return;
    }
    _rejectedPointers.remove(event.pointer);
    super.handleEvent(event);
  }

  @override
  void rejectGesture(int pointer) {
    _rejectedPointers.remove(pointer);
    _gestureAccepted = false;
    _cumulativeDelta = Offset.zero;
    super.rejectGesture(pointer);
  }

  @override
  void acceptGesture(int pointer) {
    _rejectedPointers.remove(pointer);
    if (pointerCountGetter() < 2) {
      resolve(GestureDisposition.rejected);
      return;
    }

    // Only accept if horizontal movement is dominant
    final dy = _cumulativeDelta.dy.abs();
    final dx = _cumulativeDelta.dx.abs();
    // Require vertical to be clearly dominant (30% more) before rejecting horizontal
    if (dy > dx * 1.3) {
      // Vertical is clearly dominant - reject this horizontal gesture
      resolve(GestureDisposition.rejected);
      return;
    }

    _gestureAccepted = true;
    super.acceptGesture(pointer);
  }

  @override
  void didStopTrackingLastPointer(int pointer) {
    _gestureAccepted = false;
    _cumulativeDelta = Offset.zero;
    super.didStopTrackingLastPointer(pointer);
  }
}

/// A [PanGestureRecognizer] that only accepts gestures when
/// a single pointer is active. Multi-finger drags are rejected,
/// allowing other recognizers (like [MultitouchVerticalDragGestureRecognizer]) to win.
class SingleTouchPanGestureRecognizer extends PanGestureRecognizer {
  SingleTouchPanGestureRecognizer({
    required this.pointerCountGetter,
    super.supportedDevices,
  });

  final int Function() pointerCountGetter;
  final Set<int> _rejectedPointers = {};
  bool _gestureAccepted = false;

  @override
  void addAllowedPointer(PointerDownEvent event) {
    // Always track, but mark for rejection if multi-touch
    super.addAllowedPointer(event);
  }

  @override
  void handleEvent(PointerEvent event) {
    // Once gesture is accepted, continue processing regardless of pointer count
    if (_gestureAccepted) {
      super.handleEvent(event);
      return;
    }
    // If multi-touch and not yet accepted, reject
    if (pointerCountGetter() > 1) {
      if (!_rejectedPointers.contains(event.pointer)) {
        _rejectedPointers.add(event.pointer);
        resolve(GestureDisposition.rejected);
      }
      return;
    }
    _rejectedPointers.remove(event.pointer);
    super.handleEvent(event);
  }

  @override
  void rejectGesture(int pointer) {
    _rejectedPointers.remove(pointer);
    _gestureAccepted = false;
    super.rejectGesture(pointer);
  }

  @override
  void acceptGesture(int pointer) {
    _rejectedPointers.remove(pointer);
    if (pointerCountGetter() > 1) {
      resolve(GestureDisposition.rejected);
      return;
    }
    _gestureAccepted = true;
    super.acceptGesture(pointer);
  }

  @override
  void didStopTrackingLastPointer(int pointer) {
    _gestureAccepted = false;
    super.didStopTrackingLastPointer(pointer);
  }
}

/// A gesture recognizer that detects concurrent multi-finger taps.
///
/// Extends [MultiTapGestureRecognizer] and batches taps that are lifted
/// within a short time window, reporting the count of concurrent fingers.
///
/// Taps held down too long are rejected via the built-in long tap detection.
class ConcurrentTapGestureRecognizer extends MultiTapGestureRecognizer {
  ConcurrentTapGestureRecognizer({
    this.onClick,
    this.onDoubleClick,
    super.debugOwner,
    super.supportedDevices,
  }) : super(longTapDelay: const Duration(milliseconds: 200)) {
    onTapDown = _handleTapDown;
    onTapUp = _handleTapUp;
    onTapCancel = _handleTapCancel;
    onLongTapDown = _handleLongTapDown;
  }

  /// Called when concurrent taps are detected, with the number of fingers.
  void Function(int pointerCount)? onClick;

  void Function()? onDoubleClick;

  /// Pointers currently being tracked.
  final Set<int> _activePointers = {};

  /// Pointers that are known to be registered with the parent's _gestureMap.
  /// Used to avoid calling super.rejectGesture for pointers that were already removed.
  final Set<int> _registeredPointers = {};

  @override
  void addAllowedPointer(PointerDownEvent event) {
    super.addAllowedPointer(event);
    _registeredPointers.add(event.pointer);
  }

  void _handleTapDown(int pointer, TapDownDetails details) {
    _activePointers.add(pointer);

    if (_activePointers.length == 1 && _pendingTaps.length == 1) {
      acceptGesture(pointer);
      _activePointers.clear();
      _stopTimer();
      _pendingTaps.clear();
      onDoubleClick?.call();
      return;
    }
  }

  void _handleTapUp(int pointer, TapUpDetails details) {
    if (_activePointers.remove(pointer)) {
      _pendingTaps.add(pointer);
      _batchTimer ??= Timer(_kBatchWindow, _resolveBatch);
    }
  }

  void _handleTapCancel(int pointer) {
    _activePointers.remove(pointer);
  }

  void _handleLongTapDown(int pointer, TapDownDetails details) {
    rejectGesture(pointer);

    _activePointers.clear();
    _pendingTaps.clear();
  }

  @override
  void acceptGesture(int pointer) {
    // Must call super for the specific pointer being accepted by the arena
    // This sets _wonArena = true in the parent's _TapGesture, allowing taps to complete
    if (_registeredPointers.remove(pointer)) {
      super.acceptGesture(pointer);
    }
    // Don't remove from _activePointers here - pointers should remain active
    // until tap up/cancel. Removing here caused multi-finger taps to lose track
    // of earlier fingers when the arena accepted them.
  }

  @override
  void rejectGesture(int pointer) {
    // If any pointer rejects - reject all
    // Reject all active pointers
    for (final p in _activePointers.toList()) {
      // Only call super.rejectGesture if we know this pointer is still registered
      // with the parent's _gestureMap. The parent asserts the pointer exists,
      // and it may have already been removed through normal gesture resolution.
      if (_registeredPointers.remove(p)) {
        super.rejectGesture(p);
      }
    }

    _pendingTaps.clear();
    _activePointers.clear();
    _registeredPointers.clear();
    _stopTimer();
  }

  void _stopTimer() {
    _batchTimer?.cancel();
    _batchTimer = null;
  }

  /// Time window to batch concurrent taps together.
  static const Duration _kBatchWindow = Duration(milliseconds: 100);

  final _pendingTaps = <int>{};
  Timer? _batchTimer;

  void _resolveBatch() {
    _batchTimer = null;
    if (_pendingTaps.isNotEmpty) {
      final count = _pendingTaps.length;
      _pendingTaps.clear();
      onClick?.call(count);
    }
  }

  @override
  void dispose() {
    _batchTimer?.cancel();
    _pendingTaps.clear();
    _activePointers.clear();
    _registeredPointers.clear();
    super.dispose();
  }

  @override
  String get debugDescription => 'concurrent tap';
}
