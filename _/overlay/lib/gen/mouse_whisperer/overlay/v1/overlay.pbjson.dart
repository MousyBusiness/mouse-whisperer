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

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use displayMessageDescriptor instead')
const DisplayMessage$json = {
  '1': 'DisplayMessage',
  '2': [
    {'1': 'command_text', '3': 1, '4': 1, '5': 9, '9': 0, '10': 'commandText'},
    {'1': 'commands_list', '3': 2, '4': 1, '5': 11, '6': '.mouse_whisperer.overlay.v1.CommandsList', '9': 0, '10': 'commandsList'},
    {'1': 'command_config', '3': 3, '4': 1, '5': 11, '6': '.mouse_whisperer.overlay.v1.CommandConfig', '9': 0, '10': 'commandConfig'},
  ],
  '8': [
    {'1': 'content'},
  ],
};

/// Descriptor for `DisplayMessage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List displayMessageDescriptor = $convert.base64Decode(
    'Cg5EaXNwbGF5TWVzc2FnZRIjCgxjb21tYW5kX3RleHQYASABKAlIAFILY29tbWFuZFRleHQSTw'
    'oNY29tbWFuZHNfbGlzdBgCIAEoCzIoLm1vdXNlX3doaXNwZXJlci5vdmVybGF5LnYxLkNvbW1h'
    'bmRzTGlzdEgAUgxjb21tYW5kc0xpc3QSUgoOY29tbWFuZF9jb25maWcYAyABKAsyKS5tb3VzZV'
    '93aGlzcGVyZXIub3ZlcmxheS52MS5Db21tYW5kQ29uZmlnSABSDWNvbW1hbmRDb25maWdCCQoH'
    'Y29udGVudA==');

@$core.Deprecated('Use commandsListDescriptor instead')
const CommandsList$json = {
  '1': 'CommandsList',
  '2': [
    {'1': 'title', '3': 1, '4': 1, '5': 9, '10': 'title'},
    {'1': 'items', '3': 2, '4': 3, '5': 9, '10': 'items'},
    {'1': 'dismiss_seconds', '3': 3, '4': 1, '5': 5, '10': 'dismissSeconds'},
  ],
};

/// Descriptor for `CommandsList`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commandsListDescriptor = $convert.base64Decode(
    'CgxDb21tYW5kc0xpc3QSFAoFdGl0bGUYASABKAlSBXRpdGxlEhQKBWl0ZW1zGAIgAygJUgVpdG'
    'VtcxInCg9kaXNtaXNzX3NlY29uZHMYAyABKAVSDmRpc21pc3NTZWNvbmRz');

@$core.Deprecated('Use commandConfigDescriptor instead')
const CommandConfig$json = {
  '1': 'CommandConfig',
  '2': [
    {'1': 'name', '3': 1, '4': 1, '5': 9, '10': 'name'},
    {'1': 'description', '3': 2, '4': 1, '5': 9, '10': 'description'},
    {'1': 'triggers', '3': 3, '4': 3, '5': 9, '10': 'triggers'},
    {'1': 'dismiss_seconds', '3': 4, '4': 1, '5': 5, '10': 'dismissSeconds'},
  ],
};

/// Descriptor for `CommandConfig`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commandConfigDescriptor = $convert.base64Decode(
    'Cg1Db21tYW5kQ29uZmlnEhIKBG5hbWUYASABKAlSBG5hbWUSIAoLZGVzY3JpcHRpb24YAiABKA'
    'lSC2Rlc2NyaXB0aW9uEhoKCHRyaWdnZXJzGAMgAygJUgh0cmlnZ2VycxInCg9kaXNtaXNzX3Nl'
    'Y29uZHMYBCABKAVSDmRpc21pc3NTZWNvbmRz');

