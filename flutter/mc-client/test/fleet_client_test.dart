import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:mc_client/main.dart';
import 'package:test/test.dart';

void main() {
  group('FleetClient.healthCheck', () {
    test('returns ok=true on 200', () async {
      final mock = MockClient((req) async {
        expect(req.url.path, '/healthz');
        return http.Response('ok', 200);
      });
      final resolver = FleetEndpointResolver(
        tailscale: {
          'node-a': FleetEndpoint(
            transport: FleetTransport.tailscale,
            uri: Uri.parse('http://node-a.foo.ts.net:9302'),
            node: 'node-a',
          ),
        },
        ociJump: const {},
      );
      final client = FleetClient(resolver: resolver, httpClient: mock);
      final result = await client.healthCheck('node-a');
      expect(result.ok, isTrue);
      expect(result.statusCode, 200);
      expect(result.transport, FleetTransport.tailscale);
    });

    test('returns ok=false on 503', () async {
      final mock = MockClient((req) async => http.Response('down', 503));
      final resolver = FleetEndpointResolver(
        tailscale: {
          'node-b': FleetEndpoint(
            transport: FleetTransport.tailscale,
            uri: Uri.parse('http://node-b.foo.ts.net:9302'),
            node: 'node-b',
          ),
        },
        ociJump: const {},
      );
      final client = FleetClient(resolver: resolver, httpClient: mock);
      final result = await client.healthCheck('node-b');
      expect(result.ok, isFalse);
      expect(result.statusCode, 503);
    });

    test('returns error for unknown node', () async {
      final mock = MockClient((req) async => http.Response('', 200));
      final resolver = FleetEndpointResolver(
        tailscale: const {},
        ociJump: const {},
      );
      final client = FleetClient(resolver: resolver, httpClient: mock);
      final result = await client.healthCheck('ghost');
      expect(result.ok, isFalse);
      expect(result.error, contains('no endpoint configured'));
    });

    test('falls back to OCI jump when tailscale entry absent', () async {
      final mock = MockClient((req) async {
        expect(req.url.host, 'localhost');
        expect(req.url.port, 19302);
        return http.Response('ok', 200);
      });
      final resolver = FleetEndpointResolver(
        tailscale: const {},
        ociJump: {
          'node-a': FleetEndpoint(
            transport: FleetTransport.ociJump,
            uri: Uri.parse('http://localhost:19302'),
            node: 'node-a',
          ),
        },
      );
      final client = FleetClient(resolver: resolver, httpClient: mock);
      final result = await client.healthCheck('node-a');
      expect(result.ok, isTrue);
      expect(result.transport, FleetTransport.ociJump);
    });
  });

  group('FleetClient.submitCommand', () {
    test('posts envelope to /command and decodes JSON', () async {
      final mock = MockClient((req) async {
        expect(req.url.path, '/command');
        expect(req.method, 'POST');
        final body = jsonDecode(req.body) as Map<String, dynamic>;
        expect(body['text'], '/node node-a drift');
        expect(body['user_id'], 42);
        expect(body['source'], 'mc_ironclaw_bridge');
        expect(body['signature'], startsWith('v1='));
        return http.Response(jsonEncode({'ok': true, 'task_id': 't-1'}), 200);
      });
      final resolver = FleetEndpointResolver(
        tailscale: const {},
        ociJump: {
          'mc': FleetEndpoint(
            transport: FleetTransport.ociJump,
            uri: Uri.parse('http://localhost:19308'),
            node: 'mc',
          ),
        },
      );
      final client = FleetClient(resolver: resolver, httpClient: mock);
      const envelope = SignedCommandEnvelope(
        text: '/node node-a drift',
        userId: 42,
        signature: 'v1=deadbeef',
        source: 'mc_ironclaw_bridge',
      );
      final result = await client.submitCommand('mc', envelope);
      expect(result.ok, isTrue);
      expect(result.statusCode, 200);
      expect(result.responseJson['task_id'], 't-1');
    });

    test('returns ok=false on 4xx', () async {
      final mock = MockClient((req) async => http.Response('forbidden', 403));
      final resolver = FleetEndpointResolver(
        tailscale: const {},
        ociJump: {
          'mc': FleetEndpoint(
            transport: FleetTransport.ociJump,
            uri: Uri.parse('http://localhost:19308'),
            node: 'mc',
          ),
        },
      );
      final client = FleetClient(resolver: resolver, httpClient: mock);
      final result = await client.submitCommand(
        'mc',
        const SignedCommandEnvelope(text: '/x', userId: 1, signature: 'v1='),
      );
      expect(result.ok, isFalse);
      expect(result.statusCode, 403);
      expect(result.responseText, 'forbidden');
    });

    test('handles transport timeout cleanly', () async {
      final mock = MockClient((req) async {
        await Future<void>.delayed(const Duration(milliseconds: 50));
        return http.Response('late', 200);
      });
      final resolver = FleetEndpointResolver(
        tailscale: const {},
        ociJump: {
          'mc': FleetEndpoint(
            transport: FleetTransport.ociJump,
            uri: Uri.parse('http://localhost:19308'),
            node: 'mc',
          ),
        },
      );
      final client = FleetClient(
        resolver: resolver,
        httpClient: mock,
        timeout: const Duration(milliseconds: 5),
      );
      final result = await client.submitCommand(
        'mc',
        const SignedCommandEnvelope(text: '/x', userId: 1, signature: 'v1='),
      );
      expect(result.ok, isFalse);
      expect(result.error, contains('timeout'));
    });
  });
}
