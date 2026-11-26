import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:shishimon/main.dart';

void main() {
  testWidgets('home screen shows scan and manual connect entry points', (
    tester,
  ) async {
    SharedPreferences.setMockInitialValues(<String, Object>{});
    await tester.pumpWidget(const ShishimonApp());
    await tester.pumpAndSettle();

    expect(find.text('SHISHIMON'), findsOneWidget);
    expect(find.text('SCAN QR CODE'), findsOneWidget);
    expect(find.text('CONNECT'), findsOneWidget);
  });
}
