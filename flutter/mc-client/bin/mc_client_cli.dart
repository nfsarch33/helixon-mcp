// mc_client_cli is a v259 dart entrypoint that exercises the same
// data-plane core the Flutter UI consumes. It exists so we can smoke
// the compile path on CI (no Flutter SDK required) and so operators
// can run a quick `mc-client status node-a` against a running fleet
// member without launching a GUI.
//
// Usage:
//   dart run bin/mc_client_cli.dart status <node>
//   dart run bin/mc_client_cli.dart status-all
//
// Endpoints are read from MC_CLIENT_FLEET_JSON (an inline JSON blob)
// or default to a localhost-only OCI-jump preset useful for offline
// smoke. We never read 1Password or shared secrets here -- the CLI
// only does health-checks, not signed slash-commands.

import 'dart:convert';
import 'dart:io';

import 'package:mc_client/main.dart';

const String _defaultFleetJson = '''
{
  "preferred": "oci_jump",
  "tailscale": {},
  "oci_jump": {
    "node-a": "http://localhost:19302",
    "node-b": "http://localhost:19303",
    "host-a": "http://localhost:19401",
    "host-b": "http://localhost:19402",
    "mc":   "http://localhost:19308"
  }
}
''';

Future<void> main(List<String> args) async {
  if (args.isEmpty || args[0] == '--help' || args[0] == '-h') {
    stdout.writeln('mc-client-cli - mission-control fleet pinger');
    stdout.writeln('usage:');
    stdout.writeln('  mc-client-cli status <node>');
    stdout.writeln('  mc-client-cli status-all');
    exitCode = 0;
    return;
  }
  final raw = Platform.environment['MC_CLIENT_FLEET_JSON'] ?? _defaultFleetJson;
  Map<String, dynamic> parsed;
  try {
    parsed = jsonDecode(raw) as Map<String, dynamic>;
  } catch (e) {
    stderr.writeln('invalid MC_CLIENT_FLEET_JSON: $e');
    exitCode = 2;
    return;
  }
  final resolver = FleetEndpointResolver.fromJson(parsed);
  final client = FleetClient(resolver: resolver);
  try {
    switch (args[0]) {
      case 'status':
        if (args.length < 2) {
          stderr.writeln('status: node argument required');
          exitCode = 2;
          return;
        }
        final r = await client.healthCheck(args[1]);
        stdout.writeln(
          '${r.node}\t${r.ok ? "ok" : "fail"}\t'
          'transport=${r.transport?.name ?? "n/a"}\t'
          'http=${r.statusCode}\terror=${r.error ?? ""}',
        );
        exitCode = r.ok ? 0 : 1;
        return;
      case 'status-all':
        final transport =
            (parsed['preferred'] as String?) == 'oci_jump'
                ? FleetTransport.ociJump
                : FleetTransport.tailscale;
        final results = await client.healthCheckAll(transport);
        for (final r in results) {
          stdout.writeln(
            '${r.node}\t${r.ok ? "ok" : "fail"}\t'
            'transport=${r.transport?.name ?? "n/a"}\t'
            'http=${r.statusCode}\terror=${r.error ?? ""}',
          );
        }
        exitCode = results.isNotEmpty && results.every((r) => r.ok) ? 0 : 1;
        return;
      default:
        stderr.writeln('unknown subcommand ${args[0]}');
        exitCode = 2;
        return;
    }
  } finally {
    client.close();
  }
}
