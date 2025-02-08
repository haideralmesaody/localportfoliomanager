class Stock {
  final String ticker;
  final String date;
  final double openPrice;
  final double highPrice;
  final double lowPrice;
  final double closePrice;
  final double lastPrice;
  final int sharesTraded;
  final double valueTraded;
  final int numTrades;
  final double change;
  final double changePercentage;
  final List<double> sparklinePrices;
  final List<String> sparklineDates;

  Stock({
    required this.ticker,
    required this.date,
    required this.openPrice,
    required this.highPrice,
    required this.lowPrice,
    required this.closePrice,
    required this.lastPrice,
    required this.sharesTraded,
    required this.valueTraded,
    required this.numTrades,
    required this.change,
    required this.changePercentage,
    required this.sparklinePrices,
    required this.sparklineDates,
  });

  factory Stock.fromJson(Map<String, dynamic> json) {
      return Stock(
        ticker: json['ticker'] as String,
        date: json['date'] as String,
        openPrice: (json['open_price'] as num).toDouble(),
        highPrice: (json['high_price'] as num).toDouble(),
        lowPrice: (json['low_price'] as num).toDouble(),
        closePrice: (json['close_price'] as num).toDouble(),
        lastPrice: (json['close_price'] as num).toDouble(), // Using close_price as last_price
        sharesTraded: json['shares_traded'] as int,         // Changed from qty_of_shares_traded
        valueTraded: (json['value_traded'] as num).toDouble(),
        numTrades: json['num_trades'] as int,
        change: (json['change'] as num).toDouble(),
        changePercentage: (json['change_percentage'] as num).toDouble(),
        sparklinePrices: [],  // Initialize empty, will be populated later
        sparklineDates: [],   // Initialize empty, will be populated later
      );
  }
}