# Add Location Spec — How's Working There (iOS)

> **Speed Test SDK:** [SpeedcheckerSDK](https://github.com/speedchecker/speedchecker-sdk-ios) (free tier, via SPM)
> **Location Search:** [Mapbox Search SDK](https://docs.mapbox.com/ios/search/guides/) (`MapboxSearch` via SPM)
> **Decibel Metering:** `AVFoundation` (`AVAudioRecorder`)
> **Minimum iOS:** 17.0

---

## Table of Contents

1. [Overview](#1-overview)
2. [User Flow](#2-user-flow)
3. [Permissions](#3-permissions)
4. [SDK Integration](#4-sdk-integration)
5. [Data Models](#5-data-models)
6. [Architecture and File Structure](#6-architecture-and-file-structure)
7. [View Specifications](#7-view-specifications)
8. [Error Handling](#8-error-handling)
9. [Configuration](#9-configuration)
10. [Future API Integration](#10-future-api-integration)

---

## 1. Overview

The Add Location feature allows authenticated users to check in at a workspace and capture detailed environment data. The flow collects:

- **Location** — Name and address via Mapbox place search
- **Network quality** — Download/upload speed, latency, jitter, ISP, server details, packet loss via SpeedChecker SDK
- **Ambient noise** — Decibel level measurement via device microphone
- **Workspace attributes** — Outlet availability, crowdedness, ease of work, best work type

The feature is accessed by tapping the "+" button in the center tab bar. Data is stored locally until a backend API is available.

---

## 2. User Flow

The flow is presented as a full-screen sheet with a multi-step wizard.

### Step 1 — Location Search

- User taps "+" in tab bar, sheet presents the Add Location flow
- First screen shows a Mapbox PlaceAutocomplete search bar
- As the user types, autocomplete suggestions appear in a list
- User taps a result to select it
- Captured data: location name, full address, latitude, longitude
- (Future) API call checks if location exists in database; if found, existing location ID is reused. Speed/decibel tests still run since conditions change.

### Step 2 — Speed Test

- Auto-starts after location is confirmed
- Requests location permission if not already granted (required by SpeedChecker free SDK)
- Shows animated progress UI with server selection, download, and upload phases
- Captured data from SpeedTestResult:
  - `downloadSpeed.mbps` — Download speed in Mbps
  - `uploadSpeed.mbps` — Upload speed in Mbps
  - `latencyInMs` — Latency in milliseconds
  - `jitter` — Jitter value
  - `ispName` — Internet Service Provider name
  - `ipAddress` — Device IP address
  - `server.domain` — Test server domain
  - `server.country` — Test server country
  - `server.cityName` — Test server city
  - `network` — SpeedTestNetworkType (wifi / cellular)
  - `packetLoss` — Packet loss data
  - `timeToFirstByteMs` — Time to first byte
  - `downloadTransferredMb` — Total MB downloaded
  - `uploadTransferredMb` — Total MB uploaded
  - `testID` — Unique test identifier
- Network signal info: iOS does not expose raw Wi-Fi RSSI without private API. We capture SpeedTestNetworkType and available connection metadata.

### Step 3 — Decibel Test

- Auto-starts after speed test completes
- Requests microphone permission if not already granted
- Uses AVAudioRecorder with isMeteringEnabled for ambient noise sampling
- Samples averagePower(forChannel:) over approximately 5 seconds
- Converts to approximate dB SPL
- Displays animated level meter
- Captured data: average dB, peak dB, duration sampled

### Step 4 — Workspace Ratings

User taps through intuitive pill/chip selectors matching the retro theme:

- **Outlets at bar** — Toggle: Yes / No
- **Outlets at table** — Toggle: Yes / No
- **Crowdedness** — Scale 1-3: Empty, Somewhat Crowded, Crowded
- **Ease of work** — Scale 1-3: Easy, Moderate, Difficult
- **Best work type** — Solo, Team, Both

### Step 5 — Review and Submit

- Summary card showing all collected data
- Submit button sends payload (API integration TBD, logged locally for now)
- On success, dismiss sheet

---

## 3. Permissions

| Permission | Info.plist Key | When Requested |
|---|---|---|
| Location (When In Use) | `NSLocationWhenInUseUsageDescription` | Before speed test (required by SpeedChecker free SDK) |
| Microphone | `NSMicrophoneUsageDescription` | Before decibel test |

These keys are added to the Xcode build settings under INFOPLIST_KEY.

---

## 4. SDK Integration

### 4.1 SpeedChecker SDK

- **Package URL:** `https://github.com/speedchecker/speedchecker-sdk-ios`
- **Product:** `SpeedcheckerSDK`
- **Tier:** Free (no license key required, requires location permission)
- **Key class:** `InternetSpeedTest`
- **Delegate protocol:** `InternetSpeedTestDelegate`
- **Initialization:** `InternetSpeedTest(delegate: self)`
- **Start test:** `internetTest.startFreeTest { error in ... }`

#### Delegate Callbacks

```
internetTestError(error: SpeedTestError)
internetTestFinish(result: SpeedTestResult)
internetTestReceived(servers: [SpeedTestServer])
internetTestSelected(server: SpeedTestServer, latency: Int, jitter: Int)
internetTestDownloadStart()
internetTestDownloadFinish()
internetTestDownload(progress: Double, speed: SpeedTestSpeed)
internetTestUploadStart()
internetTestUploadFinish()
internetTestUpload(progress: Double, speed: SpeedTestSpeed)
```

#### SpeedTestResult Properties

```
network: SpeedTestNetwork
server: SpeedTestServer
latencyInMs: Int
jitter: Double
downloadSpeed: SpeedTestSpeed (kbps, mbps)
uploadSpeed: SpeedTestSpeed (kbps, mbps)
ipAddress: String?
ispName: String?
date: Date?
timeToFirstByteMs: Int
downloadTransferredMb: Double
uploadTransferredMb: Double
packetLoss: SCPacketLoss?
testID: String
```

#### SpeedTestServer Properties

```
ID: Int?
domain: String?
country: String?
cityName: String?
countryCode: String?
```

### 4.2 Mapbox Search SDK

- **Package URL:** `https://github.com/mapbox/search-ios`
- **Products:** `MapboxSearch`
- **Key class:** `PlaceAutocomplete`
- **Requires:** Mapbox access token (stored in Mapbox.plist, gitignored)
- **Usage:** Forward geocoding via PlaceAutocomplete suggestions API

### 4.3 Decibel Metering (AVFoundation)

- No external SDK needed
- Uses `AVAudioRecorder` configured with:
  - Format: `.appleLossless`
  - Sample rate: 44100
  - Channels: 1
  - Quality: `.max`
- Enable metering: `recorder.isMeteringEnabled = true`
- Read levels: `recorder.averagePower(forChannel: 0)` and `recorder.peakPower(forChannel: 0)`
- Convert from dBFS to approximate dB SPL by adding offset (~96 dB)
- Sample every 0.1 seconds for ~5 seconds, compute average and peak

---

## 5. Data Models

### LocationCheckIn

```swift
struct LocationCheckIn: Codable, Identifiable {
    let id: UUID
    let userId: String
    let timestamp: Date

    // Location (from Mapbox)
    let locationName: String
    let locationAddress: String
    let latitude: Double
    let longitude: Double

    // Speed Test
    let downloadSpeedMbps: Double
    let uploadSpeedMbps: Double
    let latencyMs: Int
    let jitter: Double
    let ispName: String?
    let ipAddress: String?
    let serverDomain: String?
    let serverCountry: String?
    let serverCity: String?
    let networkType: String
    let packetLossPercent: Double?
    let timeToFirstByteMs: Int
    let speedTestId: String

    // Decibel
    let averageDecibelLevel: Double
    let peakDecibelLevel: Double

    // User Ratings
    let outletsAtBar: Bool
    let outletsAtTable: Bool
    let crowdedness: Int
    let easeOfWork: Int
    let bestWorkType: String
}
```

### SpeedTestResultData

```swift
struct SpeedTestResultData {
    var downloadSpeedMbps: Double = 0
    var uploadSpeedMbps: Double = 0
    var latencyMs: Int = 0
    var jitter: Double = 0
    var ispName: String?
    var ipAddress: String?
    var serverDomain: String?
    var serverCountry: String?
    var serverCity: String?
    var networkType: String = "unknown"
    var packetLossPercent: Double?
    var timeToFirstByteMs: Int = 0
    var downloadTransferredMb: Double = 0
    var uploadTransferredMb: Double = 0
    var speedTestId: String = ""
}
```

### DecibelMeasurement

```swift
struct DecibelMeasurement {
    var averageDecibels: Double = 0
    var peakDecibels: Double = 0
    var durationSeconds: Double = 0
}
```

---

## 6. Architecture and File Structure

```
HowsWorkingThere/
  Models/
    Models.swift                    (existing)
    LocationCheckIn.swift           (new)
  ViewModels/
    AuthViewModel.swift             (existing)
    AddLocationViewModel.swift      (new - flow orchestration)
    SpeedTestManager.swift          (new - SpeedChecker wrapper)
    DecibelMeterManager.swift       (new - AVAudioRecorder wrapper)
  Views/
    MainTabView.swift               (modified - wire new flow)
    AddLocationView.swift           (replaced by new flow)
    AddLocation/
      AddLocationFlowView.swift     (new - step container)
      LocationSearchStep.swift      (new - Mapbox search)
      SpeedTestStep.swift           (new - speed test UI)
      DecibelTestStep.swift         (new - noise meter UI)
      WorkspaceRatingsStep.swift    (new - ratings input)
      AddLocationReviewStep.swift   (new - summary/submit)
```

### MVVM Pattern

- **AddLocationViewModel** is an `ObservableObject` managing the entire flow state
- It owns `SpeedTestManager` and `DecibelMeterManager` as child managers
- Published properties drive the step views
- Each step view reads from and writes to the shared view model

---

## 7. View Specifications

All views use the existing retro theme from RetroTheme.swift (RetroColors, retroCard modifier, RetroSectionHeader, etc.).

### AddLocationFlowView
- Container with navigation between steps
- Step indicator at the top showing progress (5 dots/steps)
- Back/close button in toolbar
- Transitions between steps with animation

### LocationSearchStep
- Search bar at top
- Results list below with place name, address
- Retro-styled cards for each result
- "Confirm" button after selection

### SpeedTestStep
- Animated circular progress indicator
- Phase label: "Finding Server...", "Testing Download...", "Testing Upload..."
- Live speed readout in large text
- Summary card after completion showing download, upload, latency

### DecibelTestStep
- Animated vertical bar meter showing current dB level
- Timer showing remaining seconds
- Average and peak readouts
- Auto-advances when complete

### WorkspaceRatingsStep
- Toggle rows for outlet availability (retro-styled switches)
- Pill/chip selectors for crowdedness (with people icons)
- Pill/chip selectors for ease of work (with laptop icons)
- Pill/chip selectors for work type (with person icons)
- "Next" button at bottom

### AddLocationReviewStep
- Summary cards for each data category
- Location card: name, address, mini-map placeholder
- Network card: speed, latency, ISP
- Noise card: dB level with descriptor (Quiet/Moderate/Loud)
- Ratings card: all user selections
- "Submit" button (logs locally for now)

---

## 8. Error Handling

| Scenario | Handling |
|---|---|
| Location permission denied | Show alert explaining requirement, offer to open Settings |
| Microphone permission denied | Show alert, allow skipping decibel test |
| Speed test fails | Show error message, offer retry or skip |
| Mapbox search fails | Show error message, allow manual entry fallback |
| No network connection | Show error before starting flow |

---

## 9. Configuration

### Mapbox.plist

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>AccessToken</key>
    <string>YOUR_MAPBOX_ACCESS_TOKEN</string>
</dict>
</plist>
```

This file is added to .gitignore. A Mapbox.plist.example template is committed.

### Info.plist Keys (via Xcode build settings)

```
INFOPLIST_KEY_NSLocationWhenInUseUsageDescription = "Location access is needed to run speed tests and identify your workspace location."
INFOPLIST_KEY_NSMicrophoneUsageDescription = "Microphone access is needed to measure ambient noise levels at your workspace."
```

---

## 10. Future API Integration

The submission flow is stubbed with a protocol for future backend integration:

```swift
protocol LocationCheckInService {
    func submit(_ checkIn: LocationCheckIn) async throws
    func fetchExistingLocation(name: String, latitude: Double, longitude: Double) async throws -> LocationCheckIn?
}
```

A `LocalLocationCheckInService` implementation stores check-ins to UserDefaults/JSON for offline use. When the backend API is ready, a `RemoteLocationCheckInService` will replace it.
