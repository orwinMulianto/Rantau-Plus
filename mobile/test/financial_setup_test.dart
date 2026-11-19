import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:rantau_plus/app/rantau_app.dart';
import 'package:rantau_plus/core/network/rantau_api.dart';
import 'package:rantau_plus/features/dashboard/data/finance_summary_api.dart';

void main() {
  testWidgets('Financial Setup requires a current preview before saving', (tester) async {
    tester.view.physicalSize = const Size(900, 1200);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    Map<String, dynamic>? saved;
    var previews = 0;
    final client = MockClient((request) async {
      switch ((request.method, request.url.path)) {
        case ('GET', '/api/v1/demo/finance/summary'):
          return http.Response('', 404);
        case ('POST', '/api/v1/auth/login'):
          return _payload({'access_token': 'sample-access', 'refresh_token': 'sample-refresh'});
        case ('GET', '/api/v1/finance/setup'):
          expect(request.headers['authorization'], 'Bearer sample-access');
          return _payload({'today': '2026-11-18', 'setup': null});
        case ('POST', '/api/v1/finance/setup/preview'):
          expect(request.headers['authorization'], 'Bearer sample-access');
          previews++;
          return _payload(_summary(jsonDecode(request.body) as Map<String, dynamic>));
        case ('PUT', '/api/v1/finance/setup'):
          expect(request.headers['authorization'], 'Bearer sample-access');
          saved = jsonDecode(request.body) as Map<String, dynamic>;
          return _payload(_summary(saved!));
        case ('GET', '/api/v1/finance/summary'):
          return saved == null ? http.Response('', 204) : _payload(_summary(saved!));
        default:
          fail('Unexpected request ${request.method} ${request.url.path}');
      }
    });
    addTearDown(client.close);
    final baseUrl = Uri.parse('http://127.0.0.1:8080');
    await tester.pumpWidget(RantauApp(
      summaryApi: FinanceSummaryApi(client: client, baseUrl: baseUrl),
      api: RantauApi(client: client, baseUrl: baseUrl),
    ));
    await tester.pumpAndSettle();

    Future<void> tap(String text) async {
      final target = find.text(text);
      await tester.ensureVisible(target);
      await tester.tap(target);
      await tester.pumpAndSettle();
    }

    Future<void> enter(String label, String value) async {
      final field = find.widgetWithText(TextFormField, label);
      await tester.ensureVisible(field);
      await tester.enterText(field, value);
      await tester.pumpAndSettle();
    }

    await tap('Financial Setup');
    await enter('Email', 'student@example.test');
    await enter('Password', 'a long test password');
    await tester.tap(find.widgetWithText(FilledButton, 'Masuk'));
    await tester.pumpAndSettle();

    await enter('Saldo awal', '2.000.000');
    await tap('Pilih tanggal');
    await tap('28');
    await tap('Pilih');
    await enter('Dana cadangan', '200000');
    await tap('Tambah kewajiban');
    await enter('Nama kewajiban', 'Kos');
    await enter('Nominal kewajiban', '500000');
    expect(find.text('Simpan pengaturan'), findsNothing);
    expect(saved, isNull);

    await tap('Lihat pratinjau');
    expect(find.text('Rp1.300.000'), findsOneWidget);
    expect(find.text('Rp130.000/hari'), findsOneWidget);
    expect(saved, isNull);
    await enter('Saldo awal', '2100000');
    expect(find.text('Simpan pengaturan'), findsNothing);

    await tap('Lihat pratinjau');
    await tap('Simpan pengaturan');
    expect(previews, 2);
    expect(saved, {
      'opening_balance': 2100000,
      'next_income_date': '2026-11-28',
      'reserve_amount': 200000,
      'obligations': [{'name': 'Kos', 'amount': 500000}],
    });
    expect(find.text('Ringkasan keuangan'), findsOneWidget);
    expect(find.text('Rp140.000/hari'), findsOneWidget);
  });
}

http.Response _payload(Map<String, Object?> data) => http.Response(
  jsonEncode({'data': data}), 200, headers: {'content-type': 'application/json'},
);

Map<String, Object?> _summary(Map<String, dynamic> draft) {
  final obligations = draft['obligations'] as List<dynamic>;
  final unpaid = obligations.fold<int>(0, (total, item) => total + (item['amount'] as int));
  final reserve = draft['reserve_amount'] as int;
  final available = (draft['opening_balance'] as int) - reserve - unpaid;
  return {
    'currency': 'IDR',
    'period': {
      'as_of_date': '2026-11-18',
      'next_income_date': draft['next_income_date'],
      'remaining_days': 10,
    },
    'available_funds': available,
    'safe_to_spend_daily': available ~/ 10,
    'runway_days': null,
    'average_daily_spending': 0,
    'projected_balance': available,
    'unpaid_obligations': unpaid,
    'reserve': reserve,
    'risk_status': 'BELUM_CUKUP_DATA',
  };
}
