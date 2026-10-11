//
//  Generated code. Do not modify.
//  source: mouse_whisperer/companion/v1/companion.proto
//
// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import '../../../google/protobuf/timestamp.pb.dart' as $0;
import 'companion.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'companion.pbenum.dart';

class MouseEvent extends $pb.GeneratedMessage {
  factory MouseEvent({
    MouseEvent_EventType? type,
    $core.int? pointerCount,
    $core.int? pointerSequence,
    $core.Iterable<Modifier>? modifiers,
    $core.double? deltaX,
    $core.double? deltaY,
  }) {
    final result = create();
    if (type != null) result.type = type;
    if (pointerCount != null) result.pointerCount = pointerCount;
    if (pointerSequence != null) result.pointerSequence = pointerSequence;
    if (modifiers != null) result.modifiers.addAll(modifiers);
    if (deltaX != null) result.deltaX = deltaX;
    if (deltaY != null) result.deltaY = deltaY;
    return result;
  }

  MouseEvent._();

  factory MouseEvent.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory MouseEvent.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'MouseEvent', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.companion.v1'), createEmptyInstance: create)
    ..e<MouseEvent_EventType>(1, _omitFieldNames ? '' : 'type', $pb.PbFieldType.OE, defaultOrMaker: MouseEvent_EventType.EVENT_TYPE_UNSPECIFIED, valueOf: MouseEvent_EventType.valueOf, enumValues: MouseEvent_EventType.values)
    ..a<$core.int>(2, _omitFieldNames ? '' : 'pointerCount', $pb.PbFieldType.O3)
    ..a<$core.int>(3, _omitFieldNames ? '' : 'pointerSequence', $pb.PbFieldType.O3)
    ..pc<Modifier>(4, _omitFieldNames ? '' : 'modifiers', $pb.PbFieldType.KE, valueOf: Modifier.valueOf, enumValues: Modifier.values, defaultEnumValue: Modifier.MODIFIER_UNSPECIFIED)
    ..a<$core.double>(5, _omitFieldNames ? '' : 'deltaX', $pb.PbFieldType.OF)
    ..a<$core.double>(6, _omitFieldNames ? '' : 'deltaY', $pb.PbFieldType.OF)
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MouseEvent clone() => MouseEvent()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  MouseEvent copyWith(void Function(MouseEvent) updates) => super.copyWith((message) => updates(message as MouseEvent)) as MouseEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static MouseEvent create() => MouseEvent._();
  @$core.override
  MouseEvent createEmptyInstance() => create();
  static $pb.PbList<MouseEvent> createRepeated() => $pb.PbList<MouseEvent>();
  @$core.pragma('dart2js:noInline')
  static MouseEvent getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<MouseEvent>(create);
  static MouseEvent? _defaultInstance;

