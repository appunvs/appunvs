// Hand-written protojson wire types mirroring shared/proto/appunvs.proto.
//
// On the wire we use canonical protojson: snake_case field names and enum
// values encoded as short lowercase strings (e.g. "provider", "upsert",
// "mobile"). We do not use reflection; all mappings live in the tables below.

/// Platform mirrors appunvs.v1.Platform.
enum Platform { unspecified, browser, desktop, mobile }

/// Role mirrors appunvs.v1.Role.
enum Role { unspecified, provider, connector, both }

/// Op mirrors appunvs.v1.Op.
enum Op {
  unspecified,
  upsert,
  delete,
  tableCreate,
  tableDelete,
  columnAdd,
  columnDelete,
  quotaExceeded,
}

const Map<Platform, String> _platformToWire = <Platform, String>{
  Platform.unspecified: 'PLATFORM_UNSPECIFIED',
  Platform.browser: 'browser',
  Platform.desktop: 'desktop',
  Platform.mobile: 'mobile',
};

const Map<String, Platform> _wireToPlatform = <String, Platform>{
  'PLATFORM_UNSPECIFIED': Platform.unspecified,
  'browser': Platform.browser,
  'desktop': Platform.desktop,
  'mobile': Platform.mobile,
};

const Map<Role, String> _roleToWire = <Role, String>{
  Role.unspecified: 'ROLE_UNSPECIFIED',
  Role.provider: 'provider',
  Role.connector: 'connector',
  Role.both: 'both',
};

const Map<String, Role> _wireToRole = <String, Role>{
  'ROLE_UNSPECIFIED': Role.unspecified,
  'provider': Role.provider,
  'connector': Role.connector,
  'both': Role.both,
};

const Map<Op, String> _opToWire = <Op, String>{
  Op.unspecified: 'OP_UNSPECIFIED',
  Op.upsert: 'upsert',
  Op.delete: 'delete',
  Op.tableCreate: 'table_create',
  Op.tableDelete: 'table_delete',
  Op.columnAdd: 'column_add',
  Op.columnDelete: 'column_delete',
  Op.quotaExceeded: 'quota_exceeded',
};

const Map<String, Op> _wireToOp = <String, Op>{
  'OP_UNSPECIFIED': Op.unspecified,
  'upsert': Op.upsert,
  'delete': Op.delete,
  'table_create': Op.tableCreate,
  'table_delete': Op.tableDelete,
  'column_add': Op.columnAdd,
  'column_delete': Op.columnDelete,
  'quota_exceeded': Op.quotaExceeded,
};

String platformToWire(Platform p) => _platformToWire[p] ?? 'PLATFORM_UNSPECIFIED';
Platform platformFromWire(Object? v) =>
    _wireToPlatform[v?.toString() ?? ''] ?? Platform.unspecified;

String roleToWire(Role r) => _roleToWire[r] ?? 'ROLE_UNSPECIFIED';
Role roleFromWire(Object? v) => _wireToRole[v?.toString() ?? ''] ?? Role.unspecified;

String opToWire(Op o) => _opToWire[o] ?? 'OP_UNSPECIFIED';
Op opFromWire(Object? v) => _wireToOp[v?.toString() ?? ''] ?? Op.unspecified;

/// Message mirrors appunvs.v1.Message.
class Message {
  Message({
    this.seq = 0,
    this.deviceId = '',
    this.userId = '',
    this.namespace = '',
    this.role = Role.unspecified,
    this.op = Op.unspecified,
    this.table = '',
    this.payload,
    this.ts = 0,
  });

  final int seq;
  final String deviceId;
  final String userId;
  final String namespace;
  final Role role;
  final Op op;
  final String table;
  final Map<String, dynamic>? payload;
  final int ts;

  Message copyWith({
    int? seq,
    String? deviceId,
    String? userId,
    String? namespace,
    Role? role,
    Op? op,
    String? table,
    Map<String, dynamic>? payload,
    int? ts,
  }) {
    return Message(
      seq: seq ?? this.seq,
      deviceId: deviceId ?? this.deviceId,
      userId: userId ?? this.userId,
      namespace: namespace ?? this.namespace,
      role: role ?? this.role,
      op: op ?? this.op,
      table: table ?? this.table,
      payload: payload ?? this.payload,
      ts: ts ?? this.ts,
    );
  }

  /// Serialize to canonical protojson.
  /// `seq` is omitted when 0 (the relay assigns it).
  Map<String, dynamic> toJson() {
    final Map<String, dynamic> j = <String, dynamic>{};
    if (seq != 0) j['seq'] = seq;
    if (deviceId.isNotEmpty) j['device_id'] = deviceId;
    if (userId.isNotEmpty) j['user_id'] = userId;
    if (namespace.isNotEmpty) j['namespace'] = namespace;
    if (role != Role.unspecified) j['role'] = roleToWire(role);
    if (op != Op.unspecified) j['op'] = opToWire(op);
    if (table.isNotEmpty) j['table'] = table;
    if (payload != null) j['payload'] = payload;
    if (ts != 0) j['ts'] = ts;
    return j;
  }

