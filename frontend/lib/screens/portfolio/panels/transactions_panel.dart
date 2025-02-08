import 'package:flutter/material.dart';
import '../../../models/portfolio.dart';
import '../../../models/transaction.dart';
import '../../../services/api_service.dart';  // Add back the ApiService import
import '../../../utils/formatters.dart';  // Add this for formatting functions
import 'package:intl/intl.dart';
import 'dart:convert';

extension StringExtension on String {
  String capitalize() {
    if (isEmpty) return this;
    return "${this[0].toUpperCase()}${substring(1).toLowerCase()}";
  }
}

class TransactionsPanel extends StatefulWidget {
  final Portfolio portfolio;

  const TransactionsPanel({
    Key? key,
    required this.portfolio,
  }) : super(key: key);

  @override
  State<TransactionsPanel> createState() => _TransactionsPanelState();
}

class _TransactionsPanelState extends State<TransactionsPanel> {
  final ApiService _apiService = ApiService();
  List<Transaction> transactions = [];
  bool isLoading = true;
  String? error;
  int? currentPortfolioId;  // Add this to track portfolio changes

  @override
  void initState() {
    super.initState();
    currentPortfolioId = widget.portfolio.id;
    fetchTransactions();
  }

  @override
  void didUpdateWidget(TransactionsPanel oldWidget) {
    super.didUpdateWidget(oldWidget);
    // Check if portfolio ID changed
    if (oldWidget.portfolio.id != widget.portfolio.id) {
      print('Portfolio changed from ${oldWidget.portfolio.id} to ${widget.portfolio.id}');
      currentPortfolioId = widget.portfolio.id;
      fetchTransactions();  // Refresh data when portfolio changes
    }
  }

  Future<void> fetchTransactions() async {
    try {
      setState(() {
        isLoading = true;
        error = null;
      });

      print('Fetching transactions for portfolio ${widget.portfolio.id}');
      final response = await _apiService.get('portfolios/${widget.portfolio.id}/transactions');
      print('Raw response: $response');

      if (response == null) {
        print('Response is null for portfolio ${widget.portfolio.id}');
        if (currentPortfolioId == widget.portfolio.id) {
          setState(() {
            transactions = [];
            isLoading = false;
          });
        }
        return;
      }

      if (response is! List) {
        print('Response is not a list: ${response.runtimeType}');
        throw Exception('Invalid response format');
      }

      final List<Transaction> fetchedTransactions = response
          .map((data) {
            if (data is! Map<String, dynamic>) {
              throw Exception('Invalid transaction data format');
            }
            return Transaction.fromJson(data);
          })
          .toList();

      print('Processed ${fetchedTransactions.length} transactions for portfolio ${widget.portfolio.id}');

      // Only update state if this is still the current portfolio
      if (currentPortfolioId == widget.portfolio.id) {
        setState(() {
          transactions = fetchedTransactions;
          isLoading = false;
        });
        print('Updated transactions for portfolio ${widget.portfolio.id}');
      } else {
        print('Portfolio changed, discarding results');
      }
    } catch (e, stackTrace) {
      print('Error fetching transactions: $e');
      print('Stack trace: $stackTrace');
      if (currentPortfolioId == widget.portfolio.id) {
        setState(() {
          error = e.toString();
          isLoading = false;
        });
      }
    }
  }

