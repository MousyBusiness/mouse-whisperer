import 'dart:async';

import 'package:flutter/material.dart';
import 'package:mouse_companion/discovery.dart';
import 'package:mouse_companion/extensions.dart';
import 'package:mouse_companion/gestures.dart';
import 'package:mouse_companion/scribble.dart';
import 'package:mouse_companion/theme/app_colors.dart';
import 'package:mouse_companion/theme/app_theme.dart';
import 'package:mouse_companion/udp.dart';
import 'package:mouse_companion/gen/mouse_whisperer/companion/v1/companion.pb.dart'
    as pb;
import 'package:mouse_companion/gen/google/protobuf/timestamp.pb.dart';

void main() {
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Mouse Whisperer',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: Colors.deepPurple,
          brightness: Brightness.dark,
        ),
      ),
      home: const ConnectionScreen(),
    );
  }
}

/// Screen for discovering and connecting to Mouse Whisperer servers.
class ConnectionScreen extends StatefulWidget {
  const ConnectionScreen({super.key});

  @override
  State<ConnectionScreen> createState() => _ConnectionScreenState();
}

class _ConnectionScreenState extends State<ConnectionScreen> {
  final _discovery = ServiceDiscovery();
  StreamSubscription? _subscription;
  Set<DiscoveredService> _services = {};

  @override
  void initState() {
    super.initState();
    _startDiscovery();
  }

  Future<void> _startDiscovery() async {
    _subscription = _discovery.servicesStream.listen((services) {
      setState(() => _services = services);
    });
    await _discovery.start();
  }

  @override
  void dispose() {
    _subscription?.cancel();
    _discovery.dispose();
    super.dispose();
  }

  void _connectToService(DiscoveredService service) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) =>
            InputScreen(serverAddress: service.ip, serverPort: service.port),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Connections'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: () async {
              await _discovery.stop();
              setState(() => _services = {});
              await _discovery.start();
            },
          ),
        ],
      ),
      body: _services.isEmpty
          ? const Center(
              child: Column(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  CircularProgressIndicator(),
                  SizedBox(height: 16),
                  Text('Searching for servers...'),
                ],
              ),
            )
          : ListView.builder(
              itemCount: _services.length,
              itemBuilder: (context, index) {
                final service = _services.elementAt(index);
                return ListTile(
                  leading: const Icon(Icons.computer),
                  title: Text(service.name),
                  subtitle: Text('${service.ip}:${service.port}'),
                  trailing: const Icon(Icons.arrow_forward_ios),
                  onTap: () => _connectToService(service),
                );
              },
            ),
    );
  }
}

/// Main input screen for sending mouse events.
class InputScreen extends StatefulWidget {
  const InputScreen({
    super.key,
    required this.serverAddress,
    required this.serverPort,
  });

  final String serverAddress;
  final int serverPort;

  @override
  State<InputScreen> createState() => _InputScreenState();
}

class _InputScreenState extends State<InputScreen> {
  final _udpClient = UdpClient();

  // Event aggregation for throttling UDP packets
  static const _aggregationInterval = Duration(milliseconds: 15);
  Timer? _aggregationTimer;

  // Pending aggregated event data
  String? _pendingEventGroupId;
  pb.MouseEvent_EventType? _pendingEventType;
  int _pendingPointerCount = 0;
  double _pendingDeltaX = 0;
  double _pendingDeltaY = 0;

  bool get _hasPendingEvent => _pendingEventGroupId != null;

  @override
  void initState() {
    super.initState();
    _connectUdp();
    _startAggregationTimer();
  }

  void _startAggregationTimer() {
    _aggregationTimer = Timer.periodic(_aggregationInterval, (_) {
      _flushPendingEvent();
    });
  }

