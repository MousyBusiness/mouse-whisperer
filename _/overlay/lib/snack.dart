import 'dart:math';

import 'package:flutter/material.dart';
import 'package:mouse_whisperer/app_theme.dart';
import 'package:mouse_whisperer/code/custom_code_box.dart';
import 'package:toastification/toastification.dart';

Widget _buildTitle(String text) {
  return Text(text, style: AppTheme.subtitle1);
}

Widget _buildDescription(String text) {
  return Text(text, style: AppTheme.body1);
}

extension XSnack on String {
  ToastificationType get toastType => switch (this) {
        "error" => ToastificationType.error,
        "warn" => ToastificationType.warning,
        "info" => ToastificationType.info,
        _ => ToastificationType.info,
      };
}

void showSnack(
  String title,
  String description, {
  Duration? duration,
  ToastificationType type = ToastificationType.success,
  bool progress = false,
}) {
  toastification.show(
    // context: context,
    type: type,
    style: ToastificationStyle.flat,
    title: _buildTitle(title),
    description: _buildDescription(description),
    alignment: Alignment.topLeft,
    autoCloseDuration: duration ?? const Duration(seconds: 2),
    borderRadius: BorderRadius.circular(12.0),
    boxShadow: [
      BoxShadow(
        color: AppTheme.rainbowColors[Random().nextInt(AppTheme.rainbowColors.length-1)],
        blurRadius: 8,
        offset: Offset(0, 8),
        spreadRadius: 0,
      )
    ],borderSide: const BorderSide(color: Colors.black12),
    backgroundColor: darkBackground,
    closeOnClick: false,
    dragToClose: true,
    showProgressBar: progress,
    primaryColor: Colors.blueAccent,
  );
}
