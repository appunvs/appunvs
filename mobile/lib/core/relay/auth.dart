import 'dart:convert';

import 'package:http/http.dart' as http;

import '../../pb/wire.dart';

/// HTTP client for the relay's /auth/* endpoints.
///
/// Mirrors the proto-defined API exactly:
///   POST /auth/signup      public     {email, password}       → SessionResponse
///   POST /auth/login       public     {email, password}       → SessionResponse
///   POST /auth/register    session    {device_id, platform}   → DeviceRegistration
///   GET  /auth/me          session                            → MeResponse
///
/// Session-authed requests carry `Authorization: Bearer <session_token>`.
class AccountClient {
  AccountClient(this.baseUrl, {http.Client? client})
      : _client = client ?? http.Client();

  final String baseUrl;
  final http.Client _client;

  /// Create a new account.
  Future<SessionResponse> signup(String email, String password) async {
    final Uri uri = Uri.parse('${_stripSlash(baseUrl)}/auth/signup');
    final http.Response res = await _client.post(
      uri,
      headers: <String, String>{'content-type': 'application/json'},
      body: jsonEncode(SignupRequest(email: email, password: password).toJson()),
    );
    return _decodeSession(res, op: 'signup');
  }

  /// Log into an existing account.
  Future<SessionResponse> login(String email, String password) async {
    final Uri uri = Uri.parse('${_stripSlash(baseUrl)}/auth/login');
    final http.Response res = await _client.post(
      uri,
      headers: <String, String>{'content-type': 'application/json'},
      body: jsonEncode(LoginRequest(email: email, password: password).toJson()),
    );
    return _decodeSession(res, op: 'login');
  }

  /// Register this install as a device for the logged-in user.
  /// Requires a valid session token.
  Future<DeviceRegistration> registerDevice({
    required String sessionToken,
    required String deviceId,
    required Platform platform,
  }) async {
    final Uri uri = Uri.parse('${_stripSlash(baseUrl)}/auth/register');
    final http.Response res = await _client.post(
      uri,
      headers: <String, String>{
        'content-type': 'application/json',
        'authorization': 'Bearer $sessionToken',
      },
      body: jsonEncode(
        RegisterRequest(deviceId: deviceId, platform: platform).toJson(),
      ),
    );
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw AuthException(res.statusCode, _readError(res));
    }
    final Object? decoded = jsonDecode(res.body);
    if (decoded is! Map<String, dynamic>) {
      throw AuthException(res.statusCode, 'non-object response: ${res.body}');
    }
    final RegisterResponse parsed = RegisterResponse.fromJson(decoded);
    return DeviceRegistration(token: parsed.token, userId: parsed.userId);
  }

  /// Fetch the logged-in user's profile + registered devices.
  Future<MeResponse> me({required String sessionToken}) async {
    final Uri uri = Uri.parse('${_stripSlash(baseUrl)}/auth/me');
    final http.Response res = await _client.get(
      uri,
      headers: <String, String>{'authorization': 'Bearer $sessionToken'},
    );
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw AuthException(res.statusCode, _readError(res));
    }
    final Object? decoded = jsonDecode(res.body);
    if (decoded is! Map<String, dynamic>) {
      throw AuthException(res.statusCode, 'non-object response: ${res.body}');
    }
    return MeResponse.fromJson(decoded);
  }

  void close() => _client.close();

  SessionResponse _decodeSession(http.Response res, {required String op}) {
    if (res.statusCode < 200 || res.statusCode >= 300) {
      throw AuthException(res.statusCode, _readError(res));
    }
    final Object? decoded = jsonDecode(res.body);
    if (decoded is! Map<String, dynamic>) {
      throw AuthException(res.statusCode, 'non-object $op response: ${res.body}');
    }
    return SessionResponse.fromJson(decoded);
  }

  String _readError(http.Response res) {
    try {
      final Object? decoded = jsonDecode(res.body);
      if (decoded is Map<String, dynamic>) {
        final Object? err = decoded['error'];
        if (err is String && err.isNotEmpty) return err;
      }
    } catch (_) {
      // fall through
    }
    return res.body;
  }
}

/// Typed result of /auth/register.
class DeviceRegistration {
  DeviceRegistration({required this.token, required this.userId});
  final String token;
  final String userId;
}

class AuthException implements Exception {
  AuthException(this.status, this.body);
  final int status;
  final String body;
  @override
  String toString() => 'AuthException($status): $body';
}

String _stripSlash(String s) => s.endsWith('/') ? s.substring(0, s.length - 1) : s;
