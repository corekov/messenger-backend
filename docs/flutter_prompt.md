# Flutter Messenger App Prompt

**You are an expert Flutter and Dart developer, software architect, and mobile security specialist.** Your task is to design and build a secure, real-time messenger mobile application (iOS and Android) using Flutter.

## Backend Context & Architecture
A production-ready Go backend is already developed and running. You must build the Flutter frontend to consume this API perfectly. Here is how the backend operates:

### 1. Authentication (JWT)
- **Mechanism:** The backend uses standard JWT-based authentication. 
- **Endpoints:** `/api/v1/auth/register`, `/api/v1/auth/login`, `/api/v1/auth/refresh`, `/api/v1/auth/logout`.
- **Flow:** Login/Registration returns an `access_token` (short-lived) and a `refresh_token` (long-lived).
- **Client Duty:** The Flutter app must securely store these tokens (e.g., using `flutter_secure_storage`). Every API request to protected routes must include the header: `Authorization: Bearer <access_token>`. The app must automatically intercept 401 Unauthorized errors, use the refresh token to get a new access token, and retry the failed request.

### 2. REST API (Users & Messaging)
- **Base URL:** `http://localhost:8080/api/v1` (configurable via env).
- **Users:** Endpoints exist to fetch the current user's profile and search for other users by username.
- **Chats & Messages:** Endpoints exist to fetch a list of active 1-on-1 and group chats, fetch message history for a specific chat with pagination, mark messages as read, and delete or leave a chat.

### 3. Real-Time Communication (WebSocket)
- **Connection:** The app must establish a persistent WebSocket connection to the backend after logging in.
- **Payloads:** The WebSocket handles JSON payloads with a specific `type` field (e.g., `chat_message`, `typing_indicator`, `user_presence`).
- **Client Duty:** The app must listen for incoming messages and update the UI instantly without needing a pull-to-refresh. It should also send "typing..." indicators over the WebSocket when the user types.

### 4. End-to-End Encryption (E2EE) Readiness
- The backend is designed as a "zero-knowledge" server. It does not want to read the message content.
- **Client Duty:** The Flutter app must implement client-side key generation. Before sending a message over the WebSocket or REST API, the Flutter app must encrypt the payload. When receiving a message, the app must decrypt it locally before displaying it in the UI.

## Requirements for the Flutter MVP
1. **State Management:** Use a modern, robust state management solution (e.g., Riverpod or BLoC).
2. **Routing:** Use `go_router` for deep linking and declarative navigation.
3. **UI/UX:** Implement a premium, responsive dark-mode UI with smooth micro-animations. It should feel like a modern, native application (similar to Telegram or Signal).
4. **Local Database (Optional but recommended):** Use `sqflite` or `isar` to cache messages locally so the app works offline or on poor connections.

## Your First Task
Do not write the entire application at once. Let's work step-by-step.
**Step 1:** Outline the folder architecture for the Flutter project, recommend the best packages to use for HTTP requests, WebSockets, Local Storage, and State Management, and provide the initial code for the Authentication Service that handles the JWT login/refresh cycle securely.

Please begin with Step 1.
