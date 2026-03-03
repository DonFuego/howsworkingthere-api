# Authentication Spec — How's Working There (iOS)

> **Provider:** Auth0  
> **SDK:** [Auth0.swift](https://github.com/auth0/Auth0.swift) (latest, via Swift Package Manager)  
> **Minimum iOS:** 17.0  

---

## Table of Contents

1. [Overview](#1-overview)
2. [Auth0 Dashboard Configuration](#2-auth0-dashboard-configuration)
3. [iOS Project Configuration](#3-ios-project-configuration)
4. [Authentication Flows](#4-authentication-flows)
5. [Token & Credential Storage](#5-token--credential-storage)
6. [Session & User ID Management](#6-session--user-id-management)
7. [Security Best Practices](#7-security-best-practices)
8. [Code Impact & Architecture](#8-code-impact--architecture)
9. [Error Handling](#9-error-handling)
10. [Testing Strategy](#10-testing-strategy)

---

## 1. Overview

The app uses Auth0 as its sole authentication provider. Users can:

- **Sign up / Log in** with email and password
- **Reset a forgotten password** via email
- **Sign in with Apple**
- **Sign in with Google**

After authentication, the user's Auth0 `sub` claim is stored as the canonical **user ID** and attached to every API interaction (saving locations, reviews, friend activity, etc.).

---

## 2. Auth0 Dashboard Configuration

### 2.1 Create an Application

| Setting | Value |
|---|---|
| **Application Type** | Native |
| **Name** | How's Working There (iOS) |
| **Allowed Callback URLs** | `https://YOUR_AUTH0_DOMAIN/ios/YOUR_BUNDLE_ID/callback` |
| **Allowed Logout URLs** | `https://YOUR_AUTH0_DOMAIN/ios/YOUR_BUNDLE_ID/callback` |
| **Token Endpoint Auth Method** | None (public client) |

> Replace `YOUR_AUTH0_DOMAIN` with your tenant domain (e.g. `dev-xxxx.us.auth0.com`) and `YOUR_BUNDLE_ID` with the app's bundle identifier (e.g. `com.howsworkingthere.ios`).

### 2.2 Configuration Items

These are the values required from the Auth0 dashboard:

| Config Item | Where to Find | Purpose |
|---|---|---|
| **Domain** | Application → Settings → Domain | Base URL for all Auth0 API calls |
| **Client ID** | Application → Settings → Client ID | Identifies this native app to Auth0 |
| **Database Connection** | Authentication → Database → Username-Password-Authentication | Email/password login & signup |
| **Apple Social Connection** | Authentication → Social → Apple | Sign in with Apple |
| **Google Social Connection** | Authentication → Social → Google / Gmail | Sign in with Google |

### 2.3 Enable Social Connections

#### Sign in with Apple

1. Go to **Authentication → Social → Apple** in the Auth0 dashboard.
2. Enable the connection.
3. Provide the required Apple Developer configuration:
   - **Services ID** (from Apple Developer → Certificates, Identifiers & Profiles)
   - **Apple Team ID**
   - **Key ID** and **Private Key** (.p8 file from Apple)
4. Map scopes: `name`, `email`.
5. Enable this connection for the "How's Working There (iOS)" application.

#### Sign in with Google

1. Go to **Authentication → Social → Google / Gmail** in the Auth0 dashboard.
2. Enable the connection.
3. Create OAuth 2.0 credentials in the [Google Cloud Console](https://console.cloud.google.com/):
   - Create an **OAuth Client ID** (application type: iOS).
   - Provide the app's bundle identifier.
4. Enter the **Client ID** and **Client Secret** from Google into Auth0.
5. Map scopes: `openid`, `profile`, `email`.
6. Enable this connection for the "How's Working There (iOS)" application.

### 2.4 Enable Forgot Password / Change Password

Auth0's built-in **Database Connection** supports password reset out of the box:

- **Authentication → Database → Username-Password-Authentication → Settings**
- Ensure "Disable Sign Ups" is **off** (users can self-register).
- Customize the **Change Password** email template under **Branding → Email Templates → Change Password**.
- Optionally configure a custom password policy (minimum length, complexity rules).

---

## 3. iOS Project Configuration

### 3.1 Add Auth0.swift via Swift Package Manager

In Xcode:

1. **File → Add Package Dependencies…**
2. Enter the URL: `https://github.com/auth0/Auth0.swift`
3. Set the dependency rule to **Up to Next Major Version** from the latest release.
4. Add the `Auth0` library to the `HowsWorkingThere` target.

### 3.2 Create `Auth0.plist`

Add a property list file named `Auth0.plist` to the app target:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>ClientId</key>
    <string>YOUR_AUTH0_CLIENT_ID</string>
    <key>Domain</key>
    <string>YOUR_AUTH0_DOMAIN</string>
</dict>
</plist>
```

> **Do not commit real credentials.** Use a `.plist.example` for version control and add the real `Auth0.plist` to `.gitignore`.

### 3.3 Configure Universal Links (Callback URL)

Auth0.swift uses Universal Links on iOS 17.4+ for secure callback handling.

1. **Xcode → Target → Signing & Capabilities → Associated Domains**
2. Add: `webcredentials:YOUR_AUTH0_DOMAIN`
3. Add: `applinks:YOUR_AUTH0_DOMAIN`

This enables the SDK to call `.useHTTPS()` on Web Auth flows, which is required for the most secure callback URL handling (no custom URL schemes).

### 3.4 Sign in with Apple Capability

1. **Xcode → Target → Signing & Capabilities → + Capability → Sign in with Apple**
2. This registers the entitlement in the app's provisioning profile.

---

## 4. Authentication Flows

### 4.1 Email & Password — Login

Uses the Auth0 Authentication API (Resource Owner Password Grant):

```swift
import Auth0

Auth0
    .authentication()
    .login(
        usernameOrEmail: email,
        password: password,
        realmOrConnection: "Username-Password-Authentication",
        scope: "openid profile email offline_access"
    )
    .start { result in
        switch result {
        case .success(let credentials):
            // Store credentials (see Section 5)
            // Extract user ID (see Section 6)
        case .failure(let error):
            // Handle error (see Section 9)
        }
    }
```

**Scopes requested:**

| Scope | Purpose |
|---|---|
| `openid` | Returns an ID token (JWT) with the user's `sub` claim |
| `profile` | Includes `name`, `nickname`, `picture` in the ID token |
| `email` | Includes `email` and `email_verified` in the ID token |
| `offline_access` | Returns a refresh token for silent session renewal |

> **Note:** The Resource Owner Password Grant must be enabled for the database connection in the Auth0 dashboard under **Application → Settings → Advanced → Grant Types**. Check the "Password" grant.

### 4.2 Email & Password — Sign Up

```swift
Auth0
    .authentication()
    .createUser(
        email: email,
        password: password,
        connection: "Username-Password-Authentication",
        userMetadata: ["name": name]
    )
    .start { result in
        switch result {
        case .success(let user):
            // User created; now log them in automatically via 4.1
        case .failure(let error):
            // Handle error (duplicate email, weak password, etc.)
        }
    }
```

After successful creation, immediately call the login flow (Section 4.1) so the user gets tokens and enters the app.

### 4.3 Forgot Password

Triggers Auth0's password reset email:

```swift
Auth0
    .authentication()
    .resetPassword(
        email: email,
        connection: "Username-Password-Authentication"
    )
    .start { result in
        switch result {
        case .success:
            // Show confirmation: "Check your email for a reset link"
        case .failure(let error):
            // Handle error
        }
    }
```

The user receives an email with a link to Auth0's hosted Change Password page. No in-app password change form is needed.

### 4.4 Sign in with Apple

Uses Auth0 Web Auth to handle the Apple OAuth flow:

```swift
Auth0
    .webAuth()
    .useHTTPS()
    .connection("apple")
    .scope("openid profile email offline_access")
    .start { result in
        switch result {
        case .success(let credentials):
            // Store credentials & extract user ID
        case .failure(let error):
            // Handle error or user cancellation
        }
    }
```

Alternatively, for a native Apple Sign In experience using `ASAuthorizationController`:

```swift
// After obtaining the Apple authorization code from ASAuthorizationController:
Auth0
    .authentication()
    .login(appleAuthorizationCode: authorizationCode)
    .start { result in
        switch result {
        case .success(let credentials):
            // Store credentials & extract user ID
        case .failure(let error):
            // Handle error
        }
    }
```

The native approach provides a more seamless UX (Face ID / Touch ID prompt with no browser redirect), but requires the "Sign in with Apple" capability and more boilerplate via `AuthenticationServices`.

### 4.5 Sign in with Google

Uses Auth0 Web Auth to handle the Google OAuth flow:

```swift
Auth0
    .webAuth()
    .useHTTPS()
    .connection("google-oauth2")
    .scope("openid profile email offline_access")
    .start { result in
        switch result {
        case .success(let credentials):
            // Store credentials & extract user ID
        case .failure(let error):
            // Handle error or user cancellation
        }
    }
```

This opens a secure in-app browser (ASWebAuthenticationSession) for the Google consent screen, then returns credentials via the Universal Link callback.

---

## 5. Token & Credential Storage

### 5.1 Overview

All tokens are stored using Auth0's **`CredentialsManager`**, which wraps the iOS **Keychain**. Nothing is stored in `UserDefaults`, files, or plain text.

| Token | Purpose | Lifetime |
|---|---|---|
| **Access Token** | Authorize API requests | Short-lived (default 24h, configurable in Auth0) |
| **ID Token** | Contains user profile claims (`sub`, `email`, `name`) | Short-lived, same as access token |
| **Refresh Token** | Silently obtain new access/ID tokens without re-login | Long-lived (configurable; rotation recommended) |

### 5.2 CredentialsManager Setup

```swift
import Auth0

let credentialsManager = CredentialsManager(authentication: Auth0.authentication())
```

The `CredentialsManager` should be created once and shared across the app (e.g., as a property on `AuthViewModel` or injected via the environment).

### 5.3 Storing Credentials

After any successful login (email/password, Apple, Google):

```swift
let stored = credentialsManager.store(credentials: credentials)
// stored == true if Keychain write succeeded
```

### 5.4 Retrieving Credentials (with Auto-Renewal)

On app launch or when an API call is needed:

```swift
credentialsManager.credentials { result in
    switch result {
    case .success(let credentials):
        // credentials.accessToken is valid (renewed if it was expired)
        // Proceed with API call
    case .failure(let error):
        // No valid session; redirect to login
    }
}
```

`CredentialsManager.credentials()` automatically uses the stored refresh token to renew expired access tokens. If the refresh token is also invalid, it returns a failure, and the user must log in again.

### 5.5 Checking for Existing Session

```swift
guard credentialsManager.canRenew() else {
    // No stored credentials or no refresh token — show login
    return
}
```

Use this on app launch to decide whether to show the login screen or skip to the main app.

### 5.6 Optional: Biometric Protection

To require Face ID / Touch ID before retrieving tokens from Keychain:

```swift
credentialsManager.enableBiometrics(
    withTitle: "Unlock How's Working There",
    cancelTitle: "Cancel",
    fallbackTitle: "Use Passcode"
)
```

When enabled, `credentialsManager.credentials()` will trigger a biometric prompt before returning tokens. This adds a second layer of protection if the device is compromised.

### 5.7 Clearing Credentials (Logout)

```swift
let didClear = credentialsManager.clear()
```

This removes all tokens from the Keychain. For social logins, also clear the Auth0 session cookie:

```swift
Auth0
    .webAuth()
    .useHTTPS()
    .clearSession { result in
        switch result {
        case .success:
            let _ = credentialsManager.clear()
            // Navigate to login screen
        case .failure(let error):
            // Handle error; still clear local credentials
            let _ = credentialsManager.clear()
        }
    }
```

---

## 6. Session & User ID Management

### 6.1 Extracting the User ID

The Auth0 **ID Token** is a JWT containing a `sub` (subject) claim. This is the unique, stable user identifier across all Auth0 connections.

Example `sub` values:
- Email/password: `auth0|64a1b2c3d4e5f6g7h8i9j0`
- Apple: `apple|001234.abcdef1234567890.1234`
- Google: `google-oauth2|123456789012345678901`

Extract it after login:

```swift
import JWTDecode  // Included with Auth0.swift

func extractUserId(from credentials: Credentials) -> String? {
    guard let jwt = try? decode(jwt: credentials.idToken) else { return nil }
    return jwt.subject  // The "sub" claim
}
```

### 6.2 Storing User ID in the App Session

The `AuthViewModel` holds the current user ID for the lifetime of the session:

```swift
class AuthViewModel: ObservableObject {
    @Published var isAuthenticated = false
    @Published var currentUserId: String?
    @Published var userName: String?
    @Published var userEmail: String?
    
    private let credentialsManager = CredentialsManager(
        authentication: Auth0.authentication()
    )
    
    // Called after any successful login
    func handleCredentials(_ credentials: Credentials) {
        guard credentialsManager.store(credentials: credentials) else { return }
        
        if let jwt = try? decode(jwt: credentials.idToken) {
            currentUserId = jwt.subject
            userName = jwt["name"].string
            userEmail = jwt["email"].string
        }
        
        isAuthenticated = true
    }
}
```

### 6.3 Using the User ID in API Calls

All API interactions that create or query user-owned data must include the user ID:

```swift
// Example: saving a new location
func saveLocation(_ location: NewLocation) async throws {
    guard let userId = authViewModel.currentUserId else {
        throw AppError.notAuthenticated
    }
    
    var request = URLRequest(url: apiURL.appendingPathComponent("/locations"))
    request.httpMethod = "POST"
    request.setValue("Bearer \(accessToken)", forHTTPHeaderField: "Authorization")
    
    let body = SaveLocationRequest(
        userId: userId,
        name: location.name,
        city: location.city,
        country: location.country,
        rating: location.rating
    )
    request.httpBody = try JSONEncoder().encode(body)
    
    let (_, response) = try await URLSession.shared.data(for: request)
    // Handle response...
}
```

> **Important:** The backend should **also** validate the user ID from the access token's `sub` claim (server-side). The client-sent `userId` in the body is a convenience; the backend must treat the token's `sub` as authoritative to prevent spoofing.

### 6.4 Session Restoration on App Launch

On app start, attempt to restore the session before showing any UI:

```swift
// In HowsWorkingThereApp.swift or AuthViewModel.init
func restoreSession() {
    guard credentialsManager.canRenew() else {
        isAuthenticated = false
        return
    }
    
    credentialsManager.credentials { [weak self] result in
        DispatchQueue.main.async {
            switch result {
            case .success(let credentials):
                self?.handleCredentials(credentials)
            case .failure:
                self?.isAuthenticated = false
            }
        }
    }
}
```

---

## 7. Security Best Practices

### 7.1 PKCE (Proof Key for Code Exchange)

All Auth0 Web Auth flows (Apple, Google) use **PKCE** by default in the Auth0.swift SDK. This prevents authorization code interception attacks. No additional configuration is needed — the SDK generates a `code_verifier` and `code_challenge` automatically.

### 7.2 HTTPS-Only Callbacks

Always use `.useHTTPS()` on Web Auth calls. This uses Universal Links instead of custom URL schemes, preventing other apps from intercepting the callback:

```swift
Auth0.webAuth().useHTTPS()  // Required on every webAuth() call
```

### 7.3 Token Rotation

Enable **Refresh Token Rotation** in the Auth0 dashboard:

- **Application → Settings → Refresh Token Rotation → Enabled**
- Set **Reuse Interval** to a small window (e.g., 3 seconds) to handle network retries.

With rotation, each use of a refresh token issues a new refresh token and invalidates the old one. If a stolen refresh token is replayed, Auth0 detects the reuse and revokes the entire token family.

### 7.4 Refresh Token Expiration

Configure absolute and inactivity expiration:

- **Application → Settings → Refresh Token Expiration**
  - **Absolute Lifetime:** 30 days (user must re-authenticate after this period)
  - **Inactivity Lifetime:** 7 days (token expires if unused for 7 days)

### 7.5 Keychain Security

`CredentialsManager` stores tokens in the iOS Keychain with the following protections:

- **Encrypted at rest** by the Secure Enclave.
- **Scoped to the app** via the app's Keychain Access Group (default: app bundle ID).
- **Not included in backups** (tokens are not synced to iCloud Keychain or iTunes backups).
- **Biometric gating** (optional, see Section 5.6) adds an additional unlock requirement.

### 7.6 No Sensitive Data in Logs

Never log access tokens, refresh tokens, or ID tokens in production:

```swift
#if DEBUG
print("Token: \(credentials.accessToken)")  // OK in debug only
#endif
```

### 7.7 Input Validation

Before sending credentials to Auth0:

| Field | Validation |
|---|---|
| **Email** | Non-empty, valid email format (RFC 5322) |
| **Password** | Non-empty, minimum 8 characters (match Auth0 password policy) |
| **Name** | Non-empty, trimmed whitespace, max 100 characters |

Perform client-side validation to provide immediate feedback; Auth0 enforces its own server-side rules as well.

### 7.8 Rate Limiting & Brute Force Protection

Enable **Brute Force Protection** and **Suspicious IP Throttling** in the Auth0 dashboard:

- **Security → Attack Protection → Brute Force Protection → Enabled**
- **Security → Attack Protection → Suspicious IP Throttling → Enabled**

These are server-side protections that block repeated failed login attempts.

### 7.9 Secure Logout

A complete logout must:

1. Clear the Auth0 session cookie (for Web Auth flows): `Auth0.webAuth().useHTTPS().clearSession()`
2. Clear all tokens from Keychain: `credentialsManager.clear()`
3. Reset all in-memory state: `currentUserId = nil`, `isAuthenticated = false`
4. Navigate to the login screen.

### 7.10 Certificate Pinning (Optional)

For high-security deployments, consider enabling SSL certificate pinning for Auth0 API calls using `URLSessionDelegate` or a library like TrustKit. This prevents MITM attacks even if a rogue CA certificate is installed on the device.

---

## 8. Code Impact & Architecture

### 8.1 New Files

| File | Purpose |
|---|---|
| `Auth0.plist` | Auth0 client ID and domain configuration |
| `Auth0.plist.example` | Template for version control (real plist in `.gitignore`) |

### 8.2 Modified Files

| File | Changes |
|---|---|
| `HowsWorkingThereApp.swift` | Initialize `CredentialsManager`, call `restoreSession()` on launch |
| `AuthViewModel.swift` | Replace mock login/signup/logout with real Auth0 calls; add `currentUserId`, `userEmail`, `credentialsManager`; add `handleCredentials()`, `restoreSession()`, `resetPassword()` methods |
| `SplashLoginView.swift` | Wire "Forgot Password?" button to `resetPassword()`; wire social buttons to `loginWithApple()` / `loginWithGoogle()` on the view model |
| `ContentView.swift` | Show loading state while `restoreSession()` runs |
| `ProfileView.swift` | Display real user name/email from `AuthViewModel`; ensure logout calls the full secure logout flow |
| `Models.swift` | Add a `User` model if needed for profile data |
| `.gitignore` | Add `Auth0.plist` |

### 8.3 Dependency

| Package | URL | Version |
|---|---|---|
| Auth0.swift | `https://github.com/auth0/Auth0.swift` | Latest major (via SPM) |

The `Auth0` package includes `JWTDecode` as a transitive dependency, so no separate JWT library is needed.

### 8.4 Architecture Diagram

```
┌──────────────────────────────────────────────────┐
│                  SwiftUI Views                    │
│  SplashLoginView  │  ContentView  │  ProfileView │
└────────┬──────────┴───────┬───────┴──────┬───────┘
         │                  │              │
         ▼                  ▼              ▼
┌──────────────────────────────────────────────────┐
│                  AuthViewModel                    │
│  @Published isAuthenticated                       │
│  @Published currentUserId                         │
│  @Published userName / userEmail                  │
│                                                   │
│  login(email:password:)                           │
│  signUp(name:email:password:)                     │
│  loginWithApple()                                 │
│  loginWithGoogle()                                │
│  resetPassword(email:)                            │
│  restoreSession()                                 │
│  logout()                                         │
└────────┬─────────────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────────────┐
│              Auth0.swift SDK                      │
│  Authentication API  │  Web Auth  │  JWTDecode   │
└────────┬─────────────┴─────┬─────┴───────────────┘
         │                   │
         ▼                   ▼
┌──────────────────┐  ┌────────────────────────────┐
│ CredentialsManager│  │  ASWebAuthenticationSession │
│ (iOS Keychain)   │  │  (Social Login Browser)     │
└──────────────────┘  └────────────────────────────┘
```

---

## 9. Error Handling

### 9.1 Auth0 Error Types

The Auth0.swift SDK returns typed errors. Key cases to handle:

| Error | User-Facing Message | Action |
|---|---|---|
| `invalid_grant` (wrong password) | "Incorrect email or password." | Show inline error, stay on login |
| `too_many_attempts` | "Too many attempts. Try again later." | Disable login button temporarily |
| `unauthorized` (email not verified) | "Please verify your email first." | Show re-send verification option |
| `password_leaked` | "This password has appeared in a data breach." | Prompt user to reset password |
| `invalid_signup` (duplicate email) | "An account with this email already exists." | Suggest login or password reset |
| `password_strength_error` | "Password doesn't meet requirements." | Show password policy hint |
| User cancelled (social login) | No message | Silently return to login screen |
| Network error | "No internet connection. Please try again." | Show retry option |

### 9.2 Error Display

Errors should appear as inline messages below the relevant form field or as a banner at the top of the login card, styled to match the retro theme.

---

## 10. Testing Strategy

### 10.1 Unit Tests

- **AuthViewModel:** Test `handleCredentials()` correctly extracts user ID, name, email from a known JWT.
- **Input validation:** Test email format validation, password length checks.
- **Session restore:** Mock `CredentialsManager` to test both "has session" and "no session" paths.

### 10.2 Integration Tests

- **Login flow:** Verify email/password login returns valid credentials (use an Auth0 test user).
- **Sign up flow:** Verify new user creation and subsequent login.
- **Password reset:** Verify the reset request completes without error.
- **Token renewal:** Verify `CredentialsManager.credentials()` renews an expired access token.

### 10.3 UI Tests

- **Login screen:** Verify form validation messages appear for empty fields, invalid email.
- **Social buttons:** Verify Apple and Google buttons trigger the correct Web Auth flow.
- **Logout:** Verify tapping logout returns to the login screen and clears the session.

### 10.4 Auth0 Test Users

Create dedicated test users in the Auth0 dashboard (or via the Management API) for automated testing. Do not use production accounts.
