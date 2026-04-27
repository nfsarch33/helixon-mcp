class FleetNode {
  const FleetNode({required this.name, required this.status});

  final String name;
  final String status;
}

String renderFleetNode(FleetNode node) => '${node.name}:${node.status}';
