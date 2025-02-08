class PortfolioHolding {
  final int id;
  final int portfolioId;
  final String ticker;
  final int shares;
  final double purchaseCostAverage;
  final double purchaseCostFifo;
  final double? currentPrice;
  final DateTime? priceLastDate;
  final double? positionCostAverage;
  final double? positionCostFifo;
  final double? unrealizedGainAverage;
  final double? unrealizedGainFifo;
  final double? targetPercentage;
  final double? currentPercentage;
  final double? adjustmentPercentage;
  final double? adjustmentValue;
  final int? adjustmentQuantity;
  final DateTime createdAt;
  final DateTime updatedAt;

  PortfolioHolding({
    required this.id,
    required this.portfolioId,
    required this.ticker,
    required this.shares,
    required this.purchaseCostAverage,
    required this.purchaseCostFifo,
    this.currentPrice,
    this.priceLastDate,
    this.positionCostAverage,
    this.positionCostFifo,
    this.unrealizedGainAverage,
    this.unrealizedGainFifo,
    this.targetPercentage,
    this.currentPercentage,
    this.adjustmentPercentage,
    this.adjustmentValue,
    this.adjustmentQuantity,
    required this.createdAt,
    required this.updatedAt,
  });

  factory PortfolioHolding.fromJson(Map<String, dynamic> json) {
    return PortfolioHolding(
      id: json['id'],
      portfolioId: json['portfolio_id'],
      ticker: json['ticker'],
      shares: json['shares'],
      purchaseCostAverage: json['purchase_cost_average'].toDouble(),
      purchaseCostFifo: json['purchase_cost_fifo'].toDouble(),
      currentPrice: json['current_price']?.toDouble(),
      priceLastDate: json['price_last_date'] != null 
          ? DateTime.parse(json['price_last_date']) 
          : null,
      positionCostAverage: json['position_cost_average']?.toDouble(),
      positionCostFifo: json['position_cost_fifo']?.toDouble(),
      unrealizedGainAverage: json['unrealized_gain_average']?.toDouble(),
      unrealizedGainFifo: json['unrealized_gain_fifo']?.toDouble(),
      targetPercentage: json['target_percentage']?.toDouble(),
      currentPercentage: json['current_percentage']?.toDouble(),
      adjustmentPercentage: json['adjustment_percentage']?.toDouble(),
      adjustmentValue: json['adjustment_value']?.toDouble(),
      adjustmentQuantity: json['adjustment_quantity'],
      createdAt: DateTime.parse(json['created_at']),
      updatedAt: DateTime.parse(json['updated_at']),
    );
  }
} 