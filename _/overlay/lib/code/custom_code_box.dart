import 'dart:async';
import 'dart:math';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:linked_scroll_controller/linked_scroll_controller.dart';
import 'package:mouse_whisperer/app_theme.dart';
import 'package:mouse_whisperer/code/code_controller.dart';
import 'package:mouse_whisperer/code/line_number_controller.dart';
import 'themes/a11y-dark.dart';
import 'themes/agate.dart';
import 'themes/an-old-hope.dart';
import 'themes/androidstudio.dart';
import 'themes/arta.dart';
import 'themes/atelier-plateau-dark.dart';
import 'themes/atelier-sulphurpool-dark.dart';
import 'themes/atom-one-dark-reasonable.dart';
import 'themes/atom-one-dark.dart';
import 'themes/github-dark-dimmed.dart';
import 'themes/github-dark.dart';
import 'themes/devibeans.dart';
import 'themes/felipec.dart';
import 'themes/hybrid.dart';
import 'themes/ir-black.dart';
import 'themes/lioshi.dart';
import 'themes/monokai-sublime.dart';
import 'themes/monokai.dart';
import 'themes/night-owl.dart';
import 'themes/nord.dart';
import 'themes/obsidian.dart';
import 'themes/panda-syntax-dark.dart';
import 'themes/paraiso-dark.dart';
import 'themes/qtcreator-dark.dart';
import 'themes/rainbow.dart';
import 'themes/shades-of-purple.dart';
import 'themes/srcery.dart';
import 'themes/stackoverflow-dark.dart';
import 'themes/sunburst.dart';
import 'themes/tokyo-night-dark.dart';
import 'themes/tomorrow-night-blue.dart';
import 'themes/tomorrow-night-bright.dart';
import 'themes/vs2015.dart';

Color get darkBackground =>
    codeThemes["an-old-hope"]!["root"]!.backgroundColor!;

const codeThemes = {
  'an-old-hope': anOldHopeTheme,
  'panda-syntax-dark': pandaSyntaxDarkTheme,
  'night-owl': nightOwlTheme,
  'github-dark': githubDarkTheme,
  'github-dark-dimmed': githubDarkDimmedTheme,
  'monokai': monokaiTheme,
  'monokai-sublime': monokaiSublimeTheme,
  'atom-one-dark': atomOneDarkTheme,
  'atom-one-dark-reasonable': atomOneDarkReasonableTheme,
  'androidstudio': androidstudioTheme,
  'stackoverflow-dark': stackoverflowDarkTheme,
  'vs2015': vs2015Theme,
  'ir-black': irBlackTheme,
  'felipec': felipecTheme,
  'hybrid': hybridTheme,
  'arta': artaTheme,
  'a11y-dark': a11yDarkTheme,
  'devibeans': devibeansTheme,
  'rainbow': rainbowTheme,
  'agate': agateTheme,
  'shades-of-purple': shadesOfPurpleTheme,
  'lioshi': lioshiTheme,
  'nord': nordTheme,
  'obsidian': obsidianTheme,
  'paraiso-dark': paraisoDarkTheme,
  'qtcreator-dark': qtcreatorDarkTheme,
  'sunburst': sunburstTheme,
  'tokyo-night-dark': tokyoNightDarkTheme,
  'tomorrow-night-blue': tomorrowNightBlueTheme,
  'tomorrow-night-bright': tomorrowNightBrightTheme,
  'srcery': srceryTheme,
  'atelier-plateau-dark': atelierPlateauDarkTheme,
  'atelier-sulphurpool-dark': atelierSulphurpoolDarkTheme,
};

const lineNumberColumnExtraWidth = 8.0;
const EdgeInsets lineNumberPadding = EdgeInsets.only(left: 4, right: 8);
const EdgeInsets codeBackgroundPadding =
    EdgeInsets.only(left: 8, top: 6, bottom: 12, right: 8);

double get lineHeight {
  return characterSize.height;
}

double get charH {
  return characterSize.height;
}

double get charW {
  return characterSize.width;
}

Size? _charSize;

Size get characterSize {
  _charSize ??= measureText("W", AppTheme.codeStyleBase);
  return _charSize!;
}

Size measureText(String text, TextStyle style,
    {double maxWidth = double.infinity}) {
  TextSpan span = TextSpan(style: style, text: text);
  TextPainter textPainter = TextPainter(
      text: span, textAlign: TextAlign.left, textDirection: TextDirection.ltr);
  textPainter.layout(maxWidth: maxWidth);
  return textPainter.size;
}

class CustomCodeBox extends StatefulWidget {
  const CustomCodeBox({
    super.key,
    required this.language,
    this.backgroundColor,
    this.startingContent,
    this.selectionControls,
    this.contextMenuBuilder,
    this.focusNode,
    this.minLines,
    this.maxLines,
    this.onTap,
    this.onTextChanged,
    this.onAnyChanged,
    this.onSubmitted,
    this.onScroll,
    this.textAlignment = TextAlign.start,
    this.readOnly = false,
    this.wrap = false,
    this.lineNumbers = true,
    this.lineNumberOffset = 0,
    this.textStyle,
    this.highlightBuilder,
    this.clip = true,
    this.theme = "an-old-hope",
    this.enabled = true,
    this.fontColorOverride,
  });

