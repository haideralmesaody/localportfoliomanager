import 'package:flutter/material.dart';
import '../../services/api_service.dart';
import '../../models/portfolio.dart';
import 'package:split_view/split_view.dart';
import 'panels/transactions_panel.dart';
import 'panels/performance_panel.dart';
import 'panels/actions_panel.dart';
import 'panels/holdings_panel.dart';

class PortfolioDashboard extends StatefulWidget {
  const PortfolioDashboard({Key? key}) : super(key: key);

  @override
  State<PortfolioDashboard> createState() => _PortfolioDashboardState();
}

class _PortfolioDashboardState extends State<PortfolioDashboard> {
  final ApiService _apiService = ApiService();
  List<Portfolio> portfolios = [];
  bool isLoading = true;
  String? error;
  Portfolio? selectedPortfolio;
  int _selectedPanelIndex = 0;

  @override
  void initState() {
    super.initState();
    fetchPortfolios();
  }

  Future<void> fetchPortfolios() async {
    try {
      setState(() {
        isLoading = true;
        error = null;
      });

      final response = await _apiService.get('portfolios');
      if (response == null) {
        throw Exception('No response received from server');
      }

      final List<Portfolio> fetchedPortfolios = (response as List)
          .map((data) => Portfolio.fromJson(data as Map<String, dynamic>))
          .toList();

      setState(() {
        portfolios = fetchedPortfolios;
        isLoading = false;
      });
    } catch (e) {
      setState(() {
        error = e.toString();
        isLoading = false;
      });
    }
  }