@$core.Deprecated('Use toastDescriptor instead')
const Toast$json = {
  '1': 'Toast',
  '2': [
    {'1': 'title', '3': 1, '4': 1, '5': 9, '10': 'title'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
    {'1': 'level', '3': 3, '4': 1, '5': 9, '10': 'level'},
  ],
};

/// Descriptor for `Toast`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List toastDescriptor = $convert.base64Decode(
    'CgVUb2FzdBIUCgV0aXRsZRgBIAEoCVIFdGl0bGUSGAoHbWVzc2FnZRgCIAEoCVIHbWVzc2FnZR'
    'IUCgVsZXZlbBgDIAEoCVIFbGV2ZWw=');

@$core.Deprecated('Use eyesPositionDescriptor instead')
const EyesPosition$json = {
  '1': 'EyesPosition',
  '2': [
    {'1': 'x', '3': 1, '4': 1, '5': 1, '10': 'x'},
    {'1': 'y', '3': 2, '4': 1, '5': 1, '10': 'y'},
    {'1': 'clear', '3': 3, '4': 1, '5': 8, '10': 'clear'},
  ],
};

/// Descriptor for `EyesPosition`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List eyesPositionDescriptor = $convert.base64Decode(
    'CgxFeWVzUG9zaXRpb24SDAoBeBgBIAEoAVIBeBIMCgF5GAIgASgBUgF5EhQKBWNsZWFyGAMgAS'
    'gIUgVjbGVhcg==');

@$core.Deprecated('Use imageDescriptor instead')
const Image$json = {
  '1': 'Image',
  '2': [
    {'1': 'image_base64', '3': 1, '4': 1, '5': 9, '10': 'imageBase64'},
    {'1': 'delete', '3': 2, '4': 1, '5': 8, '10': 'delete'},
  ],
};

/// Descriptor for `Image`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List imageDescriptor = $convert.base64Decode(
    'CgVJbWFnZRIhCgxpbWFnZV9iYXNlNjQYASABKAlSC2ltYWdlQmFzZTY0EhYKBmRlbGV0ZRgCIA'
    'EoCFIGZGVsZXRl');

@$core.Deprecated('Use aIResponseDescriptor instead')
const AIResponse$json = {
  '1': 'AIResponse',
  '2': [
    {'1': 'prompt', '3': 1, '4': 1, '5': 9, '10': 'prompt'},
    {'1': 'ai_response', '3': 2, '4': 1, '5': 9, '10': 'aiResponse'},
    {'1': 'delete', '3': 3, '4': 1, '5': 8, '10': 'delete'},
  ],
};

/// Descriptor for `AIResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List aIResponseDescriptor = $convert.base64Decode(
    'CgpBSVJlc3BvbnNlEhYKBnByb21wdBgBIAEoCVIGcHJvbXB0Eh8KC2FpX3Jlc3BvbnNlGAIgAS'
    'gJUgphaVJlc3BvbnNlEhYKBmRlbGV0ZRgDIAEoCFIGZGVsZXRl');

@$core.Deprecated('Use uIActionDescriptor instead')
const UIAction$json = {
  '1': 'UIAction',
  '2': [
    {'1': 'action', '3': 1, '4': 1, '5': 9, '10': 'action'},
    {'1': 'amount', '3': 2, '4': 1, '5': 5, '10': 'amount'},
  ],
};

/// Descriptor for `UIAction`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uIActionDescriptor = $convert.base64Decode(
    'CghVSUFjdGlvbhIWCgZhY3Rpb24YASABKAlSBmFjdGlvbhIWCgZhbW91bnQYAiABKAVSBmFtb3'
    'VudA==');

@$core.Deprecated('Use tunnelVisionDescriptor instead')
const TunnelVision$json = {
  '1': 'TunnelVision',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
    {'1': 'radius', '3': 2, '4': 1, '5': 5, '10': 'radius'},
    {'1': 'opacity', '3': 3, '4': 1, '5': 1, '10': 'opacity'},
    {'1': 'blend_distance', '3': 4, '4': 1, '5': 1, '10': 'blendDistance'},
  ],
};

