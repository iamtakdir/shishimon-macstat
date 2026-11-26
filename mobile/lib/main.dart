import 'package:flutter/material.dart';

import 'screens/home_screen.dart';
import 'theme.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  runApp(const ShishimonApp());
}

class ShishimonApp extends StatelessWidget {
  const ShishimonApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Shishimon',
      debugShowCheckedModeBanner: false,
      theme: buildShishimonTheme(),
      home: const HomeScreen(),
    );
  }
}
