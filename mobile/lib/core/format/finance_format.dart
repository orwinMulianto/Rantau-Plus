String formatRupiah(int amount) {
  final digits = amount.toString().replaceFirst('-', '');
  final grouped = digits.replaceAllMapped(
    RegExp(r'\B(?=(\d{3})+(?!\d))'),
    (_) => '.',
  );
  return '${amount < 0 ? '-' : ''}Rp$grouped';
}

String displayDate(String date) => date.split('-').reversed.join('/');

String dateKey(DateTime date) => date.toIso8601String().split('T').first;
