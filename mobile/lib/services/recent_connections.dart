import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../dashboard_url.dart';

class RecentConnection {
  const RecentConnection({required this.url, required this.lastUsed});

  final DashboardUrl url;
  final DateTime lastUsed;

  Map<String, Object> toJson() => <String, Object>{
    'url': url.value,
    'lastUsed': lastUsed.toIso8601String(),
  };

  static RecentConnection? fromJson(Object? json) {
    if (json is! Map) {
      return null;
    }
    final rawUrl = json['url'];
    final rawLastUsed = json['lastUsed'];
    if (rawUrl is! String || rawLastUsed is! String) {
      return null;
    }
    final url = DashboardUrl.tryParse(rawUrl);
    final lastUsed = DateTime.tryParse(rawLastUsed);
    if (url == null || lastUsed == null) {
      return null;
    }
    return RecentConnection(url: url, lastUsed: lastUsed);
  }
}

/// Persists the most recently opened dashboards.
class RecentConnectionsStore {
  static const _storageKey = 'recent_connections_v1';
  static const _maxEntries = 6;

  Future<List<RecentConnection>> load() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_storageKey);
    if (raw == null || raw.isEmpty) {
      return <RecentConnection>[];
    }
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! List) {
        return <RecentConnection>[];
      }
      final entries = decoded
          .map(RecentConnection.fromJson)
          .whereType<RecentConnection>()
          .toList();
      entries.sort((a, b) => b.lastUsed.compareTo(a.lastUsed));
      return entries;
    } on FormatException {
      return <RecentConnection>[];
    }
  }

  Future<List<RecentConnection>> add(DashboardUrl url) async {
    final entries = await load();
    entries.removeWhere((entry) => entry.url == url);
    entries.insert(0, RecentConnection(url: url, lastUsed: DateTime.now()));
    final trimmed = entries.take(_maxEntries).toList();
    await _save(trimmed);
    return trimmed;
  }

  Future<List<RecentConnection>> remove(DashboardUrl url) async {
    final entries = await load();
    entries.removeWhere((entry) => entry.url == url);
    await _save(entries);
    return entries;
  }

  Future<void> _save(List<RecentConnection> entries) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(
      _storageKey,
      jsonEncode(entries.map((entry) => entry.toJson()).toList()),
    );
  }
}
