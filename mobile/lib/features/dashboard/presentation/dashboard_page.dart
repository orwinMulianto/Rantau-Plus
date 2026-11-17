import 'package:flutter/material.dart';

import '../data/finance_summary.dart';

class DashboardPage extends StatefulWidget {
  const DashboardPage({super.key, required this.loadSummary});

  final Future<FinanceSummary?> Function() loadSummary;

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
                    message:
                        'Ringkasan demo gagal dimuat. Periksa koneksi ke backend.',
                    onRetry: _reload,
                  );
                }
                final summary = snapshot.data;
                if (summary == null) {
                  return _StatusMessage(
                    message: 'Ringkasan demo belum tersedia.',
                    onRetry: _reload,
                  );
                }
                return _SummaryContent(summary: summary);
              },
            ),
          ),
        ),
      ),
    );
  }
}

class _StatusMessage extends StatelessWidget {
  const _StatusMessage({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(message, textAlign: TextAlign.center),
          const SizedBox(height: 12),
          FilledButton(onPressed: onRetry, child: const Text('Coba lagi')),
        ],
      ),
    );
  }
}

class _SummaryContent extends StatelessWidget {
  const _SummaryContent({required this.summary});

  final FinanceSummary summary;

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
      padding: const EdgeInsets.fromLTRB(20, 16, 20, 32),
      children: [
        Text('Ringkasan keuangan', style: theme.textTheme.headlineMedium),
        const SizedBox(height: 6),
        Text(
          'Data demo · ${_displayDate(summary.asOfDate)}',
          style: theme.textTheme.bodyMedium?.copyWith(
            color: scheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 20),
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
                  _rupiah(summary.availableFunds),
                  style: theme.textTheme.headlineMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                    color: scheme.onPrimaryContainer,
                  ),
                ),
                const SizedBox(height: 12),
                Text('Status keuangan · $risk'),
                Text(
                  'Pemasukan berikutnya ${_displayDate(summary.nextIncomeDate)}'
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
                    value: '${_rupiah(summary.safeToSpendDaily)}/hari',
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
        _AmountRow(label: 'Kewajiban belum dibayar', value: summary.unpaidObligations),
        _AmountRow(label: 'Dana cadangan', value: summary.reserve),
        _AmountRow(label: 'Rata-rata pengeluaran harian', value: summary.averageDailySpending),
        _AmountRow(label: 'Proyeksi saldo akhir', value: summary.projectedBalance),
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
          Flexible(child: Text(_rupiah(value), textAlign: TextAlign.end)),
        ],
      ),
    );
  }
}

String _rupiah(int amount) {
  final digits = amount.toString().replaceFirst('-', '');
  final grouped = digits.replaceAllMapped(
    RegExp(r'\B(?=(\d{3})+(?!\d))'),
    (_) => '.',
  );
  return '${amount < 0 ? '-' : ''}Rp$grouped';
}

String _displayDate(String date) => date.split('-').reversed.join('/');
