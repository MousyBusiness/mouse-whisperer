import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'dart:ui';

import 'package:mouse_whisperer/gen/mouse_whisperer/overlay/v1/overlay.pb.dart'
    as pb;
import 'package:mouse_whisperer/snack.dart';

class Server {
  Server({
    required this.onCommandText,
    required this.onCommandsList,
    required this.onCommandConfig,
    required this.onAiResponse,
    required this.onImage,
    required this.onEyePosition,
    required this.onUIAction,
    required this.onTunnelVision,
    required this.onGhostVisibility,
    required this.onRecordingIndicator,
    required this.onMouseDownIndicator,
    required this.onNotListeningIndicator,
    required this.onDictation,
  });

  final void Function(String) onCommandText;
  final void Function(pb.CommandsList?) onCommandsList;
  final void Function(pb.CommandConfig?) onCommandConfig;
  final void Function((String, String)?) onAiResponse;
  final void Function(Uint8List?) onImage;
  final void Function(Offset?) onEyePosition;
  final void Function(pb.UIAction) onUIAction;
  final void Function(
          bool enabled, int radius, double opacity, double blendDistance)
      onTunnelVision;
  final void Function(bool visible) onGhostVisibility;
  final void Function(bool recording, int timeoutMs) onRecordingIndicator;
  final void Function(bool active) onMouseDownIndicator;
  final void Function(bool enabled) onNotListeningIndicator;
  final void Function((String, String, List<String>)?) onDictation;

  void handleDisplayMessage(HttpRequest req, Map<String, dynamic> json) async {
    final msg = pb.DisplayMessage()..mergeFromProto3Json(json);

    switch (msg.whichContent()) {
      case pb.DisplayMessage_Content.commandText:
        onCommandText(msg.commandText);
        break;
      case pb.DisplayMessage_Content.commandsList:
        onAiResponse(null); // Clear AI response (mutually exclusive)
        onCommandConfig(null); // Clear any config
        onCommandsList(msg.commandsList);
        break;
      case pb.DisplayMessage_Content.commandConfig:
        onAiResponse(null); // Clear AI response (mutually exclusive)
        onCommandsList(null); // Clear any list
        onCommandConfig(msg.commandConfig);
        break;
      case pb.DisplayMessage_Content.notSet:
        onCommandsList(null);
        onCommandConfig(null);
        onAiResponse(null); // Also clear AI response on dismiss
        break;
    }

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleToast(HttpRequest req, Map<String, dynamic> json) async {
    final toast = pb.Toast()..mergeFromProto3Json(json);

    showSnack(toast.title, toast.message, type: toast.level.toastType);
    print("Showing snack");
    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleEyesUpdate(HttpRequest req, Map<String, dynamic> json) async {
    try {
      final eyesPos = pb.EyesPosition()..mergeFromProto3Json(json);

      if (eyesPos.clear_3) {
        onEyePosition(null);
      } else {
        onEyePosition(Offset(eyesPos.x, eyesPos.y));
      }
      req.response
        ..statusCode = HttpStatus.ok
        ..write('OK')
        ..close();
    } catch (e) {
      req.response
        ..statusCode = HttpStatus.badRequest
        ..write('Invalid request: $e')
        ..close();
    }
  }

  void handleImage(HttpRequest request, Map<String, dynamic> json) async {
    request.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();

    final img = pb.Image()..mergeFromProto3Json(json);

    if (img.delete) {
      onImage(null);
      return;
    }
    final Uint8List imageBytes = base64Decode(img.imageBase64);
    onImage(imageBytes);
  }

  void handleAIResponse(HttpRequest req, Map<String, dynamic> json) async {
    final aiResp = pb.AIResponse()..mergeFromProto3Json(json);

    if (aiResp.delete) {
      onAiResponse(null);
    } else {
      // Clear commands panels (mutually exclusive)
      onCommandsList(null);
      onCommandConfig(null);
      onAiResponse((aiResp.prompt, aiResp.aiResponse));
    }

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleUIAction(HttpRequest req, Map<String, dynamic> json) async {
    final action = pb.UIAction()..mergeFromProto3Json(json);
    onUIAction(action);

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleTunnelVision(HttpRequest req, Map<String, dynamic> json) async {
    final tv = pb.TunnelVision()..mergeFromProto3Json(json);
    onTunnelVision(
      tv.enabled,
      tv.radius > 0 ? tv.radius : 600,
      tv.opacity > 0 ? tv.opacity : 0.85,
      tv.blendDistance > 0 ? tv.blendDistance : 0.5,
    );

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleGhostVisibility(HttpRequest req, Map<String, dynamic> json) async {
    final gv = pb.GhostVisibility()..mergeFromProto3Json(json);
    onGhostVisibility(gv.visible);

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleRecordingIndicator(
      HttpRequest req, Map<String, dynamic> json) async {
    final ri = pb.RecordingIndicator()..mergeFromProto3Json(json);
    onRecordingIndicator(ri.recording, ri.timeoutMs);

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleMouseDownIndicator(
      HttpRequest req, Map<String, dynamic> json) async {
    final mdi = pb.MouseDownIndicator()..mergeFromProto3Json(json);
    onMouseDownIndicator(mdi.active);

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleNotListeningIndicator(
      HttpRequest req, Map<String, dynamic> json) async {
    final nli = pb.NotListeningIndicator()..mergeFromProto3Json(json);
    onNotListeningIndicator(nli.enabled);

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  void handleDictation(HttpRequest req, Map<String, dynamic> json) async {
    final d = pb.Dictation()..mergeFromProto3Json(json);
    if (d.delete) {
      onDictation(null);
    } else {
      onDictation((d.text, d.pending, d.hints.toList()));
    }

    req.response
      ..statusCode = HttpStatus.ok
      ..write('OK')
      ..close();
  }

  Future start() async {
    final HttpServer server = await HttpServer.bind(
      InternetAddress.loopbackIPv4,
      51012,
    );
    print('Listening on localhost:${server.port}');

    await for (HttpRequest request in server) {
      switch (request.method) {
        case 'GET':
          request.response
            ..statusCode = HttpStatus.ok
            ..write("OK")
            ..close();
          break;
        case 'POST':
          final jsonString = await utf8.decoder.bind(request).join();
          final Map<String, dynamic> json = jsonDecode(jsonString);

          if (request.uri.path == "/display") {
            handleDisplayMessage(request, json);
            break;
          }
          if (request.uri.path == "/ai") {
            handleAIResponse(request, json);
            break;
          }
          if (request.uri.path == "/toast") {
            handleToast(request, json);
            break;
          }
          if (request.uri.path == "/eyes") {
            handleEyesUpdate(request, json);
            break;
          }
          if (request.uri.path == "/image") {
            handleImage(request, json);
            break;
          }
          if (request.uri.path == "/action") {
            handleUIAction(request, json);
            break;
          }
          if (request.uri.path == "/tunnel") {
            handleTunnelVision(request, json);
            break;
          }
          if (request.uri.path == "/ghost") {
            handleGhostVisibility(request, json);
            break;
          }
          if (request.uri.path == "/recording") {
            handleRecordingIndicator(request, json);
            break;
          }
          if (request.uri.path == "/mousedown") {
            handleMouseDownIndicator(request, json);
            break;
          }
          if (request.uri.path == "/notlistening") {
            handleNotListeningIndicator(request, json);
            break;
          }
          if (request.uri.path == "/dictation") {
            handleDictation(request, json);
            break;
          }
      }
    }
  }
}
