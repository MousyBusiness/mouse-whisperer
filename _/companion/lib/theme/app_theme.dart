import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'app_colors.dart';

/*
 * How to use:
 * Text("Hello", style: Theme.of(context).textTheme.headline1,),
 */

class AppTheme {
  static final headline1 = GoogleFonts.poppins(
    textStyle: const TextStyle(
      fontSize: 72.0,
      fontWeight: FontWeight.w200,
      letterSpacing: -1.5,
      color: AppColors.white,
    ),
  );

  static final headline2 = GoogleFonts.poppins(
    textStyle: const TextStyle(
      fontSize: 60.0,
      fontWeight: FontWeight.w200,
      letterSpacing: -0.5,
      color: AppColors.white,
    ),
  );

  static final headline3 = GoogleFonts.poppins(
    textStyle: const TextStyle(fontSize: 48.0, fontWeight: FontWeight.w200, color: AppColors.white),
  );

  static final headline4 = GoogleFonts.poppins(
    textStyle: const TextStyle(fontSize: 32.0, fontWeight: FontWeight.w200, color: AppColors.white),
  );

  static final headline5 = GoogleFonts.poppins(
    textStyle: const TextStyle(
      fontSize: 24.0,
      fontWeight: FontWeight.w200,
      letterSpacing: 0.18,
      color: AppColors.white,
    ),
  );

  static final headline6 = GoogleFonts.poppins(
    textStyle: const TextStyle(
      fontSize: 20.0,
      fontWeight: FontWeight.w200,
      letterSpacing: 0.15,
      color: AppColors.white,
    ),
  );

  static final subtitle1 = GoogleFonts.poppins(
    textStyle: const TextStyle(
      fontSize: 14.0,
      fontWeight: FontWeight.bold,
      letterSpacing: 0.15,
      color: AppColors.white,
    ),
  );

  static final subtitle2 = GoogleFonts.poppins(
    textStyle: const TextStyle(fontSize: 12.0, fontWeight: FontWeight.w600, letterSpacing: 0.1, color: AppColors.white),
  );

  static final body1 = GoogleFonts.poppins(
    textStyle: const TextStyle(
      fontSize: 14.0,
      fontWeight: FontWeight.normal,
      letterSpacing: 0.5,
      color: AppColors.white,
    ),
  );

  static final body2 = GoogleFonts.poppins(
    textStyle: const TextStyle(
      fontSize: 12.0,
      fontWeight: FontWeight.normal,
      letterSpacing: 0.25,
      color: AppColors.white,
    ),
  );

  static final button = GoogleFonts.poppins(
    textStyle: const TextStyle(fontSize: 12.0, fontWeight: FontWeight.w500, color: AppColors.white),
  );

  static final overline = GoogleFonts.poppins(
    textStyle: const TextStyle(fontSize: 9.0, fontWeight: FontWeight.w600, letterSpacing: 1, color: AppColors.white),
  );

  static final caption = GoogleFonts.poppins(
    textStyle: const TextStyle(fontSize: 12.0, fontWeight: FontWeight.w500, letterSpacing: 0.4, color: AppColors.white),
  );

  static final codeStyleBase = GoogleFonts.jetBrainsMono(
    textStyle: const TextStyle(
      inherit: false,
      fontSize: 13.0,
      textBaseline: TextBaseline.alphabetic,
      height: 1.2,
      letterSpacing: 1,
    ),
  );

  static ThemeData get lightTheme {
    return ThemeData(primaryColor: AppColors.primary, scaffoldBackgroundColor: Colors.white);
  }
}
