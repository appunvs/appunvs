import 'package:drift/drift.dart';

import '../../pb/wire.dart' as wire;
import 'app_db.dart';

part 'records_dao.g.dart';

/// DAO for the Records table.
@DriftAccessor(tables: <Type>[Records])
class RecordsDao extends DatabaseAccessor<AppDb> with _$RecordsDaoMixin {
  RecordsDao(super.db);

  /// Watch all records, newest first.
  Stream<List<Record>> watchAll() {
    final SimpleSelectStatement<$RecordsTable, Record> q = select(records)
      ..orderBy(<OrderClauseGenerator<$RecordsTable>>[
        ($RecordsTable t) => OrderingTerm(expression: t.updatedAt, mode: OrderingMode.desc),
      ]);
    return q.watch();
  }

  /// Upsert a record from a local user action. Does not assign a seq
  /// (that happens server-side; applyBroadcast fills it in once the relay
  /// echoes our own message back).
  Future<void> upsert(String id, String data) {
    return into(records).insertOnConflictUpdate(
      RecordsCompanion(
        id: Value<String>(id),
        data: Value<String>(data),
        updatedAt: Value<DateTime>(DateTime.now()),
      ),
    );
  }

  /// Delete a record by id. Used by both local UI and applyBroadcast.
  Future<int> deleteById(String id) {
    return (delete(records)..where(($RecordsTable t) => t.id.equals(id))).go();
  }

  /// Maximum seq seen. Used as `last_seq` on (re)connect for replay.
  Future<int> maxSeq() async {
    final Expression<int> expr = records.seq.max();
    final TypedResult row =
        await (selectOnly(records)..addColumns(<Expression<Object>>[expr])).getSingle();
    return row.read(expr) ?? 0;
  }

  /// Apply a Message received over the relay to the local store.
  ///
  /// Upsert: writes id/data and records the relay-assigned seq.
  /// Delete: removes the row if present.
  Future<void> applyBroadcast(wire.Message m) async {
    final Map<String, dynamic>? p = m.payload;
    if (p == null) return;
    final Object? rawId = p['id'];
    if (rawId is! String || rawId.isEmpty) return;

    switch (m.op) {
      case wire.Op.upsert:
        final Object? rawData = p['data'];
        final String data = rawData is String ? rawData : (rawData?.toString() ?? '');
        await into(records).insertOnConflictUpdate(
          RecordsCompanion(
            id: Value<String>(rawId),
            data: Value<String>(data),
            seq: Value<int>(m.seq),
            updatedAt: Value<DateTime>(DateTime.fromMillisecondsSinceEpoch(
              m.ts == 0 ? DateTime.now().millisecondsSinceEpoch : m.ts,
            )),
          ),
        );
        break;
      case wire.Op.delete:
        await (delete(records)..where(($RecordsTable t) => t.id.equals(rawId))).go();
        break;
      case wire.Op.unspecified:
      // Schema mutations and billing guardrails don't touch the records
      // store — they target `_schema` or echo back to the sender only.
      // The UI layer can subscribe to the raw message stream if it cares.
      case wire.Op.tableCreate:
      case wire.Op.tableDelete:
      case wire.Op.columnAdd:
      case wire.Op.columnDelete:
      case wire.Op.quotaExceeded:
        break;
    }
  }
}
