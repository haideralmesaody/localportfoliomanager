class Transaction {
  final int id;
  final int portfolioId;
  final String type;
  final String? ticker;
  final double? shares;
  final double? price;
  final double amount;
  final double fee;
  final String? notes;
  final DateTime transactionAt;
  final DateTime createdAt;
  final double cashBalanceBefore;
  final double cashBalanceAfter;
  final double sharesCountBefore;
  final double sharesCountAfter;
  final double averageCostBefore;
  final double averageCostAfter;
  final double realizedGainAvg;
  final double realizedGainFifo;

  // Add helper getters
  bool get isCashTransaction => type == 'DEPOSIT' || type == 'WITHDRAW';
  bool get isStockTransaction => type == 'BUY' || type == 'SELL';
  bool get isDividend => type == 'DIVIDEND';
  
  // Add formatted getters
  String get formattedAmount => amount.toStringAsFixed(2);
  String get formattedShares => shares?.toStringAsFixed(6) ?? '0.000000';
  String get formattedPrice => price?.toStringAsFixed(3) ?? '0.000';
  String get formattedFee => fee.toStringAsFixed(2);

  Transaction({
    required this.id,
    required this.portfolioId,
    required this.type,
    this.ticker,
    this.shares,
    this.price,
    required this.amount,
    required this.fee,
    this.notes,
    required this.transactionAt,
    required this.createdAt,
    required this.cashBalanceBefore,
    required this.cashBalanceAfter,
    required this.sharesCountBefore,
    required this.sharesCountAfter,
    required this.averageCostBefore,
    required this.averageCostAfter,
    required this.realizedGainAvg,
    required this.realizedGainFifo,
  });

  factory Transaction.fromJson(Map<String, dynamic> json) {
    return Transaction(
      id: json['id'],
      portfolioId: json['portfolio_id'],
      type: json['type'],
      ticker: json['ticker'],
      shares: json['shares']?.toDouble(),
      price: json['price']?.toDouble(),
      amount: json['amount'].toDouble(),
      fee: json['fee'].toDouble(),
      notes: json['notes'],
      transactionAt: DateTime.parse(json['transaction_at']),
      createdAt: DateTime.parse(json['created_at']),
      cashBalanceBefore: json['cash_balance_before'].toDouble(),
      cashBalanceAfter: json['cash_balance_after'].toDouble(),
      sharesCountBefore: json['shares_count_before'].toDouble(),
      sharesCountAfter: json['shares_count_after'].toDouble(),
      averageCostBefore: json['average_cost_before'].toDouble(),
      averageCostAfter: json['average_cost_after'].toDouble(),
      realizedGainAvg: json['realized_gain_avg']?.toDouble() ?? 0,
      realizedGainFifo: json['realized_gain_fifo']?.toDouble() ?? 0,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'portfolio_id': portfolioId,
      'type': type,
      'ticker': ticker,
      'shares': shares,
      'price': price,
      'amount': amount,
      'fee': fee,
      'notes': notes,
      'transaction_at': transactionAt.toIso8601String(),
      'created_at': createdAt.toIso8601String(),
      'cash_balance_before': cashBalanceBefore,
      'cash_balance_after': cashBalanceAfter,
      'shares_count_before': sharesCountBefore,
      'shares_count_after': sharesCountAfter,
      'average_cost_before': averageCostBefore,
      'average_cost_after': averageCostAfter,
      'realized_gain_avg': realizedGainAvg,
      'realized_gain_fifo': realizedGainFifo,
    };
  }
} 