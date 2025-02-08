class TransactionForm extends StatefulWidget {
  final int portfolioId;
  final Function onTransactionCreated;

  const TransactionForm({
    Key? key, 
    required this.portfolioId,
    required this.onTransactionCreated,
  }) : super(key: key);

  @override
  _TransactionFormState createState() => _TransactionFormState();
}

class _TransactionFormState extends State<TransactionForm> {
  final _formKey = GlobalKey<FormState>();
  String _type = 'BUY';
  final _tickerController = TextEditingController();
  final _sharesController = TextEditingController();
  final _priceController = TextEditingController();
  final _amountController = TextEditingController();
  final _feeController = TextEditingController();
  final _notesController = TextEditingController();
  DateTime _transactionDate = DateTime.now();
  bool _isLoading = false;

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('Create Transaction'),
      content: SingleChildScrollView(
        child: Form(
          key: _formKey,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Focus(
                onFocusChange: (hasFocus) {
                  if (!hasFocus) {
                    // Validate on focus loss
                    _formKey.currentState?.validate();
                  }
                },
                child: DropdownButtonFormField<String>(
                  value: _type,
                  items: ['BUY', 'SELL', 'DEPOSIT', 'WITHDRAW']
                      .map((type) => DropdownMenuItem(
                            value: type,
                            child: Text(type),
                          ))
                      .toList(),
                  onChanged: (value) {
                    setState(() {
                      _type = value!;
                      // Clear fields based on type
                      if (_type == 'DEPOSIT' || _type == 'WITHDRAW') {
                        _tickerController.clear();
                        _sharesController.clear();
                        _priceController.clear();
                      }
                    });
                  },
                ),
              ),
              if (_type == 'BUY' || _type == 'SELL') ...[
                TextFormField(
                  controller: _tickerController,
                  decoration: InputDecoration(labelText: 'Ticker'),
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter a ticker';
                    }
                    return null;
                  },
                ),
                TextFormField(
                  controller: _sharesController,
                  decoration: InputDecoration(labelText: 'Shares'),
                  keyboardType: TextInputType.number,
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter number of shares';
                    }
                    return null;
                  },
                ),
                TextFormField(
                  controller: _priceController,
                  decoration: InputDecoration(labelText: 'Price'),
                  keyboardType: TextInputType.number,
                  validator: (value) {
                    if (value == null || value.isEmpty) {
                      return 'Please enter price';
                    }
                    return null;
                  },
                ),
              ],
              if (_type == 'DEPOSIT' || _type == 'WITHDRAW')
                Material(
                  child: TextFormField(
                    controller: _amountController,
                    decoration: InputDecoration(
                      labelText: 'Amount',
                      border: OutlineInputBorder(),
                    ),
                    keyboardType: TextInputType.numberWithOptions(decimal: true),
                  ),
                ),
              TextFormField(
                controller: _feeController,
                decoration: InputDecoration(labelText: 'Fee'),
                keyboardType: TextInputType.number,
              ),
              TextFormField(
                controller: _notesController,
                decoration: InputDecoration(labelText: 'Notes'),
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: Text('Cancel'),
        ),
        ElevatedButton(
          onPressed: _isLoading ? null : _submitForm,
          child: _isLoading 
              ? SizedBox(
                  width: 20,
                  height: 20,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : Text('Submit'),
        ),
      ],
    );
  }

  void _submitForm() async {
    if (_formKey.currentState!.validate()) {
      setState(() {
        _isLoading = true;
      });

      try {
        final transaction = {
          'type': _type,
          'ticker': _type == 'BUY' || _type == 'SELL' 
              ? _tickerController.text 
              : null,
          'shares': _type == 'BUY' || _type == 'SELL'
              ? double.parse(_sharesController.text)
              : null,
          'price': _type == 'BUY' || _type == 'SELL'
              ? double.parse(_priceController.text)
              : null,
          'amount': _type == 'DEPOSIT' || _type == 'WITHDRAW'
              ? double.parse(_amountController.text)
              : _type == 'BUY' || _type == 'SELL'
                  ? double.parse(_sharesController.text) * 
                    double.parse(_priceController.text)
                  : 0,
          'fee': _feeController.text.isNotEmpty 
              ? double.parse(_feeController.text) 
              : 0,
          'notes': _notesController.text,
          'transaction_at': _transactionDate.toIso8601String(),
        };

        await ApiService().createTransaction(
          widget.portfolioId,
          transaction,
        );

        widget.onTransactionCreated();
        Navigator.of(context).pop();
      } catch (e) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Error: $e')),
        );
      } finally {
        setState(() {
          _isLoading = false;
        });
      }
    }
  }
} 