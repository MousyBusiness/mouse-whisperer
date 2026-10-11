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

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use modifierDescriptor instead')
const Modifier$json = {
  '1': 'Modifier',
  '2': [
    {'1': 'MODIFIER_UNSPECIFIED', '2': 0},
    {'1': 'MODIFIER_COMMAND', '2': 1},
    {'1': 'MODIFIER_CONTROL', '2': 2},
    {'1': 'MODIFIER_OPTION', '2': 3},
    {'1': 'MODIFIER_SHIFT', '2': 4},
    {'1': 'MODIFIER_SUPER', '2': 5},
  ],
};

/// Descriptor for `Modifier`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List modifierDescriptor = $convert.base64Decode(
    'CghNb2RpZmllchIYChRNT0RJRklFUl9VTlNQRUNJRklFRBAAEhQKEE1PRElGSUVSX0NPTU1BTk'
    'QQARIUChBNT0RJRklFUl9DT05UUk9MEAISEwoPTU9ESUZJRVJfT1BUSU9OEAMSEgoOTU9ESUZJ'
    'RVJfU0hJRlQQBBISCg5NT0RJRklFUl9TVVBFUhAF');

@$core.Deprecated('Use mouseEventDescriptor instead')
const MouseEvent$json = {
  '1': 'MouseEvent',
  '2': [
    {'1': 'type', '3': 1, '4': 1, '5': 14, '6': '.mouse_whisperer.companion.v1.MouseEvent.EventType', '10': 'type'},
    {'1': 'pointer_count', '3': 2, '4': 1, '5': 5, '10': 'pointerCount'},
    {'1': 'pointer_sequence', '3': 3, '4': 1, '5': 5, '10': 'pointerSequence'},
    {'1': 'modifiers', '3': 4, '4': 3, '5': 14, '6': '.mouse_whisperer.companion.v1.Modifier', '10': 'modifiers'},
    {'1': 'delta_x', '3': 5, '4': 1, '5': 2, '9': 0, '10': 'deltaX', '17': true},
    {'1': 'delta_y', '3': 6, '4': 1, '5': 2, '9': 1, '10': 'deltaY', '17': true},
  ],
  '4': [MouseEvent_EventType$json],
  '8': [
    {'1': '_delta_x'},
    {'1': '_delta_y'},
  ],
};

@$core.Deprecated('Use mouseEventDescriptor instead')
const MouseEvent_EventType$json = {
  '1': 'EventType',
  '2': [
    {'1': 'EVENT_TYPE_UNSPECIFIED', '2': 0},
    {'1': 'EVENT_TYPE_CLICK', '2': 1},
    {'1': 'EVENT_TYPE_LONG_PRESS', '2': 2},
    {'1': 'EVENT_TYPE_DRAG', '2': 3},
    {'1': 'EVENT_TYPE_FLING', '2': 4},
    {'1': 'EVENT_TYPE_FINISH', '2': 10},
  ],
};

/// Descriptor for `MouseEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List mouseEventDescriptor = $convert.base64Decode(
    'CgpNb3VzZUV2ZW50EkYKBHR5cGUYASABKA4yMi5tb3VzZV93aGlzcGVyZXIuY29tcGFuaW9uLn'
    'YxLk1vdXNlRXZlbnQuRXZlbnRUeXBlUgR0eXBlEiMKDXBvaW50ZXJfY291bnQYAiABKAVSDHBv'
    'aW50ZXJDb3VudBIpChBwb2ludGVyX3NlcXVlbmNlGAMgASgFUg9wb2ludGVyU2VxdWVuY2USRA'
    'oJbW9kaWZpZXJzGAQgAygOMiYubW91c2Vfd2hpc3BlcmVyLmNvbXBhbmlvbi52MS5Nb2RpZmll'
    'clIJbW9kaWZpZXJzEhwKB2RlbHRhX3gYBSABKAJIAFIGZGVsdGFYiAEBEhwKB2RlbHRhX3kYBi'
    'ABKAJIAVIGZGVsdGFZiAEBIpoBCglFdmVudFR5cGUSGgoWRVZFTlRfVFlQRV9VTlNQRUNJRklF'
    'RBAAEhQKEEVWRU5UX1RZUEVfQ0xJQ0sQARIZChVFVkVOVF9UWVBFX0xPTkdfUFJFU1MQAhITCg'
    '9FVkVOVF9UWVBFX0RSQUcQAxIUChBFVkVOVF9UWVBFX0ZMSU5HEAQSFQoRRVZFTlRfVFlQRV9G'
    'SU5JU0gQCkIKCghfZGVsdGFfeEIKCghfZGVsdGFfeQ==');