  Future<void> _showCreateTransactionDialog() async {
    await showDialog(
      context: context,
      builder: (context) => TransactionForm(
        portfolioId: widget.portfolio.id,
        onTransactionCreated: () {
          print('Transaction created, refreshing transactions for portfolio ${widget.portfolio.id}');
          fetchTransactions();
        },
      ),
    );
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
              onPressed: fetchTransactions,
              child: const Text('Retry'),
            ),
          ],
        ),
      );
    }

    return Card(
      margin: const EdgeInsets.all(8.0),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(8.0),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  'Transactions',
                  style: TextStyle(
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                  ),
                ),
                ElevatedButton.icon(
                  onPressed: _showCreateTransactionDialog,
                  icon: const Icon(Icons.add),
                  label: const Text('New Transaction'),
                ),
              ],
            ),
          ),
          Expanded(
            child: transactions.isEmpty
                ? Center(
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        const Icon(
                          Icons.account_balance_wallet_outlined,
                          size: 64,
                          color: Colors.grey,
                        ),
                        const SizedBox(height: 16),
                        const Text(
                          'No transactions yet',
                          style: TextStyle(
                            fontSize: 18,
                            color: Colors.grey,
                          ),
                        ),
                        const SizedBox(height: 8),
                        const Text(
                          'Create your first transaction to get started',
                          style: TextStyle(color: Colors.grey),
                        ),
                        const SizedBox(height: 16),
                        ElevatedButton.icon(
                          onPressed: _showCreateTransactionDialog,
                          icon: const Icon(Icons.add),
                          label: const Text('Create Transaction'),
                        ),
                      ],
                    ),
                  )
                : SingleChildScrollView(
                    child: DataTable(
                      columns: const [
                        DataColumn(label: Text('Date')),
                        DataColumn(label: Text('Type')),
                        DataColumn(label: Text('Ticker')),
                        DataColumn(label: Text('Shares')),
                        DataColumn(label: Text('Price')),
                        DataColumn(label: Text('Amount')),
                        DataColumn(label: Text('Fee')),
                        DataColumn(label: Text('Balance')),
                        DataColumn(label: Text('Gain')),
                      ],
                      rows: transactions.map((Transaction tx) {
                        return DataRow(
                          cells: [
                            DataCell(Text(DateFormat('yyyy-MM-dd').format(tx.transactionAt))),
                            DataCell(Text(tx.type)),
                            DataCell(Text(tx.ticker ?? '')),
                            DataCell(Text(tx.isStockTransaction ? tx.formattedShares : '-')),
                            DataCell(Text(tx.isStockTransaction ? tx.formattedPrice : '-')),
                            DataCell(Text('\$${tx.formattedAmount}')),
                            DataCell(Text('\$${tx.formattedFee}')),
                            DataCell(Text('\$${tx.cashBalanceAfter.toStringAsFixed(2)}')),
                            DataCell(
                              tx.type == 'SELL' 
                                ? Text(
                                    '\$${tx.realizedGainAvg?.toStringAsFixed(2) ?? '0.00'}',
                                    style: TextStyle(
                                      color: (tx.realizedGainAvg ?? 0) >= 0 
                                        ? Colors.green 
                                        : Colors.red,
                                    ),
                                  )
                                : const Text('-'),
                            ),
                          ],
                        );
                      }).toList(),
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}

class TransactionForm extends StatefulWidget {
  final int portfolioId;
  final Function onTransactionCreated;

  const TransactionForm({
    Key? key,
    required this.portfolioId,
    required this.onTransactionCreated,
  }) : super(key: key);

  @override
  State<TransactionForm> createState() => _TransactionFormState();
}

class _TransactionFormState extends State<TransactionForm> {
  final ApiService _apiService = ApiService();
  bool isSubmitting = false;
  String? error;
  String? _type;
  double _amount = 0;
  String _ticker = '';
  double _price = 0;
  final _formKey = GlobalKey<FormState>();
  final _tickerController = TextEditingController();
  final _sharesController = TextEditingController();
  final _priceController = TextEditingController();
  final _amountController = TextEditingController();
  final _feeController = TextEditingController(text: '0.00');
  final _notesController = TextEditingController();
  DateTime _selectedDate = DateTime.now();
  final _amountFocus = FocusNode();
  final _feeFocus = FocusNode();
  final _notesFocus = FocusNode();
  DateTime _transactionDate = DateTime.now();  // Initialize with current date

  @override
  void dispose() {
    _amountFocus.dispose();
    _feeFocus.dispose();
    _notesFocus.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('Create ${_type?[0] ?? ''}${_type?.substring(1).toLowerCase() ?? ''} Transaction'),
      content: SingleChildScrollView(
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              // Transaction Type Selector
              DropdownButtonFormField<String>(
                value: _type,
                decoration: const InputDecoration(labelText: 'Type'),
                items: ['DEPOSIT', 'WITHDRAW', 'BUY', 'SELL', 'DIVIDEND']
                    .map((type) => DropdownMenuItem(
                          value: type,
                          child: Text('${type[0]}${type.substring(1).toLowerCase()}'),
                        ))
                    .toList(),
                onChanged: (value) {
                  setState(() {
                    _type = value;
                    _resetFields();
                  });
                },
              ),

              // Conditional Fields based on transaction type
              if (_type == 'BUY' || _type == 'SELL' || _type == 'DIVIDEND')
                TextFormField(
                  controller: _tickerController,
                  decoration: const InputDecoration(labelText: 'Ticker'),
                  textCapitalization: TextCapitalization.characters,
                  validator: (value) => 
                    value?.isEmpty ?? true ? 'Ticker is required' : null,
                ),

              if (_type == 'BUY' || _type == 'SELL') ...[
                TextFormField(
                  controller: _sharesController,
                  decoration: const InputDecoration(labelText: 'Shares'),
                  keyboardType: const TextInputType.numberWithOptions(decimal: true),
                  validator: _validatePositiveNumber,
                  onChanged: (value) => _updateAmount(),
                ),
                TextFormField(
                  controller: _priceController,
                  decoration: const InputDecoration(labelText: 'Price per Share'),
                  keyboardType: const TextInputType.numberWithOptions(decimal: true),
                  validator: _validatePositiveNumber,
                  onChanged: (value) => _updateAmount(),
                ),
              ],

              if (_type == 'DEPOSIT' || _type == 'WITHDRAW' || _type == 'DIVIDEND')
                TextFormField(
                  controller: _amountController,
                  focusNode: _amountFocus,
                  decoration: const InputDecoration(
                    labelText: 'Amount',
                    border: OutlineInputBorder(),
                    contentPadding: EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                  ),
                  autofocus: false,
                  keyboardType: const TextInputType.numberWithOptions(decimal: true),
                  validator: (value) {
                    if (value == null || value.isEmpty) return 'Required';
                    final amount = double.tryParse(value);
                    if (amount == null) return 'Invalid number';
                    if (amount <= 0) return 'Must be greater than 0';
                    if (amount > 1000000000) return 'Amount exceeds maximum allowed (1 billion)';
                    return null;
                  },
                ),

              TextFormField(
                controller: _feeController,
                focusNode: _feeFocus,
                decoration: const InputDecoration(labelText: 'Fee'),
                keyboardType: const TextInputType.numberWithOptions(decimal: true),
                validator: _validateNonNegativeNumber,
              ),

              TextFormField(
                controller: _notesController,
                focusNode: _notesFocus,
                decoration: const InputDecoration(labelText: 'Notes'),
                maxLines: 2,
              ),

              // Date Picker
              ListTile(
                title: const Text('Transaction Date'),
                subtitle: Text(
                  DateFormat('yyyy-MM-dd').format(_transactionDate),
                ),
                trailing: IconButton(
                  icon: const Icon(Icons.calendar_today),
                  onPressed: () => _selectDate(context),
                ),
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Cancel'),
        ),
        ElevatedButton(
          onPressed: _submitForm,
          child: const Text('Create'),
        ),
      ],
    );
  }

  // Helper methods...
  void _resetFields() {
    if (_type == 'DEPOSIT' || _type == 'WITHDRAW') {
      _tickerController.text = 'CASH';
      _priceController.text = '1.0';
      _sharesController.text = _amountController.text;
    } else {
      _tickerController.text = '';
      _priceController.text = '';
      _sharesController.text = '';
      _amountController.text = '';
    }
  }

  String? _validatePositiveNumber(String? value) {
    if (value == null || value.isEmpty) return 'Required';
    final number = double.tryParse(value);
    if (number == null) return 'Invalid number';
    if (number <= 0) return 'Must be greater than 0';
    return null;
  }

  String? _validateNonNegativeNumber(String? value) {
    if (value == null || value.isEmpty) return 'Required';
    final number = double.tryParse(value);
    if (number == null) return 'Invalid number';
    if (number < 0) return 'Cannot be negative';
    return null;
  }

  void _updateAmount() {
    if (_type == 'BUY' || _type == 'SELL') {
      final shares = double.tryParse(_sharesController.text) ?? 0;
      final price = double.tryParse(_priceController.text) ?? 0;
      _amountController.text = (shares * price).toStringAsFixed(2);
    }
  }

  Future<void> _selectDate(BuildContext context) async {
    final DateTime? picked = await showDatePicker(
      context: context,
      initialDate: _transactionDate,
      firstDate: DateTime(2000),
      lastDate: DateTime(2025),
    );
    if (picked != null && picked != _transactionDate) {
      setState(() {
        _transactionDate = picked;
      });
    }
  }

  Future<void> _submitForm() async {
    if (!_formKey.currentState!.validate()) return;

    try {
      final formattedDate = _selectedDate.toUtc().toIso8601String();
      
      // Specify Map<String, dynamic> to allow different value types
      final Map<String, dynamic> transaction = {
        'type': _type,
        'notes': _notesController.text,
        'transaction_at': formattedDate,
      };

      // Add type-specific fields
      if (_type == 'DEPOSIT' || _type == 'WITHDRAW') {
        final amount = double.parse(_amountController.text);
        transaction.addAll(<String, dynamic>{
          'ticker': 'CASH',
          'shares': amount,  // Now works with double
          'price': 1.0,     // Now works with double
          'amount': amount, // Now works with double
          'fee': 0.0,      // Now works with double
        });
      } else if (_type == 'BUY' || _type == 'SELL') {
        final shares = double.parse(_sharesController.text);
        final price = double.parse(_priceController.text);
        final fee = double.parse(_feeController.text);
        transaction.addAll(<String, dynamic>{
          'ticker': _tickerController.text.toUpperCase(),
          'shares': shares,  // Now works with double
          'price': price,   // Now works with double
          'amount': shares * price, // Now works with double
          'fee': fee,      // Now works with double
        });
      } else if (_type == 'DIVIDEND') {
        final amount = double.parse(_amountController.text);
        transaction.addAll(<String, dynamic>{
          'ticker': _tickerController.text.toUpperCase(),
          'amount': amount, // Now works with double
          'shares': 0.0,   // Use 0.0 instead of 0
          'price': 0.0,    // Use 0.0 instead of 0
          'fee': 0.0,      // Use 0.0 instead of 0
        });
      }

      print('Submitting transaction: ${json.encode(transaction)}');

      await _submitTransaction(transaction);
    } catch (e) {
      print('Error submitting form: $e');
      String errorMessage = 'Error creating transaction: ';
      
      if (e.toString().contains('amount exceeds maximum')) {
        errorMessage += 'Amount is too large (maximum 1 billion)';
      } else if (e.toString().contains('Failed to update portfolio holdings')) {
        errorMessage += 'Failed to update holdings. Please check if the ticker exists and try again.';
      } else if (e.toString().contains('Network error')) {
        errorMessage += 'Could not connect to server. Please check your connection.';
      } else if (e.toString().contains('timed out')) {
        errorMessage += 'Request timed out. Please try again.';
      } else {
        errorMessage += e.toString();
      }

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(errorMessage),
          backgroundColor: Colors.red,
          duration: const Duration(seconds: 5),
          action: SnackBarAction(
            label: 'DISMISS',
            onPressed: () {
              ScaffoldMessenger.of(context).hideCurrentSnackBar();
            },
          ),
        ),
      );
    }
  }

  Future<void> _submitTransaction(Map<String, dynamic> formData) async {
    try {
      setState(() {
        isSubmitting = true;
        error = null;
      });

      // Validate amount is not too large
      final amount = double.parse(formData['amount'].toString());
      if (amount > 1000000000) {
        throw Exception('Amount exceeds maximum allowed (1 billion)');
      }

      // Format transaction data
      final transaction = <String, dynamic>{
        'type': formData['type'].toString().toUpperCase(),
        'notes': formData['notes']?.toString() ?? '',
        'transaction_at': _transactionDate.toUtc().toIso8601String(),
        'amount': amount,
        'ticker': formData['ticker']?.toString()?.toUpperCase() ?? 'CASH',
        'shares': double.parse(formData['shares']?.toString() ?? '0'),
        'price': double.parse(formData['price']?.toString() ?? '0'),
        'fee': double.parse(formData['fee']?.toString() ?? '0'),
      };

      print('Submitting transaction: ${json.encode(transaction)}');

      final response = await _apiService.createTransaction(
        widget.portfolioId,
        transaction,
      );

      if (mounted) {
        Navigator.of(context).pop();
        widget.onTransactionCreated();
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('${transaction['type']} transaction created successfully'),
            backgroundColor: Colors.green,
          ),
        );
      }
    } catch (e) {
      print('Error submitting transaction: $e');
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Error: $e'),
            backgroundColor: Colors.red,
            duration: const Duration(seconds: 5),
          ),
        );
      }
    } finally {
      setState(() {
        isSubmitting = false;
      });
    }
  }

  bool _validateForm() {
    if (_type == null) {
      _setError('Please select a transaction type');
      return false;
    }
    // ... rest of validation
    return true;
  }

  void _setError(String message) {
    setState(() {
      error = message;
    });
  }
}

class TransactionListItem extends StatelessWidget {
  final Transaction transaction;

  const TransactionListItem({Key? key, required this.transaction}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        title: Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text('${transaction.type} ${transaction.ticker ?? ''}'),
            Text(formatCurrency(transaction.amount)),
          ],
        ),
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text(formatDate(transaction.transactionAt)),
                if (transaction.shares != null)
                  Text('${formatNumber(transaction.shares!)} @ ${formatCurrency(transaction.price!)}'),
              ],
            ),
            if (transaction.type == 'SELL' && 
                (transaction.realizedGainAvg != 0 || transaction.realizedGainFifo != 0))
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  const Text('Realized Gain (Avg):'),
                  Text(
                    formatCurrency(transaction.realizedGainAvg),
                    style: TextStyle(
                      color: transaction.realizedGainAvg >= 0 
                          ? Colors.green 
                          : Colors.red,
                    ),
                  ),
                ],
              ),
          ],
        ),
      ),
    );
  }
}