  Future<void> _showAddPortfolioDialog() async {
    final nameController = TextEditingController();
    final descriptionController = TextEditingController();

    return showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Create Portfolio'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: nameController,
              decoration: const InputDecoration(labelText: 'Portfolio Name'),
            ),
            TextField(
              controller: descriptionController,
              decoration: const InputDecoration(labelText: 'Description'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () async {
              try {
                final response = await _apiService.post('portfolios', {
                  'name': nameController.text,
                  'description': descriptionController.text,
                });
                final newPortfolio = Portfolio.fromJson(response);
                setState(() {
                  portfolios = [newPortfolio, ...portfolios];
                });
                Navigator.pop(context);
              } catch (e) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('Error creating portfolio: $e')),
                );
              }
            },
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  Future<void> _showEditPortfolioDialog(Portfolio portfolio) async {
    final nameController = TextEditingController(text: portfolio.name);
    final descriptionController = TextEditingController(text: portfolio.description);

    return showDialog(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Edit Portfolio'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            TextField(
              controller: nameController,
              decoration: const InputDecoration(labelText: 'Portfolio Name'),
            ),
            TextField(
              controller: descriptionController,
              decoration: const InputDecoration(labelText: 'Description'),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () async {
              try {
                await _apiService.put('portfolios/${portfolio.id}/rename', {
                  'new_name': nameController.text,
                  'description': descriptionController.text,
                });
                await fetchPortfolios(); // Refresh the list
                Navigator.pop(context);
                ScaffoldMessenger.of(context).showSnackBar(
                  const SnackBar(content: Text('Portfolio updated successfully')),
                );
              } catch (e) {
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(content: Text('Error updating portfolio: $e')),
                );
              }
            },
            child: const Text('Update'),
          ),
        ],
      ),
    );
  }

  Future<void> _deletePortfolio(Portfolio portfolio) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Delete Portfolio'),
        content: Text('Are you sure you want to delete "${portfolio.name}"?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.pop(context, true),
            style: ElevatedButton.styleFrom(backgroundColor: Colors.red),
            child: const Text('Delete'),
          ),
        ],
      ),
    );

    if (confirmed == true) {
      try {
        print('Deleting portfolio ${portfolio.id}'); // Debug log
        await _apiService.delete('portfolios/${portfolio.id}');
        setState(() {
          portfolios.removeWhere((p) => p.id == portfolio.id);
          if (selectedPortfolio?.id == portfolio.id) {
            selectedPortfolio = null;
          }
        });
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text('Portfolio deleted successfully')),
          );
        }
      } catch (e) {
        print('Delete error: $e'); // Debug log
        if (mounted) {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(content: Text('Error deleting portfolio: $e')),
          );
        }
      }
    }
  }

  Widget _buildPortfolioList() {
    if (isLoading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (error != null) {
      return Center(
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(error!),
            ElevatedButton(
              onPressed: fetchPortfolios,
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    return ListView.builder(
      itemCount: portfolios.length,
      itemBuilder: (context, index) {
        final portfolio = portfolios[index];
        return ListTile(
          selected: selectedPortfolio?.id == portfolio.id,
          title: Row(
            children: [
              // Display ID in a chip
              Chip(
                label: Text('#${portfolio.id}'),
                backgroundColor: Colors.grey[200],
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(portfolio.name),
              ),
            ],
          ),
          subtitle: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(portfolio.description),
              const SizedBox(height: 4),
              Text(
                'Created: ${_formatDate(portfolio.createdAt)}',
                style: TextStyle(fontSize: 12, color: Colors.grey[600]),
              ),
            ],
          ),
          onTap: () {
            setState(() {
              selectedPortfolio = portfolio;
            });
          },
          trailing: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              IconButton(
                icon: const Icon(Icons.edit),
                tooltip: 'Edit Portfolio #${portfolio.id}',
                onPressed: () => _showEditPortfolioDialog(portfolio),
              ),
              IconButton(
                icon: const Icon(Icons.delete),
                tooltip: 'Delete Portfolio #${portfolio.id}',
                onPressed: () => _deletePortfolio(portfolio),
              ),
            ],
          ),
        );
      },
    );
  }

  String _formatDate(DateTime date) {
    return '${date.year}-${date.month.toString().padLeft(2, '0')}-${date.day.toString().padLeft(2, '0')}';
  }

  Widget _buildRightPanel() {
    if (selectedPortfolio == null) {
      return const Card(
        margin: EdgeInsets.all(8.0),
        child: Center(
          child: Text('Select a portfolio to view details'),
        ),
      );
    }

    return Column(
      children: [
        Card(
          margin: const EdgeInsets.all(8.0),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.all(8.0),
                child: Text(
                  'Portfolio #${selectedPortfolio!.id}: ${selectedPortfolio!.name}',
                  style: const TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
              NavigationBar(
                selectedIndex: _selectedPanelIndex,
                onDestinationSelected: (int index) {
                  setState(() {
                    _selectedPanelIndex = index;
                  });
                },
                destinations: const [
                  NavigationDestination(
                    icon: Icon(Icons.account_balance_wallet),
                    label: 'Holdings',
                  ),
                  NavigationDestination(
                    icon: Icon(Icons.show_chart),
                    label: 'Performance',
                  ),
                  NavigationDestination(
                    icon: Icon(Icons.receipt_long),
                    label: 'Transactions',
                  ),
                  NavigationDestination(
                    icon: Icon(Icons.build),
                    label: 'Actions',
                  ),
                ],
              ),
            ],
          ),
        ),
        Expanded(
          child: IndexedStack(
            index: _selectedPanelIndex,
            children: [
              HoldingsPanel(portfolio: selectedPortfolio!),
              PerformancePanel(portfolio: selectedPortfolio!),
              TransactionsPanel(portfolio: selectedPortfolio!),
              ActionsPanel(portfolio: selectedPortfolio!),
            ],
          ),
        ),
      ],
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Portfolio Management'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: fetchPortfolios,
          ),
        ],
      ),
      body: SplitView(
        viewMode: SplitViewMode.Horizontal,
        indicator: const SplitIndicator(viewMode: SplitViewMode.Horizontal),
        activeIndicator: const SplitIndicator(
          viewMode: SplitViewMode.Horizontal,
          isActive: true,
        ),
        controller: SplitViewController(weights: [0.3, 0.7]),
        children: [
          // Left panel - Portfolio List
          Card(
            margin: const EdgeInsets.all(8.0),
            child: Column(
              children: [
                Padding(
                  padding: const EdgeInsets.all(8.0),
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        'Portfolios',
                        style: TextStyle(
                          fontSize: 20,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      IconButton(
                        icon: const Icon(Icons.add),
                        onPressed: _showAddPortfolioDialog,
                      ),
                    ],
                  ),
                ),
                Expanded(child: _buildPortfolioList()),
              ],
            ),
          ),
          // Right panel - Now using the new _buildRightPanel method
          _buildRightPanel(),
        ],
      ),
    );
  }
} 