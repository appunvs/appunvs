/// RelayState describes the WebSocket connection's lifecycle.
enum RelayState { disconnected, connecting, connected, reconnecting }

extension RelayStateLabel on RelayState {
  String get label {
    switch (this) {
      case RelayState.disconnected:
        return 'disconnected';
      case RelayState.connecting:
        return 'connecting';
      case RelayState.connected:
        return 'connected';
      case RelayState.reconnecting:
        return 'reconnecting';
    }
  }
}
