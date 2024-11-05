import 'package:flutter/material.dart';
import 'package:google_fonts/google_fonts.dart';
import 'package:speedplus/core/util/colors.dart';

class AppTextTheme {
  AppTextTheme._();

  // light text theme
  static TextTheme lightTextTheme = TextTheme(
    // title texts
    titleSmall: GoogleFonts.inter(
      fontWeight: FontWeight.w400,
      color: black,
    ),
    titleMedium: GoogleFonts.inter(
      fontWeight: FontWeight.w600,
      color: black,
    ),
    titleLarge: GoogleFonts.inter(
      fontWeight: FontWeight.w700,
      color: black,
    ),

    // body texts
    bodySmall: GoogleFonts.inter(
      fontWeight: FontWeight.w300,
      color: black,
    ),
    bodyMedium: GoogleFonts.inter(
      fontWeight: FontWeight.w400,
      color: black,
    ),
    bodyLarge: GoogleFonts.inter(
      fontWeight: FontWeight.w500,
      color: black,
    ),

    // label texts
    labelSmall: GoogleFonts.inter(
      fontWeight: FontWeight.w400,
      color: black,
    ),
    labelMedium: GoogleFonts.inter(
      fontWeight: FontWeight.w500,
      color: black,
    ),
    labelLarge: GoogleFonts.inter(
      fontWeight: FontWeight.w600,
      color: black,
    ),
  );

  // dark text theme
  static TextTheme darkTextTheme = TextTheme(
    // title texts
    titleSmall: GoogleFonts.inter(
      fontWeight: FontWeight.w400,
      color: white,
    ),
    titleMedium: GoogleFonts.inter(
      fontWeight: FontWeight.w600,
      color: white,
    ),
    titleLarge: GoogleFonts.inter(
      fontWeight: FontWeight.w700,
      color: white,
    ),

    // body texts
    bodySmall: GoogleFonts.inter(
      fontWeight: FontWeight.w300,
      color: white,
    ),
    bodyMedium: GoogleFonts.inter(
      fontWeight: FontWeight.w400,
      color: white,
    ),
    bodyLarge: GoogleFonts.inter(
      fontWeight: FontWeight.w500,
      color: white,
    ),

    // label texts
    labelSmall: GoogleFonts.inter(
      fontWeight: FontWeight.w400,
      color: white,
    ),
    labelMedium: GoogleFonts.inter(
      fontWeight: FontWeight.w500,
      color: white,
    ),
    labelLarge: GoogleFonts.inter(
      fontWeight: FontWeight.w600,
      color: white,
    ),
  );
}
