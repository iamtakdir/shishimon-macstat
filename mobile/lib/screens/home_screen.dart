import 'package:flutter/material.dart';

import '../dashboard_url.dart';
import '../services/recent_connections.dart';
import '../theme.dart';
import 'dashboard_screen.dart';
import 'scanner_screen.dart';

class HomeScreen extends StatefulWidget {
  const HomeScreen({super.key});

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  final RecentConnectionsStore _store = RecentConnectionsStore();
  final TextEditingController _urlController = TextEditingController();
  List<RecentConnection> _recent = const <RecentConnection>[];
  bool _loadingRecent = true;

  @override
  void initState() {
    super.initState();
    _loadRecent();
  }

  @override
  void dispose() {
    _urlController.dispose();
    super.dispose();
  }

  Future<void> _loadRecent() async {
    final entries = await _store.load();
    if (!mounted) {
      return;
    }
    setState(() {
      _recent = entries;
      _loadingRecent = false;
    });
  }

  Future<void> _open(DashboardUrl url) async {
    final entries = await _store.add(url);
    if (!mounted) {
      return;
    }
    setState(() => _recent = entries);
    await Navigator.of(
      context,
    ).push(MaterialPageRoute<void>(builder: (_) => DashboardScreen(url: url)));
  }

  Future<void> _scan() async {
    final raw = await Navigator.of(context).push<String>(
      MaterialPageRoute<String>(builder: (_) => const ScannerScreen()),
    );
    if (raw == null || !mounted) {
      return;
    }
    final url = DashboardUrl.tryParse(raw);
    if (url == null) {
      _showMessage('That QR code is not a Shishimon dashboard link.');
      return;
    }
    await _open(url);
  }

  Future<void> _connectManually() async {
    final url = DashboardUrl.tryParse(_urlController.text);
    if (url == null) {
      _showMessage(
        'Enter a dashboard URL with a token, e.g. '
        'http://192.168.0.105:8080/?token=1234',
      );
      return;
    }
    FocusScope.of(context).unfocus();
    _urlController.clear();
    await _open(url);
  }

  Future<void> _forget(RecentConnection connection) async {
    final entries = await _store.remove(connection.url);
    if (!mounted) {
      return;
    }
    setState(() => _recent = entries);
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: SafeArea(
        child: ListView(
          padding: const EdgeInsets.fromLTRB(20, 26, 20, 32),
          children: [
            const _Header(),
            const SizedBox(height: 28),
            _ScanCard(onTap: _scan),
            const SizedBox(height: 26),
            const _SectionLabel('OR CONNECT MANUALLY'),
            const SizedBox(height: 10),
            TextField(
              controller: _urlController,
              keyboardType: TextInputType.url,
              autocorrect: false,
              enableSuggestions: false,
              textInputAction: TextInputAction.go,
              onSubmitted: (_) => _connectManually(),
              style: const TextStyle(fontSize: 12.5, color: AppColors.text),
              decoration: const InputDecoration(
                hintText: 'http://192.168.0.105:8080/?token=…',
                prefixIcon: Icon(Icons.link, size: 18, color: AppColors.muted),
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Use the URL printed by bro-start. The token changes when the '
              'server restarts.',
              style: TextStyle(
                color: AppColors.muted,
                fontSize: 10.5,
                height: 1.5,
              ),
            ),
            const SizedBox(height: 12),
            ElevatedButton.icon(
              onPressed: _connectManually,
              icon: const Icon(Icons.arrow_forward, size: 18),
              label: const Text('CONNECT'),
            ),
            if (!_loadingRecent && _recent.isNotEmpty) ...[
              const SizedBox(height: 30),
              const _SectionLabel('RECENT CONNECTIONS'),
              const SizedBox(height: 10),
              for (final connection in _recent)
                Padding(
                  padding: const EdgeInsets.only(bottom: 8),
                  child: _RecentTile(
                    connection: connection,
                    onOpen: () => _open(connection.url),
                    onForget: () => _forget(connection),
                  ),
                ),
            ],
            const SizedBox(height: 30),
            const _FooterNote(),
          ],
        ),
      ),
    );
  }
}

class _Header extends StatelessWidget {
  const _Header();

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Container(
          width: 44,
          height: 44,
          decoration: BoxDecoration(
            color: AppColors.green,
            borderRadius: BorderRadius.circular(4),
          ),
          child: const Icon(
            Icons.monitor_heart_outlined,
            color: Color(0xFF06110D),
            size: 26,
          ),
        ),
        const SizedBox(width: 12),
        const Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'SHISHIMON',
                style: TextStyle(
                  color: AppColors.text,
                  fontSize: 22,
                  fontWeight: FontWeight.w700,
                  letterSpacing: 2.4,
                ),
              ),
              SizedBox(height: 3),
              Text(
                'MAC TELEMETRY // MOBILE',
                style: TextStyle(
                  color: AppColors.muted,
                  fontSize: 9.5,
                  letterSpacing: 1.6,
                ),
              ),
            ],
          ),
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          decoration: BoxDecoration(
            border: Border.all(color: AppColors.line),
            borderRadius: BorderRadius.circular(3),
          ),
          child: const Text(
            'READY',
            style: TextStyle(
              color: AppColors.green,
              fontSize: 9,
              letterSpacing: 1.2,
            ),
          ),
        ),
      ],
    );
  }
}

