import 'dart:convert';

import 'package:http/http.dart' as http;

import 'finance_summary.dart';

class FinanceSummaryApi {
  const FinanceSummaryApi({required this.client, required this.baseUrl});

  final http.Client client;
  final Uri baseUrl;

  Future<FinanceSummary?> fetch() async {
    final endpoint = baseUrl.resolve('/api/v1/demo/finance/summary');
    final response = await client
        .get(endpoint, headers: {'Accept': 'application/json'})
        .timeout(const Duration(seconds: 8));

    if (response.statusCode == 204) return null;
    if (response.statusCode != 200) {
      throw http.ClientException('Ringkasan demo gagal dimuat.', endpoint);
    }

    final payload = jsonDecode(utf8.decode(response.bodyBytes))
        as Map<String, dynamic>;
    final data = payload['data'] as Map<String, dynamic>?;
    return data == null ? null : FinanceSummary.fromJson(data);
  }
}
