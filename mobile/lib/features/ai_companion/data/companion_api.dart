import '../../../core/format/finance_format.dart';
import '../../../core/network/rantau_api.dart';

class CompanionReply {
  const CompanionReply({
    required this.mode,
    required this.message,
    required this.details,
    required this.actions,
    required this.sources,
  });

  final String mode;
  final String message;
  final List<({String label, String value})> details;
  final List<String> actions;
  final List<String> sources;

  static Future<CompanionReply> ask(RantauApi api, String message) async {
    final response = await api.request(
      'POST',
      '/api/v1/assistant/chat',
      private: true,
      body: {'message': message},
    );
    final data = RantauApi.data(response);
    final details = <({String label, String value})>[];
    if (data['financial'] case {
      'available_funds': final int available,
      'safe_to_spend_daily': final int dailyLimit,
      'remaining_days': final int days,
      'runway_days': final num? runway,
      'projected_balance': final int projected,
    }) {
      details.addAll([
        (label: 'Safe-to-Spend', value: '${formatRupiah(dailyLimit)}/hari'),
        (label: 'Dana tersedia', value: formatRupiah(available)),
        (label: 'Sisa waktu hingga pemasukan', value: '$days hari'),
        if (runway != null && runway > 0)
          (
            label: 'Financial Runway',
            value: '${runway.toStringAsFixed(1)} hari',
          ),
        (label: 'Proyeksi saldo', value: formatRupiah(projected)),
      ]);
    }
    if (data['simulation'] case {
      'amount': final int amount,
      'available_after': final int available,
      'daily_limit_after': final int dailyLimit,
      'projected_after': final int projected,
    }) {
      details.addAll([
        (label: 'Nominal simulasi', value: formatRupiah(amount)),
        (
          label: 'Dana tersedia setelah belanja',
          value: formatRupiah(available),
        ),
        (label: 'Batas harian baru', value: '${formatRupiah(dailyLimit)}/hari'),
        (
          label: 'Proyeksi saldo setelah belanja',
          value: formatRupiah(projected),
        ),
      ]);
    }
    return CompanionReply(
      mode: data['mode'] as String,
      message: data['answer'] as String,
      details: details,
      actions: (data['actions'] as List?)?.whereType<String>().toList() ?? [],
      sources: [
        for (final source in data['sources'] as List? ?? [])
          if (source case {'title': final String title}) title,
      ],
    );
  }
}
