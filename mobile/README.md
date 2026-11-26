# Shishimon Mobile

Flutter companion app for the Shishimon Mac dashboard.

## What it does

- Scans the QR code printed by `bro-start`.
- Opens the live dashboard in a WebView.
- Keeps the screen awake while the dashboard is open.
- Remembers recent connections (with a manual URL fallback).
- Shows clear errors for unreachable hosts and expired access tokens.

## Run

```sh
flutter pub get
flutter run
```

The phone must be on the same Wi-Fi or local network as the Mac.

## Build

Android:

```sh
flutter build apk --debug
# build/app/outputs/flutter-apk/app-debug.apk
```

iOS builds require a full Xcode installation and CocoaPods.

## Platform notes

- **Android**: cleartext HTTP and camera/internet permissions are declared in `android/app/src/main/AndroidManifest.xml`.
- **iOS**: camera and local-network usage descriptions plus ATS exceptions are declared in `ios/Runner/Info.plist`.
- **Screen wake**: `wakelock_plus` enables the platform keep-screen-on flag only while the dashboard route is active.
- **Token**: the server generates a new token on every launch. Rescan the QR code when the token expires.

## Checks

```sh
flutter analyze
flutter test
```
