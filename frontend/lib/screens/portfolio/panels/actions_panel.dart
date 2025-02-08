import 'package:flutter/material.dart';
import '../../../models/portfolio.dart';

class ActionsPanel extends StatelessWidget {
  final Portfolio portfolio;

  const ActionsPanel({
    Key? key,
    required this.portfolio,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Card(
      margin: const EdgeInsets.all(8.0),
      child: Center(
        child: Text('Actions for ${portfolio.name}\nComing soon...'),
      ),
    );
  }
}
