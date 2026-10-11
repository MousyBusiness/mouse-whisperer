import 'dart:io';
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:mouse_companion/extensions.dart';
import 'package:mouse_companion/theme/app_colors.dart';

class ScribbleInput extends StatefulWidget {
  const ScribbleInput({super.key, this.onTextChanged, this.onSubmitted});

  final void Function(String text, Offset offset)? onTextChanged;
  final void Function(String text, Offset offset)? onSubmitted;

  @override
  State<ScribbleInput> createState() => _ScribbleInputState();
}

class _ScribbleInputState extends State<ScribbleInput>
    with TextInputClient, SingleTickerProviderStateMixin
    implements ScribbleClient {
  // Key to get widget bounds for ScribbleClient
  final GlobalKey _widgetKey = GlobalKey();

  // Text input connection to the platform
  TextInputConnection? _textInputConnection;

  // Current text editing state
  TextEditingValue _textEditingValue = TextEditingValue.empty;

  // Track last sent text to avoid duplicates
  String _lastSentText = '';

  late final AnimationController _controller;

  // Heuristic for iPad corner radius
  double get _cornerRadius => Platform.isIOS ? 20.0 : 8.0;

  @override
  void initState() {
    super.initState();
    debugPrint(
      '[Scribble] initState - registering element: $elementIdentifier',
    );

    _attachTextInput();

    // Initialize with any provided text
    _textEditingValue = TextEditingValue(
      text: '',
      selection: TextSelection.collapsed(offset: 0),
    );

    _controller = AnimationController(
      vsync: this,
      duration: const Duration(seconds: 4),
    )..repeat();

    WidgetsBinding.instance.addPostFrameCallback((timeStamp) {
      // Register this ScribbleClient with the system
      TextInput.registerScribbleElement(elementIdentifier, this);
      debugPrint('[Scribble] Registered ScribbleClient with bounds: $bounds');
    });
  }

  @override
  void dispose() {
    debugPrint(
      '[Scribble] dispose - unregistering element: $elementIdentifier',
    );
    // Unregister ScribbleClient
    TextInput.unregisterScribbleElement(elementIdentifier);

    // Close text input connection
    _detachTextInput();

    _controller.dispose();

    super.dispose();
  }

  void _attachTextInput() {
    if (_textInputConnection != null && _textInputConnection!.attached) {
      debugPrint('[Scribble] TextInput already attached, skipping');
      return;
    }

    debugPrint('[Scribble] Attaching TextInputConnection');
    _textInputConnection = TextInput.attach(
      this,
      const TextInputConfiguration(
        inputType: TextInputType.text,
        inputAction: TextInputAction.done,
        autocorrect: false,
        enableSuggestions: false,
      ),
    );

    _textInputConnection!.show();
    _textInputConnection!.setEditingState(_textEditingValue);
    debugPrint('[Scribble] TextInputConnection attached and shown');
  }

  void _detachTextInput() {
    debugPrint('[Scribble] Detaching TextInputConnection');
    _textInputConnection?.close();
    _textInputConnection = null;
  }

  Rect _getWidgetBounds() {
    final renderBox =
        _widgetKey.currentContext?.findRenderObject() as RenderBox?;
    if (renderBox == null) {
      return Rect.zero;
    }
    final offset = renderBox.localToGlobal(Offset.zero);
    return offset & renderBox.size;
  }

  // ============================================
  // ScribbleClient Implementation
  // ============================================

  @override
  String get elementIdentifier => 'scribble_input_$hashCode';

  @override
  Rect get bounds => _getWidgetBounds();

  Offset? offset;
  @override
  bool isInScribbleRect(Rect rect) {
    final center = rect.center;
    final contains = bounds.contains(center);
    debugPrint(
      '[Scribble] isInScribbleRect: rect=$rect, bounds=$bounds, center=$center, contains=$contains',
    );

    return contains;
  }

  @override
  void onScribbleFocus(Offset offset) {
    debugPrint('[Scribble] 🖊️ onScribbleFocus called at offset: $offset');
    this.offset = offset;

    // Reset tracking for new scribble session
    _lastSentText = '';

    // Ensure the text input connection is active
    if (_textInputConnection == null || !_textInputConnection!.attached) {
      _attachTextInput();
      debugPrint('[Scribble] TextInputConnection attached and shown');
    } else {
      _textInputConnection!.show();
      debugPrint('[Scribble] TextInputConnection shown');
    }
  }

  // ============================================
  // TextInputClient Implementation
  // ============================================

  @override
  TextEditingValue? get currentTextEditingValue => _textEditingValue;

  @override
  AutofillScope? get currentAutofillScope => null;

  @override
  void updateEditingValue(TextEditingValue value) {
    if (_textEditingValue == value) return;

    debugPrint(
      '[Scribble] 📝 updateEditingValue: "${value.text}" selection=${value.selection}',
    );

    _textEditingValue = value;

    // Extract complete words (text before last space, or entire text if it ends with space)
    final text = value.text;
    String completeWords = '';
    String remainingText = text;

    // Check if text ends with space - means word is complete
    if (text.endsWith(' ') || text.endsWith('\n')) {
      completeWords = text.trimRight();
      remainingText = '';
    } else {
      // Find last space and extract complete words before it
      final lastSpaceIndex = text.lastIndexOf(' ');
      if (lastSpaceIndex > 0) {
        completeWords = text.substring(0, lastSpaceIndex).trim();
        remainingText = text.substring(lastSpaceIndex + 1);
      }
    }

    // Only send if we have complete words that weren't sent before
    if (completeWords.isNotEmpty && completeWords != _lastSentText) {
      // Extract only the new words (difference from last sent)
      String newWords = completeWords;
      if (_lastSentText.isNotEmpty && completeWords.startsWith(_lastSentText)) {
        newWords = completeWords.substring(_lastSentText.length).trim();
      }

      if (newWords.isNotEmpty) {
        debugPrint('[Scribble] Sending complete words: "$newWords"');
        final callbackOffset = offset ?? bounds.center;
        widget.onTextChanged?.call(newWords, callbackOffset);
        _lastSentText = completeWords;
      }
    }

    // Update text input to only keep the remaining incomplete word
    _textEditingValue = TextEditingValue(
      text: remainingText,
      selection: TextSelection.collapsed(offset: remainingText.length),
    );
    _textInputConnection!.setEditingState(_textEditingValue);
  }

  @override
  void performAction(TextInputAction action) {
    debugPrint('[Scribble] performAction: $action');
    switch (action) {
      case TextInputAction.done:
      case TextInputAction.go:
      case TextInputAction.send:
      case TextInputAction.search:
        // Use center of bounds as default offset if onScribbleFocus hasn't been called yet
        final callbackOffset = offset ?? bounds.center;
        widget.onSubmitted?.call(_textEditingValue.text, callbackOffset);
        break;
      case TextInputAction.next:
        break;
      case TextInputAction.previous:
        break;
      default:
        break;
    }
  }

  @override
  void performPrivateCommand(String action, Map<String, dynamic> data) {
    debugPrint('[Scribble] performPrivateCommand: action=$action, data=$data');
  }

  @override
  void updateFloatingCursor(RawFloatingCursorPoint point) {
    debugPrint(
      '[Scribble] updateFloatingCursor: state=${point.state}, offset=${point.offset}',
    );
  }

  @override
  void showAutocorrectionPromptRect(int start, int end) {
    debugPrint(
      '[Scribble] showAutocorrectionPromptRect: start=$start, end=$end',
    );
  }

  @override
  void connectionClosed() {
    debugPrint('[Scribble] connectionClosed called');
    if (_textInputConnection?.attached ?? false) {
      _textInputConnection?.connectionClosedReceived();
    }
  }

  // ============================================
  // Scribble-Specific TextInputClient Methods
  // ============================================

  @override
  void insertTextPlaceholder(Size size) {
    debugPrint('[Scribble] 🖊️ insertTextPlaceholder: size=$size');
    // Called when iOS Scribble needs space for handwriting input
    setState(() {
      // _hasPlaceholder = true;
      // _placeholderSize = size;
    });
  }

  @override
  void removeTextPlaceholder() {
    debugPrint(
      '[Scribble] 🖊️ removeTextPlaceholder - Scribble recognition complete',
    );
    // Called when Scribble recognition is complete
    // Reset tracking for next scribble session
    _lastSentText = '';
    setState(() {
      // _hasPlaceholder = false;
      // _placeholderSize = null;
      // _isScribbling = false;
    });
  }

  @override
  void showToolbar() {
    debugPrint('[Scribble] 🖊️ showToolbar - iOS requested toolbar display');
    // Called when iOS requests showing the text editing toolbar
    // This would typically show copy/paste/select all options
  }

  // ============================================
  // Additional TextInputClient Methods
  // ============================================

  @override
  void performSelector(String selectorName) {
    debugPrint('[Scribble] performSelector: $selectorName');
  }

  @override
  void insertContent(KeyboardInsertedContent content) {
    debugPrint(
      '[Scribble] insertContent: mimeType=${content.mimeType}, uri=${content.uri}',
    );
  }

  @override
  void didChangeInputControl(
    TextInputControl? oldControl,
    TextInputControl? newControl,
  ) {
    debugPrint(
      '[Scribble] didChangeInputControl: old=$oldControl, new=$newControl',
    );
  }

  // ============================================
  // Build UI
  // ============================================

  @override
  Widget build(BuildContext context) {
    return AnimatedBuilder(
      key: _widgetKey,

      animation: _controller,
      builder: (context, child) {
        return CustomPaint(
          painter: _GradientBorderPainter(
            animation: _controller,
            color: AppColors.secondary,
            borderRadius: _cornerRadius,
          ),
          child: Container(color: Colors.transparent),
        );
      },
    );
  }
}

