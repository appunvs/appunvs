// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'records_dao.dart';

// ignore_for_file: type=lint
mixin _$RecordsDaoMixin on DatabaseAccessor<AppDb> {
  $RecordsTable get records => attachedDatabase.records;
  RecordsDaoManager get managers => RecordsDaoManager(this);
}

class RecordsDaoManager {
  final _$RecordsDaoMixin _db;
  RecordsDaoManager(this._db);
  $$RecordsTableTableManager get records =>
      $$RecordsTableTableManager(_db.attachedDatabase, _db.records);
}
