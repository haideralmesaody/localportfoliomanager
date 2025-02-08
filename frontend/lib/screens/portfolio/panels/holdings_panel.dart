import 'package:flutter/material.dart';
import '../../../models/portfolio.dart';
import '../../../models/portfolio_holding.dart';
import '../../../services/api_service.dart';

class HoldingsPanel extends StatefulWidget {
  final Portfolio portfolio;

  const HoldingsPanel({
    Key? key,
    required this.portfolio,
  }) : super(key: key);

  @override
  State<HoldingsPanel> createState() => _HoldingsPanelState();
}

class _HoldingsPanelState extends State<HoldingsPanel> {
  final ApiService _apiService = ApiService();
  List<PortfolioHolding> holdings = [];
  bool isLoading = true;
  String? error;
  int? currentPortfolioId;

  @override
  void initState() {
    super.initState();
    currentPortfolioId = widget.portfolio.id;
    fetchHoldings();
  }

  @override
  void didUpdateWidget(HoldingsPanel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.portfolio.id != widget.portfolio.id) {
      print('Portfolio changed from ${oldWidget.portfolio.id} to ${widget.portfolio.id}');
      currentPortfolioId = widget.portfolio.id;
      fetchHoldings();
    }
  }

  Future<void> fetchHoldings() async {
    try {
      setState(() {
        isLoading = true;
        error = null;
      });

      print('Fetching holdings for portfolio ${widget.portfolio.id}');
      final response = await _apiService.get('portfolios/${widget.portfolio.id}/holdings');
      print('Holdings response: $response');
      
      if (response == null) {
        print('No holdings found for portfolio ${widget.portfolio.id}');
        setState(() {
          holdings = [];
          isLoading = false;
        });
        return;
      }

      final List<PortfolioHolding> fetchedHoldings = (response as List)
          .map((data) {
            print('Processing holding: $data');
            return PortfolioHolding.fromJson(data as Map<String, dynamic>);
          })
          .toList();

      if (currentPortfolioId == widget.portfolio.id) {
        setState(() {
          holdings = fetchedHoldings;
          isLoading = false;
        });
        print('Updated holdings for portfolio ${widget.portfolio.id}: ${holdings.length} holdings');
      }
    } catch (e, stackTrace) {
      print('Error fetching holdings: $e');
      print('Stack trace: $stackTrace');
      if (currentPortfolioId == widget.portfolio.id) {
        setState(() {
          error = e.toString();
          isLoading = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text('Error: $error'),
            const SizedBox(height: 16),
            ElevatedButton(
              onPressed: fetchHoldings,
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    if (holdings.isEmpty) {
      return Card(
        margin: const EdgeInsets.all(8.0),
        child: Center(
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(
                Icons.account_balance_wallet_outlined,
                size: 48,
                color: Colors.grey,
              ),
              const SizedBox(height: 16),
              Text(
                'No Holdings',
                style: Theme.of(context).textTheme.titleLarge?.copyWith(
                  color: Colors.grey[600],
                ),
              ),
              const SizedBox(height: 8),
              Text(
                'This portfolio has no active holdings',
                style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                  color: Colors.grey[500],
                ),
              ),
            ],
          ),
        ),
      );
    }

    return Card(
      margin: const EdgeInsets.all(8.0),
      child: SingleChildScrollView(
        child: DataTable(
          columns: const [
            DataColumn(label: Text('Ticker')),
            DataColumn(label: Text('Shares')),
            DataColumn(label: Text('Avg Cost')),
            DataColumn(label: Text('Current Price')),
            DataColumn(label: Text('Market Value')),
            DataColumn(label: Text('Unrealized Gain')),
            DataColumn(label: Text('% of Portfolio')),
          ],
          rows: holdings.map((holding) {
            final marketValue = 
                (holding.currentPrice ?? 0) * holding.shares;
            
            return DataRow(
              cells: [
                DataCell(Text(holding.ticker)),
                DataCell(Text(holding.shares.toStringAsFixed(2))),
                DataCell(Text('\$${holding.purchaseCostAverage.toStringAsFixed(3)}')),
                DataCell(Text(holding.currentPrice != null 
                    ? '\$${holding.currentPrice!.toStringAsFixed(3)}'
                    : 'N/A')),
                DataCell(Text(holding.currentPrice != null 
                    ? '\$${(holding.currentPrice! * holding.shares).toStringAsFixed(2)}'
                    : 'N/A')),
                DataCell(
                  Text(
                    holding.unrealizedGainAverage != null
                        ? '\$${holding.unrealizedGainAverage!.toStringAsFixed(2)}'
                        : 'N/A',
                    style: TextStyle(
                      color: holding.unrealizedGainAverage != null
                          ? (holding.unrealizedGainAverage! >= 0 ? Colors.green : Colors.red)
                          : null,
                    ),
                  ),
                ),
                DataCell(Text(holding.currentPercentage != null
                    ? '${holding.currentPercentage!.toStringAsFixed(2)}%'
                    : 'N/A')),
              ],
            );
          }).toList(),
        ),
      ),
    );
  }
} 