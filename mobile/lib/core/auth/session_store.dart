import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:uuid/uuid.dart';

/// Persisted auth state for the mobile provider.
///
/// Uses `flutter_secure_storage` for all keys — on iOS this is the Keychain
/// and on Android the AndroidKeystore-backed EncryptedSharedPreferences.
///
/// Mirrors the browser layout at a higher level:
/// - session_token + user_id + email for the account
/// - device_token for the WS handshake
/// - device_id stays put across logout so devices table entries don't spray
class SessionStore {
  SessionStore({FlutterSecureStorage? storage, Uuid? uuid})
      : _storage = storage ?? const FlutterSecureStorage(),
        _uuid = uuid ?? const Uuid();

  final FlutterSecureStorage _storage;
  final Uuid _uuid;

  static const String _kSession = 'appunvs.session_token';
  static const String _kDeviceToken = 'appunvs.token';
  static const String _kDeviceId = 'appunvs.device_id';
  static const String _kUserId = 'appunvs.user_id';
  static const String _kEmail = 'appunvs.email';

  Future<String> sessionToken() async =>
      (await _storage.read(key: _kSession)) ?? '';

  Future<String> deviceToken() async =>
      (await _storage.read(key: _kDeviceToken)) ?? '';

  Future<String> userId() async =>
      (await _storage.read(key: _kUserId)) ?? '';

  Future<String> email() async => (await _storage.read(key: _kEmail)) ?? '';

  Future<String> deviceId() async =>
      (await _storage.read(key: _kDeviceId)) ?? '';

  /// Save the account session returned by /auth/signup or /auth/login.
  ///
  /// Clears any previously-cached device token: a new session always
  /// forces a fresh /auth/register so server + client stay in sync.
  Future<void> saveSession(String userId, String sessionToken, String email) async {
    await _storage.write(key: _kSession, value: sessionToken);
    await _storage.write(key: _kUserId, value: userId);
    await _storage.write(key: _kEmail, value: email);
    await _storage.delete(key: _kDeviceToken);
  }

  /// Save the device token returned by /auth/register.
  Future<void> saveDeviceToken(String token) async {
    await _storage.write(key: _kDeviceToken, value: token);
  }

  /// Returns a stable device id, generating one on first call.
  Future<String> ensureDeviceId() async {
    final String? existing = await _storage.read(key: _kDeviceId);
    if (existing != null && existing.isNotEmpty) return existing;
    final String fresh = _uuid.v4();
    await _storage.write(key: _kDeviceId, value: fresh);
    return fresh;
  }

  /// Clear session + device token + account identity.  Keeps device_id so
  /// the same install does not spray fresh entries into the devices table
  /// across logout/login cycles.
  Future<void> logout() async {
    await _storage.delete(key: _kSession);
    await _storage.delete(key: _kDeviceToken);
    await _storage.delete(key: _kUserId);
    await _storage.delete(key: _kEmail);
  }
}