  final bool wrap;
  final bool readOnly;
  final int? minLines;
  final int? maxLines;
  final int lineNumberOffset;
  final FocusNode? focusNode;
  final VoidCallback? onTap;
  final ValueSetter<String>? onTextChanged;
  final ValueSetter<String>? onSubmitted;
  final VoidCallback? onAnyChanged;
  final ValueSetter<double>? onScroll;
  final TextAlign textAlignment;
  final String language;
  final String theme;
  final String? startingContent;
  final TextSelectionControls? selectionControls;
  final EditableTextContextMenuBuilder? contextMenuBuilder;
  final bool lineNumbers;
  final Color? backgroundColor;
  final TextStyle? textStyle;
  final bool clip;
  final bool enabled;
  final Color? fontColorOverride;
  final WidgetBuilder? highlightBuilder;

  @override
  CustomCodeBoxState createState() => CustomCodeBoxState();
}

class CustomCodeBoxState extends State<CustomCodeBox> {
  static const _rootKey = 'root';
  late FocusNode _focusNode;
  late CodeController _textController;
  final _scrollController = ScrollController();
  final _waitFirstFrame = Completer();
  late TextStyle textStyle;
  late Map<String, TextStyle> styles;
  late LinkedScrollControllerGroup _controllers;
  late ScrollController _numberScroll;
  late ScrollController _codeScroll;
  late LineNumberController _numberController;
  late Color cursorColor;
  late List<String> _lines;
  late double numberColumnWidth;

  String get text => _textController.text;

  TextSelection get cursorPosition => _textController.getCursor();

  set cursorPosition(TextSelection cursor) {
    try {
      _textController.setCursor(cursor);
    } catch (e, stack) {
      final text = _textController.text;
    }
  }

  set text(String t) {
    _textController.text = t;
  }

  TextEditingController get controller => _textController;

  String get _codeColorTheme => widget.theme;

  @override
  void initState() {
    super.initState();
    styles = {...codeThemes[_codeColorTheme]!};
    styles[_rootKey] =
        TextStyle(color: codeThemes[_codeColorTheme]![_rootKey]!.color);
    textStyle = (widget.textStyle ?? AppTheme.codeStyleBase).copyWith(
      color: widget.fontColorOverride ?? styles[_rootKey]!.color,
    );
    _textController = CodeController(
      text: widget.startingContent,
      language: widget.language,
      styles: styles,
    );
    WidgetsBinding.instance.addPostFrameCallback((timeStamp) async {
      _waitFirstFrame.complete();
    });
    _recomputeSize();

    _populateStyles();
    _controllers = LinkedScrollControllerGroup();
    _numberScroll = _controllers.addAndGet();
    _codeScroll = _controllers.addAndGet();
    _numberController = LineNumberController(null);
    _textController.addListener(_onTextChanged);
    _focusNode = widget.focusNode ?? FocusNode();
    _onTextChanged(refresh: false);
  }

  bool _suppressScroll = false;

  void _onScroll() {
    if (_suppressScroll) {
      return;
    }
    widget.onScroll?.call(_scrollController.offset);
  }

  void jumpTo(double position, {bool animate = false}) async {
    await _waitFirstFrame.future;
    _suppressScroll = true;
    WidgetsBinding.instance.addPostFrameCallback((timeStamp) {
      _suppressScroll = false;
    });
    if (animate) {
      unawaited(_scrollController.animateTo(position,
          duration: const Duration(milliseconds: 200), curve: Curves.linear));
    } else {
      _scrollController.jumpTo(position);
    }
  }

  @override
  void dispose() {
    _textController.removeListener(_onTextChanged);
    _textController.dispose();
    _numberScroll.dispose();
    _codeScroll.dispose();
    _numberController.dispose();
    super.dispose();
  }

  Size measure() {
    return Size(width, height);
  }

  late double _textHeight;
  late double _textWidth;
  String _longestLine = '';

  void _recomputeSize() {
    _longestLine = '';
    final lines = _textController.text.split('\n');
    final buf = <String>[];

    for (var k = 0; k < lines.length; k++) {
      buf.add((k + 1).toString());
    }
    // Find longest line
    _longestLine = '';
    for (var line in lines) {
      if (line.length > _longestLine.length) _longestLine = line;
    }

    // Note this isn't a perfect measure but about 10x quicker than an actual measure.
    // Height seems to be fine, but width has a minor inaccuracy per line (between 10 - 1).
    _textWidth = charW * _longestLine.length;
    _textHeight = lineHeight * lines.length;
  }

  double get height => _textHeight + codeBackgroundPadding.vertical;

