/// FleetEndpoint resolves a logical fleet-member name (e.g. "node-a")
/// into a concrete URL the mc-client should hit. v259 supports two
/// transports:
///
///   1. Direct Tailscale: https://<host>.<tailnet>.ts.net:<port>
///   2. OCI SSH-jump:    https://localhost:<local-forwarded-port>
///
/// The transport is chosen at runtime so the same client binary can
/// run on the MacBook (which always has Tailscale) and inside a
/// session on a roaming device that only has the OCI jump available.
enum FleetTransport { tailscale, ociJump }

class FleetEndpoint {
  const FleetEndpoint({
    required this.transport,
    required this.uri,
    required this.node,
  });

  final FleetTransport transport;
  final Uri uri;
  final String node;

  /// healthCheck is the path the bridges expose for a 200 OK liveness
  /// probe. Each MC bridge listens on its own port so we don't hit
  /// /healthz on the wrong service.
  Uri get healthCheck => uri.replace(path: '/healthz');

  /// command is the slash-command POST endpoint the dart client uses
  /// for the IronClaw-bridge transport.
  Uri get command => uri.replace(path: '/command');
}

/// FleetEndpointResolver turns a logical fleet-member name into a
/// concrete FleetEndpoint based on a static config map. The config
/// is loaded from $HOME/Code/global-kb/fleet/nodes.yaml on the host
/// and shipped into the client as JSON at build time -- no runtime
/// network discovery, deterministic for tests.
class FleetEndpointResolver {
  FleetEndpointResolver({
    required Map<String, FleetEndpoint> tailscale,
    required Map<String, FleetEndpoint> ociJump,
    this.preferred = FleetTransport.tailscale,
  }) : _tailscale = Map.unmodifiable(tailscale),
       _ociJump = Map.unmodifiable(ociJump);

  final Map<String, FleetEndpoint> _tailscale;
  final Map<String, FleetEndpoint> _ociJump;
  final FleetTransport preferred;

  /// resolve returns the endpoint for [node] using the preferred
  /// transport. If the preferred entry is missing we fall back to
  /// the alternate transport so the client still works on a partial
  /// network.
  FleetEndpoint? resolve(String node) {
    final FleetTransport first = preferred;
    final FleetTransport second = preferred == FleetTransport.tailscale
        ? FleetTransport.ociJump
        : FleetTransport.tailscale;
    return _byTransport(first)[node] ?? _byTransport(second)[node];
  }

  /// resolveAll yields every known endpoint under [transport].
  Iterable<FleetEndpoint> resolveAll(FleetTransport transport) =>
      _byTransport(transport).values;

  Map<String, FleetEndpoint> _byTransport(FleetTransport t) =>
      t == FleetTransport.tailscale ? _tailscale : _ociJump;

  /// fromJson builds a resolver from the JSON shape the build pipeline
  /// emits:
  ///   {
  ///     "preferred": "tailscale",
  ///     "tailscale": {"node-a": "https://node-a.foo.ts.net:9302", ...},
  ///     "oci_jump":  {"node-a": "https://localhost:19302", ...}
  ///   }
  factory FleetEndpointResolver.fromJson(Map<String, dynamic> json) {
    final pref = (json['preferred'] as String?) ?? 'tailscale';
    final tailscale = _parseMap(
      json['tailscale'] as Map<String, dynamic>? ?? const {},
      FleetTransport.tailscale,
    );
    final ociJump = _parseMap(
      json['oci_jump'] as Map<String, dynamic>? ?? const {},
      FleetTransport.ociJump,
    );
    return FleetEndpointResolver(
      tailscale: tailscale,
      ociJump: ociJump,
      preferred: pref == 'oci_jump'
          ? FleetTransport.ociJump
          : FleetTransport.tailscale,
    );
  }

  static Map<String, FleetEndpoint> _parseMap(
    Map<String, dynamic> raw,
    FleetTransport transport,
  ) {
    final out = <String, FleetEndpoint>{};
    raw.forEach((node, value) {
      out[node] = FleetEndpoint(
        transport: transport,
        uri: Uri.parse(value as String),
        node: node,
      );
    });
    return out;
  }
}