/// Descriptor for `TunnelVision`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List tunnelVisionDescriptor = $convert.base64Decode(
    'CgxUdW5uZWxWaXNpb24SGAoHZW5hYmxlZBgBIAEoCFIHZW5hYmxlZBIWCgZyYWRpdXMYAiABKA'
    'VSBnJhZGl1cxIYCgdvcGFjaXR5GAMgASgBUgdvcGFjaXR5EiUKDmJsZW5kX2Rpc3RhbmNlGAQg'
    'ASgBUg1ibGVuZERpc3RhbmNl');

@$core.Deprecated('Use ghostVisibilityDescriptor instead')
const GhostVisibility$json = {
  '1': 'GhostVisibility',
  '2': [
    {'1': 'visible', '3': 1, '4': 1, '5': 8, '10': 'visible'},
  ],
};

/// Descriptor for `GhostVisibility`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List ghostVisibilityDescriptor = $convert.base64Decode(
    'Cg9HaG9zdFZpc2liaWxpdHkSGAoHdmlzaWJsZRgBIAEoCFIHdmlzaWJsZQ==');

@$core.Deprecated('Use recordingIndicatorDescriptor instead')
const RecordingIndicator$json = {
  '1': 'RecordingIndicator',
  '2': [
    {'1': 'recording', '3': 1, '4': 1, '5': 8, '10': 'recording'},
    {'1': 'timeout_ms', '3': 2, '4': 1, '5': 5, '10': 'timeoutMs'},
  ],
};

/// Descriptor for `RecordingIndicator`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List recordingIndicatorDescriptor = $convert.base64Decode(
    'ChJSZWNvcmRpbmdJbmRpY2F0b3ISHAoJcmVjb3JkaW5nGAEgASgIUglyZWNvcmRpbmcSHQoKdG'
    'ltZW91dF9tcxgCIAEoBVIJdGltZW91dE1z');

@$core.Deprecated('Use mouseDownIndicatorDescriptor instead')
const MouseDownIndicator$json = {
  '1': 'MouseDownIndicator',
  '2': [
    {'1': 'active', '3': 1, '4': 1, '5': 8, '10': 'active'},
  ],
};

/// Descriptor for `MouseDownIndicator`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List mouseDownIndicatorDescriptor = $convert.base64Decode(
    'ChJNb3VzZURvd25JbmRpY2F0b3ISFgoGYWN0aXZlGAEgASgIUgZhY3RpdmU=');

@$core.Deprecated('Use notListeningIndicatorDescriptor instead')
const NotListeningIndicator$json = {
  '1': 'NotListeningIndicator',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
  ],
};

/// Descriptor for `NotListeningIndicator`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List notListeningIndicatorDescriptor = $convert.base64Decode(
    'ChVOb3RMaXN0ZW5pbmdJbmRpY2F0b3ISGAoHZW5hYmxlZBgBIAEoCFIHZW5hYmxlZA==');

@$core.Deprecated('Use dictationDescriptor instead')
const Dictation$json = {
  '1': 'Dictation',
  '2': [
    {'1': 'text', '3': 1, '4': 1, '5': 9, '10': 'text'},
    {'1': 'pending', '3': 2, '4': 1, '5': 9, '10': 'pending'},
    {'1': 'delete', '3': 3, '4': 1, '5': 8, '10': 'delete'},
    {'1': 'hints', '3': 4, '4': 3, '5': 9, '10': 'hints'},
  ],
};

/// Descriptor for `Dictation`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List dictationDescriptor = $convert.base64Decode(
    'CglEaWN0YXRpb24SEgoEdGV4dBgBIAEoCVIEdGV4dBIYCgdwZW5kaW5nGAIgASgJUgdwZW5kaW'
    '5nEhYKBmRlbGV0ZRgDIAEoCFIGZGVsZXRlEhQKBWhpbnRzGAQgAygJUgVoaW50cw==');

