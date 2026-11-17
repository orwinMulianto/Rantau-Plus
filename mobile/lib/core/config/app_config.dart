import 'package:flutter/foundation.dart';

abstract final class AppConfig {
  static Uri get apiBaseUrl {
    const configured = String.fromEnvironment('API_BASE_URL');
    if (configured.isNotEmpty) return Uri.parse(configured);

    final host = defaultTargetPlatform == TargetPlatform.android
        ? '10.0.2.2'
        : '127.0.0.1';
    return Uri.parse('http://$host:8080');
  }
}
