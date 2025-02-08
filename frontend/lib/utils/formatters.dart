import 'package:intl/intl.dart';

String formatCurrency(double value) {
  return NumberFormat.currency(symbol: '\$', decimalDigits: 2).format(value);
}

String formatNumber(double value) {
  return NumberFormat.decimalPattern().format(value);
}

String formatDate(DateTime date) {
  return DateFormat('yyyy-MM-dd').format(date);
} 