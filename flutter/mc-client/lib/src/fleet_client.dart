import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'fleet_endpoint.dart';
import 'fleet_node.dart';

/// FleetClient is the high-level dart-side mc-client. It wraps an
/// HTTP transport and a FleetEndpointResolver so the UI layer never
/// has to know whether it's talking to Tailscale or the OCI-jump
/// tunnel. Operations:
///
///   - [healthCheck]: GET /healthz on a logical fleet member.
///   - [submitCommand]: POST /command to the IronClaw sidecar with
///     a pre-signed envelope; returns the bridge's response body.
class FleetClient {
  FleetClient({
    required this.resolver,
    http.Client? httpClient,
    Duration? timeout,
  }) : _httpClient = httpClient ?? http.Client(),
       _timeout = timeout ?? const Duration(seconds: 10);

  final FleetEndpointResolver resolver;
  final http.Client _httpClient;
  final Duration _timeout;

  Future<HealthCheckResult> healthCheck(String node) async {
    final endpoint = resolver.resolve(node);
    if (endpoint == null) {
      return HealthCheckResult(
        node: node,
        ok: false,
        statusCode: 0,
        error: 'no endpoint configured for node $node',
      );
    }
    try {
      final resp = await _httpClient
          .get(endpoint.healthCheck)
          .timeout(_timeout);
      return HealthCheckResult(
        node: node,
        ok: resp.statusCode == 200,
        statusCode: resp.statusCode,
        transport: endpoint.transport,
      );
    } on TimeoutException {
      return HealthCheckResult(
        node: node,
        ok: false,
        statusCode: 0,
        transport: endpoint.transport,
        error: 'timeout after ${_timeout.inSeconds}s',
      );
    } catch (e) {
      return HealthCheckResult(
        node: node,
        ok: false,
        statusCode: 0,
        transport: endpoint.transport,
        error: e.toString(),
      );
    }
  }

  /// healthCheckAll fans out to every endpoint in the configured
  /// transport, useful for the dashboard bring-up screen.
  Future<List<HealthCheckResult>> healthCheckAll(
    FleetTransport transport,
  ) async {
    final endpoints = resolver.resolveAll(transport).toList();
    final results = await Future.wait(
      endpoints.map((e) => healthCheck(e.node)),
    );
    return results;
  }

  /// submitCommand POSTs an IronClaw-bridge envelope. The signature
  /// must already be computed by the caller using the same canonical
  /// form the Go-side mcsidecar.IronclawSignBody emits. Keeping the
  /// signing here would force this dart code to know the shared HMAC
  /// secret; instead we let a higher layer (a 1Password-backed
  /// provider) inject the pre-signed envelope.
  Future<CommandResult> submitCommand(
    String node,
    SignedCommandEnvelope envelope,
  ) async {
    final endpoint = resolver.resolve(node);
    if (endpoint == null) {
      return CommandResult(
        ok: false,
        error: 'no endpoint configured for node $node',
        node: node,
      );
    }
    try {
      final resp = await _httpClient
          .post(
            endpoint.command,
            headers: {'content-type': 'application/json'},
            body: jsonEncode(envelope.toJson()),
          )
          .timeout(_timeout);
      Map<String, dynamic> body = const {};
      try {
        if (resp.body.isNotEmpty) {
          body = jsonDecode(resp.body) as Map<String, dynamic>;
        }
      } catch (_) {
        // Non-JSON body is fine; we surface raw text below.
      }
      return CommandResult(
        ok: resp.statusCode >= 200 && resp.statusCode < 300,
        statusCode: resp.statusCode,
        node: node,
        transport: endpoint.transport,
        responseText: resp.body,
        responseJson: body,
      );
    } on TimeoutException {
      return CommandResult(
        ok: false,
        node: node,
        transport: endpoint.transport,
        error: 'timeout after ${_timeout.inSeconds}s',
      );
    } catch (e) {
      return CommandResult(
        ok: false,
        node: node,
        transport: endpoint.transport,
        error: e.toString(),
      );
    }
  }

  void close() => _httpClient.close();
}

class HealthCheckResult {
  const HealthCheckResult({
    required this.node,
    required this.ok,
    required this.statusCode,
    this.transport,
    this.error,
  });

  final String node;
  final bool ok;
  final int statusCode;
  final FleetTransport? transport;
  final String? error;

  FleetNode toFleetNode() => FleetNode(
    name: node,
    status: ok ? 'healthy' : 'unhealthy',
    lastSeen: DateTime.now(),
  );
}

class SignedCommandEnvelope {
  const SignedCommandEnvelope({
    required this.text,
    required this.userId,
    required this.signature,
    this.channelId = '',
    this.source = 'mc_ironclaw_bridge',
  });

  final String text;
  final int userId;
  final String channelId;
  final String source;
  final String signature;

  Map<String, dynamic> toJson() => {
    'text': text,
    'user_id': userId,
    'channel_id': channelId,
    'source': source,
    'signature': signature,
  };
}

class CommandResult {
  const CommandResult({
    required this.ok,
    required this.node,
    this.statusCode = 0,
    this.transport,
    this.responseText = '',
    this.responseJson = const {},
    this.error,
  });

  final bool ok;
  final int statusCode;
  final String node;
  final FleetTransport? transport;
  final String responseText;
  final Map<String, dynamic> responseJson;
  final String? error;
}
