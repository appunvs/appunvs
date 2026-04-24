import 'package:drift/drift.dart';
import 'package:drift_flutter/drift_flutter.dart';

import 'records_dao.dart';

part 'app_db.g.dart';

/// Records table: the authoritative local copy of synced records.
@DataClassName('Record')
class Records extends Table {
  TextColumn get id => text()();
  TextColumn get data => text()();
  IntColumn get seq => integer().withDefault(const Constant(0))();
  DateTimeColumn get updatedAt => dateTime()();

  @override
  Set<Column> get primaryKey => <Column>{id};
}

@DriftDatabase(tables: <Type>[Records], daos: <Type>[RecordsDao])
class AppDb extends _$AppDb {
  AppDb() : super(_openConnection());

  /// Secondary ctor for tests that want to inject an in-memory connection.
  AppDb.forExecutor(super.e);

  @override
  int get schemaVersion => 1;
}

QueryExecutor _openConnection() => driftDatabase(name: 'appunvs');
