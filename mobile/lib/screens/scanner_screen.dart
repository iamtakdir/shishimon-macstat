import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../dashboard_url.dart';
import '../theme.dart';

class ScannerScreen extends StatefulWidget {
  const ScannerScreen({super.key});

  @override
  State<ScannerScreen> createState() => _ScannerScreenState();
}

class _ScannerScreenState extends State<ScannerScreen> {
  final MobileScannerController _controller = MobileScannerController(
    formats: const [BarcodeFormat.qrCode],
    detectionSpeed: DetectionSpeed.noDuplicates,
  );
  bool _handled = false;
  DateTime _lastInvalidAt = DateTime.fromMillisecondsSinceEpoch(0);

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _onDetect(BarcodeCapture capture) {
    if (_handled) {
      return;
    }
    for (final barcode in capture.barcodes) {
      final raw = barcode.rawValue;
      if (raw == null || raw.isEmpty) {
        continue;
      }
      final url = DashboardUrl.tryParse(raw);
      if (url == null) {
        _showInvalid();
        continue;
      }
      _handled = true;
      Navigator.of(context).pop(url.value);
      return;
    }
  }

  void _showInvalid() {
    final now = DateTime.now();
    if (now.difference(_lastInvalidAt) < const Duration(seconds: 2)) {
      return;
    }
    _lastInvalidAt = now;
    ScaffoldMessenger.of(context)
      ..hideCurrentSnackBar()
      ..showSnackBar(
        const SnackBar(
          content: Text('That QR code is not a Shishimon dashboard link.'),
        ),
      );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: Colors.black,
      body: Stack(
        fit: StackFit.expand,
        children: [
          MobileScanner(
            controller: _controller,
            onDetect: _onDetect,
            errorBuilder: (context, error) => _CameraError(error: error),
            overlayBuilder: (context, constraints) =>
                _ScannerOverlay(constraints: constraints),
          ),
          SafeArea(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  _RoundButton(
                    icon: Icons.close,
                    tooltip: 'Close scanner',
                    onPressed: () => Navigator.of(context).pop(),
                  ),
                  ValueListenableBuilder<MobileScannerState>(
                    valueListenable: _controller,
                    builder: (context, state, _) {
                      final torchOn = state.torchState == TorchState.on;
                      return _RoundButton(
                        icon: torchOn ? Icons.flash_on : Icons.flash_off,
                        tooltip: torchOn ? 'Turn torch off' : 'Turn torch on',
                        active: torchOn,
                        onPressed: () => _controller.toggleTorch(),
                      );
                    },
                  ),
                ],
              ),
            ),
          ),
          const SafeArea(
            child: Align(
              alignment: Alignment.bottomCenter,
              child: Padding(
                padding: EdgeInsets.fromLTRB(24, 0, 24, 28),
                child: Text(
                  'ALIGN THE QR CODE INSIDE THE FRAME',
                  textAlign: TextAlign.center,
                  style: TextStyle(
                    color: AppColors.muted,
                    fontSize: 10.5,
                    letterSpacing: 1.6,
                  ),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _RoundButton extends StatelessWidget {
  const _RoundButton({
    required this.icon,
    required this.tooltip,
    required this.onPressed,
    this.active = false,
  });

  final IconData icon;
  final String tooltip;
  final VoidCallback onPressed;
  final bool active;

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.black.withValues(alpha: 0.55),
      shape: const CircleBorder(),
      child: IconButton(
        tooltip: tooltip,
        onPressed: onPressed,
        icon: Icon(
          icon,
          color: active ? AppColors.green : Colors.white,
          size: 22,
        ),
      ),
    );
  }
}

class _ScannerOverlay extends StatelessWidget {
  const _ScannerOverlay({required this.constraints});

  final BoxConstraints constraints;

  @override
  Widget build(BuildContext context) {
    final side = math.min(constraints.maxWidth * 0.72, 280.0);
    final window = Rect.fromCenter(
      center: Offset(constraints.maxWidth / 2, constraints.maxHeight / 2),
      width: side,
      height: side,
    );
    return IgnorePointer(
      child: CustomPaint(
        size: Size(constraints.maxWidth, constraints.maxHeight),
        painter: _ScannerOverlayPainter(window),
      ),
    );
  }
}

class _ScannerOverlayPainter extends CustomPainter {
  const _ScannerOverlayPainter(this.window);

  final Rect window;

  @override
  void paint(Canvas canvas, Size size) {
    final dim = Paint()..color = Colors.black.withValues(alpha: 0.62);
    final overlay = Path()..addRect(Offset.zero & size);
    final hole = Path()
      ..addRRect(RRect.fromRectAndRadius(window, const Radius.circular(18)));
    canvas.drawPath(Path.combine(PathOperation.difference, overlay, hole), dim);

    final frame = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 1.5
      ..color = AppColors.green.withValues(alpha: 0.45);
    canvas.drawRRect(
      RRect.fromRectAndRadius(window, const Radius.circular(18)),
      frame,
    );

    final corner = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 4
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round
      ..color = AppColors.green;
    const length = 26.0;
    final rect = window;

    void drawCorner(Offset start, Offset pivot, Offset end) {
      canvas.drawPath(
        Path()
          ..moveTo(start.dx, start.dy)
          ..lineTo(pivot.dx, pivot.dy)
          ..lineTo(end.dx, end.dy),
        corner,
      );
    }

    drawCorner(
      Offset(rect.left, rect.top + length),
      Offset(rect.left, rect.top),
      Offset(rect.left + length, rect.top),
    );
    drawCorner(
      Offset(rect.right - length, rect.top),
      Offset(rect.right, rect.top),
      Offset(rect.right, rect.top + length),
    );
    drawCorner(
      Offset(rect.right, rect.bottom - length),
      Offset(rect.right, rect.bottom),
      Offset(rect.right - length, rect.bottom),
    );
    drawCorner(
      Offset(rect.left + length, rect.bottom),
      Offset(rect.left, rect.bottom),
      Offset(rect.left, rect.bottom - length),
    );
  }

  @override
  bool shouldRepaint(covariant _ScannerOverlayPainter oldDelegate) =>
      oldDelegate.window != window;
}

class _CameraError extends StatelessWidget {
  const _CameraError({required this.error});

  final MobileScannerException error;

  @override
  Widget build(BuildContext context) {
    final denied = error.errorCode == MobileScannerErrorCode.permissionDenied;
    final detail = error.errorDetails?.message;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(
              Icons.no_photography_outlined,
              color: AppColors.red,
              size: 48,
            ),
            const SizedBox(height: 18),
            Text(
              denied ? 'CAMERA PERMISSION DENIED' : 'CAMERA UNAVAILABLE',
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: AppColors.text,
                fontSize: 14,
                fontWeight: FontWeight.w700,
                letterSpacing: 1.2,
              ),
            ),
            const SizedBox(height: 10),
            Text(
              denied
                  ? 'Allow camera access for Shishimon in system settings, '
                        'then reopen the scanner.'
                  : detail ?? 'The camera could not be started on this device.',
              textAlign: TextAlign.center,
              style: const TextStyle(
                color: AppColors.muted,
                fontSize: 11.5,
                height: 1.6,
              ),
            ),
          ],
        ),
      ),
    );
  }
}