  void _flushPendingEvent() {
    if (_hasPendingEvent) {
      final payload = pb.Payload(
        eventGroupId: _pendingEventGroupId,
        timestamp: Timestamp.fromDateTime(DateTime.now()),
        mouse: pb.MouseEvent(
          type: _pendingEventType,
          pointerCount: _pendingPointerCount,
          deltaX: _pendingDeltaX,
          deltaY: _pendingDeltaY,
        ),
      );
      final bytes = payload.writeToBuffer();
      _udpClient.send(bytes);
      _clearPendingEvent();
    }
  }

  void _clearPendingEvent() {
    _pendingEventGroupId = null;
    _pendingEventType = null;
    _pendingPointerCount = 0;
    _pendingDeltaX = 0;
    _pendingDeltaY = 0;
  }

  Future<void> _connectUdp() async {
    debugPrint(
      'Connecting to UDP server at ${widget.serverAddress}:${widget.serverPort}',
    );
    try {
      await _udpClient.connect(
        serverAddress: widget.serverAddress,
        serverPort: widget.serverPort,
      );
      debugPrint('UDP client connected');
    } catch (e) {
      debugPrint('Failed to connect UDP client: $e');
    }
  }

  @override
  void dispose() {
    _aggregationTimer?.cancel();
    _flushPendingEvent(); // Send any remaining event
    _udpClient.close();
    super.dispose();
  }

  void _handleMouseEvent(pb.MouseEvent mouseEvent, String eventGroupId) {
    final eventType = mouseEvent.type;

    // Aggregate drag, scroll (drag with pointerCount > 1), and fling events (high frequency)
    final shouldAggregate =
        eventType == pb.MouseEvent_EventType.EVENT_TYPE_DRAG ||
        eventType == pb.MouseEvent_EventType.EVENT_TYPE_FLING;

    if (shouldAggregate) {
      // Aggregate with pending event if same type, group, and pointer count
      if (_hasPendingEvent &&
          _pendingEventGroupId == eventGroupId &&
          _pendingEventType == eventType &&
          _pendingPointerCount == mouseEvent.pointerCount) {
        // Sum the deltas
        _pendingDeltaX += mouseEvent.deltaX;
        _pendingDeltaY += mouseEvent.deltaY;
      } else {
        // Flush any different pending event first
        _flushPendingEvent();
        // Start new pending event
        _pendingEventGroupId = eventGroupId;
        _pendingEventType = eventType;
        _pendingPointerCount = mouseEvent.pointerCount;
        _pendingDeltaX = mouseEvent.deltaX;
        _pendingDeltaY = mouseEvent.deltaY;
      }
      return;
    }

    // For non-aggregated events (click, long press, finish), flush pending and send immediately
    _flushPendingEvent();
    final payload = pb.Payload(
      eventGroupId: eventGroupId,
      timestamp: Timestamp.fromDateTime(DateTime.now()),
      mouse: mouseEvent,
    );
    final bytes = payload.writeToBuffer();
    _udpClient.send(bytes);
  }

  void _sendScribble(String scribble) {
    final payload = pb.Payload(
      timestamp: Timestamp.fromDateTime(DateTime.now()),
      keyboard: pb.KeyboardEvent(
        type: pb.KeyboardEvent_EventType.EVENT_TYPE_TYPE_STRING,
        typeString: scribble,
      ),
    );
    final bytes = payload.writeToBuffer();
    _udpClient.send(bytes);
  }

  void _sendCommand(String command) {
    final payload = pb.Payload(
      timestamp: Timestamp.fromDateTime(DateTime.now()),
      command: command.toLowerCase().trim(),
    );
    final bytes = payload.writeToBuffer();
    _udpClient.send(bytes);
  }

  final _scribbleController = TextEditingController();
  final _scribbleFocusNode = FocusNode()..requestFocus();

  bool _showKeyboard = false;
  int _scribbleKey = 0; // Bumped on submit to reset ScribbleInput
  final List<String> _commandHistory = [];

