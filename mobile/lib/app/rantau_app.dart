import 'package:flutter/material.dart';

import '../core/theme/app_theme.dart';
import '../features/dashboard/data/finance_summary_api.dart';
import '../features/dashboard/presentation/dashboard_page.dart';

class RantauApp extends StatelessWidget {
  const RantauApp({super.key, required this.summaryApi});

  final FinanceSummaryApi summaryApi;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Rantau+',
      debugShowCheckedModeBanner: false,
      theme: buildAppTheme(),
      home: DashboardPage(loadSummary: summaryApi.fetch),
    );
  }
}
