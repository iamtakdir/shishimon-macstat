/// A validated Shishimon dashboard URL, including its access token.
class DashboardUrl {
  const DashboardUrl(this.uri);

  final Uri uri;

  /// Parses user input or a scanned QR payload.
  ///
  /// Accepts full URLs (`http://192.168.0.105:8080/?token=1234`) and bare
  /// `host:port/?token=...` values. A token query parameter is required,
  /// because the server rejects requests without it.
  static DashboardUrl? tryParse(String raw) {
    var value = raw.trim();
    if (value.isEmpty) {
      return null;
    }
    if (!value.contains('://')) {
      value = 'http://$value';
    }
    final uri = Uri.tryParse(value);
    if (uri == null || !uri.hasAuthority) {
      return null;
    }
    if (uri.scheme != 'http' && uri.scheme != 'https') {
      return null;
    }
    if (uri.host.isEmpty) {
      return null;
    }
    final token = uri.queryParameters['token'];
    if (token == null || token.trim().isEmpty) {
      return null;
    }
    return DashboardUrl(uri);
  }

  /// `host` or `host:port`, used in titles and the recent list.
  String get authority => uri.hasPort ? '${uri.host}:${uri.port}' : uri.host;

  String get value => uri.toString();

  @override
  bool operator ==(Object other) =>
      other is DashboardUrl && other.value == value;

  @override
  int get hashCode => value.hashCode;
}
