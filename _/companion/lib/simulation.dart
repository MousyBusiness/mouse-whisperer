import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';

class Notifier {
  Notifier({required this.onUpdate, required this.onEnd});

  final void Function(Offset) onUpdate;
  final void Function() onEnd;
}

class VelocitySimulation {
  Timer? _trackpadTimer;
  VelocityTracker? _tracker;

  // Velocity sample tracking for average calculation
  final List<double> _velocitySamples = [];

  double get averageVelocity {
    if (_velocitySamples.isEmpty) return 0.0;
    return _velocitySamples.reduce((a, b) => a + b) / _velocitySamples.length;
  }

  void recordVelocitySample() {
    final v = velocity.distance;
    if (v > 0) {
      _velocitySamples.add(v);
    }
    if(_velocitySamples.length > 20) {
      _velocitySamples.removeAt(0);
    }
  }

  void dispose() {
    _trackpadTimer?.cancel();
  }

  bool get hasTracker => _tracker != null;

  bool get pending => _trackpadTimer != null;

  Offset get velocity => _tracker?.getVelocity().pixelsPerSecond ?? Offset.zero;

  void reset() {
    _trackpadTimer?.cancel();
    _trackpadTimer = null;
    _velocitySamples.clear();
  }

  VelocityTracker generateTracker(PointerEvent event) {
    _trackpadTimer?.cancel();
    _tracker = MacOSScrollViewFlingVelocityTracker(PointerDeviceKind.trackpad);
    _tracker!.addPosition(event.timeStamp, event.position);
    return _tracker!;
  }

  void continueTrackpadPanMomentum({required DragEndDetails details, required Notifier notifier, required Axis axis, required double distance, required double spikeThreshold}) {
    assert(_tracker != null);
    assert(_trackpadTimer == null);
    if (_velocitySamples.length < 2) {
      debugPrint('not enough velocity samples to compute fling: ${_velocitySamples.length}');
      return;
    }

    // Use average velocity magnitude with direction from end velocity
    final o = velocity;
    var avgMagnitude = velocity.distance;
    debugPrint('distance: ${distance}');
    debugPrint('velocitySamples: ${_velocitySamples.length}');
    debugPrint('averageVelocity: ${averageVelocity}, actual: ${velocity.distance}');

    if (velocity.distance < kMinFlingVelocity * 2) {
      debugPrint('velocity too low to fling: ${velocity.distance}');
      return;
    }

    if (velocity.distance > averageVelocity) {
      debugPrint('velocity exceeded average velocity: ${velocity.distance}');
      avgMagnitude = averageVelocity;
    }

    final spikeDetection = avgMagnitude / distance;
    debugPrint('spikeDetection: $spikeDetection');
    if (spikeDetection < spikeThreshold) {
      debugPrint('spike detected: $spikeDetection');
      return;
    }

    // Get the sign (direction) from the end velocity for the relevant axis
    final sign = axis == Axis.vertical ? (o.dy >= 0 ? 1.0 : -1.0) : (o.dx >= 0 ? 1.0 : -1.0);
    final flingVelocity = avgMagnitude * sign;

    final position = details.globalPosition;
    late final Simulation simulation;
    late final double startPos;

    if (axis == Axis.vertical) {
      startPos = position.dy;
      simulation = ClampingScrollSimulation(
        friction: 0.01,
        position: startPos,
        velocity: flingVelocity,
        tolerance: const Tolerance(velocity: 1, distance: 1),
      );
    } else {
      startPos = position.dx;
      simulation = ClampingScrollSimulation(
        friction: 0.01,
        position: startPos,
        velocity: flingVelocity,
        tolerance: const Tolerance(velocity: 1, distance: 1),
      );
    }

    var startTime = DateTime.now().millisecondsSinceEpoch;
    _trackpadTimer = Timer.periodic(const Duration(milliseconds: 8), (timer) {
      var currentTime = DateTime.now().millisecondsSinceEpoch;
      var elapsedTime = (currentTime - startTime) / 1000;

      final newPos = simulation.x(elapsedTime);
      final next = axis == Axis.vertical ? Offset(position.dx, newPos) : Offset(newPos, position.dy);
      notifier.onUpdate(next);

      if (simulation.isDone(elapsedTime)) {
        notifier.onEnd();
        reset();
      }
    });
  }
}
