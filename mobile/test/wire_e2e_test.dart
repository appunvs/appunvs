// Cross-language E2E: drives the real Go relay using the Dart wire codec.
//
// Skips gracefully when APPUNVS_RELAY_BASE (default http://localhost:8080)
// is unreachable so `flutter test` on a laptop stays green.
import 'dart:convert';
import 'dart:io' as io;

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import 'package:appunvs_mobile/core/relay/auth.dart';
import 'package:appunvs_mobile/pb/wire.dart' as pb;

String get _base =>
    io.Platform.environment['APPUNVS_RELAY_BASE'] ?? 'http://localhost:8080';

String get _wsBase => _base.replaceFirst('http', 'ws');

Future<bool> _relayUp() async {
  try {
    final http.Response r = await http
        .get(Uri.parse('$_base/health'))
        .timeout(const Duration(milliseconds: 500));
    return r.statusCode == 200;
  } catch (_) {
    return false;
  }
}

String _uniqueEmail(String tag) =>
    '$tag-${DateTime.now().microsecondsSinceEpoch}@mobile-e2e.local';

Future<({pb.SessionResponse session, DeviceRegistration device})> _signupAndRegister(
  AccountClient client,
  String deviceId,
) async {
  final pb.SessionResponse session =
      await client.signup(_uniqueEmail('dart-e2e'), 'hunter2hunter2');
  final DeviceRegistration device = await client.registerDevice(
    sessionToken: session.sessionToken,
    deviceId: deviceId,
    platform: pb.Platform.mobile,
  );
  return (session: session, device: device);
}

Future<WebSocketChannel> _dial(String token, {int? lastSeq}) async {
  final Uri u = Uri.parse('$_wsBase/ws').replace(
    queryParameters: <String, String>{
      'token': token,
      if (lastSeq != null && lastSeq > 0) 'last_seq': lastSeq.toString(),
    },
  );
  final IOWebSocketChannel ch = IOWebSocketChannel.connect(u);
  await ch.ready;
  return ch;
}

Future<pb.Message> _nextMessage(WebSocketChannel ch,
    {Duration within = const Duration(seconds: 3)}) async {
  final Object? raw = await ch.stream.first.timeout(within);
  final Map<String, dynamic> decoded =
      jsonDecode(raw.toString()) as Map<String, dynamic>;
  return pb.Message.fromJson(decoded);
}

void main() {
  group('relay e2e (Dart wire <-> Go relay)', () {
    test('provider upsert echoes back with a relay-assigned seq', () async {
      if (!await _relayUp()) {
        // ignore: avoid_print
        print('relay not reachable; skipping');
        return;
      }
      final AccountClient client = AccountClient(_base);
      final ({pb.SessionResponse session, DeviceRegistration device}) reg =
          await _signupAndRegister(client, 'dart-e2e-device');
      final WebSocketChannel ch = await _dial(reg.device.token);

      final pb.Message msg = pb.Message(
        deviceId: 'dart-e2e-device',
        userId: reg.device.userId,
        namespace: reg.device.userId,
        role: pb.Role.provider,
        op: pb.Op.upsert,
        table: 'records',
        payload: <String, dynamic>{'id': 'r1', 'data': 'dart-says-hi'},
        ts: DateTime.now().millisecondsSinceEpoch,
      );
      ch.sink.add(jsonEncode(msg.toJson()));

      final pb.Message echo = await _nextMessage(ch);
      expect(echo.seq, greaterThan(0));
      expect(echo.namespace, reg.device.userId);
      expect(echo.role, pb.Role.provider);
      expect(echo.op, pb.Op.upsert);
      expect(echo.payload?['id'], 'r1');

      await ch.sink.close();
      client.close();
    }, timeout: const Timeout(Duration(seconds: 15)));
  });
}
