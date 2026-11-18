import 'package:flutter/material.dart';

import '../core/network/rantau_api.dart';
import '../core/theme/app_theme.dart';
import '../features/dashboard/data/finance_summary_api.dart';
import '../features/dashboard/presentation/dashboard_page.dart';
import '../features/financial_setup/data/financial_setup_api.dart';
import '../features/financial_setup/presentation/financial_setup_page.dart';
import '../features/financial_setup/presentation/setup_sign_in_page.dart';

class RantauApp extends StatelessWidget {
  const RantauApp({super.key, required this.summaryApi, required this.api});

  final FinanceSummaryApi summaryApi;
  final RantauApi api;

  @override
  Widget build(BuildContext context) {
    final setupApi = FinancialSetupApi(api);
    return MaterialApp(
      title: 'Rantau+',
      debugShowCheckedModeBanner: false,
      theme: buildAppTheme(),
      home: DashboardPage(
        loadSummary: () =>
            api.signedIn ? setupApi.summary() : summaryApi.fetch(),
        isDemo: () => !api.signedIn,
        openSetup: (context) async {
          if (!api.signedIn) {
            final signedIn = await Navigator.of(context).push<bool>(
              MaterialPageRoute(
                builder: (context) => SetupSignInPage(api: api),
              ),
            );
            if (signedIn != true || !context.mounted) return;
          }
          final saved = await Navigator.of(context).push<bool>(
            MaterialPageRoute(
              builder: (context) => FinancialSetupPage(api: setupApi),
            ),
          );
          if (saved == true && context.mounted) {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(content: Text('Pengaturan keuangan disimpan.')),
            );
          }
        },
      ),
    );
  }
}
