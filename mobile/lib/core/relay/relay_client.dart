import 'dart:async';
import 'dart:convert';

import 'package:web_socket_channel/web_socket_channel.dart';

import '../../pb/wire.dart';
import 'relay_state.dart';

/// A tiny exponential backoff (1s, 2s, 4s, … capped at 30s).
class _Backoff {
  static const Duration _base = Duration(seconds: 1);
  static const Duration _cap = Duration(seconds: 30);
  int _attempt = 0;

  Duration next() {
    final int mult = 1 << (_attempt.clamp(0, 5));
    _attempt += 1;
    final Duration d = _base * mult;
    return d > _cap ? _cap : d;
  }

  void reset() => _attempt = 0;
}

/// WebSocket wrapper around the relay's `/ws` endpoint.
///
/// - `connect(baseUrl, token, lastSeq)` opens the socket and emits
///   decoded [Message]s on [messages] and lifecycle changes on [state].
/// - On close or error, reconnects after exponential backoff (1..30s),
///   re-reading `last_seq` via the supplied [maxSeqProvider] first.
/// - `send(Message)` serializes and writes; caller is responsible for
///   checking connectivity — no buffering.
class RelayClient {
  RelayClient({required Future<int> Function() maxSeqProvider})
      : _maxSeqProvider = maxSeqProvider;

  final Future<int> Function() _maxSeqProvider;

  final StreamController<RelayState> _stateCtl =
      StreamController<RelayState>.broadcast();
  final StreamController<Message> _msgCtl = StreamController<Message>.broadcast();

  Stream<RelayState> get state => _stateCtl.stream;
  Stream<Message> get messages => _msgCtl.stream;

  RelayState _current = RelayState.disconnected;
  RelayState get currentState => _current;

  WebSocketChannel? _channel;
  StreamSubscription<Object?>? _sub;
  final _Backoff _backoff = _Backoff();
  Timer? _reconnectTimer;

  String? _baseUrl;
  String? _token;
  bool _stopped = false;

  /// Opens the initial WebSocket connection.
  Future<void> connect(String baseUrl, String token, int lastSeq) async {
    _baseUrl = baseUrl;
    _token = token;
    _stopped = false;
    await _open(lastSeq);
  }

  /// Tear down and stop reconnecting.
  Future<void> close() async {
    _stopped = true;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    await _sub?.cancel();
    _sub = null;
    await _channel?.sink.close();
    _channel = null;
    _setState(RelayState.disconnected);
  }

  /// Serialize and write `m`. Returns true when a socket was available.
  bool send(Message m) {
    final WebSocketChannel? ch = _channel;
    if (ch == null || _current != RelayState.connected) return false;
    ch.sink.add(jsonEncode(m.toJson()));
    return true;
  }

  // --- internals ---

  Future<void> _open(int lastSeq) async {
    if (_stopped) return;
    final String? base = _baseUrl;
    final String? token = _token;
    if (base == null || token == null) return;

    _setState(RelayState.connecting);
    final Uri wsUri = _buildWsUri(base, token, lastSeq);

    try {
      final WebSocketChannel ch = WebSocketChannel.connect(wsUri);
      await ch.ready;
      _channel = ch;
      _backoff.reset();
      _setState(RelayState.connected);

      _sub = ch.stream.listen(
        _onData,
        onError: (Object err, StackTrace st) => _scheduleReconnect(),
        onDone: _scheduleReconnect,
        cancelOnError: true,
      );
    } catch (_) {
      _scheduleReconnect();
    }
  }

  void _onData(Object? data) {
    if (data == null) return;
    final String raw = data is String ? data : (data is List<int> ? utf8.decode(data) : data.toString());
    try {
      final Object? decoded = jsonDecode(raw);
      if (decoded is Map<String, dynamic>) {
        _msgCtl.add(Message.fromJson(decoded));
      }
    } catch (_) {
      // Drop malformed frames silently; server is source of truth.
    }
  }

  void _scheduleReconnect() {
    if (_stopped) return;
    _reconnectTimer?.cancel();
    _sub?.cancel();
    _sub = null;
    _channel = null;
    _setState(RelayState.reconnecting);

    final Duration wait = _backoff.next();
    _reconnectTimer = Timer(wait, () async {
      final int seq = await _maxSeqProvider();
      await _open(seq);
    });
  }

  void _setState(RelayState s) {
    if (_current == s) return;
    _current = s;
    _stateCtl.add(s);
  }

  Uri _buildWsUri(String baseUrl, String token, int lastSeq) {
    final Uri http = Uri.parse(baseUrl);
    final String scheme = (http.scheme == 'https') ? 'wss' : 'ws';
    return Uri(
      scheme: scheme,
      host: http.host,
      port: http.hasPort ? http.port : null,
      path: '${http.path.isEmpty ? '' : http.path.replaceAll(RegExp(r'/+$'), '')}/ws',
      queryParameters: <String, String>{
        'token': token,
        'last_seq': lastSeq.toString(),
      },
    );
  }
}
