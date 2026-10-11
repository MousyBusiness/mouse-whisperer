import 'dart:async';
import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_animate/flutter_animate.dart';
import 'package:mouse_whisperer/app_theme.dart';
import 'package:mouse_whisperer/code/custom_code_box.dart';
import 'package:mouse_whisperer/gen/mouse_whisperer/overlay/v1/overlay.pb.dart'
    as pb;
import 'package:mouse_whisperer/server.dart';
import 'package:toastification/toastification.dart';
import 'package:window_manager/window_manager.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return const ToastificationWrapper(
      child: MaterialApp(
        debugShowCheckedModeBanner: false,
        home: App(),
      ),
    );
  }
}

class App extends StatefulWidget {
  const App({super.key});

  @override
  State<App> createState() => _AppState();
}

class _CommandText {
  _CommandText({required this.id, required this.text});
  final int id;
  final String text;
}

class _ModifierKeys {
  final bool command;
  final bool control;
  final bool alt;
  final bool shift;

  const _ModifierKeys({
    this.command = false,
    this.control = false,
    this.alt = false,
    this.shift = false,
  });

  bool get hasAny => command || control || alt || shift;
}

class _AppState extends State<App> {
  final _texts = <_CommandText>[];
  int _textIdCounter = 0;
  Uint8List? screenshot;
  Offset? eyes;
  (String, String)? aiResponse;
  (String, String)? dictation; // (committed buffer, pending utterance)
  List<String> dictationHints = []; // cached from the last non-empty send
  pb.CommandsList? commandsList;
  pb.CommandConfig? commandConfig;
  _ModifierKeys? modifierKeys;
  bool ghostVisible = false;
  bool isRecording = false;
  bool isNotListening = false;
  Timer? _recordingTimer;
  bool isMouseDown = false;
  bool tunnelVisionEnabled = false;
  int tunnelVisionRadius = 600;
  double tunnelVisionOpacity = 0.95;
  double tunnelVisionBlendDistance = 0.5;
  Offset? _tunnelVisionAnimatedPos; // Smoothly animated position
  Timer? _tunnelVisionAnimationTimer;
  late Server server;

  // Scroll controllers for commands panels
  final ScrollController _commandsListScrollController = ScrollController();
  final ScrollController _commandConfigScrollController = ScrollController();

  // Global modifier keys monitoring
  static const _modifierKeysChannel =
      EventChannel('mouse_whisperer/modifier_keys');
  StreamSubscription? _modifierKeysSubscription;

  static Future initializeWindow() async {
    // await windowManager.ensureInitialized();

    // WindowOptions windowOptions = const WindowOptions(
    //   skipTaskbar: true,
    // );

    // await windowManager.waitUntilReadyToShow(windowOptions, () async {
    //   await windowManager.show();
    //   await windowManager.focus();
    //   await windowManager.setAlwaysOnTop(true);
    //   await windowManager.setVisibleOnAllWorkspaces(true,
    //       visibleOnFullScreen: true);
    // });
  }

  late Timer timer;

