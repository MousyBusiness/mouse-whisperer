import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';

class AppTheme {
   static List<Color> rainbowColors = [
    Color(0xFFDEF268), // 0
    Color(0xFF5CED81), // 1
    Color(0xFF62D5E4), // 2
    Color(0xFFFFFFFF), // 3
    Color(0xFFBB8EF5), // 4
    Color(0xFFE874B4), // 5
    Color(0xFFFF7F6B), // 6
    Color(0xFFF2C65E), // 7
  ];

  static final codeStyleBase = GoogleFonts.jetBrainsMono(
    textStyle: const TextStyle(
      fontSize: 13.0,
      height: 1.2,
    ),
  );

  static final headline1 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 72.0,
        fontWeight: FontWeight.w200,
        letterSpacing: -1.5,
        color: Colors.white),
  );

  static final headline2 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 60.0,
        fontWeight: FontWeight.normal,
        letterSpacing: -0.5,
        color: Colors.white),
  );

  static final headline3 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 48.0, fontWeight: FontWeight.w600, color: Colors.white),
  );

  static final headline4 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 32.0, fontWeight: FontWeight.w600, color: Colors.white),
  );

  static final headline5 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 24.0,
        fontWeight: FontWeight.bold,
        letterSpacing: 0.18,
        color: Colors.white),
  );

  static final headline6 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 20.0,
        fontWeight: FontWeight.w600,
        letterSpacing: 0.15,
        color: Colors.white),
  );

  static final subtitle1 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 16.0,
        fontWeight: FontWeight.bold,
        letterSpacing: 0.15,
        color: Colors.white),
  );

  static final subtitle2 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 14.0,
        fontWeight: FontWeight.w600,
        letterSpacing: 0.1,
        color: Colors.white),
  );

  static final body1 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 16.0,
        fontWeight: FontWeight.normal,
        letterSpacing: 0.5,
        color: Colors.white),
  );

  static final body2 = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 14.0,
        fontWeight: FontWeight.normal,
        letterSpacing: 0.25,
        color: Colors.white),
  );

  static final button = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 14.0, fontWeight: FontWeight.w500, color: Colors.white),
  );

  static final overline = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 9.0,
        fontWeight: FontWeight.w600,
        letterSpacing: 1,
        color: Colors.white),
  );

  static final caption = GoogleFonts.poppins(
    textStyle: const TextStyle(
        fontSize: 12.0,
        fontWeight: FontWeight.w500,
        letterSpacing: 0.4,
        color: Colors.white),
  );

}
