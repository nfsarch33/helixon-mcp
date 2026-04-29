// Public API for mc-client v259. The data-plane is pure Dart so
// `dart test` and `dart compile` work without a Flutter SDK; the
// optional Flutter shell consumes the same public symbols.
//
// We re-export the v258 FleetNode + renderFleetNode from `src/`
// so the existing v258 tests keep passing without any changes.

export 'src/fleet_client.dart';
export 'src/fleet_endpoint.dart';
export 'src/fleet_node.dart';
