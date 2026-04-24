import 'dart:async';

import 'package:flutter/material.dart';

import 'core/auth/session_store.dart';
import 'core/config.dart';
import 'core/database/app_db.dart';
import 'core/database/records_dao.dart';
import 'core/relay/auth.dart';
import 'core/relay/relay_client.dart';
import 'core/sync/provider_sync.dart';
import 'features/auth/login_page.dart';
import 'features/home/home_page.dart';
import 'pb/wire.dart' as pb;

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const AppunvsApp());
}

class AppunvsApp extends StatelessWidget {
  const AppunvsApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'appunvs mobile',
      theme: ThemeData(useMaterial3: true, colorSchemeSeed: Colors.indigo),
      initialRoute: '/',
      routes: <String, WidgetBuilder>{
        '/': (_) => const _Bootstrap(),
        '/login': (_) => const LoginPage(),
        '/home': (_) => const _HomeBootstrap(),
      },
    );
  }
}

/// Startup screen: consults [SessionStore] and routes to /login or /home.
class _Bootstrap extends StatefulWidget {
  const _Bootstrap();

  @override
  State<_Bootstrap> createState() => _BootstrapState();
}

class _BootstrapState extends State<_Bootstrap> {
  final SessionStore _store = SessionStore();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) => _route());
  }

  Future<void> _route() async {
    final String token = await _store.sessionToken();
    if (!mounted) return;
    unawaited(
      Navigator.of(context)
          .pushReplacementNamed(token.isEmpty ? '/login' : '/home'),
    );
  }

  @override
  Widget build(BuildContext context) {
    return const Scaffold(body: Center(child: CircularProgressIndicator()));
  }
}

/// Home screen wrapper that owns the db + relay lifecycle.
///
/// Constructs lazily on entry: opens AppDb, ensures a device token (using the
/// session token to re-register if needed), connects the WS, and only then
/// renders [HomePage].
class _HomeBootstrap extends StatefulWidget {
  const _HomeBootstrap();

  @override
  State<_HomeBootstrap> createState() => _HomeBootstrapState();
}

class _HomeBootstrapState extends State<_HomeBootstrap>
    with WidgetsBindingObserver {
  final SessionStore _store = SessionStore();
  final AccountClient _auth = AccountClient(AppConfig.relayBase);

  AppDb? _db;
  RecordsDao? _dao;
  RelayClient? _relay;
  ProviderSync? _sync;
  String? _error;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _init();
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    unawaited(_tearDown());
    _auth.close();
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    switch (state) {
      case AppLifecycleState.resumed:
        unawaited(_reconnectIfNeeded());
        break;
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
      case AppLifecycleState.hidden:
        unawaited(_relay?.close() ?? Future<void>.value());
        break;
      case AppLifecycleState.inactive:
        break;
    }
  }

  Future<void> _init() async {
    try {
      final String sessionToken = await _store.sessionToken();
      if (sessionToken.isEmpty) {
        // Someone landed at /home without a session — send them to /login.
        if (!mounted) return;
        unawaited(Navigator.of(context).pushReplacementNamed('/login'));
        return;
      }

      final String deviceId = await _store.ensureDeviceId();
      String deviceToken = await _store.deviceToken();
      String userId = await _store.userId();

      if (deviceToken.isEmpty) {
        final DeviceRegistration reg = await _auth.registerDevice(
          sessionToken: sessionToken,
          deviceId: deviceId,
          platform: pb.Platform.mobile,
        );
        deviceToken = reg.token;
        userId = reg.userId;
        await _store.saveDeviceToken(deviceToken);
      }

      final AppDb db = AppDb();
      final RecordsDao dao = db.recordsDao;

      final RelayClient relay = RelayClient(maxSeqProvider: dao.maxSeq);
      final ProviderSync sync = ProviderSync(
        dao: dao,
        relay: relay,
        deviceId: deviceId,
        userId: userId,
      );
      sync.start();

      final int lastSeq = await dao.maxSeq();
      await relay.connect(AppConfig.relayBase, deviceToken, lastSeq);

      if (!mounted) return;
      setState(() {
        _db = db;
        _dao = dao;
        _relay = relay;
        _sync = sync;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    }
  }

  Future<void> _reconnectIfNeeded() async {
    final RelayClient? r = _relay;
    final RecordsDao? dao = _dao;
    if (r == null || dao == null) return;
    final String token = await _store.deviceToken();
    if (token.isEmpty) return;
    final int lastSeq = await dao.maxSeq();
    await r.connect(AppConfig.relayBase, token, lastSeq);
  }

  Future<void> _tearDown() async {
    await _sync?.stop();
    await _relay?.close();
    await _db?.close();
    _sync = null;
    _relay = null;
    _dao = null;
    _db = null;
  }

  @override
  Widget build(BuildContext context) {
    if (_error != null) {
      return Scaffold(
        appBar: AppBar(title: const Text('appunvs mobile')),
        body: Padding(
          padding: const EdgeInsets.all(24),
          child: Center(child: Text('Failed to start: $_error')),
        ),
      );
    }
    final RecordsDao? dao = _dao;
    final RelayClient? relay = _relay;
    final ProviderSync? sync = _sync;
    if (dao == null || relay == null || sync == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    return HomePage(
      dao: dao,
      relay: relay,
      sync: sync,
      sessionStore: _store,
      onSignOut: _tearDown,
    );
  }
}