class _ScanCard extends StatelessWidget {
  const _ScanCard({required this.onTap});

  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.panel,
      borderRadius: BorderRadius.circular(4),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(4),
        child: Container(
          padding: const EdgeInsets.all(20),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(4),
            border: Border.all(color: AppColors.green.withValues(alpha: 0.45)),
          ),
          child: Row(
            children: [
              Container(
                padding: const EdgeInsets.all(14),
                decoration: BoxDecoration(
                  color: AppColors.green.withValues(alpha: 0.08),
                  border: Border.all(color: AppColors.line),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: const Icon(
                  Icons.qr_code_scanner,
                  color: AppColors.green,
                  size: 34,
                ),
              ),
              const SizedBox(width: 16),
              const Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'SCAN QR CODE',
                      style: TextStyle(
                        color: AppColors.text,
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                        letterSpacing: 1.4,
                      ),
                    ),
                    SizedBox(height: 6),
                    Text(
                      'Point your camera at the code printed by bro-start.',
                      style: TextStyle(
                        color: AppColors.muted,
                        fontSize: 11,
                        height: 1.5,
                      ),
                    ),
                  ],
                ),
              ),
              const Icon(Icons.chevron_right, color: AppColors.green),
            ],
          ),
        ),
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.label);

  final String label;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        Text(
          label,
          style: const TextStyle(
            color: AppColors.muted,
            fontSize: 9.5,
            fontWeight: FontWeight.w600,
            letterSpacing: 1.6,
          ),
        ),
        const SizedBox(width: 10),
        const Expanded(child: Divider(height: 1)),
      ],
    );
  }
}

class _RecentTile extends StatelessWidget {
  const _RecentTile({
    required this.connection,
    required this.onOpen,
    required this.onForget,
  });

  final RecentConnection connection;
  final VoidCallback onOpen;
  final VoidCallback onForget;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: AppColors.panel,
      borderRadius: BorderRadius.circular(4),
      child: InkWell(
        onTap: onOpen,
        borderRadius: BorderRadius.circular(4),
        child: Container(
          padding: const EdgeInsets.fromLTRB(14, 10, 4, 10),
          decoration: BoxDecoration(
            border: Border.all(color: AppColors.line),
            borderRadius: BorderRadius.circular(4),
          ),
          child: Row(
            children: [
              const Icon(Icons.lan_outlined, color: AppColors.cyan, size: 20),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      connection.url.authority,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                        color: AppColors.text,
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 3),
                    Text(
                      'Last used ${_relativeTime(connection.lastUsed)}',
                      style: const TextStyle(
                        color: AppColors.muted,
                        fontSize: 10,
                      ),
                    ),
                  ],
                ),
              ),
              IconButton(
                tooltip: 'Forget connection',
                onPressed: onForget,
                icon: const Icon(
                  Icons.delete_outline,
                  size: 18,
                  color: AppColors.muted,
                ),
              ),
              const Icon(Icons.chevron_right, color: AppColors.muted, size: 20),
            ],
          ),
        ),
      ),
    );
  }
}

class _FooterNote extends StatelessWidget {
  const _FooterNote();

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: AppColors.panel,
        border: Border.all(color: AppColors.line),
        borderRadius: BorderRadius.circular(4),
      ),
      child: const Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.info_outline, size: 16, color: AppColors.muted),
          SizedBox(width: 10),
          Expanded(
            child: Text(
              'Phone and Mac must be on the same Wi-Fi. The screen stays '
              'awake while the dashboard is open.',
              style: TextStyle(
                color: AppColors.muted,
                fontSize: 10.5,
                height: 1.5,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

String _relativeTime(DateTime time) {
  final difference = DateTime.now().difference(time);
  if (difference.inMinutes < 1) {
    return 'just now';
  }
  if (difference.inMinutes < 60) {
    return '${difference.inMinutes}m ago';
  }
  if (difference.inHours < 24) {
    return '${difference.inHours}h ago';
  }
  if (difference.inDays < 7) {
    return '${difference.inDays}d ago';
  }
  final month = time.month.toString().padLeft(2, '0');
  final day = time.day.toString().padLeft(2, '0');
  return '${time.year}-$month-$day';
}
