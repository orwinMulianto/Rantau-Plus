import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import 'app/rantau_app.dart';
import 'core/config/app_config.dart';
import 'core/network/rantau_api.dart';
import 'features/dashboard/data/finance_summary_api.dart';

void main() {
  final client = http.Client();
  final baseUrl = AppConfig.apiBaseUrl;
  runApp(
    RantauApp(
      summaryApi: FinanceSummaryApi(client: client, baseUrl: baseUrl),
      api: RantauApi(client: client, baseUrl: baseUrl),
    ),
  );
}
