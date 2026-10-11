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

class Modifier extends $pb.ProtobufEnum {
  static const Modifier MODIFIER_UNSPECIFIED = Modifier._(0, _omitEnumNames ? '' : 'MODIFIER_UNSPECIFIED');
  static const Modifier MODIFIER_COMMAND = Modifier._(1, _omitEnumNames ? '' : 'MODIFIER_COMMAND');
  static const Modifier MODIFIER_CONTROL = Modifier._(2, _omitEnumNames ? '' : 'MODIFIER_CONTROL');
  static const Modifier MODIFIER_OPTION = Modifier._(3, _omitEnumNames ? '' : 'MODIFIER_OPTION');
  static const Modifier MODIFIER_SHIFT = Modifier._(4, _omitEnumNames ? '' : 'MODIFIER_SHIFT');
  static const Modifier MODIFIER_SUPER = Modifier._(5, _omitEnumNames ? '' : 'MODIFIER_SUPER');

  static const $core.List<Modifier> values = <Modifier> [
    MODIFIER_UNSPECIFIED,
    MODIFIER_COMMAND,
    MODIFIER_CONTROL,
    MODIFIER_OPTION,
    MODIFIER_SHIFT,
    MODIFIER_SUPER,
  ];

  static final $core.List<Modifier?> _byValue = $pb.ProtobufEnum.$_initByValueList(values, 5);
  static Modifier? valueOf($core.int value) =>  value < 0 || value >= _byValue.length ? null : _byValue[value];

  const Modifier._(super.value, super.name);
}

class MouseEvent_EventType extends $pb.ProtobufEnum {
  static const MouseEvent_EventType EVENT_TYPE_UNSPECIFIED = MouseEvent_EventType._(0, _omitEnumNames ? '' : 'EVENT_TYPE_UNSPECIFIED');
  static const MouseEvent_EventType EVENT_TYPE_CLICK = MouseEvent_EventType._(1, _omitEnumNames ? '' : 'EVENT_TYPE_CLICK');
  static const MouseEvent_EventType EVENT_TYPE_LONG_PRESS = MouseEvent_EventType._(2, _omitEnumNames ? '' : 'EVENT_TYPE_LONG_PRESS');
  static const MouseEvent_EventType EVENT_TYPE_DRAG = MouseEvent_EventType._(3, _omitEnumNames ? '' : 'EVENT_TYPE_DRAG');
  static const MouseEvent_EventType EVENT_TYPE_FLING = MouseEvent_EventType._(4, _omitEnumNames ? '' : 'EVENT_TYPE_FLING');
  static const MouseEvent_EventType EVENT_TYPE_FINISH = MouseEvent_EventType._(10, _omitEnumNames ? '' : 'EVENT_TYPE_FINISH');

  static const $core.List<MouseEvent_EventType> values = <MouseEvent_EventType> [
    EVENT_TYPE_UNSPECIFIED,
    EVENT_TYPE_CLICK,
    EVENT_TYPE_LONG_PRESS,
    EVENT_TYPE_DRAG,
    EVENT_TYPE_FLING,
    EVENT_TYPE_FINISH,
  ];

  static final $core.Map<$core.int, MouseEvent_EventType> _byValue = $pb.ProtobufEnum.initByValue(values);
  static MouseEvent_EventType? valueOf($core.int value) => _byValue[value];

  const MouseEvent_EventType._(super.value, super.name);
}

class KeyboardEvent_EventType extends $pb.ProtobufEnum {
  static const KeyboardEvent_EventType EVENT_TYPE_UNSPECIFIED = KeyboardEvent_EventType._(0, _omitEnumNames ? '' : 'EVENT_TYPE_UNSPECIFIED');
  static const KeyboardEvent_EventType EVENT_TYPE_TYPE_STRING = KeyboardEvent_EventType._(1, _omitEnumNames ? '' : 'EVENT_TYPE_TYPE_STRING');

  static const $core.List<KeyboardEvent_EventType> values = <KeyboardEvent_EventType> [
    EVENT_TYPE_UNSPECIFIED,
    EVENT_TYPE_TYPE_STRING,
  ];

  static final $core.List<KeyboardEvent_EventType?> _byValue = $pb.ProtobufEnum.$_initByValueList(values, 1);
  static KeyboardEvent_EventType? valueOf($core.int value) =>  value < 0 || value >= _byValue.length ? null : _byValue[value];

  const KeyboardEvent_EventType._(super.value, super.name);
}


const $core.bool _omitEnumNames = $core.bool.fromEnvironment('protobuf.omit_enum_names');