  void _addToCommandHistory(String command) {
    if (command.trim().isEmpty) return;
    setState(() {
      // Remove duplicate if exists and add to front
      _commandHistory.remove(command);
      _commandHistory.insert(0, command);
      // Keep only last 10 commands
      if (_commandHistory.length > 10) {
        _commandHistory.removeLast();
      }
    });
  }

  Widget _buildCommandHistory() {
    // Limit to 8 items to fit approximately 2 rows
    final displayHistory = _commandHistory.take(8).toList();
    return Container(
      decoration: BoxDecoration(
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.midGrey),
      ),
      child: ListView.builder(
        itemBuilder: (context, index) {
          final command = displayHistory[index];
          return InputChip(
            label: Text(command),
            deleteIcon: const Icon(Icons.close, size: 18),
            onDeleted: () {
              setState(() {
                _commandHistory.remove(command);
              });
            },
            onPressed: () {
              _sendCommand(command);
            },
          );
        },
      ),
    );
  }

  Widget _buildNavigationKeypad() {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        // Up button
        IconButton(
          onPressed: () => _sendCommand("up"),
          icon: const Icon(Icons.arrow_upward),
          iconSize: 32,
        ),
        // Left, Down, Right buttons
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            IconButton(
              onPressed: () => _sendCommand("left"),
              icon: const Icon(Icons.arrow_back),
              iconSize: 32,
            ),
            IconButton(
              onPressed: () => _sendCommand("down"),
              icon: const Icon(Icons.arrow_downward),
              iconSize: 32,
            ),
            IconButton(
              onPressed: () => _sendCommand("right"),
              icon: const Icon(Icons.arrow_forward),
              iconSize: 32,
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildScribbleInput() {
    return Stack(
      children: [
        Container(
          color: Colors.transparent,
          child: TextField(
            key: ValueKey(_showKeyboard),
            controller: _scribbleController,
            stylusHandwritingEnabled:
                false, // Disabled - ScribbleInput handles scribble
            focusNode: _scribbleFocusNode,
            keyboardType: _showKeyboard
                ? TextInputType.text
                : TextInputType.none,
            maxLines: 2,
            minLines: 2,
            // keyboardType: TextInputType.none,
            // textInputAction: TextInputAction.done,
            decoration: InputDecoration(
              border: OutlineInputBorder(),
              floatingLabelBehavior: FloatingLabelBehavior.always,
              labelText: 'Command / Text',
            ),
          ),
        ).h(80).alignTop,
        ScribbleInput(
          key: ValueKey(_scribbleKey),
          onTextChanged: (text, _) {
            final currentText = _scribbleController.text;
            final cursorPosition = _scribbleController.selection.baseOffset;
            final noPosition = cursorPosition == -1;

            debugPrint('[Scribble] Received text: "$text"');

            // Handle hyphen as deletion
            if (text.replaceAll("-", "").trim().isEmpty) {
              final hyphenCount = text.replaceAll(" ", "").length;
              final toDelete = hyphenCount.clamp(
                0,
                noPosition ? currentText.length : cursorPosition,
              );

              if (toDelete > 0) {
                if (noPosition) {
                  _scribbleController.text = currentText.substring(
                    0,
                    currentText.length - toDelete,
                  );
                } else {
                  final newText =
                      currentText.substring(0, cursorPosition - toDelete) +
                      currentText.substring(cursorPosition);
                  final newCursorPosition = cursorPosition - toDelete;

                  _scribbleController.value = TextEditingValue(
                    text: newText,
                    selection: TextSelection.collapsed(
                      offset: newCursorPosition,
                    ),
                  );
                }
              }
              return;
            }

            // Add the text
            final normalizedText = text.toLowerCase().trim();
            if (normalizedText.isEmpty) return;

            if (_scribbleController.text.isEmpty) {
              _scribbleController.text = normalizedText;
            } else {
              if (noPosition || cursorPosition >= currentText.length) {
                _scribbleController.text += " $normalizedText";
              } else {
                final newText =
                    "${currentText.substring(0, cursorPosition)} $normalizedText${currentText.substring(cursorPosition)}";
                final newCursorPosition =
                    cursorPosition + normalizedText.length + 1;

                _scribbleController.value = TextEditingValue(
                  text: newText,
                  selection: TextSelection.collapsed(
                    offset: newCursorPosition.clamp(0, newText.length),
                  ),
                );
              }
            }
          },
          onSubmitted: (text, offset) {
            // _sendScribble(text);
          },
        ).posFill(top: 80),
      ],
    );
  }

  Widget _buildInputs() {
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _buildScribbleInput().h(200),
          Wrap(
            spacing: 8,
            runSpacing: 8,

            children: [
              IconButton(
                onPressed: () {
                  setState(() {
                    _showKeyboard = !_showKeyboard;
                  });
                },
                icon: const Icon(Icons.keyboard),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _scribbleController.clear();
                  _scribbleFocusNode.requestFocus();
                },
                icon: const Icon(Icons.clear),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("undo");
                },
                icon: const Icon(Icons.undo),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("redo");
                },
                icon: const Icon(Icons.redo),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("delete that");
                },
                icon: const Icon(Icons.backspace),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("copy");
                },
                icon: const Icon(Icons.content_copy),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("paste that");
                },
                icon: const Icon(Icons.content_paste),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("ghost");
                },
                icon: const Icon(Icons.circle_outlined),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("refresh");
                },
                icon: const Icon(Icons.refresh),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("terminal search");
                },
                icon: const Icon(Icons.terminal),
              ),
              SizedBox(width: 8),
              IconButton(
                onPressed: () {
                  _sendCommand("select click");
                },
                icon: const Icon(Icons.mouse),
              ),
              SizedBox(width: 24),
              IconButton(
                onPressed: () {
                  _sendCommand("select");
                },
                icon: const Icon(Icons.select_all),
              ),
              SizedBox(width: 24),
              IconButton(
                onPressed: () {
                  _sendCommand("enter");
                },
                icon: const Icon(Icons.keyboard_return),
              ),
              SizedBox(width: 24),
              ElevatedButton(
                onPressed: () {
                  final command = _scribbleController.text;
                  _addToCommandHistory(command);
                  _sendCommand(command);
                  setState(() {
                    _scribbleKey++; // Reset ScribbleInput
                  });
                },
                child: const Text('Command'),
              ),
              SizedBox(width: 8),
              ElevatedButton(
                onPressed: () {
                  _sendScribble(_scribbleController.text);
                  _scribbleController.clear();
                  _scribbleFocusNode.requestFocus();
                  setState(() {
                    _scribbleKey++; // Reset ScribbleInput
                  });
                },
                child: const Text('Type'),
              ),
            ],
          ).w(double.infinity).t(8),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final isIpad = MediaQuery.of(context).size.width > 400;
    debugPrint('isIpad: $isIpad');
    return Scaffold(
      body: Stack(
        children: [
          Gestures(
            onMouseEvent: _handleMouseEvent,
            child: Container(
              color: Colors.black,
              child: Center(
                child:
                    Text(
                      'Mouse Whisperer',
                      style: AppTheme.body1.copyWith(
                        color: Colors.white.withValues(alpha: 0.1),
                      ),
                    ).shimmer(
                      Duration(milliseconds: 2000),
                      Duration(milliseconds: 3000),
                    ),
              ),
            ),
          ).posFill(),
          if (isIpad)
            Row(
              children: [
                _buildInputs().expand,
                // _buildCommandHistory().w(160),
                SizedBox(width: 16),
                _buildNavigationKeypad().h(200).w(160).center,
              ],
            ).pos(top: 32, left: 8, right: 0),
          IconButton(
            icon: Icon(
              Icons.home_outlined,
              size: 20,
              color: Colors.white.withValues(alpha: 0.3),
            ),
            onPressed: () => Navigator.of(context).pop(),
          ).pos(top: 32, right: 8),
        ],
      ),
    );
  }
}
