import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import 'app/rantau_app.dart';
import 'core/config/app_config.dart';
import 'features/dashboard/data/finance_summary_api.dart';

void main() {
  runApp(
    RantauApp(
      summaryApi: FinanceSummaryApi(
        client: http.Client(),
        baseUrl: AppConfig.apiBaseUrl,
      ),
    ),
  );
}
