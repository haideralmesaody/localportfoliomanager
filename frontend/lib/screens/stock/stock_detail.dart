import 'package:flutter/material.dart';
import 'package:split_view/split_view.dart';
import '../../models/stock.dart';
import '../../services/api_service.dart';
import 'package:intl/intl.dart';

/// Page to display detailed information about a stock, including
/// historical data in a table and an interactive candlestick chart.
class StockDetailPage extends StatefulWidget {
  final String ticker;
  const StockDetailPage({Key? key, required this.ticker}): super(key: key);

  @override
  State<StockDetailPage> createState() => _StockDetailPageState();
}

class _StockDetailPageState extends State<StockDetailPage> {
  final ApiService _apiService = ApiService();

  /// Indicates if the page is loading data.
  bool isLoading = true;

  /// Stores any error message encountered during data fetching.
  String? error;

  /// The name of the company.
  String companyName = '';

  /// List of historical stock data.
  List<Stock> historicalData = [];

  /// Controls the visibility of the data table.
  bool isTableVisible = true;

  /// Controls the visibility of the candlestick chart.
  bool isChartVisible = true;

  int _sortColumnIndex = 0;  // Default sort by date
  bool _sortAscending = false;  // Default sort descending (newest first)

  @override
  void initState() {
    super.initState();
    fetchStockDetails();
  }

  /// Fetches the stock details from the API.
  Future<void> fetchStockDetails() async {
    try {
      setState(() {
        isLoading = true;
        error = null;
      });

      print('Fetching details for ticker: ${widget.ticker}');
      final data = await _apiService.get('stocks/${widget.ticker}/details');
      print('Received data: $data');

      if (data == null) {
        throw Exception('No data received from server');
      }

      if (!data.containsKey('prices') || !data.containsKey('company_name')) {
        throw Exception('Invalid data format received');
      }

      final List<Stock> stocks = (data['prices'] as List? ?? [])
          .map((item) {
            print('Processing stock item: $item');
            return Stock.fromJson(item as Map<String, dynamic>);
          })
          .toList();

      if (stocks.isEmpty) {
        throw Exception('No historical data available');
      }

      setState(() {
        companyName = data['company_name'] ?? '';
        historicalData = stocks;
        isLoading = false;
      });
    } catch (e) {
      print('Error fetching stock details: $e');
      setState(() {
        error = e.toString();
        isLoading = false;
      });
    }
  }

  void _sort<T>(T Function(Stock stock) getField, int columnIndex, bool ascending) {
    setState(() {
      _sortColumnIndex = columnIndex;
      _sortAscending = ascending;
      
      historicalData.sort((a, b) {
        final aValue = getField(a);
        final bValue = getField(b);
        
        return ascending
            ? Comparable.compare(aValue as Comparable, bValue as Comparable)
            : Comparable.compare(bValue as Comparable, aValue as Comparable);
      });
    });
  }