  /// Parse a protojson message. Unknown keys are ignored.
  static Message fromJson(Map<String, dynamic> j) {
    return Message(
      seq: _asInt(j['seq']),
      deviceId: _asString(j['device_id']),
      userId: _asString(j['user_id']),
      namespace: _asString(j['namespace']),
      role: roleFromWire(j['role']),
      op: opFromWire(j['op']),
      table: _asString(j['table']),
      payload: _asPayload(j['payload']),
      ts: _asInt(j['ts']),
    );
  }
}

/// RegisterRequest mirrors appunvs.v1.RegisterRequest.
class RegisterRequest {
  RegisterRequest({required this.deviceId, required this.platform});
  final String deviceId;
  final Platform platform;

  Map<String, dynamic> toJson() => <String, dynamic>{
        'device_id': deviceId,
        'platform': platformToWire(platform),
      };
}

/// RegisterResponse mirrors appunvs.v1.RegisterResponse.
class RegisterResponse {
  RegisterResponse({required this.token, required this.userId});
  final String token;
  final String userId;

  static RegisterResponse fromJson(Map<String, dynamic> j) =>
      RegisterResponse(token: _asString(j['token']), userId: _asString(j['user_id']));
}

/// SignupRequest mirrors appunvs.v1.SignupRequest.
class SignupRequest {
  SignupRequest({required this.email, required this.password});
  final String email;
  final String password;

  Map<String, dynamic> toJson() => <String, dynamic>{
        'email': email,
        'password': password,
      };
}

/// LoginRequest mirrors appunvs.v1.LoginRequest.
class LoginRequest {
  LoginRequest({required this.email, required this.password});
  final String email;
  final String password;

  Map<String, dynamic> toJson() => <String, dynamic>{
        'email': email,
        'password': password,
      };
}

/// SessionResponse mirrors appunvs.v1.SessionResponse.
class SessionResponse {
  SessionResponse({required this.userId, required this.sessionToken});
  final String userId;
  final String sessionToken;

  static SessionResponse fromJson(Map<String, dynamic> j) => SessionResponse(
        userId: _asString(j['user_id']),
        sessionToken: _asString(j['session_token']),
      );
}

/// Device mirrors appunvs.v1.Device.
class Device {
  Device({
    this.id = '',
    this.userId = '',
    this.platform = Platform.unspecified,
    this.createdAt = 0,
    this.lastSeen = 0,
  });

  final String id;
  final String userId;
  final Platform platform;
  final int createdAt;
  final int lastSeen;

  static Device fromJson(Map<String, dynamic> j) => Device(
        id: _asString(j['id']),
        userId: _asString(j['user_id']),
        platform: platformFromWire(j['platform']),
        createdAt: _asInt(j['created_at']),
        lastSeen: _asInt(j['last_seen']),
      );
}

/// MeResponse mirrors appunvs.v1.MeResponse.
class MeResponse {
  MeResponse({
    this.userId = '',
    this.email = '',
    this.createdAt = 0,
    this.devices = const <Device>[],
  });

  final String userId;
  final String email;
  final int createdAt;
  final List<Device> devices;

  static MeResponse fromJson(Map<String, dynamic> j) {
    final Object? rawDevices = j['devices'];
    final List<Device> devs = <Device>[];
    if (rawDevices is List) {
      for (final Object? d in rawDevices) {
        if (d is Map<String, dynamic>) {
          devs.add(Device.fromJson(d));
        } else if (d is Map) {
          devs.add(Device.fromJson(d.map<String, dynamic>(
            (Object? k, Object? v) => MapEntry<String, dynamic>(k.toString(), v),
          )));
        }
      }
    }
    return MeResponse(
      userId: _asString(j['user_id']),
      email: _asString(j['email']),
      createdAt: _asInt(j['created_at']),
      devices: devs,
    );
  }
}

int _asInt(Object? v) {
  if (v == null) return 0;
  if (v is int) return v;
  if (v is num) return v.toInt();
  if (v is String) return int.tryParse(v) ?? 0;
  return 0;
}

String _asString(Object? v) {
  if (v == null) return '';
  if (v is String) return v;
  return v.toString();
}

Map<String, dynamic>? _asPayload(Object? v) {
  if (v == null) return null;
  if (v is Map<String, dynamic>) return v;
  if (v is Map) {
    return v.map<String, dynamic>(
      (Object? k, Object? val) => MapEntry<String, dynamic>(k.toString(), val),
    );
  }
  return null;
}
