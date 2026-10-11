//
//  Generated code. Do not modify.
//  source: mouse_whisperer/overlay/v1/overlay.proto
//
// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

enum DisplayMessage_Content {
  commandText, 
  commandsList, 
  commandConfig, 
  notSet
}

/// DisplayMessage is sent to show text content in the UI
class DisplayMessage extends $pb.GeneratedMessage {
  factory DisplayMessage({
    $core.String? commandText,
    CommandsList? commandsList,
    CommandConfig? commandConfig,
  }) {
    final result = create();
    if (commandText != null) result.commandText = commandText;
    if (commandsList != null) result.commandsList = commandsList;
    if (commandConfig != null) result.commandConfig = commandConfig;
    return result;
  }

  DisplayMessage._();

  factory DisplayMessage.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory DisplayMessage.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, DisplayMessage_Content> _DisplayMessage_ContentByTag = {
    1 : DisplayMessage_Content.commandText,
    2 : DisplayMessage_Content.commandsList,
    3 : DisplayMessage_Content.commandConfig,
    0 : DisplayMessage_Content.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'DisplayMessage', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..oo(0, [1, 2, 3])
    ..aOS(1, _omitFieldNames ? '' : 'commandText')
    ..aOM<CommandsList>(2, _omitFieldNames ? '' : 'commandsList', subBuilder: CommandsList.create)
    ..aOM<CommandConfig>(3, _omitFieldNames ? '' : 'commandConfig', subBuilder: CommandConfig.create)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisplayMessage clone() => DisplayMessage()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisplayMessage copyWith(void Function(DisplayMessage) updates) => super.copyWith((message) => updates(message as DisplayMessage)) as DisplayMessage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DisplayMessage create() => DisplayMessage._();
  @$core.override
  DisplayMessage createEmptyInstance() => create();
  static $pb.PbList<DisplayMessage> createRepeated() => $pb.PbList<DisplayMessage>();
  @$core.pragma('dart2js:noInline')
  static DisplayMessage getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DisplayMessage>(create);
  static DisplayMessage? _defaultInstance;

  DisplayMessage_Content whichContent() => _DisplayMessage_ContentByTag[$_whichOneof(0)]!;
  void clearContent() => $_clearField($_whichOneof(0));

  /// Brief command text shown at bottom of screen (auto-dismiss ~1s)
  @$pb.TagNumber(1)
  $core.String get commandText => $_getSZ(0);
  @$pb.TagNumber(1)
  set commandText($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCommandText() => $_has(0);
  @$pb.TagNumber(1)
  void clearCommandText() => $_clearField(1);

  /// Commands list panel shown on the right (auto-dismiss ~15s, tap to close)
  @$pb.TagNumber(2)
  CommandsList get commandsList => $_getN(1);
  @$pb.TagNumber(2)
  set commandsList(CommandsList value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCommandsList() => $_has(1);
  @$pb.TagNumber(2)
  void clearCommandsList() => $_clearField(2);
  @$pb.TagNumber(2)
  CommandsList ensureCommandsList() => $_ensure(1);

  /// Command config/spec panel (auto-dismiss ~20s, tap to close)
  @$pb.TagNumber(3)
  CommandConfig get commandConfig => $_getN(2);
  @$pb.TagNumber(3)
  set commandConfig(CommandConfig value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasCommandConfig() => $_has(2);
  @$pb.TagNumber(3)
  void clearCommandConfig() => $_clearField(3);
  @$pb.TagNumber(3)
  CommandConfig ensureCommandConfig() => $_ensure(2);
}

class CommandsList extends $pb.GeneratedMessage {
  factory CommandsList({
    $core.String? title,
    $core.Iterable<$core.String>? items,
    $core.int? dismissSeconds,
  }) {
    final result = create();
    if (title != null) result.title = title;
    if (items != null) result.items.addAll(items);
    if (dismissSeconds != null) result.dismissSeconds = dismissSeconds;
    return result;
  }

  CommandsList._();

  factory CommandsList.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory CommandsList.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CommandsList', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'title')
    ..pPS(2, _omitFieldNames ? '' : 'items')
    ..a<$core.int>(3, _omitFieldNames ? '' : 'dismissSeconds', $pb.PbFieldType.O3)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandsList clone() => CommandsList()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandsList copyWith(void Function(CommandsList) updates) => super.copyWith((message) => updates(message as CommandsList)) as CommandsList;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommandsList create() => CommandsList._();
  @$core.override
  CommandsList createEmptyInstance() => create();
  static $pb.PbList<CommandsList> createRepeated() => $pb.PbList<CommandsList>();
  @$core.pragma('dart2js:noInline')
  static CommandsList getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CommandsList>(create);
  static CommandsList? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get title => $_getSZ(0);
  @$pb.TagNumber(1)
  set title($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTitle() => $_has(0);
  @$pb.TagNumber(1)
  void clearTitle() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get items => $_getList(1);

  @$pb.TagNumber(3)
  $core.int get dismissSeconds => $_getIZ(2);
  @$pb.TagNumber(3)
  set dismissSeconds($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDismissSeconds() => $_has(2);
  @$pb.TagNumber(3)
  void clearDismissSeconds() => $_clearField(3);
}

class CommandConfig extends $pb.GeneratedMessage {
  factory CommandConfig({
    $core.String? name,
    $core.String? description,
    $core.Iterable<$core.String>? triggers,
    $core.int? dismissSeconds,
  }) {
    final result = create();
    if (name != null) result.name = name;
    if (description != null) result.description = description;
    if (triggers != null) result.triggers.addAll(triggers);
    if (dismissSeconds != null) result.dismissSeconds = dismissSeconds;
    return result;
  }

  CommandConfig._();

  factory CommandConfig.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory CommandConfig.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'CommandConfig', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'name')
    ..aOS(2, _omitFieldNames ? '' : 'description')
    ..pPS(3, _omitFieldNames ? '' : 'triggers')
    ..a<$core.int>(4, _omitFieldNames ? '' : 'dismissSeconds', $pb.PbFieldType.O3)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandConfig clone() => CommandConfig()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandConfig copyWith(void Function(CommandConfig) updates) => super.copyWith((message) => updates(message as CommandConfig)) as CommandConfig;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommandConfig create() => CommandConfig._();
  @$core.override
  CommandConfig createEmptyInstance() => create();
  static $pb.PbList<CommandConfig> createRepeated() => $pb.PbList<CommandConfig>();
  @$core.pragma('dart2js:noInline')
  static CommandConfig getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<CommandConfig>(create);
  static CommandConfig? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get name => $_getSZ(0);
  @$pb.TagNumber(1)
  set name($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasName() => $_has(0);
  @$pb.TagNumber(1)
  void clearName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get description => $_getSZ(1);
  @$pb.TagNumber(2)
  set description($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDescription() => $_has(1);
  @$pb.TagNumber(2)
  void clearDescription() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get triggers => $_getList(2);

  @$pb.TagNumber(4)
  $core.int get dismissSeconds => $_getIZ(3);
  @$pb.TagNumber(4)
  set dismissSeconds($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDismissSeconds() => $_has(3);
  @$pb.TagNumber(4)
  void clearDismissSeconds() => $_clearField(4);
}

class Toast extends $pb.GeneratedMessage {
  factory Toast({
    $core.String? title,
    $core.String? message,
    $core.String? level,
  }) {
    final result = create();
    if (title != null) result.title = title;
    if (message != null) result.message = message;
    if (level != null) result.level = level;
    return result;
  }

  Toast._();

  factory Toast.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory Toast.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Toast', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'title')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..aOS(3, _omitFieldNames ? '' : 'level')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Toast clone() => Toast()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Toast copyWith(void Function(Toast) updates) => super.copyWith((message) => updates(message as Toast)) as Toast;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Toast create() => Toast._();
  @$core.override
  Toast createEmptyInstance() => create();
  static $pb.PbList<Toast> createRepeated() => $pb.PbList<Toast>();
  @$core.pragma('dart2js:noInline')
  static Toast getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Toast>(create);
  static Toast? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get title => $_getSZ(0);
  @$pb.TagNumber(1)
  set title($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTitle() => $_has(0);
  @$pb.TagNumber(1)
  void clearTitle() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get level => $_getSZ(2);
  @$pb.TagNumber(3)
  set level($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLevel() => $_has(2);
  @$pb.TagNumber(3)
  void clearLevel() => $_clearField(3);
}

class EyesPosition extends $pb.GeneratedMessage {
  factory EyesPosition({
    $core.double? x,
    $core.double? y,
    $core.bool? clear_3,
  }) {
    final result = create();
    if (x != null) result.x = x;
    if (y != null) result.y = y;
    if (clear_3 != null) result.clear_3 = clear_3;
    return result;
  }

  EyesPosition._();

  factory EyesPosition.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory EyesPosition.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'EyesPosition', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..a<$core.double>(1, _omitFieldNames ? '' : 'x', $pb.PbFieldType.OD)
    ..a<$core.double>(2, _omitFieldNames ? '' : 'y', $pb.PbFieldType.OD)
    ..aOB(3, _omitFieldNames ? '' : 'clear')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EyesPosition clone() => EyesPosition()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EyesPosition copyWith(void Function(EyesPosition) updates) => super.copyWith((message) => updates(message as EyesPosition)) as EyesPosition;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EyesPosition create() => EyesPosition._();
  @$core.override
  EyesPosition createEmptyInstance() => create();
  static $pb.PbList<EyesPosition> createRepeated() => $pb.PbList<EyesPosition>();
  @$core.pragma('dart2js:noInline')
  static EyesPosition getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<EyesPosition>(create);
  static EyesPosition? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get x => $_getN(0);
  @$pb.TagNumber(1)
  set x($core.double value) => $_setDouble(0, value);
  @$pb.TagNumber(1)
  $core.bool hasX() => $_has(0);
  @$pb.TagNumber(1)
  void clearX() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get y => $_getN(1);
  @$pb.TagNumber(2)
  set y($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasY() => $_has(1);
  @$pb.TagNumber(2)
  void clearY() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get clear_3 => $_getBF(2);
  @$pb.TagNumber(3)
  set clear_3($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasClear_3() => $_has(2);
  @$pb.TagNumber(3)
  void clearClear_3() => $_clearField(3);
}

class Image extends $pb.GeneratedMessage {
  factory Image({
    $core.String? imageBase64,
    $core.bool? delete,
  }) {
    final result = create();
    if (imageBase64 != null) result.imageBase64 = imageBase64;
    if (delete != null) result.delete = delete;
    return result;
  }

  Image._();

  factory Image.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory Image.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Image', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'imageBase64')
    ..aOB(2, _omitFieldNames ? '' : 'delete')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Image clone() => Image()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Image copyWith(void Function(Image) updates) => super.copyWith((message) => updates(message as Image)) as Image;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Image create() => Image._();
  @$core.override
  Image createEmptyInstance() => create();
  static $pb.PbList<Image> createRepeated() => $pb.PbList<Image>();
  @$core.pragma('dart2js:noInline')
  static Image getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Image>(create);
  static Image? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get imageBase64 => $_getSZ(0);
  @$pb.TagNumber(1)
  set imageBase64($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasImageBase64() => $_has(0);
  @$pb.TagNumber(1)
  void clearImageBase64() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get delete => $_getBF(1);
  @$pb.TagNumber(2)
  set delete($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDelete() => $_has(1);
  @$pb.TagNumber(2)
  void clearDelete() => $_clearField(2);
}

class AIResponse extends $pb.GeneratedMessage {
  factory AIResponse({
    $core.String? prompt,
    $core.String? aiResponse,
    $core.bool? delete,
  }) {
    final result = create();
    if (prompt != null) result.prompt = prompt;
    if (aiResponse != null) result.aiResponse = aiResponse;
    if (delete != null) result.delete = delete;
    return result;
  }

  AIResponse._();

  factory AIResponse.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory AIResponse.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'AIResponse', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'prompt')
    ..aOS(2, _omitFieldNames ? '' : 'aiResponse')
    ..aOB(3, _omitFieldNames ? '' : 'delete')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AIResponse clone() => AIResponse()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AIResponse copyWith(void Function(AIResponse) updates) => super.copyWith((message) => updates(message as AIResponse)) as AIResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AIResponse create() => AIResponse._();
  @$core.override
  AIResponse createEmptyInstance() => create();
  static $pb.PbList<AIResponse> createRepeated() => $pb.PbList<AIResponse>();
  @$core.pragma('dart2js:noInline')
  static AIResponse getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<AIResponse>(create);
  static AIResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get prompt => $_getSZ(0);
  @$pb.TagNumber(1)
  set prompt($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPrompt() => $_has(0);
  @$pb.TagNumber(1)
  void clearPrompt() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get aiResponse => $_getSZ(1);
  @$pb.TagNumber(2)
  set aiResponse($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAiResponse() => $_has(1);
  @$pb.TagNumber(2)
  void clearAiResponse() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get delete => $_getBF(2);
  @$pb.TagNumber(3)
  set delete($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDelete() => $_has(2);
  @$pb.TagNumber(3)
  void clearDelete() => $_clearField(3);
}

/// UIAction is sent to perform actions on the UI (scroll, dismiss, etc.)
class UIAction extends $pb.GeneratedMessage {
  factory UIAction({
    $core.String? action,
    $core.int? amount,
  }) {
    final result = create();
    if (action != null) result.action = action;
    if (amount != null) result.amount = amount;
    return result;
  }

  UIAction._();

  factory UIAction.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory UIAction.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'UIAction', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'action')
    ..a<$core.int>(2, _omitFieldNames ? '' : 'amount', $pb.PbFieldType.O3)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UIAction clone() => UIAction()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UIAction copyWith(void Function(UIAction) updates) => super.copyWith((message) => updates(message as UIAction)) as UIAction;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UIAction create() => UIAction._();
  @$core.override
  UIAction createEmptyInstance() => create();
  static $pb.PbList<UIAction> createRepeated() => $pb.PbList<UIAction>();
  @$core.pragma('dart2js:noInline')
  static UIAction getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<UIAction>(create);
  static UIAction? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get action => $_getSZ(0);
  @$pb.TagNumber(1)
  set action($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAction() => $_has(0);
  @$pb.TagNumber(1)
  void clearAction() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get amount => $_getIZ(1);
  @$pb.TagNumber(2)
  set amount($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAmount() => $_has(1);
  @$pb.TagNumber(2)
  void clearAmount() => $_clearField(2);
}

/// TunnelVision darkens the screen except for a circle around the eye position
class TunnelVision extends $pb.GeneratedMessage {
  factory TunnelVision({
    $core.bool? enabled,
    $core.int? radius,
    $core.double? opacity,
    $core.double? blendDistance,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    if (radius != null) result.radius = radius;
    if (opacity != null) result.opacity = opacity;
    if (blendDistance != null) result.blendDistance = blendDistance;
    return result;
  }

  TunnelVision._();

  factory TunnelVision.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory TunnelVision.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'TunnelVision', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..a<$core.int>(2, _omitFieldNames ? '' : 'radius', $pb.PbFieldType.O3)
    ..a<$core.double>(3, _omitFieldNames ? '' : 'opacity', $pb.PbFieldType.OD)
    ..a<$core.double>(4, _omitFieldNames ? '' : 'blendDistance', $pb.PbFieldType.OD)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TunnelVision clone() => TunnelVision()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TunnelVision copyWith(void Function(TunnelVision) updates) => super.copyWith((message) => updates(message as TunnelVision)) as TunnelVision;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TunnelVision create() => TunnelVision._();
  @$core.override
  TunnelVision createEmptyInstance() => create();
  static $pb.PbList<TunnelVision> createRepeated() => $pb.PbList<TunnelVision>();
  @$core.pragma('dart2js:noInline')
  static TunnelVision getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<TunnelVision>(create);
  static TunnelVision? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get radius => $_getIZ(1);
  @$pb.TagNumber(2)
  set radius($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasRadius() => $_has(1);
  @$pb.TagNumber(2)
  void clearRadius() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.double get opacity => $_getN(2);
  @$pb.TagNumber(3)
  set opacity($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasOpacity() => $_has(2);
  @$pb.TagNumber(3)
  void clearOpacity() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get blendDistance => $_getN(3);
  @$pb.TagNumber(4)
  set blendDistance($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBlendDistance() => $_has(3);
  @$pb.TagNumber(4)
  void clearBlendDistance() => $_clearField(4);
}

/// GhostVisibility controls whether the ghost cursor indicator is shown
class GhostVisibility extends $pb.GeneratedMessage {
  factory GhostVisibility({
    $core.bool? visible,
  }) {
    final result = create();
    if (visible != null) result.visible = visible;
    return result;
  }

  GhostVisibility._();

  factory GhostVisibility.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory GhostVisibility.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'GhostVisibility', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'visible')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GhostVisibility clone() => GhostVisibility()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GhostVisibility copyWith(void Function(GhostVisibility) updates) => super.copyWith((message) => updates(message as GhostVisibility)) as GhostVisibility;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GhostVisibility create() => GhostVisibility._();
  @$core.override
  GhostVisibility createEmptyInstance() => create();
  static $pb.PbList<GhostVisibility> createRepeated() => $pb.PbList<GhostVisibility>();
  @$core.pragma('dart2js:noInline')
  static GhostVisibility getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<GhostVisibility>(create);
  static GhostVisibility? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get visible => $_getBF(0);
  @$pb.TagNumber(1)
  set visible($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasVisible() => $_has(0);
  @$pb.TagNumber(1)
  void clearVisible() => $_clearField(1);
}

/// RecordingIndicator shows when voice input is being awaited
class RecordingIndicator extends $pb.GeneratedMessage {
  factory RecordingIndicator({
    $core.bool? recording,
    $core.int? timeoutMs,
  }) {
    final result = create();
    if (recording != null) result.recording = recording;
    if (timeoutMs != null) result.timeoutMs = timeoutMs;
    return result;
  }

  RecordingIndicator._();

  factory RecordingIndicator.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory RecordingIndicator.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'RecordingIndicator', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'recording')
    ..a<$core.int>(2, _omitFieldNames ? '' : 'timeoutMs', $pb.PbFieldType.O3)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RecordingIndicator clone() => RecordingIndicator()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RecordingIndicator copyWith(void Function(RecordingIndicator) updates) => super.copyWith((message) => updates(message as RecordingIndicator)) as RecordingIndicator;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RecordingIndicator create() => RecordingIndicator._();
  @$core.override
  RecordingIndicator createEmptyInstance() => create();
  static $pb.PbList<RecordingIndicator> createRepeated() => $pb.PbList<RecordingIndicator>();
  @$core.pragma('dart2js:noInline')
  static RecordingIndicator getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RecordingIndicator>(create);
  static RecordingIndicator? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get recording => $_getBF(0);
  @$pb.TagNumber(1)
  set recording($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRecording() => $_has(0);
  @$pb.TagNumber(1)
  void clearRecording() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get timeoutMs => $_getIZ(1);
  @$pb.TagNumber(2)
  set timeoutMs($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTimeoutMs() => $_has(1);
  @$pb.TagNumber(2)
  void clearTimeoutMs() => $_clearField(2);
}

/// MouseDownIndicator shows when the mouse button is being held down
class MouseDownIndicator extends $pb.GeneratedMessage {
  factory MouseDownIndicator({
    $core.bool? active,
  }) {
    final result = create();
    if (active != null) result.active = active;
    return result;
  }

  MouseDownIndicator._();

  factory MouseDownIndicator.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory MouseDownIndicator.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'MouseDownIndicator', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'active')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MouseDownIndicator clone() => MouseDownIndicator()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MouseDownIndicator copyWith(void Function(MouseDownIndicator) updates) => super.copyWith((message) => updates(message as MouseDownIndicator)) as MouseDownIndicator;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MouseDownIndicator create() => MouseDownIndicator._();
  @$core.override
  MouseDownIndicator createEmptyInstance() => create();
  static $pb.PbList<MouseDownIndicator> createRepeated() => $pb.PbList<MouseDownIndicator>();
  @$core.pragma('dart2js:noInline')
  static MouseDownIndicator getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<MouseDownIndicator>(create);
  static MouseDownIndicator? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get active => $_getBF(0);
  @$pb.TagNumber(1)
  set active($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasActive() => $_has(0);
  @$pb.TagNumber(1)
  void clearActive() => $_clearField(1);
}

/// NotListeningIndicator shows when voice input is explicitly disabled
class NotListeningIndicator extends $pb.GeneratedMessage {
  factory NotListeningIndicator({
    $core.bool? enabled,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    return result;
  }

  NotListeningIndicator._();

  factory NotListeningIndicator.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory NotListeningIndicator.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'NotListeningIndicator', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NotListeningIndicator clone() => NotListeningIndicator()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  NotListeningIndicator copyWith(void Function(NotListeningIndicator) updates) => super.copyWith((message) => updates(message as NotListeningIndicator)) as NotListeningIndicator;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static NotListeningIndicator create() => NotListeningIndicator._();
  @$core.override
  NotListeningIndicator createEmptyInstance() => create();
  static $pb.PbList<NotListeningIndicator> createRepeated() => $pb.PbList<NotListeningIndicator>();
  @$core.pragma('dart2js:noInline')
  static NotListeningIndicator getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<NotListeningIndicator>(create);
  static NotListeningIndicator? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);
}

/// Dictation shows the live prose buffer while dictating
class Dictation extends $pb.GeneratedMessage {
  factory Dictation({
    $core.String? text,
    $core.String? pending,
    $core.bool? delete,
    $core.Iterable<$core.String>? hints,
  }) {
    final result = create();
    if (text != null) result.text = text;
    if (pending != null) result.pending = pending;
    if (delete != null) result.delete = delete;
    if (hints != null) result.hints.addAll(hints);
    return result;
  }

  Dictation._();

  factory Dictation.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory Dictation.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Dictation', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.overlay.v1'), createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'text')
    ..aOS(2, _omitFieldNames ? '' : 'pending')
    ..aOB(3, _omitFieldNames ? '' : 'delete')
    ..pPS(4, _omitFieldNames ? '' : 'hints')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Dictation clone() => Dictation()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Dictation copyWith(void Function(Dictation) updates) => super.copyWith((message) => updates(message as Dictation)) as Dictation;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Dictation create() => Dictation._();
  @$core.override
  Dictation createEmptyInstance() => create();
  static $pb.PbList<Dictation> createRepeated() => $pb.PbList<Dictation>();
  @$core.pragma('dart2js:noInline')
  static Dictation getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Dictation>(create);
  static Dictation? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get text => $_getSZ(0);
  @$pb.TagNumber(1)
  set text($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasText() => $_has(0);
  @$pb.TagNumber(1)
  void clearText() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get pending => $_getSZ(1);
  @$pb.TagNumber(2)
  set pending($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPending() => $_has(1);
  @$pb.TagNumber(2)
  void clearPending() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get delete => $_getBF(2);
  @$pb.TagNumber(3)
  set delete($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDelete() => $_has(2);
  @$pb.TagNumber(3)
  void clearDelete() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<$core.String> get hints => $_getList(3);
}


const $core.bool _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