  /// Builds the historical data table.
  Widget buildTable() {
    return Card(
      margin: const EdgeInsets.all(8),
      elevation: 2,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: const EdgeInsets.all(16.0),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  'Historical Data',
                  style: TextStyle(
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: () {
                    setState(() {
                      isTableVisible = false;
                      if (!isChartVisible) isChartVisible = true;
                    });
                  },
                ),
              ],
            ),
          ),
          Expanded(
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: SingleChildScrollView(
                child: DataTable(
                  showCheckboxColumn: false,
                  sortColumnIndex: _sortColumnIndex,
                  sortAscending: _sortAscending,
                  columnSpacing: 16,
                  horizontalMargin: 16,
                  headingRowHeight: 48,
                  dataRowHeight: 52,
                  headingTextStyle: const TextStyle(
                    fontWeight: FontWeight.w600,
                    color: Colors.black87,
                    fontSize: 13,
                  ),
                  dataTextStyle: const TextStyle(
                    fontSize: 13,
                    color: Colors.black87,
                  ),
                  columns: [
                    DataColumn(
                      label: const SizedBox(width: 100, child: Text('Date')),
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => DateTime.parse(stock.date),
                        columnIndex,
                        ascending,
                      ),
                    ),
                    DataColumn(
                      label: const SizedBox(width: 90, child: Text('Open')),
                      numeric: true,
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => stock.openPrice,
                        columnIndex,
                        ascending,
                      ),
                    ),
                    DataColumn(
                      label: const SizedBox(width: 90, child: Text('High')),
                      numeric: true,
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => stock.highPrice,
                        columnIndex,
                        ascending,
                      ),
                    ),
                    DataColumn(
                      label: const SizedBox(width: 90, child: Text('Low')),
                      numeric: true,
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => stock.lowPrice,
                        columnIndex,
                        ascending,
                      ),
                    ),
                    DataColumn(
                      label: const SizedBox(width: 90, child: Text('Close')),
                      numeric: true,
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => stock.closePrice,
                        columnIndex,
                        ascending,
                      ),
                    ),
                    DataColumn(
                      label: const SizedBox(width: 90, child: Text('Change')),
                      numeric: true,
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => stock.change,
                        columnIndex,
                        ascending,
                      ),
                    ),
                    DataColumn(
                      label: const SizedBox(width: 90, child: Text('Change%')),
                      numeric: true,
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => stock.changePercentage,
                        columnIndex,
                        ascending,
                      ),
                    ),
                    DataColumn(
                      label: const SizedBox(width: 90, child: Text('Volume')),
                      numeric: true,
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => stock.sharesTraded,
                        columnIndex,
                        ascending,
                      ),
                    ),
                    DataColumn(
                      label: const SizedBox(width: 90, child: Text('Value Traded')),
                      numeric: true,
                      onSort: (columnIndex, ascending) => _sort(
                        (stock) => stock.valueTraded,
                        columnIndex,
                        ascending,
                      ),
                    ),
                  ],
                  rows: historicalData.map((stock) => DataRow(
                    cells: [
                      DataCell(Text(
                        DateFormat('yyyy-MM-dd').format(DateTime.parse(stock.date)),
                        style: TextStyle(
                          color: getDateColor(stock.date),
                          fontWeight: FontWeight.w500,
                        ),
                      )),
                      DataCell(Text(
                        '\$${stock.openPrice.toStringAsFixed(2)}',
                        textAlign: TextAlign.right,
                      )),
                      DataCell(Text(
                        '\$${stock.highPrice.toStringAsFixed(2)}',
                        textAlign: TextAlign.right,
                      )),
                      DataCell(Text(
                        '\$${stock.lowPrice.toStringAsFixed(2)}',
                        textAlign: TextAlign.right,
                      )),
                      DataCell(Text(
                        '\$${stock.closePrice.toStringAsFixed(2)}',
                        textAlign: TextAlign.right,
                      )),
                      DataCell(Text(
                        '${stock.change >= 0 ? '+' : ''}${stock.change.toStringAsFixed(2)}',
                        style: TextStyle(
                          color: getChangeColor(stock.change),
                          fontWeight: FontWeight.w500,
                        ),
                        textAlign: TextAlign.right,
                      )),
                      DataCell(Text(
                        '${stock.changePercentage >= 0 ? '+' : ''}${stock.changePercentage.toStringAsFixed(2)}%',
                        style: TextStyle(
                          color: getChangeColor(stock.changePercentage),
                          fontWeight: FontWeight.w500,
                        ),
                        textAlign: TextAlign.right,
                      )),
                      DataCell(Text(
                        formatNumber(stock.sharesTraded),
                        textAlign: TextAlign.right,
                      )),
                      DataCell(Text(
                        formatNumber(stock.valueTraded),
                        textAlign: TextAlign.right,
                      )),
                    ],
                  )).toList(),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// Determines the color for the date in the table.
  Color getDateColor(String dateStr) {
    final date = DateTime.parse(dateStr);
    final today = DateTime.now();
    final difference = today.difference(date).inDays;
    return difference > 5? Colors.red: Colors.black;
  }

  /// Determines the color for change values (green for positive, red for negative).
  Color getChangeColor(num value) {
    if (value > 0) return Colors.green;
    if (value < 0) return Colors.red;
    return Colors.black;
  }

  /// Builds the candlestick chart using ECharts.
  Widget buildChart() {
    return Card(
      margin: const EdgeInsets.all(8),
      child: Column(
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Padding(
                padding: EdgeInsets.all(8.0),
                child: Text(
                  'Price Chart - Under Construction',
                  style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                ),
              ),
              IconButton(
                icon: const Icon(Icons.close),
                onPressed: () {
                  setState(() {
                    isChartVisible = false;
                    if (!isTableVisible) isTableVisible = true;
                  });
                },
              ),
            ],
          ),
          const Expanded(
            child: Center(
              child: Text(
                '🚧 Chart Feature Coming Soon 🚧',
                style: TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.bold,
                  color: Colors.grey,
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  /// Formats a number with commas for better readability.
  String formatNumber(num value) {
    return value.toString().replaceAllMapped(
        RegExp(r'(\d{1,3})(?=(\d{3})+(?!\d))'), (Match m) => '${m[1]},');
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          onPressed: () => Navigator.pop(context),
        ),
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(widget.ticker),
            Text(
              isLoading? 'Loading...': companyName,
              style: const TextStyle(fontSize: 14),
            ),
          ],
        ),
        actions: [
          if (!isTableVisible)
            IconButton(
              icon: const Icon(Icons.table_chart),
              onPressed: () => setState(() => isTableVisible = true),
              tooltip: 'Show Table',
            ),
          if (!isChartVisible)
            IconButton(
              icon: const Icon(Icons.candlestick_chart),
              onPressed: () => setState(() => isChartVisible = true),
              tooltip: 'Show Chart',
            ),
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: fetchStockDetails,
          ),
        ],
      ),
      body: isLoading
        ? const Center(child: CircularProgressIndicator())
        : error != null
            ? Center(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    Text(error!),
                    ElevatedButton(
                      onPressed: fetchStockDetails,
                      child: const Text('Retry'),
                    ),
                  ],
                ),
              )
            : SplitView(
                viewMode: SplitViewMode.Horizontal,
                indicator: const SplitIndicator(
                  viewMode: SplitViewMode.Horizontal,
                  color: Colors.grey,
                ),
                activeIndicator: const SplitIndicator(
                  viewMode: SplitViewMode.Horizontal,
                  color: Colors.black,
                ),
                controller: SplitViewController(
                  weights: isTableVisible && isChartVisible ? [0.5, 0.5] : [1.0],
                ),
                children: [
                  if (isTableVisible) buildTable(),
                  if (isChartVisible) buildChart(),
                ],
              ),
    );
  }
}