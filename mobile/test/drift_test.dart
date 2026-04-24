// Drift check: loads the repo-wide golden fixture and asserts the Dart wire
// codec roundtrips each case. Paths are resolved relative to the mobile/
// package root because `flutter test` sets that as CWD.
import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:appunvs_mobile/pb/wire.dart';

Object? _sort(Object? v) {
  if (v is Map<String, dynamic>) {
    final List<String> keys = v.keys.toList()..sort();
    return <String, dynamic>{
      for (final String k in keys) k: _sort(v[k]),
    };
  }
  if (v is Map) {
    // Map<String, Object?> or similar
    final Map<String, Object?> m =
        v.map((Object? k, Object? val) => MapEntry<String, Object?>(k.toString(), val));
    return _sort(m);
  }
  if (v is List) return v.map(_sort).toList();
  return v;
}

void main() {
  final File golden = File('../shared/proto/testdata/messages.json');
  if (!golden.existsSync()) {
    throw StateError('golden fixture missing at ${golden.absolute.path}');
  }
  final List<dynamic> cases =
      jsonDecode(golden.readAsStringSync()) as List<dynamic>;
  if (cases.isEmpty) {
    throw StateError('golden fixture is empty');
  }

  group('wire drift against shared/proto/testdata', () {
    for (final dynamic c in cases) {
      final Map<String, dynamic> entry = c as Map<String, dynamic>;
      final String name = entry['name'] as String;
      final Map<String, dynamic> msg = entry['message'] as Map<String, dynamic>;

      test(name, () {
        final Message parsed = Message.fromJson(msg);
        final Map<String, dynamic> produced = parsed.toJson();
        expect(_sort(produced), equals(_sort(msg)));
      });
    }
  });
}
