import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import 'package:intl/intl.dart';

class SparklineWidget extends StatelessWidget {
  final List<double> prices;
  final List<String> dates;
  final Color? color;
  final double height;
  final double width;
  final double changePercentage;

  const SparklineWidget({
    Key? key,
    required this.prices,
    required this.dates,
    this.color,
    this.height = 50,
    this.width = 100,
    required this.changePercentage,
  }) : super(key: key);

  Color _getChangeColor(BuildContext context) {
    if (changePercentage > 0) return Colors.green;
    if (changePercentage < 0) return Colors.red;
    return Theme.of(context).textTheme.bodyLarge?.color ?? Colors.grey;
  }

  @override
  Widget build(BuildContext context) {
    if (prices.isEmpty) {
      return SizedBox(height: height, width: width);
    }

    final spots = List.generate(
      prices.length,
      (index) => FlSpot(index.toDouble(), prices[index]),
    );

    return SizedBox(
      height: height,
      width: width,
      child: LineChart(
        LineChartData(
          gridData: const FlGridData(show: false),
          titlesData: const FlTitlesData(show: false),
          borderData: FlBorderData(show: false),
          minX: 0,
          maxX: spots.length.toDouble() - 1,
          minY: prices.reduce((a, b) => a < b ? a : b),
          maxY: prices.reduce((a, b) => a > b ? a : b),
          lineBarsData: [
            LineChartBarData(
              spots: spots,
              isCurved: true,
              color: color ?? _getChangeColor(context),
              barWidth: 2,
              isStrokeCapRound: true,
              dotData: const FlDotData(show: false),
              belowBarData: BarAreaData(show: false),
            ),
          ],
          lineTouchData: LineTouchData(
            enabled: true,
            touchTooltipData: LineTouchTooltipData(
              tooltipBgColor: Colors.black.withOpacity(0.8),
              getTooltipItems: (List<LineBarSpot> touchedSpots) {
                return touchedSpots.map((LineBarSpot touchedSpot) {
                  final int index = touchedSpot.x.toInt();
                  final double price = touchedSpot.y;
                  final String date = dates[index];
                  return LineTooltipItem(
                    '${DateFormat('MMM dd').format(DateTime.parse(date))}\n\$${price.toStringAsFixed(2)}',
                    const TextStyle(
                      color: Colors.white,
                      fontSize: 12,
                      fontWeight: FontWeight.bold,
                    ),
                  );
                }).toList();
              },
            ),
            touchCallback: (FlTouchEvent event, LineTouchResponse? touchResponse) {},
            handleBuiltInTouches: true,
          ),
        ),
      ),
    );
  }
} 