import 'package:flutter/material.dart';

/// Palette shared with the web dashboard (web/index.html).
abstract final class AppColors {
  static const background = Color(0xFF070B0D);
  static const panel = Color(0xFF0D1417);
  static const line = Color(0xFF1D3432);
  static const text = Color(0xFFD9FFF3);
  static const muted = Color(0xFF71918B);
  static const green = Color(0xFF57F2B1);
  static const cyan = Color(0xFF4DE4F5);
  static const amber = Color(0xFFFFD166);
  static const red = Color(0xFFFF6B7A);
}

ThemeData buildShishimonTheme() {
  const scheme = ColorScheme.dark(
    primary: AppColors.green,
    onPrimary: Color(0xFF06110D),
    secondary: AppColors.cyan,
    onSecondary: Color(0xFF041316),
    surface: AppColors.panel,
    onSurface: AppColors.text,
    error: AppColors.red,
    onError: Color(0xFF1A0508),
  );

  final base = ThemeData(
    useMaterial3: true,
    brightness: Brightness.dark,
    colorScheme: scheme,
    scaffoldBackgroundColor: AppColors.background,
    fontFamily: 'monospace',
  );

  final inputBorder = OutlineInputBorder(
    borderRadius: BorderRadius.circular(4),
    borderSide: const BorderSide(color: AppColors.line),
  );

  return base.copyWith(
    appBarTheme: const AppBarTheme(
      backgroundColor: AppColors.background,
      foregroundColor: AppColors.text,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      centerTitle: false,
      titleTextStyle: TextStyle(
        fontFamily: 'monospace',
        fontSize: 16,
        fontWeight: FontWeight.w700,
        letterSpacing: 1.2,
        color: AppColors.text,
      ),
    ),
    inputDecorationTheme: InputDecorationTheme(
      filled: true,
      fillColor: AppColors.panel,
      hintStyle: const TextStyle(color: AppColors.muted, fontSize: 12),
      contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
      enabledBorder: inputBorder,
      focusedBorder: inputBorder.copyWith(
        borderSide: const BorderSide(color: AppColors.green),
      ),
      errorBorder: inputBorder.copyWith(
        borderSide: const BorderSide(color: AppColors.red),
      ),
      focusedErrorBorder: inputBorder.copyWith(
        borderSide: const BorderSide(color: AppColors.red),
      ),
    ),
    snackBarTheme: const SnackBarThemeData(
      backgroundColor: AppColors.panel,
      contentTextStyle: TextStyle(color: AppColors.text, fontSize: 12),
      behavior: SnackBarBehavior.floating,
    ),
    elevatedButtonTheme: ElevatedButtonThemeData(
      style: ElevatedButton.styleFrom(
        backgroundColor: AppColors.green,
        foregroundColor: const Color(0xFF06110D),
        minimumSize: const Size.fromHeight(48),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
        textStyle: const TextStyle(
          fontFamily: 'monospace',
          fontWeight: FontWeight.w700,
          letterSpacing: 1.2,
        ),
      ),
    ),
    outlinedButtonTheme: OutlinedButtonThemeData(
      style: OutlinedButton.styleFrom(
        foregroundColor: AppColors.cyan,
        side: const BorderSide(color: AppColors.line),
        minimumSize: const Size(0, 44),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(4)),
        textStyle: const TextStyle(
          fontFamily: 'monospace',
          fontWeight: FontWeight.w700,
          letterSpacing: 1,
        ),
      ),
    ),
    textButtonTheme: TextButtonThemeData(
      style: TextButton.styleFrom(foregroundColor: AppColors.cyan),
    ),
    dividerTheme: const DividerThemeData(color: AppColors.line, thickness: 1),
    progressIndicatorTheme: const ProgressIndicatorThemeData(
      color: AppColors.green,
      linearTrackColor: AppColors.panel,
    ),
    popupMenuTheme: const PopupMenuThemeData(
      color: AppColors.panel,
      surfaceTintColor: Colors.transparent,
      textStyle: TextStyle(color: AppColors.text, fontSize: 13),
    ),
  );
}