  @override
  void initState() {
    super.initState();
    timer = Timer.periodic(Duration(seconds: 5), (t) {
      currentColor++;
      if (currentColor >= rainbowColors.length) {
        currentColor = 0;
      }
      setState(() {});
    });
    initializeWindow();
    server = Server(
      onCommandText: onCommandText,
      onCommandsList: onCommandsList,
      onCommandConfig: onCommandConfig,
      onImage: onImage,
      onAiResponse: onAiResponse,
      onEyePosition: onEyePosition,
      onUIAction: onUIAction,
      onTunnelVision: onTunnelVision,
      onGhostVisibility: onGhostVisibility,
      onRecordingIndicator: onRecordingIndicator,
      onMouseDownIndicator: onMouseDownIndicator,
      onNotListeningIndicator: onNotListeningIndicator,
      onDictation: onDictation,
    );

    Future.delayed(Duration.zero, () async {
      await server.start();
    });

    // Monitor global modifier keys from native macOS (delayed to ensure native handler is registered)
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _modifierKeysSubscription =
          _modifierKeysChannel.receiveBroadcastStream().listen(_onModifierKeys);
    });

    SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
  }

  @override
  void dispose() {
    timer.cancel();
    _tunnelVisionAnimationTimer?.cancel();
    _recordingTimer?.cancel();
    _modifierKeysSubscription?.cancel();
    _commandsListScrollController.dispose();
    _commandConfigScrollController.dispose();
    super.dispose();
  }

  void onCommandText(String text) {
    setState(() {
      _texts.clear();
      final id = _textIdCounter++;
      _texts.add(_CommandText(id: id, text: text));
      Future.delayed(const Duration(seconds: 1), () {
        setState(() {
          _texts.removeWhere((e) => e.id == id);
        });
      });
    });
  }

  void onCommandsList(pb.CommandsList? list) {
    setState(() {
      commandsList = list;
      if (list != null) {
        final dismissSeconds =
            list.dismissSeconds > 0 ? list.dismissSeconds : 15;
        Future.delayed(Duration(seconds: dismissSeconds), () {
          if (commandsList == list) {
            setState(() {
              commandsList = null;
            });
          }
        });
      }
    });
  }

  void onCommandConfig(pb.CommandConfig? config) {
    setState(() {
      commandConfig = config;
      if (config != null) {
        final dismissSeconds =
            config.dismissSeconds > 0 ? config.dismissSeconds : 20;
        Future.delayed(Duration(seconds: dismissSeconds), () {
          if (commandConfig == config) {
            setState(() {
              commandConfig = null;
            });
          }
        });
      }
    });
  }

  void onAiResponse((String, String)? s) {
    setState(() {
      aiResponse = s;
    });
  }

  void onDictation((String, String, List<String>)? d) {
    setState(() {
      if (d == null) {
        dictation = null;
        return;
      }
      dictation = (d.$1, d.$2);
      if (d.$3.isNotEmpty) {
        dictationHints = d.$3;
      }
    });
  }

  void onImage(Uint8List? image) {
    setState(() {
      screenshot = image;
    });
  }

  void onEyePosition(Offset? pos) {
    setState(() {
      eyes = pos;
    });
  }

  void onUIAction(pb.UIAction action) {
    // Handle dismiss action first (works for all panels including AI)
    if (action.action == 'dismiss') {
      setState(() {
        commandsList = null;
        commandConfig = null;
        aiResponse = null;
        dictation = null;
      });
      return;
    }

    // Determine which scroll controller to use based on active panel
    ScrollController? activeController;
    if (commandsList != null) {
      activeController = _commandsListScrollController;
    } else if (commandConfig != null) {
      activeController = _commandConfigScrollController;
    }

    if (activeController == null || !activeController.hasClients) return;

    switch (action.action) {
      case 'scroll_down':
        final offset = activeController.offset +
            (action.amount > 0 ? action.amount.toDouble() : 200);
        activeController.animateTo(
          offset.clamp(0, activeController.position.maxScrollExtent),
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
        break;
      case 'scroll_up':
        final offset = activeController.offset -
            (action.amount > 0 ? action.amount.toDouble() : 200);
        activeController.animateTo(
          offset.clamp(0, activeController.position.maxScrollExtent),
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
        break;
    }
  }

  void onTunnelVision(
      bool enabled, int radius, double opacity, double blendDistance) {
    setState(() {
      tunnelVisionEnabled = enabled;
      tunnelVisionRadius = radius;
      tunnelVisionOpacity = opacity;
      tunnelVisionBlendDistance = blendDistance;
    });

    if (enabled) {
      _startTunnelVisionAnimation();
    } else {
      _stopTunnelVisionAnimation();
    }
  }

  void onGhostVisibility(bool visible) {
    setState(() {
      ghostVisible = visible;
    });
  }

  void onRecordingIndicator(bool recording, int timeoutMs) {
    // Cancel any existing timer
    _recordingTimer?.cancel();
    _recordingTimer = null;

    setState(() {
      isRecording = recording;
    });

    // Set up timeout timer if recording started
    if (recording && timeoutMs > 0) {
      _recordingTimer = Timer(Duration(milliseconds: timeoutMs + 500), () {
        // Add small buffer to timeout
        if (isRecording) {
          setState(() {
            isRecording = false;
          });
        }
      });
    }
  }

  void onMouseDownIndicator(bool active) {
    setState(() {
      isMouseDown = active;
    });
  }

  void onNotListeningIndicator(bool enabled) {
    setState(() {
      isNotListening = enabled;
    });
  }

  void _startTunnelVisionAnimation() {
    // Cancel any existing timer
    _tunnelVisionAnimationTimer?.cancel();

    // Run at ~60fps for smooth animation
    _tunnelVisionAnimationTimer =
        Timer.periodic(const Duration(milliseconds: 16), (_) {
      if (!tunnelVisionEnabled || eyes == null) return;

      final target = eyes!;

      // Initialize animated position if not set yet
      if (_tunnelVisionAnimatedPos == null) {
        setState(() {
          _tunnelVisionAnimatedPos = target;
        });
        return;
      }

      final current = _tunnelVisionAnimatedPos!;

      // Calculate distance from current center to target
      final dx = target.dx - current.dx;
      final dy = target.dy - current.dy;
      final distance = math.sqrt(dx * dx + dy * dy);

      // Dead zone: don't move if target is within 50% of radius from center
      final deadZoneRadius = tunnelVisionRadius * 0.5;
      if (distance < deadZoneRadius) {
        return; // Target is within dead zone, don't move
      }

      // Lerp factor: lower = slower/smoother, higher = faster/snappier
      const lerpFactor = 0.02;
      final newPos = Offset(
        current.dx + dx * lerpFactor,
        current.dy + dy * lerpFactor,
      );

      // Update the animated position
      setState(() {
        _tunnelVisionAnimatedPos = newPos;
      });
    });
  }

  void _stopTunnelVisionAnimation() {
    _tunnelVisionAnimationTimer?.cancel();
    _tunnelVisionAnimationTimer = null;
    _tunnelVisionAnimatedPos = null;
  }

  void _onModifierKeys(dynamic event) {
    if (event is! Map) return;
    final map = Map<String, dynamic>.from(event);
    final newKeys = _ModifierKeys(
      command: map['command'] == true,
      control: map['control'] == true,
      alt: map['alt'] == true,
      shift: map['shift'] == true,
    );

    setState(() {
      modifierKeys = newKeys.hasAny ? newKeys : null;
    });
  }

  int currentColor = 0;
  List<Color> rainbowColors = [
    Colors.redAccent,
    Colors.blueAccent,
    Colors.green,
  ];
  double ghostSize = 4;

  Widget _buildAiResponse() {
    return Positioned(
      key: ValueKey(aiResponse),
      right: 40,
      top: 100,
      child: GestureDetector(
        onTap: () {
          setState(() {
            aiResponse = null;
          });
        },
        child: Container(
          width: 420,
          constraints: const BoxConstraints(maxHeight: 600),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(12),
            color: darkBackground.withValues(alpha: 0.95),
            border: Border.all(
                color: Colors.purpleAccent.withValues(alpha: 0.3), width: 1),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Header
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                decoration: BoxDecoration(
                  border: Border(
                    bottom: BorderSide(
                      color: Colors.purpleAccent.withValues(alpha: 0.2),
                      width: 1,
                    ),
                  ),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    const Expanded(
                      child: Text(
                        'AI Response',
                        style: TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                          color: Colors.purpleAccent,
                        ),
                      ),
                    ),
                    const Text(
                      "'dismiss'",
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.white38,
                      ),
                    ),
                  ],
                ),
              ),
              // Content
              Flexible(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        "// Prompt: ${aiResponse!.$1}",
                        style: AppTheme.codeStyleBase
                            .copyWith(color: Colors.white38),
                      ),
                      const SizedBox(height: 8),
                      CustomCodeBox(
                        language: "bash",
                        minLines: 1,
                        maxLines: 10,
                        readOnly: true,
                        backgroundColor: Colors.transparent,
                        startingContent: aiResponse!.$2,
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        )
            .animate()
            .fadeIn()
            .slideX(begin: 0.2, duration: Duration(milliseconds: 300)),
      ),
    );
  }

  Widget _buildDictationPanel() {
    final committed = dictation!.$1;
    final pending = dictation!.$2;

    // Constant key: updates per transcription tick must not replay the
    // entry animation (the AI panel flickers because it keys on content)
    return Positioned(
      key: const ValueKey('dictation-panel'),
      left: 0,
      right: 0,
      bottom: 96,
      child: Center(
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxWidth: MediaQuery.of(context).size.width * 0.6,
          ),
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(12),
              color: darkBackground.withValues(alpha: 0.92),
              border: Border.all(
                color: Colors.tealAccent.withValues(alpha: 0.3),
                width: 1,
              ),
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'DICTATING',
                  style: TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.bold,
                    color: Colors.tealAccent,
                    letterSpacing: 1.5,
                  ),
                ),
                const SizedBox(height: 8),
                Text.rich(
                  TextSpan(
                    children: [
                      if (committed.isNotEmpty)
                        TextSpan(
                          text: committed,
                          style: TextStyle(
                            color: Colors.white.withValues(alpha: 0.85),
                          ),
                        ),
                      if (committed.isNotEmpty && pending.isNotEmpty)
                        const TextSpan(text: ' '),
                      if (pending.isNotEmpty)
                        TextSpan(
                          text: pending,
                          style: TextStyle(
                            color: Colors.white.withValues(alpha: 0.4),
                            fontStyle: FontStyle.italic,
                          ),
                        ),
                    ],
                    style: const TextStyle(fontSize: 15, height: 1.4),
                  ),
                ),
                if (dictationHints.isNotEmpty) ...[
                  const SizedBox(height: 10),
                  Wrap(
                    spacing: 6,
                    runSpacing: 4,
                    children: dictationHints
                        .map(
                          (h) => Container(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 8,
                              vertical: 2,
                            ),
                            decoration: BoxDecoration(
                              borderRadius: BorderRadius.circular(4),
                              color: Colors.tealAccent.withValues(alpha: 0.08),
                            ),
                            child: Text(
                              h,
                              style: const TextStyle(
                                fontSize: 11,
                                color: Colors.white38,
                              ),
                            ),
                          ),
                        )
                        .toList(),
                  ),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildCommandsList() {
    final list = commandsList!;

    return Positioned(
      key: ValueKey(list.hashCode),
      right: 40,
      top: 100,
      child: GestureDetector(
        onTap: () {
          setState(() {
            commandsList = null;
          });
        },
        child: Container(
          width: 400,
          constraints: const BoxConstraints(maxHeight: 600),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(12),
            color: darkBackground.withValues(alpha: 0.95),
            border: Border.all(
                color: Colors.blueAccent.withValues(alpha: 0.3), width: 1),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Header
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                decoration: BoxDecoration(
                  border: Border(
                    bottom: BorderSide(
                      color: Colors.blueAccent.withValues(alpha: 0.2),
                      width: 1,
                    ),
                  ),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Expanded(
                      child: Text(
                        list.title.isNotEmpty ? list.title : 'Hints',
                        style: const TextStyle(
                          fontSize: 20,
                          fontWeight: FontWeight.bold,
                          color: Colors.blueAccent,
                        ),
                      ),
                    ),
                    const Text(
                      "'more' / 'less' / 'dismiss'",
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.white38,
                      ),
                    ),
                  ],
                ),
              ),
              // Scrollable items list
              Flexible(
                child: SingleChildScrollView(
                  controller: _commandsListScrollController,
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: list.items.map((item) {
                      return Padding(
                        padding: const EdgeInsets.symmetric(vertical: 4),
                        child: Text(
                          item,
                          style: const TextStyle(
                            fontSize: 14,
                            color: Colors.white70,
                            fontFamily: 'monospace',
                            height: 1.4,
                          ),
                        ),
                      );
                    }).toList(),
                  ),
                ),
              ),
            ],
          ),
        )
            .animate()
            .fadeIn()
            .slideX(begin: 0.2, duration: Duration(milliseconds: 300)),
      ),
    );
  }

  Widget _buildCommandConfig() {
    final config = commandConfig!;

    return Positioned(
      key: ValueKey(config.hashCode),
      right: 40,
      top: 100,
      child: GestureDetector(
        onTap: () {
          setState(() {
            commandConfig = null;
          });
        },
        child: Container(
          width: 420,
          constraints: const BoxConstraints(maxHeight: 600),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(12),
            color: darkBackground.withValues(alpha: 0.95),
            border: Border.all(
                color: Colors.orangeAccent.withValues(alpha: 0.3), width: 1),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Header
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                decoration: BoxDecoration(
                  border: Border(
                    bottom: BorderSide(
                      color: Colors.orangeAccent.withValues(alpha: 0.2),
                      width: 1,
                    ),
                  ),
                ),
                child: Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Expanded(
                      child: Text(
                        config.name.toUpperCase(),
                        style: const TextStyle(
                          fontSize: 18,
                          fontWeight: FontWeight.bold,
                          color: Colors.orangeAccent,
                          fontFamily: 'monospace',
                        ),
                      ),
                    ),
                    const Text(
                      "'more' / 'less' / 'dismiss'",
                      style: TextStyle(
                        fontSize: 12,
                        color: Colors.white38,
                      ),
                    ),
                  ],
                ),
              ),
              // Config content
              Flexible(
                child: SingleChildScrollView(
                  controller: _commandConfigScrollController,
                  padding: const EdgeInsets.all(16),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      // Description (if present)
                      if (config.description.isNotEmpty) ...[
                        Text(
                          config.description,
                          style: TextStyle(
                            fontSize: 14,
                            color: Colors.white.withValues(alpha: 0.8),
                            height: 1.4,
                          ),
                        ),
                        const SizedBox(height: 16),
                      ],
                      // Triggers
                      _configRow('triggers',
                          '[${config.triggers.map((t) => '"$t"').join(', ')}]'),
                      // Footer hint
                      const SizedBox(height: 16),
                      Text(
                        "Say 'show hints' to go back",
                        style: TextStyle(
                          fontSize: 12,
                          color: Colors.blueAccent.withValues(alpha: 0.7),
                          fontStyle: FontStyle.italic,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
        )
            .animate()
            .fadeIn()
            .slideX(begin: 0.2, duration: Duration(milliseconds: 300)),
      ),
    );
  }

  Widget _configRow(String key, String value) {
    return RichText(
      text: TextSpan(
        children: [
          TextSpan(
            text: key,
            style: const TextStyle(
              fontSize: 14,
              color: Colors.cyanAccent,
              fontFamily: 'monospace',
            ),
          ),
          const TextSpan(
            text: ' = ',
            style: TextStyle(
              fontSize: 14,
              color: Colors.white60,
              fontFamily: 'monospace',
            ),
          ),
          TextSpan(
            text: value,
            style: const TextStyle(
              fontSize: 14,
              color: Colors.greenAccent,
              fontFamily: 'monospace',
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildGhost() {
    return Positioned(
        left: eyes!.dx - ghostSize / 2,
        top: eyes!.dy - ghostSize / 2,
        child: ClipOval(
          child: AnimatedContainer(
            height: ghostSize,
            width: ghostSize,
            color: rainbowColors[currentColor].withOpacity(.4),
            duration: const Duration(seconds: 1),
          ),
        ));
  }

  Widget _buildMouseRects() {
    return Image.memory(screenshot!);
  }

  Widget _buildModifierKeys() {
    final keys = modifierKeys!;
    final activeKeys = <String>[];
    if (keys.command) activeKeys.add('⌘');
    if (keys.control) activeKeys.add('⌃');
    if (keys.alt) activeKeys.add('⌥');
    if (keys.shift) activeKeys.add('⇧');

    return Positioned(
      left: 48,
      bottom: 80,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: activeKeys.map((key) {
          return Container(
            margin: const EdgeInsets.only(right: 6),
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(6),
              color: darkBackground.withValues(alpha: 0.9),
              border: Border.all(
                color: Colors.amber.withValues(alpha: 0.6),
                width: 1.5,
              ),
              boxShadow: [
                BoxShadow(
                  color: Colors.amber.withValues(alpha: 0.2),
                  blurRadius: 8,
                  spreadRadius: 1,
                ),
              ],
            ),
            child: Text(
              key,
              style: const TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.bold,
                color: Colors.amber,
                fontFamily: 'monospace',
                letterSpacing: 1,
              ),
            ),
          );
        }).toList(),
      )
          .animate()
          .fadeIn(duration: const Duration(milliseconds: 200))
          .scale(begin: const Offset(0.9, 0.9)),
    );
  }

  Widget _buildTunnelVision() {
    // Use animated position, fall back to eyes if animation hasn't started
    final center = _tunnelVisionAnimatedPos ?? eyes;
    if (center == null) return const SizedBox.shrink();

    return Positioned.fill(
      child: CustomPaint(
        painter: TunnelVisionPainter(
          center: center,
          radius: tunnelVisionRadius.toDouble(),
          opacity: tunnelVisionOpacity,
          blendDistance: tunnelVisionBlendDistance,
        ),
      ),
    );
  }

  Widget _buildRecordingIndicator() {
    return Positioned(
      left: 0,
      right: 0,
      top: 64,
      child: Center(
        child: Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: darkBackground.withValues(alpha: 0.9),
            border: Border.all(
              color: Colors.redAccent.withValues(alpha: 0.6),
              width: 2,
            ),
            boxShadow: [
              BoxShadow(
                color: Colors.redAccent.withValues(alpha: 0.3),
                blurRadius: 20,
                spreadRadius: 5,
              ),
            ],
          ),
          child: const Icon(
            Icons.mic,
            color: Colors.redAccent,
            size: 48,
          ),
        )
            .animate(onPlay: (controller) => controller.repeat(reverse: true))
            .scale(
              begin: const Offset(1.0, 1.0),
              end: const Offset(1.1, 1.1),
              duration: const Duration(milliseconds: 800),
              curve: Curves.easeInOut,
            )
            .animate()
            .fadeIn(duration: const Duration(milliseconds: 200))
            .slideY(
              begin: -0.5,
              duration: const Duration(milliseconds: 300),
              curve: Curves.easeOut,
            ),
      ),
    );
  }

  Widget _buildMouseDownIndicator() {
    return Positioned(
      left: 48,
      top: 64,
      child: Container(
        padding: const EdgeInsets.all(20),
        decoration: BoxDecoration(
          shape: BoxShape.circle,
          color: darkBackground.withValues(alpha: 0.9),
          border: Border.all(
            color: Colors.orangeAccent.withValues(alpha: 0.6),
            width: 2,
          ),
          boxShadow: [
            BoxShadow(
              color: Colors.orangeAccent.withValues(alpha: 0.3),
              blurRadius: 20,
              spreadRadius: 5,
            ),
          ],
        ),
        child: const Icon(
          Icons.touch_app,
          color: Colors.orangeAccent,
          size: 48,
        ),
      )
          .animate(onPlay: (controller) => controller.repeat(reverse: true))
          .scale(
            begin: const Offset(1.0, 1.0),
            end: const Offset(1.1, 1.1),
            duration: const Duration(milliseconds: 800),
            curve: Curves.easeInOut,
          )
          .animate()
          .fadeIn(duration: const Duration(milliseconds: 200))
          .slideX(
            begin: -0.5,
            duration: const Duration(milliseconds: 300),
            curve: Curves.easeOut,
          ),
    );
  }

  Widget _buildNotListeningIndicator() {
    return Positioned(
      left: 0,
      right: 0,
      top: 64,
      child: Center(
        child: Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            shape: BoxShape.circle,
            color: darkBackground.withValues(alpha: 0.9),
            border: Border.all(
              color: Colors.grey.withValues(alpha: 0.6),
              width: 2,
            ),
            boxShadow: [
              BoxShadow(
                color: Colors.grey.withValues(alpha: 0.2),
                blurRadius: 15,
                spreadRadius: 2,
              ),
            ],
          ),
          child: const Icon(
            Icons.mic_off,
            color: Colors.grey,
            size: 48,
          ),
        )
            .animate(onPlay: (controller) => controller.repeat(reverse: true))
            .scale(
              begin: const Offset(1.0, 1.0),
              end: const Offset(1.05, 1.05),
              duration: const Duration(milliseconds: 1500),
              curve: Curves.easeInOut,
            )
            .animate()
            .fadeIn(duration: const Duration(milliseconds: 200))
            .slideY(
              begin: -0.5,
              duration: const Duration(milliseconds: 300),
              curve: Curves.easeOut,
            ),
      ),
    );
  }

  Widget _buildScaffold() {
    return Scaffold(
      backgroundColor: Colors.transparent,
      body: Stack(
        children: [
          Positioned.fill(
            child: Container(
              decoration: BoxDecoration(
                border: Border.all(
                    color: Colors.limeAccent.withValues(alpha: .2), width: 1),
              ),
            ),
          ),
          if (tunnelVisionEnabled &&
              (_tunnelVisionAnimatedPos != null || eyes != null))
            _buildTunnelVision(),
          if (ghostVisible && eyes != null) _buildGhost(),
          if (screenshot != null) _buildMouseRects(),
          if (aiResponse != null) _buildAiResponse(),
          if (dictation != null) _buildDictationPanel(),
          if (commandsList != null) _buildCommandsList(),
          if (commandConfig != null) _buildCommandConfig(),
          if (modifierKeys != null) _buildModifierKeys(),
          if (isRecording) _buildRecordingIndicator(),
          if (isNotListening && !isRecording) _buildNotListeningIndicator(),
          if (isMouseDown) _buildMouseDownIndicator(),
          if (_texts.isNotEmpty)
            Positioned(
              left: 0,
              right: 0,
              bottom: 32,
              child: Center(
                child: ConstrainedBox(
                  constraints: BoxConstraints(
                    maxWidth: MediaQuery.of(context).size.width * 0.6,
                  ),
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 20,
                      vertical: 12,
                    ),
                    decoration: BoxDecoration(
                      borderRadius: BorderRadius.circular(8),
                      color: darkBackground.withValues(alpha: 0.85),
                      border: Border.all(
                        color: Colors.white.withValues(alpha: 0.1),
                        width: 1,
                      ),
                    ),
                    child: Text(
                      _texts.map((t) => t.text).join(' '),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      textAlign: TextAlign.center,
                      style: TextStyle(
                        fontSize: 16,
                        fontFamily: 'monospace',
                        color: Colors.white.withValues(alpha: 0.7),
                        fontWeight: FontWeight.w500,
                        letterSpacing: 0.5,
                      ),
                    ),
                  )
                      .animate(key: ValueKey(_texts.map((t) => t.id).join()))
                      .fadeIn(duration: const Duration(milliseconds: 150))
                      .slideY(
                        begin: 0.3,
                        duration: const Duration(milliseconds: 200),
                        curve: Curves.easeOut,
                      ),
                ),
              ),
            ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return _buildScaffold();
  }
}

/// Custom painter for tunnel vision effect
/// Creates a dark overlay with a transparent gradient circle around the eye position
class TunnelVisionPainter extends CustomPainter {
  final Offset center;
  final double radius;
  final double opacity;
  final double blendDistance;

  TunnelVisionPainter({
    required this.center,
    required this.radius,
    required this.opacity,
    required this.blendDistance,
  });

  @override
  void paint(Canvas canvas, Size size) {
    // Save the canvas state
    canvas.saveLayer(Rect.fromLTWH(0, 0, size.width, size.height), Paint());

    // Fill the entire canvas with semi-transparent black
    final backgroundPaint = Paint()
      ..color = Colors.black.withValues(alpha: opacity);
    canvas.drawRect(
        Rect.fromLTWH(0, 0, size.width, size.height), backgroundPaint);

    // Create a radial gradient for the cutout
    // blendDistance controls how much of the radius is used for the gradient blend
    // 0.0 = sharp edge, 1.0 = gradient starts from center
    final innerStop = 1.0 - blendDistance;
    final midStop1 = innerStop + (blendDistance * 0.6);
    final midStop2 = innerStop + (blendDistance * 0.9);

    final cutoutPaint = Paint()
      ..blendMode = BlendMode.dstOut
      ..shader = RadialGradient(
        colors: [
          Colors.black, // Fully cut out in center
          Colors.black, // Solid until blend starts
          Colors.black.withValues(alpha: 0.6),
          Colors.black.withValues(alpha: 0.2),
          Colors.transparent, // No cutout at edge (smooth blend)
        ],
        stops: [0.0, innerStop, midStop1, midStop2, 1.0],
      ).createShader(Rect.fromCircle(center: center, radius: radius));

    // Draw the gradient circle to cut out from the dark overlay
    canvas.drawCircle(center, radius, cutoutPaint);

    // Restore the canvas
    canvas.restore();
  }

  @override
  bool shouldRepaint(TunnelVisionPainter oldDelegate) {
    return oldDelegate.center != center ||
        oldDelegate.radius != radius ||
        oldDelegate.opacity != opacity ||
        oldDelegate.blendDistance != blendDistance;
  }
}
