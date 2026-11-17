class FinanceSummary {
  const FinanceSummary({
    required this.currency,
    required this.asOfDate,
    required this.nextIncomeDate,
    required this.remainingDays,
    required this.availableFunds,
    required this.safeToSpendDaily,
    required this.runwayDays,
    required this.averageDailySpending,
    required this.projectedBalance,
    required this.unpaidObligations,
    required this.reserve,
    required this.riskStatus,
  });

  factory FinanceSummary.fromJson(Map<String, dynamic> json) {
    final period = json['period'] as Map<String, dynamic>;
    return FinanceSummary(
      currency: json['currency'] as String,
      asOfDate: period['as_of_date'] as String,
      nextIncomeDate: period['next_income_date'] as String,
      remainingDays: period['remaining_days'] as int,
      availableFunds: json['available_funds'] as int,
      safeToSpendDaily: json['safe_to_spend_daily'] as int,
      runwayDays: (json['runway_days'] as num?)?.toDouble(),
      averageDailySpending: json['average_daily_spending'] as int,
      projectedBalance: json['projected_balance'] as int,
      unpaidObligations: json['unpaid_obligations'] as int,
      reserve: json['reserve'] as int,
      riskStatus: json['risk_status'] as String,
    );
  }

  final String currency;
  final String asOfDate;
  final String nextIncomeDate;
  final int remainingDays;
  final int availableFunds;
  final int safeToSpendDaily;
  final double? runwayDays;
  final int averageDailySpending;
  final int projectedBalance;
  final int unpaidObligations;
  final int reserve;
  final String riskStatus;
}
