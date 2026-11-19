import 'package:flutter/material.dart';

import '../../../core/network/rantau_api.dart';
import '../data/companion_api.dart';

class CompanionSheet extends StatefulWidget {
  const CompanionSheet({super.key, required this.api, required this.signIn});

  final RantauApi api;
  final Future<bool> Function(BuildContext) signIn;

  @override
  State<CompanionSheet> createState() => _CompanionSheetState();
}

class _CompanionSheetState extends State<CompanionSheet> {
  static const _examples = [
    'Berapa Safe-to-Spend saya?',
    'Kalau beli barang Rp300 ribu, uangku masih cukup?',
    'Apa itu Financial Runway?',
  ];
  final _input = TextEditingController();
  final _focus = FocusNode();
  final _scroll = ScrollController();
  final _turns = <({String question, CompanionReply? reply, String? error})>[];
  bool _busy = false;

  @override
  void dispose() {
    _input.dispose();
    _focus.dispose();
    _scroll.dispose();
    super.dispose();
  }

  void _scrollToLatest() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted && _scroll.hasClients) {
        _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 220),
          curve: Curves.easeOut,
        );
      }
    });
  }

  Future<void> _send(String input, {int? retryIndex}) async {
    final question = input.trim();
    if (question.isEmpty || _busy || !widget.api.signedIn) return;
    final index = retryIndex ?? _turns.length;
    final pending = (question: question, reply: null, error: null);
    if (retryIndex == null) _input.clear();
    _focus.unfocus();
    setState(() {
      _busy = true;
      if (retryIndex == null) {
        _turns.add(pending);
      } else {
        _turns[index] = pending;
      }
    });
    _scrollToLatest();
    try {
      final reply = await CompanionReply.ask(widget.api, question);
      if (mounted) {
        setState(
          () => _turns[index] = (question: question, reply: reply, error: null),
        );
      }
    } catch (error) {
      if (!mounted) return;
      setState(() {
        if (widget.api.signedIn) {
          _turns[index] = (
            question: question,
            reply: null,
            error: error is ApiFailure
                ? error.message
                : 'Gagal memuat jawaban.',
          );
        } else {
          _turns.clear();
          _input.clear();
        }
      });
    } finally {
      if (mounted) {
        setState(() => _busy = false);
        _scrollToLatest();
      }
    }
  }

  void _tryAnotherAmount() {
    const starter = 'Kalau beli Rp';
    _input.value = TextEditingValue(
      text: starter,
      selection: TextSelection.collapsed(offset: starter.length),
    );
    _focus.requestFocus();
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.viewInsetsOf(context).bottom),
      child: FractionallySizedBox(
        heightFactor: .88,
        child: SafeArea(
          top: false,
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 680),
              child: Padding(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 12),
                child: Column(
                  children: [
                    Row(
                      children: [
                        const CircleAvatar(
                          child: Icon(Icons.smart_toy_outlined),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Text(
                            'AI Companion',
                            style: theme.textTheme.titleLarge,
                          ),
                        ),
                        IconButton(
                          tooltip: 'Tutup chat',
                          onPressed: () => Navigator.of(context).pop(),
                          icon: const Icon(Icons.close),
                        ),
                      ],
                    ),
                    const Divider(),
                    Expanded(
                      child: ListView(
                        controller: _scroll,
                        children: [
                          if (!widget.api.signedIn) ...[
                            const Text(
                              'Masuk untuk bertanya tentang kondisi keuanganmu.',
                            ),
                            Align(
                              alignment: Alignment.centerLeft,
                              child: FilledButton(
                                onPressed: () async {
                                  final signedIn = await widget.signIn(context);
                                  if (mounted && signedIn) setState(() {});
                                },
                                child: const Text('Masuk'),
                              ),
                            ),
                          ] else if (_turns.isEmpty) ...[
                            const Text(
                              'Mau cek batas harian atau coba simulasi belanja?',
                            ),
                            const SizedBox(height: 8),
                            for (final example in _examples)
                              Align(
                                alignment: Alignment.centerLeft,
                                child: ActionChip(
                                  label: Text(example),
                                  onPressed: () => _send(example),
                                ),
                              ),
                          ],
                          for (final (index, turn) in _turns.indexed) ...[
                            Align(
                              alignment: Alignment.centerRight,
                              child: Card(
                                color: theme.colorScheme.primaryContainer,
                                child: Padding(
                                  padding: const EdgeInsets.all(12),
                                  child: Text(
                                    turn.question,
                                    style: TextStyle(
                                      color:
                                          theme.colorScheme.onPrimaryContainer,
                                    ),
                                  ),
                                ),
                              ),
                            ),
                            if (turn.reply case final CompanionReply reply)
                              _ReplyCard(
                                reply: reply,
                                onSetup: () => Navigator.of(context).pop(true),
                                onTryAmount: _busy ? null : _tryAnotherAmount,
                              )
                            else if (turn.error case final String error)
                              Column(
                                crossAxisAlignment: CrossAxisAlignment.start,
                                children: [
                                  Text(error),
                                  TextButton(
                                    onPressed: _busy
                                        ? null
                                        : () => _send(
                                            turn.question,
                                            retryIndex: index,
                                          ),
                                    child: const Text('Coba lagi'),
                                  ),
                                ],
                              )
                            else
                              const Padding(
                                padding: EdgeInsets.all(12),
                                child: LinearProgressIndicator(
                                  semanticsLabel: 'Memuat jawaban',
                                ),
                              ),
                          ],
                        ],
                      ),
                    ),
                    if (widget.api.signedIn)
                      Row(
                        children: [
                          Expanded(
                            child: TextField(
                              controller: _input,
                              focusNode: _focus,
                              maxLength: 1000,
                              textInputAction: TextInputAction.send,
                              decoration: const InputDecoration(
                                labelText: 'Tulis pertanyaan',
                                counterText: '',
                              ),
                              onSubmitted: _send,
                            ),
                          ),
                          IconButton.filled(
                            tooltip: 'Kirim pesan',
                            onPressed: _busy ? null : () => _send(_input.text),
                            icon: const Icon(Icons.send),
                          ),
                        ],
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
}

class _ReplyCard extends StatelessWidget {
  const _ReplyCard({
    required this.reply,
    required this.onSetup,
    this.onTryAmount,
  });

  final CompanionReply reply;
  final VoidCallback onSetup;
  final VoidCallback? onTryAmount;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(reply.message),
            if (reply.details.isNotEmpty) const SizedBox(height: 8),
            for (final detail in reply.details)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 3),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(child: Text(detail.label)),
                    const SizedBox(width: 8),
                    Flexible(
                      child: Text(
                        detail.value,
                        style: theme.textTheme.titleSmall,
                      ),
                    ),
                  ],
                ),
              ),
            for (final action in reply.actions)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(action),
              ),
            if (reply.sources.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 8),
                child: Text(
                  'Panduan Rantau+ · ${reply.sources.join(' · ')}',
                  style: theme.textTheme.bodySmall,
                ),
              ),
            if (reply.mode == 'setup_required')
              TextButton(
                onPressed: onSetup,
                child: const Text('Buka Financial Setup'),
              ),
            if (reply.mode == 'simulation' || reply.mode == 'needs_amount')
              TextButton(
                onPressed: onTryAmount,
                child: const Text('Coba nominal lain'),
              ),
            if (reply.mode == 'personal')
              TextButton(
                onPressed: onTryAmount,
                child: const Text('Coba simulasi belanja'),
              ),
          ],
        ),
      ),
    );
  }
}