  @$pb.TagNumber(1)
  MouseEvent_EventType get type => $_getN(0);
  @$pb.TagNumber(1)
  set type(MouseEvent_EventType value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasType() => $_has(0);
  @$pb.TagNumber(1)
  void clearType() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get pointerCount => $_getIZ(1);
  @$pb.TagNumber(2)
  set pointerCount($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasPointerCount() => $_has(1);
  @$pb.TagNumber(2)
  void clearPointerCount() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get pointerSequence => $_getIZ(2);
  @$pb.TagNumber(3)
  set pointerSequence($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPointerSequence() => $_has(2);
  @$pb.TagNumber(3)
  void clearPointerSequence() => $_clearField(3);

  @$pb.TagNumber(4)
  $pb.PbList<Modifier> get modifiers => $_getList(3);

  @$pb.TagNumber(5)
  $core.double get deltaX => $_getN(4);
  @$pb.TagNumber(5)
  set deltaX($core.double value) => $_setFloat(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDeltaX() => $_has(4);
  @$pb.TagNumber(5)
  void clearDeltaX() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.double get deltaY => $_getN(5);
  @$pb.TagNumber(6)
  set deltaY($core.double value) => $_setFloat(5, value);
  @$pb.TagNumber(6)
  $core.bool hasDeltaY() => $_has(5);
  @$pb.TagNumber(6)
  void clearDeltaY() => $_clearField(6);
}

enum KeyboardEvent_Data {
  typeString, 
  notSet
}

class KeyboardEvent extends $pb.GeneratedMessage {
  factory KeyboardEvent({
    KeyboardEvent_EventType? type,
    $core.String? typeString,
  }) {
    final result = create();
    if (type != null) result.type = type;
    if (typeString != null) result.typeString = typeString;
    return result;
  }

  KeyboardEvent._();

  factory KeyboardEvent.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory KeyboardEvent.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, KeyboardEvent_Data> _KeyboardEvent_DataByTag = {
    10 : KeyboardEvent_Data.typeString,
    0 : KeyboardEvent_Data.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'KeyboardEvent', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.companion.v1'), createEmptyInstance: create)
    ..oo(0, [10])
    ..e<KeyboardEvent_EventType>(1, _omitFieldNames ? '' : 'type', $pb.PbFieldType.OE, defaultOrMaker: KeyboardEvent_EventType.EVENT_TYPE_UNSPECIFIED, valueOf: KeyboardEvent_EventType.valueOf, enumValues: KeyboardEvent_EventType.values)
    ..aOS(10, _omitFieldNames ? '' : 'typeString')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeyboardEvent clone() => KeyboardEvent()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeyboardEvent copyWith(void Function(KeyboardEvent) updates) => super.copyWith((message) => updates(message as KeyboardEvent)) as KeyboardEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static KeyboardEvent create() => KeyboardEvent._();
  @$core.override
  KeyboardEvent createEmptyInstance() => create();
  static $pb.PbList<KeyboardEvent> createRepeated() => $pb.PbList<KeyboardEvent>();
  @$core.pragma('dart2js:noInline')
  static KeyboardEvent getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<KeyboardEvent>(create);
  static KeyboardEvent? _defaultInstance;

  KeyboardEvent_Data whichData() => _KeyboardEvent_DataByTag[$_whichOneof(0)]!;
  void clearData() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  KeyboardEvent_EventType get type => $_getN(0);
  @$pb.TagNumber(1)
  set type(KeyboardEvent_EventType value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasType() => $_has(0);
  @$pb.TagNumber(1)
  void clearType() => $_clearField(1);

  @$pb.TagNumber(10)
  $core.String get typeString => $_getSZ(1);
  @$pb.TagNumber(10)
  set typeString($core.String value) => $_setString(1, value);
  @$pb.TagNumber(10)
  $core.bool hasTypeString() => $_has(1);
  @$pb.TagNumber(10)
  void clearTypeString() => $_clearField(10);
}

enum Payload_Data {
  mouse, 
  keyboard, 
  command, 
  notSet
}

class Payload extends $pb.GeneratedMessage {
  factory Payload({
    $core.String? eventGroupId,
    $0.Timestamp? timestamp,
    MouseEvent? mouse,
    KeyboardEvent? keyboard,
    $core.String? command,
  }) {
    final result = create();
    if (eventGroupId != null) result.eventGroupId = eventGroupId;
    if (timestamp != null) result.timestamp = timestamp;
    if (mouse != null) result.mouse = mouse;
    if (keyboard != null) result.keyboard = keyboard;
    if (command != null) result.command = command;
    return result;
  }

  Payload._();

  factory Payload.fromBuffer($core.List<$core.int> data, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromBuffer(data, registry);
  factory Payload.fromJson($core.String json, [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) => create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, Payload_Data> _Payload_DataByTag = {
    5 : Payload_Data.mouse,
    6 : Payload_Data.keyboard,
    7 : Payload_Data.command,
    0 : Payload_Data.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(_omitMessageNames ? '' : 'Payload', package: const $pb.PackageName(_omitMessageNames ? '' : 'mouse_whisperer.companion.v1'), createEmptyInstance: create)
    ..oo(0, [5, 6, 7])
    ..aOS(1, _omitFieldNames ? '' : 'eventGroupId')
    ..aOM<$0.Timestamp>(2, _omitFieldNames ? '' : 'timestamp', subBuilder: $0.Timestamp.create)
    ..aOM<MouseEvent>(5, _omitFieldNames ? '' : 'mouse', subBuilder: MouseEvent.create)
    ..aOM<KeyboardEvent>(6, _omitFieldNames ? '' : 'keyboard', subBuilder: KeyboardEvent.create)
    ..aOS(7, _omitFieldNames ? '' : 'command')
    ..hasRequiredFields = false
  ;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Payload clone() => Payload()..mergeFromMessage(this);
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Payload copyWith(void Function(Payload) updates) => super.copyWith((message) => updates(message as Payload)) as Payload;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Payload create() => Payload._();
  @$core.override
  Payload createEmptyInstance() => create();
  static $pb.PbList<Payload> createRepeated() => $pb.PbList<Payload>();
  @$core.pragma('dart2js:noInline')
  static Payload getDefault() => _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Payload>(create);
  static Payload? _defaultInstance;

  Payload_Data whichData() => _Payload_DataByTag[$_whichOneof(0)]!;
  void clearData() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get eventGroupId => $_getSZ(0);
  @$pb.TagNumber(1)
  set eventGroupId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEventGroupId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEventGroupId() => $_clearField(1);

  @$pb.TagNumber(2)
  $0.Timestamp get timestamp => $_getN(1);
  @$pb.TagNumber(2)
  set timestamp($0.Timestamp value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasTimestamp() => $_has(1);
  @$pb.TagNumber(2)
  void clearTimestamp() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.Timestamp ensureTimestamp() => $_ensure(1);

  @$pb.TagNumber(5)
  MouseEvent get mouse => $_getN(2);
  @$pb.TagNumber(5)
  set mouse(MouseEvent value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasMouse() => $_has(2);
  @$pb.TagNumber(5)
  void clearMouse() => $_clearField(5);
  @$pb.TagNumber(5)
  MouseEvent ensureMouse() => $_ensure(2);

  @$pb.TagNumber(6)
  KeyboardEvent get keyboard => $_getN(3);
  @$pb.TagNumber(6)
  set keyboard(KeyboardEvent value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasKeyboard() => $_has(3);
  @$pb.TagNumber(6)
  void clearKeyboard() => $_clearField(6);
  @$pb.TagNumber(6)
  KeyboardEvent ensureKeyboard() => $_ensure(3);

  @$pb.TagNumber(7)
  $core.String get command => $_getSZ(4);
  @$pb.TagNumber(7)
  set command($core.String value) => $_setString(4, value);
  @$pb.TagNumber(7)
  $core.bool hasCommand() => $_has(4);
  @$pb.TagNumber(7)
  void clearCommand() => $_clearField(7);
}


const $core.bool _omitFieldNames = $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames = $core.bool.fromEnvironment('protobuf.omit_message_names');