@$core.Deprecated('Use keyboardEventDescriptor instead')
const KeyboardEvent$json = {
  '1': 'KeyboardEvent',
  '2': [
    {'1': 'type', '3': 1, '4': 1, '5': 14, '6': '.mouse_whisperer.companion.v1.KeyboardEvent.EventType', '10': 'type'},
    {'1': 'type_string', '3': 10, '4': 1, '5': 9, '9': 0, '10': 'typeString'},
  ],
  '4': [KeyboardEvent_EventType$json],
  '8': [
    {'1': 'data'},
  ],
};

@$core.Deprecated('Use keyboardEventDescriptor instead')
const KeyboardEvent_EventType$json = {
  '1': 'EventType',
  '2': [
    {'1': 'EVENT_TYPE_UNSPECIFIED', '2': 0},
    {'1': 'EVENT_TYPE_TYPE_STRING', '2': 1},
  ],
};

/// Descriptor for `KeyboardEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List keyboardEventDescriptor = $convert.base64Decode(
    'Cg1LZXlib2FyZEV2ZW50EkkKBHR5cGUYASABKA4yNS5tb3VzZV93aGlzcGVyZXIuY29tcGFuaW'
    '9uLnYxLktleWJvYXJkRXZlbnQuRXZlbnRUeXBlUgR0eXBlEiEKC3R5cGVfc3RyaW5nGAogASgJ'
    'SABSCnR5cGVTdHJpbmciQwoJRXZlbnRUeXBlEhoKFkVWRU5UX1RZUEVfVU5TUEVDSUZJRUQQAB'
    'IaChZFVkVOVF9UWVBFX1RZUEVfU1RSSU5HEAFCBgoEZGF0YQ==');

@$core.Deprecated('Use payloadDescriptor instead')
const Payload$json = {
  '1': 'Payload',
  '2': [
    {'1': 'event_group_id', '3': 1, '4': 1, '5': 9, '10': 'eventGroupId'},
    {'1': 'timestamp', '3': 2, '4': 1, '5': 11, '6': '.google.protobuf.Timestamp', '10': 'timestamp'},
    {'1': 'mouse', '3': 5, '4': 1, '5': 11, '6': '.mouse_whisperer.companion.v1.MouseEvent', '9': 0, '10': 'mouse'},
    {'1': 'keyboard', '3': 6, '4': 1, '5': 11, '6': '.mouse_whisperer.companion.v1.KeyboardEvent', '9': 0, '10': 'keyboard'},
    {'1': 'command', '3': 7, '4': 1, '5': 9, '9': 0, '10': 'command'},
  ],
  '8': [
    {'1': 'data'},
  ],
};

/// Descriptor for `Payload`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List payloadDescriptor = $convert.base64Decode(
    'CgdQYXlsb2FkEiQKDmV2ZW50X2dyb3VwX2lkGAEgASgJUgxldmVudEdyb3VwSWQSOAoJdGltZX'
    'N0YW1wGAIgASgLMhouZ29vZ2xlLnByb3RvYnVmLlRpbWVzdGFtcFIJdGltZXN0YW1wEkAKBW1v'
    'dXNlGAUgASgLMigubW91c2Vfd2hpc3BlcmVyLmNvbXBhbmlvbi52MS5Nb3VzZUV2ZW50SABSBW'
    '1vdXNlEkkKCGtleWJvYXJkGAYgASgLMisubW91c2Vfd2hpc3BlcmVyLmNvbXBhbmlvbi52MS5L'
    'ZXlib2FyZEV2ZW50SABSCGtleWJvYXJkEhoKB2NvbW1hbmQYByABKAlIAFIHY29tbWFuZEIGCg'
    'RkYXRh');

