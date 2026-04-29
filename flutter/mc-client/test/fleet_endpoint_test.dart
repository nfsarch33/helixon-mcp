import 'package:mc_client/main.dart';
import 'package:test/test.dart';

void main() {
  group('FleetEndpointResolver', () {
    test('resolves preferred transport', () {
      final resolver = FleetEndpointResolver(
        tailscale: {
          'node-a': FleetEndpoint(
            transport: FleetTransport.tailscale,
            uri: Uri.parse('http://node-a.foo.ts.net:9302'),
            node: 'node-a',
          ),
        },
        ociJump: {
          'node-a': FleetEndpoint(
            transport: FleetTransport.ociJump,
            uri: Uri.parse('http://localhost:19302'),
            node: 'node-a',
          ),
        },
      );
      final ep = resolver.resolve('node-a');
      expect(ep, isNotNull);
      expect(ep!.transport, FleetTransport.tailscale);
      expect(ep.uri.toString(), 'http://node-a.foo.ts.net:9302');
    });

    test('falls back to alternate when preferred missing', () {
      final resolver = FleetEndpointResolver(
        tailscale: const {},
        ociJump: {
          'host-a': FleetEndpoint(
            transport: FleetTransport.ociJump,
            uri: Uri.parse('http://localhost:19401'),
            node: 'host-a',
          ),
        },
      );
      final ep = resolver.resolve('host-a');
      expect(ep, isNotNull);
      expect(ep!.transport, FleetTransport.ociJump);
    });

    test('returns null for unknown node', () {
      final resolver = FleetEndpointResolver(
        tailscale: const {},
        ociJump: const {},
      );
      expect(resolver.resolve('ghost'), isNull);
    });

    test('healthCheck path overrides input path', () {
      final ep = FleetEndpoint(
        transport: FleetTransport.tailscale,
        uri: Uri.parse('http://node-a.foo.ts.net:9302/v1'),
        node: 'node-a',
      );
      expect(ep.healthCheck.path, '/healthz');
      expect(ep.command.path, '/command');
    });

    test('fromJson decodes preferred + maps', () {
      final resolver = FleetEndpointResolver.fromJson({
        'preferred': 'oci_jump',
        'tailscale': {'node-a': 'http://node-a.foo.ts.net:9302'},
        'oci_jump': {'node-a': 'http://localhost:19302'},
      });
      expect(resolver.preferred, FleetTransport.ociJump);
      final ep = resolver.resolve('node-a');
      expect(ep!.transport, FleetTransport.ociJump);
    });
  });
}
