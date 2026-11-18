import 'package:flutter/material.dart';

import '../../../core/format/finance_format.dart';
import '../../../core/network/rantau_api.dart';
import '../../dashboard/data/finance_summary.dart';
import '../data/financial_setup_api.dart';

class FinancialSetupPage extends StatefulWidget {
  const FinancialSetupPage({super.key, required this.api});

  final FinancialSetupApi api;

  @override
  State<FinancialSetupPage> createState() => _FinancialSetupPageState();
}

class _FinancialSetupPageState extends State<FinancialSetupPage> {
  final _form = GlobalKey<FormState>();
  final _opening = TextEditingController();
  final _reserve = TextEditingController(text: '0');
  final _items = <_ObligationFields>[];
  DateTime? _today;
  DateTime? _income;
  SetupDraft? _previewedDraft;
  FinanceSummary? _preview;
  String? _error;
  bool _busy = false;
  int _revision = 0;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _error = null);
    try {
      final data = await widget.api.load();
      if (!mounted) return;
      final setup = data.setup;
      setState(() {
        _today = data.today;
        if (setup != null) {
          _opening.text = setup.openingBalance.toString();
          _reserve.text = setup.reserveAmount.toString();
          _income = DateTime.parse(setup.nextIncomeDate);
          _items.addAll(setup.obligations.map(_ObligationFields.new));
        }
      });
    } catch (error) {
      _showError(error);
    }
  }

  @override
  void dispose() {
    _opening.dispose();
    _reserve.dispose();
    for (final item in _items) {
      item.dispose();
    }
    super.dispose();
  }

  void _invalidatePreview() {
    _revision++;
    _preview = null;
    _previewedDraft = null;
    _error = null;
  }

  void _onChanged() => setState(_invalidatePreview);

  void _showError(Object error) {
    if (mounted) {
      setState(() {
        _error = error is ApiFailure
            ? error.message
            : 'Gagal terhubung. Coba lagi.';
      });
    }
  }

  Future<void> _pickIncome(FormFieldState<DateTime> field) async {
    final today = _today!;
    final tomorrow = DateTime(today.year, today.month, today.day + 1);
    final picked = await showDatePicker(
      context: context,
      initialDate: _income ?? tomorrow,
      currentDate: today,
      firstDate: tomorrow,
      lastDate: DateTime(9999, 12, 31),
      helpText: 'Pilih tanggal pemasukan',
      cancelText: 'Batal',
      confirmText: 'Pilih',
    );
    if (!mounted || picked == null) return;
    field.didChange(picked);
    setState(() {
      _income = picked;
      _invalidatePreview();
    });
  }

  Future<void> _pickDueDate(_ObligationFields item) async {
    final income = _income;
    if (income == null) return;
    final current = item.dueDate;
    final date = await showDatePicker(
      context: context,
      currentDate: _today,
      initialDate: current != null && !current.isAfter(income)
          ? current
          : income,
      firstDate: DateTime(1),
      lastDate: income,
      helpText: 'Pilih jatuh tempo',
      cancelText: 'Batal',
      confirmText: 'Pilih',
    );
    if (!mounted || date == null) return;
    setState(() {
      item.dueDate = date;
      _invalidatePreview();
    });
  }

  Future<void> _calculate() async {
    if (_busy) return;
    if (!_form.currentState!.validate()) {
      setState(() => _error = 'Periksa kolom yang ditandai, lalu coba lagi.');
      return;
    }
    final draft = SetupDraft(
      openingBalance: _amount(_opening.text)!,
      reserveAmount: _amount(_reserve.text)!,
      nextIncomeDate: dateKey(_income!),
      obligations: _items
          .map(
            (item) => SetupObligation(
              id: item.id,
              name: item.name.text.trim(),
              amount: _amount(item.amount.text)!,
              dueDate: item.dueDate == null ? null : dateKey(item.dueDate!),
            ),
          )
          .toList(),
    );
    final revision = _revision;
    FocusScope.of(context).unfocus();
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final preview = await widget.api.preview(draft);
      if (!mounted || revision != _revision) return;
      setState(() {
        _preview = preview;
        _previewedDraft = draft;
      });
    } catch (error) {
      _showError(error);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _save() async {
    final draft = _previewedDraft;
    if (_busy || draft == null) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.api.save(draft);
      if (mounted) Navigator.of(context).pop(true);
    } catch (error) {
      _showError(error);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final preview = _preview;
    return Scaffold(
      appBar: AppBar(title: const Text('Financial Setup')),
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 680),
            child: _today == null
                ? _error == null
                      ? const Center(child: CircularProgressIndicator())
                      : Padding(
                          padding: const EdgeInsets.all(24),
                          child: Column(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Text(_error!, textAlign: TextAlign.center),
                              const SizedBox(height: 12),
                              OutlinedButton(
                                onPressed: _load,
                                child: const Text('Coba lagi'),
                              ),
                            ],
                          ),
                        )
                : Form(
                    key: _form,
                    child: SingleChildScrollView(
                      padding: const EdgeInsets.fromLTRB(20, 12, 20, 32),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.stretch,
                        children: [
                          _SetupSection(
                            title: 'Saldo dan pemasukan',
                            children: [
                              _moneyField(_opening, 'Saldo awal'),
                              const SizedBox(height: 16),
                              FormField<DateTime>(
                                initialValue: _income,
                                validator: (value) => value == null
                                    ? 'Pilih tanggal pemasukan.'
                                    : null,
                                builder: (field) => _dateField(
                                  label: 'Tanggal pemasukan berikutnya',
                                  value: _income,
                                  placeholder: 'Pilih tanggal',
                                  error: field.errorText,
                                  onTap: _busy
                                      ? null
                                      : () => _pickIncome(field),
                                ),
                              ),
                              const SizedBox(height: 16),
                              _moneyField(_reserve, 'Dana cadangan'),
                            ],
                          ),
                          const SizedBox(height: 16),
                          _SetupSection(
                            title: 'Kewajiban utama',
                            children: [
                              if (_items.isEmpty)
                                const Text(
                                  'Belum ada kewajiban. Tambahkan tagihan yang perlu disisihkan.',
                                ),
                              for (final (index, item) in _items.indexed)
                                Padding(
                                  key: ObjectKey(item),
                                  padding: const EdgeInsets.only(bottom: 12),
                                  child: Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.stretch,
                                    children: [
                                      Row(
                                        children: [
                                          Expanded(
                                            child: Text(
                                              'Kewajiban ${index + 1}',
                                              style: theme.textTheme.titleSmall,
                                            ),
                                          ),
                                          IconButton(
                                            tooltip:
                                                'Hapus kewajiban ${index + 1}',
                                            onPressed: _busy
                                                ? null
                                                : () => setState(() {
                                                    _items.remove(item);
                                                    item.dispose();
                                                    _invalidatePreview();
                                                  }),
                                            icon: const Icon(Icons.close),
                                          ),
                                        ],
                                      ),
                                      TextFormField(
                                        controller: item.name,
                                        enabled: !_busy,
                                        textCapitalization:
                                            TextCapitalization.sentences,
                                        decoration: const InputDecoration(
                                          labelText: 'Nama kewajiban',
                                          hintText: 'Contoh: Uang kos',
                                          border: OutlineInputBorder(),
                                        ),
                                        onChanged: (_) => _onChanged(),
                                        validator: (value) {
                                          final name = (value ?? '').trim();
                                          return name.isEmpty ||
                                                  name.runes.length > 100
                                              ? 'Isi nama (maksimal 100 karakter).'
                                              : null;
                                        },
                                      ),
                                      const SizedBox(height: 12),
                                      _moneyField(
                                        item.amount,
                                        'Nominal kewajiban',
                                        positive: true,
                                      ),
                                      const SizedBox(height: 12),
                                      _dateField(
                                        label: 'Jatuh tempo (opsional)',
                                        value: item.dueDate,
                                        placeholder:
                                            'Mengikuti tanggal pemasukan',
                                        error:
                                            item.dueDate != null &&
                                                _income != null &&
                                                item.dueDate!.isAfter(_income!)
                                            ? 'Pilih tanggal pada atau sebelum pemasukan.'
                                            : null,
                                        onTap: _busy || _income == null
                                            ? null
                                            : () => _pickDueDate(item),
                                      ),
                                      if (item.dueDate != null)
                                        Align(
                                          alignment: Alignment.centerLeft,
                                          child: TextButton(
                                            onPressed: _busy
                                                ? null
                                                : () => setState(() {
                                                    item.dueDate = null;
                                                    _invalidatePreview();
                                                  }),
                                            child: const Text(
                                              'Ikuti tanggal pemasukan',
                                            ),
                                          ),
                                        ),
                                    ],
                                  ),
                                ),
                              Align(
                                alignment: Alignment.centerLeft,
                                child: TextButton.icon(
                                  onPressed: _busy || _items.length >= 50
                                      ? null
                                      : () => setState(() {
                                          _items.add(_ObligationFields());
                                          _invalidatePreview();
                                        }),
                                  icon: const Icon(Icons.add),
                                  label: const Text('Tambah kewajiban'),
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 16),
                          _SetupSection(
                            title: 'Pratinjau',
                            color: theme.colorScheme.primaryContainer,
                            children: [
                              if (preview == null)
                                const Text(
                                  'Lihat dana tersedia dan batas harian sebelum menyimpan.',
                                ),
                              if (preview != null) ...[
                                _metric(
                                  'Dana tersedia',
                                  formatRupiah(preview.availableFunds),
                                ),
                                const SizedBox(height: 12),
                                _metric(
                                  'Batas harian',
                                  '${formatRupiah(preview.safeToSpendDaily)}/hari',
                                ),
                                const SizedBox(height: 8),
                                Text(
                                  '${preview.remainingDays} hari hingga pemasukan berikutnya.',
                                ),
                                if (preview.availableFunds < 0)
                                  Text(
                                    'Dana tersedia masih negatif.',
                                    style: TextStyle(
                                      color: theme.colorScheme.error,
                                    ),
                                  ),
                              ],
                            ],
                          ),
                          if (_error != null) ...[
                            const SizedBox(height: 12),
                            Text(
                              _error!,
                              style: TextStyle(color: theme.colorScheme.error),
                            ),
                          ],
                          const SizedBox(height: 20),
                          FilledButton(
                            onPressed: _busy
                                ? null
                                : preview == null
                                ? _calculate
                                : _save,
                            child: Text(
                              _busy
                                  ? 'Memproses…'
                                  : preview == null
                                  ? 'Lihat pratinjau'
                                  : 'Simpan pengaturan',
                            ),
                          ),
                        ],
                      ),
                    ),
                  ),
          ),
        ),
      ),
    );
  }

  Widget _moneyField(
    TextEditingController controller,
    String label, {
    bool positive = false,
  }) {
    return TextFormField(
      controller: controller,
      enabled: !_busy,
      keyboardType: TextInputType.number,
      decoration: InputDecoration(
        labelText: label,
        prefixText: 'Rp ',
        border: const OutlineInputBorder(),
      ),
      onChanged: (_) => _onChanged(),
      validator: (value) {
        final amount = _amount(value ?? '');
        return amount == null || (positive && amount == 0)
            ? positive
                  ? 'Masukkan nominal lebih dari Rp0.'
                  : 'Masukkan nominal rupiah yang valid.'
            : null;
      },
    );
  }

  Widget _dateField({
    required String label,
    required DateTime? value,
    required String placeholder,
    String? error,
    VoidCallback? onTap,
  }) {
    return InkWell(
      onTap: onTap,
      child: InputDecorator(
        decoration: InputDecoration(
          labelText: label,
          enabled: onTap != null,
          errorText: error,
          suffixIcon: const Icon(Icons.calendar_today_outlined),
          border: const OutlineInputBorder(),
        ),
        child: Text(value == null ? placeholder : displayDate(dateKey(value))),
      ),
    );
  }

  Widget _metric(String label, String value) => Column(
    crossAxisAlignment: CrossAxisAlignment.start,
    children: [
      Text(label),
      Text(
        value,
        style: Theme.of(
          context,
        ).textTheme.headlineSmall?.copyWith(fontWeight: FontWeight.w700),
      ),
    ],
  );
}

class _SetupSection extends StatelessWidget {
  const _SetupSection({
    required this.title,
    required this.children,
    this.color,
  });

  final String title;
  final List<Widget> children;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      margin: EdgeInsets.zero,
      color: color ?? theme.colorScheme.surfaceContainerLow,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(title, style: theme.textTheme.titleLarge),
            const SizedBox(height: 16),
            ...children,
          ],
        ),
      ),
    );
  }
}

class _ObligationFields {
  _ObligationFields([SetupObligation? obligation])
    : id = obligation?.id,
      name = TextEditingController(text: obligation?.name),
      amount = TextEditingController(text: obligation?.amount.toString()),
      dueDate = obligation?.dueDate == null
          ? null
          : DateTime.parse(obligation!.dueDate!);

  final String? id;
  final TextEditingController name;
  final TextEditingController amount;
  DateTime? dueDate;

  void dispose() {
    name.dispose();
    amount.dispose();
  }
}

int? _amount(String raw) {
  final text = raw.trim();
  if (!RegExp(r'^(\d+|\d{1,3}(\.\d{3})+)$').hasMatch(text)) return null;
  final amount = int.tryParse(text.replaceAll('.', ''));
  return amount != null && amount >= 0 ? amount : null;
}
