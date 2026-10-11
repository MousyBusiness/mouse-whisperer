import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:highlight/highlight.dart';
import 'package:highlight/languages/all.dart';
import 'package:mouse_whisperer/code/modifiers.dart';

class CodeController extends TextEditingController {
  CodeController({
    required this.language,
    required this.styles,
    super.text,
  }) {
    const modifiers = [
      IndentModifier(),
      CloseBlockModifier(),
      TabModifier(),
    ];
    // Create modifier map
    for (final el in modifiers) {
      _modifierMap[el.char] = el;
    }

    final l = allLanguages[language] ?? allLanguages['plaintext']!;
    highlight.registerLanguage(language, l);
  }

  final Map<String, TextStyle> styles;
  final EditorParams params = const EditorParams();
  final _modifierMap = <String, CodeModifier>{};
  String language = '';

  TextSelection getCursor() {
    return selection;
  }

  /// Sets a specific cursor position in the text
  void setCursor(TextSelection cursor) {
    selection = cursor;
  }

  /// Replaces the current [selection] by [str]
  void insertStr(String str) {
    final sel = selection;
    text = text.replaceRange(selection.start, selection.end, str);
    final len = str.length;

    selection = sel.copyWith(
      baseOffset: sel.start + len,
      extentOffset: sel.start + len,
    );
  }

  /// Remove the char just before the cursor or the selection
  void removeChar() {
    if (selection.start < 1) {
      return;
    }

    final sel = selection;
    text = text.replaceRange(selection.start - 1, selection.start, '');

    selection = sel.copyWith(
      baseOffset: sel.start - 1,
      extentOffset: sel.start - 1,
    );
  }

  /// Remove the selected text
  void removeSelection() {
    final sel = selection;
    text = text.replaceRange(selection.start, selection.end, '');

    selection = sel.copyWith(
      baseOffset: sel.start,
      extentOffset: sel.start,
    );
  }

  /// Remove the selection or last char if the selection is empty
  void backspace() {
    if (selection.start < selection.end) {
      removeSelection();
    } else {
      removeChar();
    }
  }

  KeyEventResult onKey(RawKeyEvent event) {
    if (event.isKeyPressed(LogicalKeyboardKey.tab)) {
      text = text.replaceRange(selection.start, selection.end, '\t');
      return KeyEventResult.handled;
    }

    return KeyEventResult.ignored;
  }

  int? _insertedLoc(String a, String b) {
    final sel = selection;

    if (a.length + 1 != b.length || sel.start != sel.end) {
      return null;
    }

    return sel.start;
  }

  @override
  set value(TextEditingValue newValue) {
    final loc = _insertedLoc(text, newValue.text);

    if (loc != null && loc != -1) {
      final char = newValue.text[loc];
      final modifier = _modifierMap[char];
      final val = modifier?.updateString(super.text, selection, params);

      if (val != null) {
        // Update newValue
        newValue = newValue.copyWith(
          text: val.text,
          selection: val.selection,
        );
      }
    }
    super.value = newValue;
  }

  TextSpan _processLanguage(
    String text,
    TextStyle? style,
  ) {
    final result = highlight.parse(text, language: language);
    final nodes = result.nodes;

    final children = <TextSpan>[];
    var currentSpans = children;
    final stack = <List<TextSpan>>[];

    void traverse(Node node) {
      var val = node.value;
      final nodeChildren = node.children;
      final nodeStyle = styles[node.className];

      if (val != null) {
        var child = TextSpan(text: val, style: nodeStyle);

        currentSpans.add(child);
      } else if (nodeChildren != null) {
        List<TextSpan> tmp = [];

        currentSpans.add(TextSpan(
          children: tmp,
          style: nodeStyle,
        ));

        stack.add(currentSpans);
        currentSpans = tmp;

        for (final n in nodeChildren) {
          traverse(n);
          if (n == nodeChildren.last) {
            currentSpans = stack.isEmpty ? children : stack.removeLast();
          }
        }
      }
    }

    if (nodes != null) {
      nodes.forEach(traverse);
    }

    return TextSpan(style: style, children: children);
  }

  @override
  TextSpan buildTextSpan({
    required BuildContext context,
    TextStyle? style,
    bool? withComposing,
  }) {
    return _processLanguage(text, style);
  }
}
