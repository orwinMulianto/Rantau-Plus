import 'package:flutter/material.dart';

import '../core/network/rantau_api.dart';
import '../core/theme/app_theme.dart';
import '../features/ai_companion/presentation/companion_sheet.dart';
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

    Future<bool> signIn(BuildContext context) async {
      if (api.signedIn) return true;
      final result = await Navigator.of(context).push<bool>(
        MaterialPageRoute(builder: (context) => SetupSignInPage(api: api)),
      );
      return result == true;
    }

    Future<void> openSetup(BuildContext context) async {
      if (!await signIn(context) || !context.mounted) return;
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
    }

    return MaterialApp(
      title: 'Rantau+',
      debugShowCheckedModeBanner: false,
      theme: buildAppTheme(),
      home: DashboardPage(
        loadSummary: () =>
            api.signedIn ? setupApi.summary() : summaryApi.fetch(),
        isDemo: () => !api.signedIn,
        openSetup: openSetup,
        openCompanion: (context) => showModalBottomSheet<bool>(
          context: context,
          isScrollControlled: true,
          useSafeArea: true,
          showDragHandle: true,
          builder: (context) => CompanionSheet(api: api, signIn: signIn),
        ),
      ),
    );
  }
}
