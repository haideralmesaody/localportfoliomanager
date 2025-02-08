import 'package:flutter/material.dart';
import '../../models/stock.dart';
import '../../services/api_service.dart';
import '../../widgets/sparkline_widget.dart';
import 'package:intl/intl.dart';

class MarketOverview extends StatefulWidget {
  const MarketOverview({Key? key}) : super(key: key);

  @override
  State<MarketOverview> createState() => _MarketOverviewState();
}

class _MarketOverviewState extends State<MarketOverview> {
  List<Stock> stocks = [];
  List<Stock> filteredStocks = [];
  bool isLoading = true;
  String? error;
  final ApiService _apiService = ApiService();
  int _sortColumnIndex = 0;
  bool _sortAscending = true;
  TextEditingController searchController = TextEditingController();

  @override
  void initState() {
    super.initState();
    searchController.addListener(_filterStocks);
    Future.delayed(const Duration(milliseconds: 500), () {
      fetchLatestStocks();
    });
  }

  void _filterStocks() {
    final query = searchController.text.toLowerCase();
    setState(() {
      filteredStocks = stocks.where((stock) {
        return stock.ticker.toLowerCase().contains(query) ||
               stock.date.toLowerCase().contains(query);
      }).toList();
    });
  }

  Future<void> fetchLatestStocks() async {
    try {
      setState(() {
        isLoading = true;
        error = null;
      });

      print('Fetching latest stocks...'); // Debug log
      final jsonData = await _apiService.get('stocks/latest');
      print('Received data: $jsonData'); // Debug log

      if (jsonData == null || !jsonData.containsKey('stocks')) {
        throw Exception('Invalid response format');
      }

      final List<Stock> stocksList = (jsonData['stocks'] as List)
          .map((data) => Stock.fromJson(data))
          .toList();

      // Fetch sparkline data for each stock
      for (var stock in stocksList) {
        try {
          final sparklineData = await _apiService.get('stocks/${stock.ticker}/sparkline');
          if (sparklineData != null) {
            final List<double> prices = List<double>.from(sparklineData['prices'] ?? []);
            final List<String> dates = List<String>.from(sparklineData['dates'] ?? []);
            
            // Create a new Stock instance with updated sparkline data
            final int index = stocksList.indexOf(stock);
            stocksList[index] = Stock(
              ticker: stock.ticker,
              date: stock.date,
              openPrice: stock.openPrice,
              highPrice: stock.highPrice,
              lowPrice: stock.lowPrice,
              closePrice: stock.closePrice,
              lastPrice: stock.lastPrice,
              sharesTraded: stock.sharesTraded,
              valueTraded: stock.valueTraded,
              numTrades: stock.numTrades,
              change: stock.change,
              changePercentage: stock.changePercentage,
              sparklinePrices: prices,
              sparklineDates: dates,
            );
          }
        } catch (e) {
          print('Error fetching sparkline for ${stock.ticker}: $e');
        }
      }

      setState(() {
        stocks = stocksList;
        filteredStocks = stocksList;
        isLoading = false;
      });
    } catch (e) {
      print('Error fetching stocks: $e'); // Debug log
      setState(() {
        error = 'Error connecting to server: $e';
        isLoading = false;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Market Overview'),
        elevation: 0, // Remove shadow
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: fetchLatestStocks,
            tooltip: 'Refresh Data',
          ),
        ],
      ),
      body: Container(
        color: Colors.grey[50], // Light background
        child: Column(
          children: [
            // Search bar with better styling
            Container(
              padding: const EdgeInsets.all(16.0),
              decoration: BoxDecoration(
                color: Colors.white,
                boxShadow: [
                  BoxShadow(
                    color: Colors.grey.withOpacity(0.1),
                    spreadRadius: 1,
                    blurRadius: 3,
                  ),
                ],
              ),
              child: TextField(
                controller: searchController,
                decoration: InputDecoration(
                  labelText: 'Search Stocks',
                  hintText: 'Enter ticker or date...',
                  prefixIcon: const Icon(Icons.search),
                  suffixIcon: IconButton(
                    icon: const Icon(Icons.clear),
                    onPressed: () {
                      searchController.clear();
                      _filterStocks();
                    },
                  ),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(color: Colors.grey[300]!),
                  ),
                  enabledBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(color: Colors.grey[300]!),
                  ),
                  focusedBorder: OutlineInputBorder(
                    borderRadius: BorderRadius.circular(8),
                    borderSide: BorderSide(color: Theme.of(context).primaryColor),
                  ),
                  filled: true,
                  fillColor: Colors.white,
                ),
              ),
            ),
            // Table content
            Expanded(
              child: isLoading
                ? const Center(child: CircularProgressIndicator())
                : error != null
                  ? _buildErrorWidget()
                  : RefreshIndicator(
                      onRefresh: fetchLatestStocks,
                      child: filteredStocks.isEmpty
                        ? _buildEmptyState()
                        : Card(
                            margin: const EdgeInsets.all(16.0),
                            elevation: 2,
                            shape: RoundedRectangleBorder(
                              borderRadius: BorderRadius.circular(8),
                            ),
                            child: SingleChildScrollView(
                              scrollDirection: Axis.horizontal,
                              child: SingleChildScrollView(
                                child: buildStockTable(),
                              ),
                            ),
                          ),
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Color getDateColor(String dateStr) {
    final date = DateTime.parse(dateStr);
    final today = DateTime.now();
    final difference = today.difference(date).inDays;
    
    if (difference > 5) {
      return Colors.red;
    }
    return Colors.green;
  }

  DataTable buildStockTable() {
    return DataTable(
      showCheckboxColumn: false,
      sortAscending: _sortAscending,
      sortColumnIndex: _sortColumnIndex,
      columnSpacing: 16,  // Reduced spacing
      horizontalMargin: 16,
      headingRowHeight: 48,  // Reduced header height
      dataRowHeight: 52,    // Standard material design row height
      headingTextStyle: const TextStyle(
        fontWeight: FontWeight.w600,
        color: Colors.black87,
        fontSize: 13,
      ),
      dataTextStyle: const TextStyle(
        fontSize: 13,
        color: Colors.black87,
      ),
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(
            color: Colors.grey[200]!,
            width: 1,
          ),
        ),
      ),
      dataRowColor: MaterialStateProperty.resolveWith<Color?>(
        (Set<MaterialState> states) {
          if (states.contains(MaterialState.hovered)) {
            return Colors.grey.withOpacity(0.1);
          }
          return states.contains(MaterialState.selected) 
              ? Theme.of(context).colorScheme.primary.withOpacity(0.08)
              : null;
        },
      ),
      columns: [
        DataColumn(
          label: const SizedBox(
            width: 80,  // Fixed width for ticker
            child: Text('Ticker'),
          ),
          onSort: (columnIndex, ascending) => _sort((stock) => stock.ticker, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 100,  // Fixed width for date
            child: Text('Date'),
          ),
          onSort: (columnIndex, ascending) => _sort((stock) => stock.date, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 90,  // Fixed width for price columns
            child: Text('Open'),
          ),
          numeric: true,
          onSort: (columnIndex, ascending) => _sort((stock) => stock.openPrice, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 90,  // Fixed width for price columns
            child: Text('High'),
          ),
          numeric: true,
          onSort: (columnIndex, ascending) => _sort((stock) => stock.highPrice, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 90,  // Fixed width for price columns
            child: Text('Low'),
          ),
          numeric: true,
          onSort: (columnIndex, ascending) => _sort((stock) => stock.lowPrice, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 90,  // Fixed width for price columns
            child: Text('Close'),
          ),
          numeric: true,
          onSort: (columnIndex, ascending) => _sort((stock) => stock.closePrice, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 90,
            child: Text('Change'),
          ),
          numeric: true,
          onSort: (columnIndex, ascending) => _sort((stock) => stock.change, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 90,
            child: Text('Change%'),
          ),
          numeric: true,
          onSort: (columnIndex, ascending) => _sort((stock) => stock.changePercentage, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 100,
            child: Text('Volume'),
          ),
          numeric: true,
          onSort: (columnIndex, ascending) => _sort((stock) => stock.sharesTraded, columnIndex, ascending),
        ),
        DataColumn(
          label: const SizedBox(
            width: 120,
            child: Text('Value Traded'),
          ),
          numeric: true,
          onSort: (columnIndex, ascending) => _sort((stock) => stock.valueTraded, columnIndex, ascending),
        ),
        const DataColumn(
          label: SizedBox(
            width: 150,  // Fixed width for sparkline
            child: Text('Trend'),
          ),
        ),
      ],
      rows: filteredStocks.map((stock) => DataRow(
        onSelectChanged: (_) => _navigateToDetail(stock.ticker),
        cells: [
          DataCell(Text(
            stock.ticker,
            style: const TextStyle(fontWeight: FontWeight.bold),
          )),
          DataCell(
            Text(
              DateFormat('yyyy-MM-dd').format(DateTime.parse(stock.date)),
              style: TextStyle(
                color: getDateColor(stock.date),
                fontWeight: FontWeight.bold,
              ),
            ),
          ),
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
          DataCell(
            Container(
              alignment: Alignment.centerRight,
              child: Text(
                '${stock.change >= 0 ? '+' : ''}${stock.change.toStringAsFixed(2)}',
                style: TextStyle(
                  color: getChangeColor(stock.change),
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),
          ),
          DataCell(
            Container(
              alignment: Alignment.centerRight,
              child: Text(
                '${stock.changePercentage >= 0 ? '+' : ''}${stock.changePercentage.toStringAsFixed(2)}%',
                style: TextStyle(
                  color: getChangeColor(stock.changePercentage),
                  fontWeight: FontWeight.bold,
                ),
              ),
            ),
          ),
          DataCell(Text(
            formatNumber(stock.sharesTraded),
            textAlign: TextAlign.right,
          )),
          DataCell(Text(
            '\$${formatNumber(stock.valueTraded)}',
            textAlign: TextAlign.right,
          )),
          DataCell(
            Padding(
              padding: const EdgeInsets.symmetric(vertical: 4.0),
              child: SizedBox(
                width: 120,
                height: 35,  // Reduced sparkline height
                child: SparklineWidget(
                  prices: stock.sparklinePrices,
                  dates: stock.sparklineDates,
                  changePercentage: stock.changePercentage,
                  color: getChangeColor(stock.changePercentage),
                ),
              ),
            ),
          ),
        ],
      )).toList(),
    );
  }

  String formatNumber(num number) {
    if (number >= 1000000000) {
      return '${(number / 1000000000).toStringAsFixed(1)}B';
    }
    if (number >= 1000000) {
      return '${(number / 1000000).toStringAsFixed(1)}M';
    }
    if (number >= 1000) {
      return '${(number / 1000).toStringAsFixed(1)}K';
    }
    return number.toString();
  }

  Color getChangeColor(num value) {
    if (value > 0) return Colors.green;
    if (value < 0) return Colors.red;
    return Colors.black;
  }

  void _sort<T>(T Function(Stock stock) getField, int columnIndex, bool ascending) {
    setState(() {
      _sortColumnIndex = columnIndex;
      _sortAscending = ascending;
      
      filteredStocks.sort((a, b) {
        final aValue = getField(a);
        final bValue = getField(b);
        
        return ascending
            ? Comparable.compare(aValue as Comparable, bValue as Comparable)
            : Comparable.compare(bValue as Comparable, aValue as Comparable);
      });
    });
  }

  void _navigateToDetail(String ticker) {
    Navigator.pushNamed(
      context,
      '/stock-detail',
      arguments: ticker,
    );
  }

  @override
  void dispose() {
    searchController.dispose();
    super.dispose();
  }

  Widget _buildErrorWidget() {
    return Center(
      child: Card(
        margin: const EdgeInsets.all(16),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.error_outline, size: 48, color: Colors.red[400]),
              const SizedBox(height: 16),
              Text(error ?? 'An error occurred'),
              const SizedBox(height: 16),
              ElevatedButton(
                onPressed: fetchLatestStocks,
                child: const Text('Retry'),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(Icons.search_off, size: 64, color: Colors.grey[400]),
          const SizedBox(height: 16),
          Text(
            'No stocks found',
            style: TextStyle(
              fontSize: 18,
              color: Colors.grey[600],
              fontWeight: FontWeight.w500,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Try adjusting your search',
            style: TextStyle(
              color: Colors.grey[500],
            ),
          ),
        ],
      ),
    );
  }
} 