class _GradientBorderPainter extends CustomPainter {
  final Animation<double> animation;
  final Color color;
  final double borderRadius;

  _GradientBorderPainter({
    required this.animation,
    required this.color,
    required this.borderRadius,
  }) : super(repaint: animation);

  @override
  void paint(Canvas canvas, Size size) {
    const strokeWidth = 6.0;
    // Inset by half the stroke width so the border stays inside/centered on the bounds
    final rect = (Offset.zero & size).deflate(strokeWidth);
    final rrect = RRect.fromRectAndRadius(rect, Radius.circular(borderRadius));

    final paint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = strokeWidth
      ..shader = SweepGradient(
        colors: [
          Colors.transparent,
          color.withValues(alpha: 0.2),
          color,
          color.withValues(alpha: 0.2),
          Colors.transparent,
        ],
        stops: const [0.0, 0.2, 0.5, 0.8, 1.0],
        transform: GradientRotation(animation.value * 2 * math.pi),
      ).createShader(rect);

    canvas.drawRRect(rrect, paint);
  }

  @override
  bool shouldRepaint(covariant _GradientBorderPainter oldDelegate) {
    return oldDelegate.animation != animation ||
        oldDelegate.color != color ||
        oldDelegate.borderRadius != borderRadius;
  }
}
