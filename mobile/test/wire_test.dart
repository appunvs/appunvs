// Unit tests for the Dart protojson wire codec.  These match the same
// golden inputs used in relay/, browser/ and desktop/ to prove all four
// codecs agree on the JSON shape.
import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:appunvs_mobile/pb/wire.dart';

void main() {
  group('wire codec', () {
    test('roundtrips a canonical protojson Message', () {
      final Message src = Message(
        seq: 1024,
        deviceId: 'd1',
        userId: 'u1',
        namespace: 'u1',
        role: Role.provider,
        op: Op.upsert,
        table: 'records',
        payload: <String, dynamic>{'id': 'r1', 'data': 'hello'},
        ts: 1714000000000,
      );
      final String wire = jsonEncode(src.toJson());
      final Message parsed =
          Message.fromJson(jsonDecode(wire) as Map<String, dynamic>);
      expect(parsed.seq, src.seq);
      expect(parsed.deviceId, src.deviceId);
      expect(parsed.userId, src.userId);
      expect(parsed.namespace, src.namespace);
      expect(parsed.role, src.role);
      expect(parsed.op, src.op);
      expect(parsed.table, src.table);
      expect(parsed.payload, src.payload);
      expect(parsed.ts, src.ts);
    });

    test('omits seq when zero (relay assigns it)', () {
      final Message m = Message(
        deviceId: 'd1',
        userId: 'u1',
        namespace: 'u1',
        role: Role.provider,
        op: Op.upsert,
        table: 'records',
        payload: <String, dynamic>{'id': 'r1'},
        ts: 1,
      );
      expect(m.toJson().containsKey('seq'), isFalse);
    });

    test('serializes enums as short lowercase strings', () {
      final Message m = Message(
        deviceId: 'd',
        userId: 'u',
        namespace: 'u',
        role: Role.connector,
        op: Op.delete,
        table: 't',
        ts: 1,
      );
      final Map<String, dynamic> j = m.toJson();
      expect(j['role'], 'connector');
      expect(j['op'], 'delete');
    });

    test('tolerates int64-as-string (protojson quirk)', () {
      final Map<String, dynamic> raw = <String, dynamic>{
        'seq': '42',
        'device_id': 'd',
        'user_id': 'u',
        'namespace': 'u',
        'role': 'provider',
        'op': 'upsert',
        'table': 't',
        'ts': '1',
      };
      final Message m = Message.fromJson(raw);
      expect(m.seq, 42);
      expect(m.ts, 1);
    });

    test('unknown role falls back to unspecified (forward compatibility)', () {
      final Map<String, dynamic> raw = <String, dynamic>{
        'device_id': 'd',
        'user_id': 'u',
        'namespace': 'u',
        'role': 'superadmin',
        'op': 'upsert',
        'table': 't',
        'ts': 0,
      };
      final Message m = Message.fromJson(raw);
      expect(m.role, Role.unspecified);
    });
  });
}
