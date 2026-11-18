import 'package:flutter/material.dart';

import '../../../core/network/rantau_api.dart';

enum _Step { signIn, register, verify }

class SetupSignInPage extends StatefulWidget {
  const SetupSignInPage({super.key, required this.api});

  final RantauApi api;

  @override
  State<SetupSignInPage> createState() => _SetupSignInPageState();
}

class _SetupSignInPageState extends State<SetupSignInPage> {
  final _form = GlobalKey<FormState>();
  final _email = TextEditingController();
  final _password = TextEditingController();
  final _name = TextEditingController();
  final _token = TextEditingController();
  _Step _step = _Step.signIn;
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _email.dispose();
    _password.dispose();
    _name.dispose();
    _token.dispose();
    super.dispose();
  }

  void _setStep(_Step step) => setState(() {
    _step = step;
    _error = null;
  });

  Future<void> _submit() async {
    if (_busy || !_form.currentState!.validate()) return;
    FocusScope.of(context).unfocus();
    setState(() {
      _busy = true;
      _error = null;
    });
    final email = _email.text.trim();
    final password = _password.text;
    try {
      if (_step == _Step.register) {
        final response = await widget.api.request(
          'POST',
          '/api/v1/auth/register',
          body: {'email': email},
        );
        RantauApi.check(response);
        if (mounted) _setStep(_Step.verify);
        return;
      }
      if (_step == _Step.verify) {
        final response = await widget.api.request(
          'POST',
          '/api/v1/auth/verify-email',
          body: {
            'name': _name.text.trim(),
            'token': _token.text.trim(),
            'password': password,
          },
        );
        RantauApi.check(response);
        if (!mounted) return;
        _setStep(_Step.signIn);
      }
      await widget.api.login(email, password);
      if (mounted) Navigator.of(context).pop(true);
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _error = error is ApiFailure
            ? error.message
            : 'Gagal terhubung. Coba lagi.';
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final title = switch (_step) {
      _Step.signIn => 'Masuk',
      _Step.register => 'Buat akun',
      _Step.verify => 'Verifikasi email',
    };
    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: SafeArea(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 480),
            child: Form(
              key: _form,
              child: ListView(
                padding: const EdgeInsets.all(24),
                shrinkWrap: true,
                children: [
                  Text(
                    _step == _Step.verify
                        ? 'Masukkan token dari email yang dikirim ke ${_email.text.trim()}.'
                        : 'Gunakan akun Rantau+ untuk menyimpan pengaturan keuangan.',
                    style: theme.textTheme.bodyLarge,
                  ),
                  const SizedBox(height: 20),
                  if (_step != _Step.verify)
                    TextFormField(
                      controller: _email,
                      enabled: !_busy,
                      keyboardType: TextInputType.emailAddress,
                      autofillHints: const [AutofillHints.email],
                      decoration: const InputDecoration(
                        labelText: 'Email',
                        border: OutlineInputBorder(),
                      ),
                      validator: (value) => (value ?? '').contains('@')
                          ? null
                          : 'Masukkan email yang valid.',
                    ),
                  if (_step == _Step.verify) ...[
                    TextFormField(
                      controller: _token,
                      enabled: !_busy,
                      autocorrect: false,
                      decoration: const InputDecoration(
                        labelText: 'Token verifikasi',
                        border: OutlineInputBorder(),
                      ),
                      validator: (value) => (value ?? '').trim().isEmpty
                          ? 'Masukkan token verifikasi.'
                          : null,
                    ),
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _name,
                      enabled: !_busy,
                      autofillHints: const [AutofillHints.name],
                      textCapitalization: TextCapitalization.words,
                      decoration: const InputDecoration(
                        labelText: 'Nama',
                        border: OutlineInputBorder(),
                      ),
                      validator: (value) => (value ?? '').trim().isEmpty
                          ? 'Masukkan nama.'
                          : null,
                    ),
                  ],
                  if (_step != _Step.register) ...[
                    const SizedBox(height: 12),
                    TextFormField(
                      controller: _password,
                      enabled: !_busy,
                      obscureText: true,
                      autofillHints: _step == _Step.verify
                          ? const [AutofillHints.newPassword]
                          : const [AutofillHints.password],
                      decoration: InputDecoration(
                        labelText: 'Password',
                        helperText: _step == _Step.verify
                            ? 'Minimal 15 karakter'
                            : null,
                        border: const OutlineInputBorder(),
                      ),
                      validator: (value) =>
                          (value ?? '').isEmpty ? 'Masukkan password.' : null,
                    ),
                  ],
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Text(
                      _error!,
                      style: TextStyle(color: theme.colorScheme.error),
                    ),
                  ],
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: _busy ? null : _submit,
                    child: Text(
                      _busy
                          ? 'Memproses…'
                          : switch (_step) {
                              _Step.signIn => 'Masuk',
                              _Step.register => 'Kirim token verifikasi',
                              _Step.verify => 'Verifikasi dan masuk',
                            },
                    ),
                  ),
                  TextButton(
                    onPressed: _busy
                        ? null
                        : () => _setStep(
                            _step == _Step.signIn
                                ? _Step.register
                                : _Step.signIn,
                          ),
                    child: Text(
                      _step == _Step.signIn ? 'Buat akun' : 'Kembali ke login',
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }
}
