import 'dart:async';

import '../../pb/wire.dart';
import '../config.dart';
import '../database/records_dao.dart';
import '../relay/relay_client.dart';

/// Glue between the local DAO and the relay client.
///
/// - Local writes are pushed via [pushLocalUpsert] / [pushLocalDelete].
/// - Incoming [Message]s from providers are applied to the DAO.
/// - Incoming [Message]s from connectors are applied AND re-broadcast as
///   a provider message using this device's identity (authoritative copy).
class ProviderSync {
  ProviderSync({
    required this.dao,
    required this.relay,
    required this.deviceId,
    required this.userId,
    this.table = AppConfig.defaultTable,
  });

  final RecordsDao dao;
  final RelayClient relay;
  final String deviceId;
  final String userId;
  final String table;

  StreamSubscription<Message>? _sub;

  /// Start listening to incoming relay messages.
  void start() {
    _sub?.cancel();
    _sub = relay.messages.listen(_onIncoming);
  }

  Future<void> stop() async {
    await _sub?.cancel();
    _sub = null;
  }

  /// UI called `dao.upsert(id, data)`; now publish it.
  Future<void> pushLocalUpsert(String id, String data) async {
    final Message m = Message(
      deviceId: deviceId,
      userId: userId,
      namespace: userId,
      role: Role.provider,
      op: Op.upsert,
      table: table,
      payload: <String, dynamic>{'id': id, 'data': data},
      ts: DateTime.now().millisecondsSinceEpoch,
    );
    relay.send(m);
  }

  /// UI requested a delete; persist then publish.
  Future<void> pushLocalDelete(String id) async {
    await dao.deleteById(id);
    final Message m = Message(
      deviceId: deviceId,
      userId: userId,
      namespace: userId,
      role: Role.provider,
      op: Op.delete,
      table: table,
      payload: <String, dynamic>{'id': id},
      ts: DateTime.now().millisecondsSinceEpoch,
    );
    relay.send(m);
  }

  Future<void> _onIncoming(Message m) async {
    // Ignore our own device echoes: DAO state is already correct.
    // But we still want to capture the relay-assigned seq for our own
    // upserts so maxSeq() reflects reality on reconnect.
    switch (m.role) {
      case Role.provider:
      case Role.both:
        await dao.applyBroadcast(m);
        break;
      case Role.connector:
        await dao.applyBroadcast(m);
        // Re-broadcast with our identity so other providers pick it up.
        if (m.deviceId != deviceId) {
          final Message forward = m.copyWith(
            seq: 0, // let the relay assign a new seq
            deviceId: deviceId,
            userId: userId,
            namespace: userId,
            role: Role.provider,
            ts: DateTime.now().millisecondsSinceEpoch,
          );
          relay.send(forward);
        }
        break;
      case Role.unspecified:
        break;
    }
  }
}
