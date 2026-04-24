import 'dart:async';

import 'package:flutter/material.dart';

import '../../core/auth/session_store.dart';
import '../../core/config.dart';
import '../../core/relay/auth.dart';
import '../../pb/wire.dart' as pb;

/// Email + password screen with a toggle between sign-in and sign-up.
///
/// On success:
///   1. Persist session via [SessionStore.saveSession]
///   2. POST /auth/register to obtain a device token, persist it
///   3. Navigate to /home
///
/// Errors are shown inline below the form.
class LoginPage extends StatefulWidget {
  const LoginPage({super.key, this.sessionStore, this.accountClient});

  /// Overrides for tests. When null, new instances are used.
  final SessionStore? sessionStore;
  final AccountClient? accountClient;

  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  late final SessionStore _store;
  late final AccountClient _auth;

  final TextEditingController _emailCtl = TextEditingController();
  final TextEditingController _passwordCtl = TextEditingController();
  final GlobalKey<FormState> _formKey = GlobalKey<FormState>();

  bool _isSignup = false;
  bool _busy = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _store = widget.sessionStore ?? SessionStore();
    _auth = widget.accountClient ?? AccountClient(AppConfig.relayBase);
  }

  @override
  void dispose() {
    _emailCtl.dispose();
    _passwordCtl.dispose();
    if (widget.accountClient == null) _auth.close();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(_isSignup ? 'Create account' : 'Sign in'),
      ),
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: Form(
                key: _formKey,
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  mainAxisSize: MainAxisSize.min,
                  children: <Widget>[
                    TextFormField(
                      controller: _emailCtl,
                      keyboardType: TextInputType.emailAddress,
                      autofillHints: const <String>[AutofillHints.email],
                      decoration: const InputDecoration(
                        labelText: 'Email',
                        border: OutlineInputBorder(),
                      ),
                      validator: (String? v) {
                        final String s = (v ?? '').trim();
                        if (s.isEmpty) return 'Email is required';
                        if (!s.contains('@')) return 'Enter a valid email';
                        return null;
                      },
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _passwordCtl,
                      obscureText: true,
                      autofillHints: <String>[
                        _isSignup
                            ? AutofillHints.newPassword
                            : AutofillHints.password,
                      ],
                      decoration: const InputDecoration(
                        labelText: 'Password',
                        border: OutlineInputBorder(),
                      ),
                      validator: (String? v) {
                        final String s = v ?? '';
                        if (s.isEmpty) return 'Password is required';
                        if (_isSignup && s.length < 8) {
                          return 'Use at least 8 characters';
                        }
                        return null;
                      },
                    ),
                    const SizedBox(height: 16),
                    FilledButton(
                      onPressed: _busy ? null : _submit,
                      child: _busy
                          ? const SizedBox(
                              height: 18,
                              width: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : Text(_isSignup ? 'Create account' : 'Sign in'),
                    ),
                    const SizedBox(height: 8),
                    TextButton(
                      onPressed: _busy
                          ? null
                          : () => setState(() {
                                _isSignup = !_isSignup;
                                _error = null;
                              }),
                      child: Text(
                        _isSignup
                            ? 'Have an account? Sign in'
                            : "New here? Create an account",
                      ),
                    ),
                    if (_error != null) ...<Widget>[
                      const SizedBox(height: 12),
                      Text(
                        _error!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    ],
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<void> _submit() async {
    final FormState? form = _formKey.currentState;
    if (form == null || !form.validate()) return;

    setState(() {
      _busy = true;
      _error = null;
    });

    final String email = _emailCtl.text.trim();
    final String password = _passwordCtl.text;

    try {
      final pb.SessionResponse session = _isSignup
          ? await _auth.signup(email, password)
          : await _auth.login(email, password);
      await _store.saveSession(session.userId, session.sessionToken, email);

      final String deviceId = await _store.ensureDeviceId();
      final DeviceRegistration reg = await _auth.registerDevice(
        sessionToken: session.sessionToken,
        deviceId: deviceId,
        platform: pb.Platform.mobile,
      );
      await _store.saveDeviceToken(reg.token);

      if (!mounted) return;
      unawaited(Navigator.of(context).pushReplacementNamed('/home'));
    } on AuthException catch (e) {
      if (!mounted) return;
      setState(() => _error = e.body.isEmpty ? e.toString() : e.body);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }
}