  double get width =>
      (!widget.lineNumbers
          ? codeBackgroundPadding.horizontal
          : codeBackgroundPadding.right) +
      _textWidth +
      numberColumnWidth +
      lineNumberPadding.horizontal +
      16;

  // Wrap the codeField in a horizontal scrollView
  Widget _wrapInHorizontalScrollView(
      {required ScrollPhysics physics, required Widget child}) {
    if (widget.wrap) {
      return child;
    }

    return LayoutBuilder(builder: (context, constraints) {
      return SingleChildScrollView(
        clipBehavior: widget.clip ? Clip.hardEdge : Clip.none,
        scrollDirection: Axis.horizontal,

        physics: _lockHorizontalScroll
            ? const NeverScrollableScrollPhysics()
            : physics,
        // physics: physics,
        child: SizedBox(
          width: max(constraints.maxWidth, width),
          child: child,
        ),
      );
    });
  }

  void _populateStyles() {
    // Cursor style
    cursorColor = styles[_rootKey]!.color!;
  }

  void _onTextChanged({bool refresh = true}) async {
    // Rebuild line number
    _lines = _textController.text.split('\n');
    final buf = <String>[];

    for (var k = 0; k < _lines.length; k++) {
      buf.add((k + 1 + widget.lineNumberOffset).toString());
    }

    final lastNum = buf.last;
    TextSpan span = TextSpan(style: textStyle, text: lastNum);
    TextPainter textPainter = TextPainter(
        text: span,
        textAlign: TextAlign.left,
        textDirection: TextDirection.ltr);
    textPainter.layout();

    // Add some padding to prevent overflow by minor symbol size differences
    numberColumnWidth = textPainter.width + lineNumberColumnExtraWidth;
    _numberController.text = buf.join('\n');
    if (refresh) {
      // Always called even on text selection
      widget.onAnyChanged?.call();
      setState(() {});
    }
  }

  Widget _buildLineNumbers() {
    return RepaintBoundary(
      child: Container(
        padding: lineNumberPadding,
        // color: lineNumberStyle.background,
        child: SizedBox(
          width: numberColumnWidth,
          child: Text(
            _numberController.text,
            style: textStyle,
            textAlign: TextAlign.end,
          ),
        ),
      ),
    );
  }

  bool _lockHorizontalScroll = false;

  Widget _buildTextField() {
    return Container(
      // color: widget.backgroundColor,
      padding: !widget.lineNumbers
          ? codeBackgroundPadding
          : codeBackgroundPadding.copyWith(left: 0),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (widget.lineNumbers) _buildLineNumbers(),
          Expanded(
            child: Listener(
              onPointerDown: (d) {
                if (mounted) {
                  setState(() {
                    _lockHorizontalScroll = true;
                  });
                }
              },
              onPointerUp: (d) {
                if (mounted) {
                  setState(() {
                    _lockHorizontalScroll = false;
                  });
                }
              },
              child: TextField(
                smartDashesType: null,
                textAlign: widget.textAlignment,
                smartQuotesType: null,
                focusNode: _focusNode,
                scrollPhysics: const NeverScrollableScrollPhysics(),
                onTap: widget.onTap,
                scrollPadding: EdgeInsets.zero,
                style: textStyle,
                controller: _textController,
                minLines: widget.minLines,
                selectionControls: widget.selectionControls,
                maxLines: widget.maxLines,
                expands: false,
                scrollController: _codeScroll,
                contextMenuBuilder: widget.contextMenuBuilder,
                magnifierConfiguration: TextMagnifierConfiguration.disabled,
                enableInteractiveSelection: true,
                keyboardType: TextInputType.multiline,
                decoration: null,
                scribbleEnabled: false,
                // onAppPrivateCommand: (_, __){},
                enableIMEPersonalizedLearning: false,
                cursorColor: cursorColor,
                obscureText: false,
                autocorrect: false,
                enableSuggestions: false,
                enabled: widget.enabled,
                onChanged: (t) {
                  _recomputeSize();
                  widget.onTextChanged?.call(t);
                },
                onSubmitted: widget.onSubmitted,
                readOnly: widget.readOnly,
                clipBehavior: Clip.none,
              ),
            ),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    const physics = ClampingScrollPhysics();
    return ColoredBox(
      color: widget.backgroundColor ?? darkBackground,
      child: NotificationListener<ScrollNotification>(
        onNotification: (n) {
          _onScroll();
          return true;
        },
        child: SingleChildScrollView(
          controller: _scrollController,
          physics: physics,
          child: _wrapInHorizontalScrollView(
            physics: physics,
            child: Stack(children: [
              if (widget.highlightBuilder != null)
                Positioned.fill(child: widget.highlightBuilder!(context)),
              _buildTextField(),
            ]),
          ),
        ),
      ),
    );
  }
}

class NoActionsTextFormatter extends TextInputFormatter {
  @override
  TextEditingValue formatEditUpdate(
    TextEditingValue oldValue,
    TextEditingValue newValue,
  ) {
    return newValue;
  }
}
