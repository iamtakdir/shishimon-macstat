import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:webview_flutter/webview_flutter.dart';
import 'package:wakelock_plus/wakelock_plus.dart';

import '../dashboard_url.dart';
import '../services/recent_connections.dart';
import '../theme.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({super.key, required this.url});

  final DashboardUrl url;

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen>
    with WidgetsBindingObserver {
  late final WebViewController _controller;
  double _progress = 0;
  bool _loading = true;
  String? _error;
  bool _authError = false;
  bool _fullscreen = false;
  bool _landscapeLocked = false;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    unawaited(WakelockPlus.enable());
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setBackgroundColor(AppColors.background)
      ..setNavigationDelegate(
        NavigationDelegate(
          onProgress: (progress) {
            if (!mounted) {
              return;
            }
            setState(() => _progress = progress / 100);
          },
          onPageStarted: (_) {
            if (!mounted) {
              return;
            }
            setState(() => _loading = true);
          },
          onPageFinished: (_) {
            if (!mounted) {
              return;
            }
            setState(() {
              _loading = false;
              _progress = 1;
            });
          },
          onWebResourceError: _onResourceError,
          onHttpError: _onHttpError,
        ),
      )
      ..loadRequest(widget.url.uri);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    unawaited(SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge));
    unawaited(
      SystemChrome.setPreferredOrientations(<DeviceOrientation>[]),
    );
    unawaited(WakelockPlus.disable());
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      unawaited(WakelockPlus.enable());
    } else if (state == AppLifecycleState.paused ||
        state == AppLifecycleState.hidden ||
        state == AppLifecycleState.detached) {
      unawaited(WakelockPlus.disable());
    }
  }

  void _onResourceError(WebResourceError error) {
    // -999 is NSURLErrorCancelled, emitted for superseded navigations.
    if (error.errorCode == -999) {
      return;
    }
    if (error.isForMainFrame == false) {
      return;
    }
    if (!mounted) {
      return;
    }
    setState(() {
      _error = error.description.isEmpty
          ? 'Could not reach the dashboard.'
          : error.description;
      _authError = false;
      _loading = false;
    });
  }

  Future<void> _handleAuthError({String? detail}) async {
    await RecentConnectionsStore().remove(widget.url);
    if (!mounted) {
      return;
    }
    setState(() {
      _error = detail ??
          'Invalid or expired token. Scan the QR code from bro-start again.';
      _authError = true;
      _loading = false;
    });
  }

  void _onHttpError(HttpResponseError error) {
    if (error.response?.statusCode != 401) {
      return;
    }
    // Only treat the dashboard document itself as an auth failure.
    // WebViews report HTTP errors for subresources too (e.g. /favicon.ico),
    // which carry no token and must not cover a working dashboard.
    final uri = error.request?.uri ?? error.response?.uri;
    if (uri != null &&
        uri.queryParameters['token'] != widget.url.uri.queryParameters['token']) {
      return;
    }
    unawaited(_handleAuthError(
      detail:
          'Server returned 401 for ${uri ?? widget.url.uri}. '
          'Scan the QR code from bro-start again.',
    ));
  }

  Future<void> _reload() async {
    setState(() {
      _error = null;
      _authError = false;
      _loading = true;
      _progress = 0;
    });
    await _controller.reload();
  }

  Future<void> _toggleFullscreen() async {
    final next = !_fullscreen;
    setState(() => _fullscreen = next);
    await SystemChrome.setEnabledSystemUIMode(
      next ? SystemUiMode.immersiveSticky : SystemUiMode.edgeToEdge,
    );
  }

  Future<void> _toggleLandscape() async {
    final next = !_landscapeLocked;
    setState(() => _landscapeLocked = next);
    await SystemChrome.setPreferredOrientations(
      next
          ? const <DeviceOrientation>[
              DeviceOrientation.landscapeLeft,
              DeviceOrientation.landscapeRight,
            ]
          : <DeviceOrientation>[],
    );
  }

  @override
  Widget build(BuildContext context) {
    final scaffold = Scaffold(
      appBar: _fullscreen
          ? null
          : AppBar(
              titleSpacing: 16,
              title: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    widget.url.authority,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.w700,
                      letterSpacing: 0.4,
                    ),
                  ),
                  const Text(
                    'LIVE DASHBOARD',
                    style: TextStyle(
                      color: AppColors.muted,
                      fontSize: 9,
                      letterSpacing: 1.6,
                    ),
                  ),
                ],
              ),
              actions: [
                const Tooltip(
                  message: 'Screen stays on while this screen is open',
                  child: Padding(
                    padding: EdgeInsets.symmetric(horizontal: 4),
                    child: Icon(
                      Icons.screen_lock_portrait,
                      size: 19,
                      color: AppColors.green,
                    ),
                  ),
                ),
                IconButton(
                  tooltip: 'Refresh',
                  onPressed: _reload,
                  icon: const Icon(Icons.refresh, size: 22),
                ),
                IconButton(
                  tooltip: _fullscreen ? 'Exit fullscreen' : 'Fullscreen',
                  onPressed: _toggleFullscreen,
                  icon: Icon(
                    _fullscreen
                        ? Icons.fullscreen_exit
                        : Icons.fullscreen,
                    size: 22,
                  ),
                ),
                IconButton(
                  tooltip: _landscapeLocked
                      ? 'Unlock rotation'
                      : 'Lock landscape',
                  onPressed: _toggleLandscape,
                  icon: Icon(
                    _landscapeLocked
                        ? Icons.screen_lock_rotation
                        : Icons.screen_rotation,
                    size: 22,
                  ),
                ),
                PopupMenuButton<String>(
                  tooltip: 'More',
                  onSelected: (value) {
                    if (value == 'disconnect') {
                      Navigator.of(context).pop();
                    }
                  },
                  itemBuilder: (_) => const [
                    PopupMenuItem<String>(
                      value: 'disconnect',
                      child: Text('Disconnect'),
                    ),
                  ],
                ),
              ],
              bottom: PreferredSize(
                preferredSize: const Size.fromHeight(2),
                child: _loading
                    ? LinearProgressIndicator(
                        value: _progress <= 0 || _progress >= 1
                            ? null
                            : _progress,
                        minHeight: 2,
                      )
                    : const SizedBox(height: 2),
              ),
            ),
      body: SafeArea(
        top: !_fullscreen,
        bottom: !_fullscreen,
        child: Stack(
          children: [
            WebViewWidget(controller: _controller),
            if (_error != null)
              Positioned.fill(
                child: ColoredBox(
                  color: AppColors.background.withValues(alpha: 0.95),
                  child: _ErrorPanel(
                    message: _error!,
                    authError: _authError,
                    onRetry: _reload,
                    onDisconnect: () => Navigator.of(context).pop(),
                  ),
                ),
              ),
          ],
        ),
      ),
    );
    if (!_fullscreen) {
      return scaffold;
    }
    return Stack(
      children: [
        scaffold,
        Positioned(
          right: 12,
          bottom: 12,
          child: SafeArea(
            child: FloatingActionButton.small(
              heroTag: 'exit_fullscreen',
              tooltip: 'Exit fullscreen',
              onPressed: _toggleFullscreen,
              backgroundColor: AppColors.panel,
              foregroundColor: AppColors.green,
              child: const Icon(Icons.fullscreen_exit, size: 20),
            ),
          ),
        ),
      ],
    );
  }
}

