import 'package:flutter/material.dart';

import '../../../core/format/finance_format.dart';
import '../../../core/network/rantau_api.dart';
import '../data/finance_summary.dart';

class DashboardPage extends StatefulWidget {
  const DashboardPage({
    super.key,
    required this.loadSummary,
    required this.openSetup,
    required this.isDemo,
    required this.openCompanion,
  });

  final Future<FinanceSummary?> Function() loadSummary;
  final Future<void> Function(BuildContext) openSetup;
  final Future<bool?> Function(BuildContext) openCompanion;
  final bool Function() isDemo;

  @override
  State<DashboardPage> createState() => _DashboardPageState();
}

class _DashboardPageState extends State<DashboardPage> {
  late Future<FinanceSummary?> _summary;

  @override
  void initState() {
    super.initState();
    _summary = widget.loadSummary();
  }

  Future<void> _reload() async {
    setState(() => _summary = widget.loadSummary());
    try {
      await _summary;
    } catch (_) {
      // FutureBuilder displays the request error and retry action.
    }
  }

  Future<void> _openSetup() async {
    await widget.openSetup(context);
    if (mounted) await _reload();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Rantau+'),
        actions: [
          IconButton(
            tooltip: 'Muat ulang ringkasan',
            onPressed: _reload,
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      floatingActionButton: FloatingActionButton(
        tooltip: 'Buka AI Companion',
        onPressed: () async {
          final wasDemo = widget.isDemo();
          final needsSetup = await widget.openCompanion(context);
          if (!mounted) return;
          if (needsSetup == true) {
            await _openSetup();
          } else if (wasDemo != widget.isDemo()) {
            await _reload();
          }
        },
        child: const Icon(Icons.smart_toy_outlined),
      ),
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 680),
            child: FutureBuilder<FinanceSummary?>(
              future: _summary,
              builder: (context, snapshot) {
                if (snapshot.connectionState == ConnectionState.waiting) {
                  return const Center(child: CircularProgressIndicator());
                }
                if (snapshot.hasError) {
                  return _StatusMessage(
                    message: snapshot.error is ApiFailure
                        ? (snapshot.error as ApiFailure).message
                        : 'Ringkasan gagal dimuat. Periksa koneksi ke backend.',
                    onRetry: _reload,
                    onSetup: _openSetup,
                  );
                }
                final summary = snapshot.data;
                if (summary == null) {
                  return _StatusMessage(
                    message:
                        'Isi Financial Setup untuk melihat dana tersedia dan batas harian.',
                    onSetup: _openSetup,
                  );
                }
                return _SummaryContent(
                  summary: summary,
                  isDemo: widget.isDemo(),
                  onSetup: _openSetup,
                );
              },
            ),
          ),
        ),
      ),
    );
  }
}

class _StatusMessage extends StatelessWidget {
  const _StatusMessage({
    required this.message,
    this.onRetry,
    required this.onSetup,
  });

  final String message;
  final VoidCallback? onRetry;
  final VoidCallback onSetup;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(message, textAlign: TextAlign.center),
          const SizedBox(height: 12),
          FilledButton(
            onPressed: onSetup,
            child: const Text('Financial Setup'),
          ),
          if (onRetry != null)
            TextButton(onPressed: onRetry, child: const Text('Coba lagi')),
        ],
      ),
    );
  }
}

class _SummaryContent extends StatelessWidget {
  const _SummaryContent({
    required this.summary,
    required this.isDemo,
    required this.onSetup,
  });

  final FinanceSummary summary;
  final bool isDemo;
  final VoidCallback onSetup;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final risk = switch (summary.riskStatus) {
      'AMAN' => 'Aman',
      'WASPADA' => 'Waspada',
      'BERISIKO' => 'Berisiko',
      _ => 'Belum cukup data',
    };

    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 88),
      children: [
        Text('Ringkasan keuangan', style: theme.textTheme.headlineMedium),
        const SizedBox(height: 6),
        Text(
          '${isDemo ? 'Data demo · ' : ''}${displayDate(summary.asOfDate)}',
          style: theme.textTheme.bodyMedium?.copyWith(
            color: scheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 8),
        Align(
          alignment: Alignment.centerLeft,
          child: TextButton.icon(
            onPressed: onSetup,
            icon: const Icon(Icons.tune),
            label: const Text('Financial Setup'),
          ),
        ),
        const SizedBox(height: 12),
        Card(
          margin: EdgeInsets.zero,
          color: scheme.primaryContainer,
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('Dana tersedia', style: theme.textTheme.titleMedium),
                const SizedBox(height: 6),
                Text(
                  formatRupiah(summary.availableFunds),
                  style: theme.textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: scheme.onPrimaryContainer,
                  ),
                ),
                const SizedBox(height: 12),
                Text('Status keuangan · $risk'),
                Text(
                  'Pemasukan berikutnya ${displayDate(summary.nextIncomeDate)}'
                  ' (${summary.remainingDays} hari lagi)',
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 12),
        LayoutBuilder(
          builder: (context, constraints) {
            final width = constraints.maxWidth;
            final metricWidth = width >= 400 ? (width - 12) / 2 : width;
            final runway = summary.runwayDays;
            return Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                SizedBox(
                  width: metricWidth,
                  child: _MetricCard(
                    title: 'Safe-to-Spend',
                    value: '${formatRupiah(summary.safeToSpendDaily)}/hari',
                  ),
                ),
                SizedBox(
                  width: metricWidth,
                  child: _MetricCard(
                    title: 'Financial Runway',
                    value: runway == null
                        ? 'Belum cukup data'
                        : '${runway.toStringAsFixed(2).replaceAll('.', ',')} hari',
                  ),
                ),
              ],
            );
          },
        ),
        const SizedBox(height: 24),
        Text('Rincian periode', style: theme.textTheme.titleLarge),
        const SizedBox(height: 6),
        _AmountRow(
          label: 'Kewajiban belum dibayar',
          value: summary.unpaidObligations,
        ),
        _AmountRow(label: 'Dana cadangan', value: summary.reserve),
        _AmountRow(
          label: 'Rata-rata pengeluaran harian',
          value: summary.averageDailySpending,
        ),
        _AmountRow(
          label: 'Proyeksi saldo akhir',
          value: summary.projectedBalance,
        ),
      ],
    );
  }
}

class _MetricCard extends StatelessWidget {
  const _MetricCard({required this.title, required this.value});

  final String title;
  final String value;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: EdgeInsets.zero,
      elevation: 0,
      color: theme.colorScheme.surfaceContainerLow,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(title),
            const SizedBox(height: 6),
            Text(value, style: theme.textTheme.titleLarge),
          ],
        ),
      ),
    );
  }
}

class _AmountRow extends StatelessWidget {
  const _AmountRow({required this.label, required this.value});

  final String label;
  final int value;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(child: Text(label)),
          const SizedBox(width: 12),
          Flexible(child: Text(formatRupiah(value), textAlign: TextAlign.end)),
        ],
      ),
    );
  }
}
