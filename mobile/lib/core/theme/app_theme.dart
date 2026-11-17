import 'package:flutter/material.dart';

ThemeData buildAppTheme() {
  final scheme = ColorScheme.fromSeed(
    seedColor: const Color(0xFF246B55),
    surface: const Color(0xFFF6F7F2),
  );
  return ThemeData(
    colorScheme: scheme,
    scaffoldBackgroundColor: scheme.surface,
    appBarTheme: AppBarTheme(backgroundColor: scheme.surface),
  );
}
