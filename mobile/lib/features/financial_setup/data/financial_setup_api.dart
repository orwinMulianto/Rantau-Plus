import '../../../core/network/rantau_api.dart';
import '../../dashboard/data/finance_summary.dart';

class SetupObligation {
  const SetupObligation({
    this.id,
    required this.name,
    required this.amount,
    this.dueDate,
  });

  factory SetupObligation.fromJson(Map<String, dynamic> json) {
    return SetupObligation(
      id: json['id'] as String?,
      name: json['name'] as String,
      amount: json['amount'] as int,
      dueDate: json['due_date'] as String?,
    );
  }

  final String? id;
  final String name;
  final int amount;
  final String? dueDate;

  Map<String, Object?> toJson() => {
    if (id != null) 'id': id,
    'name': name,
    'amount': amount,
    if (dueDate != null) 'due_date': dueDate,
  };
}

class SetupDraft {
  const SetupDraft({
    required this.openingBalance,
    required this.nextIncomeDate,
    required this.reserveAmount,
    required this.obligations,
  });

  factory SetupDraft.fromJson(Map<String, dynamic> json) {
    final items = json['obligations'] as List<dynamic>;
    return SetupDraft(
      openingBalance: json['opening_balance'] as int,
      nextIncomeDate: json['next_income_date'] as String,
      reserveAmount: json['reserve_amount'] as int,
      obligations: items
          .map((item) => SetupObligation.fromJson(item as Map<String, dynamic>))
          .toList(),
    );
  }

  final int openingBalance;
  final String nextIncomeDate;
  final int reserveAmount;
  final List<SetupObligation> obligations;

  Map<String, Object?> toJson() => {
    'opening_balance': openingBalance,
    'next_income_date': nextIncomeDate,
    'reserve_amount': reserveAmount,
    'obligations': obligations.map((item) => item.toJson()).toList(),
  };
}

class FinancialSetupApi {
  const FinancialSetupApi(this.api);

  final RantauApi api;
  static const _setupPath = '/api/v1/finance/setup';

  Future<({DateTime today, SetupDraft? setup})> load() async {
    final response = await api.request('GET', _setupPath, private: true);
    final payload = RantauApi.data(response);
    final setup = payload['setup'] as Map<String, dynamic>?;
    return (
      today: DateTime.parse(payload['today'] as String),
      setup: setup == null ? null : SetupDraft.fromJson(setup),
    );
  }

  Future<FinanceSummary> preview(SetupDraft draft) async {
    final response = await api.request(
      'POST',
      '$_setupPath/preview',
      private: true,
      body: draft.toJson(),
    );
    return FinanceSummary.fromJson(RantauApi.data(response));
  }

  Future<void> save(SetupDraft draft) async {
    final response = await api.request(
      'PUT',
      _setupPath,
      private: true,
      body: draft.toJson(),
    );
    RantauApi.check(response);
  }

  Future<FinanceSummary?> summary() async {
    final response = await api.request(
      'GET',
      '/api/v1/finance/summary',
      private: true,
    );
    return response.statusCode == 204
        ? null
        : FinanceSummary.fromJson(RantauApi.data(response));
  }
}
