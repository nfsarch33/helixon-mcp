import 'package:mc_client/main.dart';
import 'package:test/test.dart';

void main() {
  test('renders fleet node status', () {
    const node = FleetNode(name: 'node-a', status: 'healthy');
    expect(renderFleetNode(node), 'node-a:healthy');
  });
}