class _ErrorPanel extends StatelessWidget {
  const _ErrorPanel({
    required this.message,
    required this.authError,
    required this.onRetry,
    required this.onDisconnect,
  });

  final String message;
  final bool authError;
  final VoidCallback onRetry;
  final VoidCallback onDisconnect;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(28),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              authError ? Icons.key_off_outlined : Icons.wifi_off_rounded,
              color: AppColors.red,
              size: 46,
            ),
            const SizedBox(height: 18),
            Text(
              authError ? 'CONNECTION REJECTED' : 'DASHBOARD UNREACHABLE',
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: AppColors.text,
                fontSize: 15,
                fontWeight: FontWeight.w700,
                letterSpacing: 1.4,
              ),
            ),
            const SizedBox(height: 10),
            Text(
              message,
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: AppColors.muted,
                fontSize: 11.5,
                height: 1.6,
              ),
            ),
            const SizedBox(height: 22),
            Wrap(
              spacing: 10,
              runSpacing: 10,
              alignment: WrapAlignment.center,
              children: [
                ElevatedButton.icon(
                  onPressed: onRetry,
                  icon: const Icon(Icons.refresh, size: 18),
                  label: const Text('RETRY'),
                  style: ElevatedButton.styleFrom(
                    minimumSize: const Size(140, 46),
                  ),
                ),
                OutlinedButton.icon(
                  onPressed: onDisconnect,
                  icon: const Icon(Icons.link_off, size: 18),
                  label: const Text('DISCONNECT'),
                  style: OutlinedButton.styleFrom(
                    minimumSize: const Size(140, 46),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
