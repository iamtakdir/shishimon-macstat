import 'package:flutter_test/flutter_test.dart';
import 'package:shishimon/dashboard_url.dart';

void main() {
  group('DashboardUrl.tryParse', () {
    test('accepts a full dashboard URL', () {
      final url = DashboardUrl.tryParse(
        'http://192.168.0.105:8080/?token=47859',
      );

      expect(url, isNotNull);
      expect(url!.uri.host, '192.168.0.105');
      expect(url.uri.port, 8080);
      expect(url.authority, '192.168.0.105:8080');
      expect(url.uri.queryParameters['token'], '47859');
    });

    test('adds http to a bare host:port value', () {
      final url = DashboardUrl.tryParse('192.168.0.105:8080/?token=abc');

      expect(url, isNotNull);
      expect(url!.uri.scheme, 'http');
      expect(url.authority, '192.168.0.105:8080');
    });

    test('accepts https and trims whitespace', () {
      final url = DashboardUrl.tryParse(
        '  https://mac.local:8443/?token=abc  ',
      );

      expect(url, isNotNull);
      expect(url!.uri.scheme, 'https');
      expect(url.authority, 'mac.local:8443');
    });

    test('rejects a URL without a token', () {
      expect(DashboardUrl.tryParse('http://192.168.0.105:8080/'), isNull);
    });

    test('rejects a blank token', () {
      expect(
        DashboardUrl.tryParse('http://192.168.0.105:8080/?token=%20'),
        isNull,
      );
    });

    test('rejects unsupported schemes', () {
      expect(DashboardUrl.tryParse('ftp://192.168.0.105/?token=abc'), isNull);
    });

    test('rejects empty input', () {
      expect(DashboardUrl.tryParse('   '), isNull);
    });

    test('compares by full URL', () {
      final first = DashboardUrl.tryParse('http://mac:8080/?token=a');
      final second = DashboardUrl.tryParse('http://mac:8080/?token=a');
      final third = DashboardUrl.tryParse('http://mac:8080/?token=b');

      expect(first, second);
      expect(first, isNot(third));
    });
  });
}
