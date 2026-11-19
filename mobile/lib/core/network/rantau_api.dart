import 'dart:convert';

import 'package:http/http.dart' as http;

class ApiFailure implements Exception {
  const ApiFailure(this.message, this.statusCode);

  final String message;
  final int statusCode;
}

class RantauApi {
  RantauApi({required this.client, required this.baseUrl});

  final http.Client client;
  final Uri baseUrl;
  ({String access, String refresh})? _session;
  Future<void>? _refreshing;

  bool get signedIn => _session != null;

  Future<void> login(String email, String password) async {
    final response = await request(
      'POST',
      '/api/v1/auth/login',
      body: {'email': email, 'password': password},
    );
    _storeTokens(data(response));
  }

  Future<http.Response> request(
    String method,
    String path, {
    Map<String, Object?>? body,
    bool private = false,
  }) async {
    if (!private) return _send(method, path, body);
    final session = _session;
    if (session == null) {
      throw const ApiFailure('Masuk kembali untuk melanjutkan.', 401);
    }
    var response = await _send(method, path, body, token: session.access);
    if (response.statusCode == 401) {
      if (_session == session) {
        _refreshing ??= _renew(session.refresh).whenComplete(() {
          _refreshing = null;
        });
        await _refreshing;
      }
      final access = _session?.access;
      if (access != null) {
        response = await _send(method, path, body, token: access);
        if (response.statusCode == 401 && _session?.access == access) {
          _session = null;
        }
      }
      if (response.statusCode == 401) {
        throw const ApiFailure(
          'Sesi berakhir. Masuk kembali untuk melanjutkan.',
          401,
        );
      }
    }
    return response;
  }

  Future<void> _renew(String refresh) async {
    try {
      final response = await _send('POST', '/api/v1/auth/refresh', {
        'refresh_token': refresh,
      });
      _storeTokens(data(response));
    } on ApiFailure catch (error) {
      if (error.statusCode == 401) {
        _session = null;
        throw const ApiFailure(
          'Sesi berakhir. Masuk kembali untuk melanjutkan.',
          401,
        );
      }
      rethrow;
    }
  }

  void _storeTokens(Map<String, dynamic> payload) {
    _session = (
      access: payload['access_token'] as String,
      refresh: payload['refresh_token'] as String,
    );
  }

  Future<http.Response> _send(
    String method,
    String path,
    Map<String, Object?>? body, {
    String? token,
  }) async {
    final request = http.Request(method, baseUrl.resolve(path));
    request.headers['Accept'] = 'application/json';
    if (token != null) request.headers['Authorization'] = 'Bearer $token';
    if (body != null) {
      request.headers['Content-Type'] = 'application/json; charset=utf-8';
      request.body = jsonEncode(body);
    }
    return (() async {
      return http.Response.fromStream(await client.send(request));
    })().timeout(const Duration(seconds: 10));
  }

  static void check(http.Response response) {
    final status = response.statusCode;
    if (status >= 200 && status < 300) return;
    String? message;
    try {
      final payload = jsonDecode(utf8.decode(response.bodyBytes));
      if (payload case {'error': {'message': final String error}}) {
        message = error;
      }
    } on FormatException {
      // Respond with the fallback below when the server returns plain text.
    }
    throw ApiFailure(message ?? 'Permintaan gagal. Coba lagi.', status);
  }

  static Map<String, dynamic> data(http.Response response) {
    check(response);
    final payload =
        jsonDecode(utf8.decode(response.bodyBytes)) as Map<String, dynamic>;
    return payload['data'] as Map<String, dynamic>;
  }
}
