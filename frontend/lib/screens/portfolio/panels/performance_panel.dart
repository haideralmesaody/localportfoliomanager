import 'package:flutter/material.dart';
import '../../../models/portfolio.dart';

class PerformancePanel extends StatelessWidget {
  final Portfolio portfolio;

  const PerformancePanel({
    Key? key,
    required this.portfolio,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.all(8.0),
      child: Center(
        child: Text('Performance metrics for ${portfolio.name}\nComing soon...'),
      ),
    );
  }
}
