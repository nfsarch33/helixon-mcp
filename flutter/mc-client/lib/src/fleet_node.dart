/// FleetNode is the on-the-wire model for a Mission Control fleet
/// member. The MC bridges expose an aggregated /status endpoint that
/// returns one of these per known member; we keep the model small and
/// JSON-friendly so both the Flutter UI and the dart-side fleet ops
/// CLI can decode the same payload.
class FleetNode {
  const FleetNode({
    required this.name,
    required this.status,
    this.tailscaleIp,
    this.role,
    this.lastSeen,
  });

  final String name;
  final String status;
  final String? tailscaleIp;
  final String? role;
  final DateTime? lastSeen;

  factory FleetNode.fromJson(Map<String, dynamic> json) {
    return FleetNode(
      name: json['name'] as String,
      status: json['status'] as String,
      tailscaleIp: json['tailscale_ip'] as String?,
      role: json['role'] as String?,
      lastSeen: json['last_seen'] != null
          ? DateTime.parse(json['last_seen'] as String)
          : null,
    );
  }

  Map<String, dynamic> toJson() => {
    'name': name,
    'status': status,
    if (tailscaleIp != null) 'tailscale_ip': tailscaleIp,
    if (role != null) 'role': role,
    if (lastSeen != null) 'last_seen': lastSeen!.toIso8601String(),
  };

  bool get isHealthy => status == 'healthy' || status == 'ok';
}

/// renderFleetNode is a tiny formatter the Flutter UI uses for the
/// per-row label. Kept top-level to mirror the v258 scaffold contract
/// the existing test file asserts against.
String renderFleetNode(FleetNode node) => '${node.name}:${node.status}';
