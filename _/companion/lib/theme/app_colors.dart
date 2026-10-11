import 'package:flutter/material.dart';

class AppColors {
  static const Color primary = Color(0xFFFDDA5F);
  static const Color secondary = Color(0xFFADB7F1);
  static const Color lightAccent = Color(0xFF6ED1D0);
  static const Color darkAccent = Color(0xFF243153);

  static const Color black = Color(0xFF2C2E33);
  static const Color darkGrey = Color(0xFF3B3F47);
  static const Color midGrey = Color(0xFF8C8E99);
  static const Color lightGrey = Color(0xFFDEDEDE);
  static const Color divider = Color.fromARGB(255, 238, 238, 238);

// backgroundColor: Color(0xff1c1d21), color: Color(0xffc0c5ce)
  static const Color codeBackground = Color(0xff1c1d21);
  static const Color inputFieldBackground = codeBackground;
  static const Color inputFieldOutline = midGrey;

  // static const Color canvas = Color(0xFF242E35); // Original
  // static const Color canvas = Color(0xFF2C3040); // Gun Metal - Poor contrast
  // static const Color canvas = Color(0xFF1F232B); // Raisin Black - Good contrast, quite dart
  // static const Color canvas = Color(0xFF2B303B); // Also Gun Metal - Poor contrast
  static const Color canvas = Color(0xFF333A47); // Charcoal - Nice contrast
  // static const Color canvas = Color(0xFF3A3F55); // Delft Blue
  static const Color canvasSurface = Color(0xFF2B303B);
  static const Color canvasSurfaceDark = Color(0xFF1F232B);

  static const Color green = Colors.green;
  static const Color orange = Colors.amber;
  static Color red = Colors.redAccent;

  static const Color paleBlue = Color(0xFFF0F7FF);

  static const Color white = Color(0xFFFFFFFF);

  static const Color toolMiddleButton = Color(0xFF373B43);
  static const Color toolSectorButton = Color(0xFF353B45);
  static const Color toolSectorButtonSelected = Color(0xFF353B45);
  static const Color toolLines = Color(0xFF373B43);

  static Map<int, Color> primarySwatch = {
    50: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .1),
    100: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .2),
    200: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .3),
    300: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .4),
    400: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .5),
    500: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .6),
    600: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .7),
    700: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .8),
    800: const Color.fromRGBO(0xFD, 0xDA, 0x5F, .9),
    900: const Color.fromRGBO(0xFD, 0xDA, 0x5F, 1),
  };

  static Map<int, Color> secondarySwatch = {
    50: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .1),
    100: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .2),
    200: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .3),
    300: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .4),
    400: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .5),
    500: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .6),
    600: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .7),
    700: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .8),
    800: const Color.fromRGBO(0xAD, 0xB7, 0xF1, .9),
    900: const Color.fromRGBO(0xAD, 0xB7, 0xF1, 1),
  };

  static Map<int, Color> accentSwatch = {
    50: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .1),
    100: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .2),
    200: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .3),
    300: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .4),
    400: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .5),
    500: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .6),
    600: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .7),
    700: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .8),
    800: const Color.fromRGBO(0x6E, 0xD1, 0xD0, .9),
    900: const Color.fromRGBO(0x6E, 0xD1, 0xD0, 1),
  };

  static Color transparent = Colors.transparent;
}
