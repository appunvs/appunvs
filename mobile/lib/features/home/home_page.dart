import 'dart:async';

import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';

import '../../core/auth/session_store.dart';
import '../../core/database/app_db.dart';
import '../../core/database/records_dao.dart';
import '../../core/relay/relay_client.dart';
import '../../core/relay/relay_state.dart';
import '../../core/sync/provider_sync.dart';

class HomePage extends StatefulWidget {
  const HomePage({
    super.key,
    required this.dao,
    required this.relay,
    required this.sync,
    this.sessionStore,
    this.onSignOut,
  });

  final RecordsDao dao;
  final RelayClient relay;
  final ProviderSync sync;
  final SessionStore? sessionStore;

  /// Optional hook called after [SessionStore.logout]; used by the app shell
  /// to tear down the relay + db before routing to /login.
  final Future<void> Function()? onSignOut;

  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final Uuid _uuid = const Uuid();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('appunvs mobile'),
        actions: <Widget>[
          IconButton(
            tooltip: 'Sign out',
            icon: const Icon(Icons.logout),
            onPressed: _signOut,
          ),
        ],
      ),
      body: Column(
        children: <Widget>[
          _StatusBar(relay: widget.relay),
          const Divider(height: 1),
          Expanded(
            child: StreamBuilder<List<Record>>(
              stream: widget.dao.watchAll(),
              builder: (BuildContext ctx, AsyncSnapshot<List<Record>> snap) {
                final List<Record> items = snap.data ?? const <Record>[];
                if (items.isEmpty) {
                  return const Center(child: Text('No records yet. Tap + to add one.'));
                }
                return ListView.separated(
                  itemCount: items.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (BuildContext c, int i) {
                    final Record r = items[i];
                    return ListTile(
                      title: Text(r.data),
                      subtitle: Text('id=${r.id}  seq=${r.seq}'),
                      trailing: IconButton(
                        icon: const Icon(Icons.delete_outline),
                        onPressed: () => widget.sync.pushLocalDelete(r.id),
                      ),
                    );
                  },
                );
              },
            ),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        onPressed: _addRecord,
        child: const Icon(Icons.add),
      ),
    );
  }

  Future<void> _addRecord() async {
    final TextEditingController idCtl = TextEditingController(text: _uuid.v4());
    final TextEditingController dataCtl = TextEditingController();
    final bool? ok = await showDialog<bool>(
      context: context,
      builder: (BuildContext ctx) {
        return AlertDialog(
          title: const Text('Add record'),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            children: <Widget>[
              TextField(
                controller: idCtl,
                decoration: const InputDecoration(labelText: 'id'),
              ),
              TextField(
                controller: dataCtl,
                decoration: const InputDecoration(labelText: 'data'),
              ),
            ],
          ),
          actions: <Widget>[
            TextButton(
              onPressed: () => Navigator.of(ctx).pop(false),
              child: const Text('Cancel'),
            ),
            FilledButton(
              onPressed: () => Navigator.of(ctx).pop(true),
              child: const Text('Save'),
            ),
          ],
        );
      },
    );

    if (ok != true) return;
    final String id = idCtl.text.trim();
    final String data = dataCtl.text;
    if (id.isEmpty) return;
    await widget.dao.upsert(id, data);
    await widget.sync.pushLocalUpsert(id, data);
  }

  Future<void> _signOut() async {
    final SessionStore store = widget.sessionStore ?? SessionStore();
    await store.logout();
    if (widget.onSignOut != null) {
      await widget.onSignOut!.call();
    }
    if (!mounted) return;
    unawaited(
      Navigator.of(context).pushNamedAndRemoveUntil('/login', (_) => false),
    );
  }
}

class _StatusBar extends StatelessWidget {
  const _StatusBar({required this.relay});
  final RelayClient relay;

  @override
  Widget build(BuildContext context) {
    return StreamBuilder<RelayState>(
      stream: relay.state,
      initialData: relay.currentState,
      builder: (BuildContext c, AsyncSnapshot<RelayState> snap) {
        final RelayState s = snap.data ?? RelayState.disconnected;
        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          alignment: Alignment.centerLeft,
          child: Row(
            children: <Widget>[
              Container(
                width: 10,
                height: 10,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: _colorFor(s),
                ),
              ),
              const SizedBox(width: 8),
              Text(s.label),
            ],
          ),
        );
      },
    );
  }

  Color _colorFor(RelayState s) {
    switch (s) {
      case RelayState.connected:
        return Colors.green;
      case RelayState.connecting:
      case RelayState.reconnecting:
        return Colors.amber;
      case RelayState.disconnected:
        return Colors.red;
    }
  }
}
