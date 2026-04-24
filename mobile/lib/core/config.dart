/// App-wide configuration values.
class AppConfig {
  /// Base URL of the Go relay. Override with:
  ///   flutter run --dart-define=RELAY_BASE=http://10.0.2.2:8080
  static const String relayBase = String.fromEnvironment(
    'RELAY_BASE',
    defaultValue: 'http://10.0.2.2:8080',
  );

  /// Default table name pushed by this provider.
  static const String defaultTable = 'records';
}
