import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:speedplus/core/util/colors.dart';

class AppElevatedButtonTheme {
  AppElevatedButtonTheme._();

  static final appElevatedButtonTheme = ElevatedButtonThemeData(
    style: ElevatedButton.styleFrom(
      backgroundColor: primary,
      foregroundColor: Colors.white,
      fixedSize: const Size(double.infinity, 48),
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8.0),
      ),
      textStyle: GoogleFonts.inter(
        fontWeight: FontWeight.w600,
        fontSize: 16.0,
      ),
    ),
  );
}
