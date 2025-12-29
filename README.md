# P2P Chat App (Hybrid P2P)

## Architecture Summary
- Hybrid P2P: server handles auth, friends, groups, presence, and WebRTC signaling; data (chat/file) goes P2P via WebRTC DataChannel.
- Backend: Golang (Gin) + MySQL for identity, social graph, groups, and signaling authorization.
- Frontend: Vue 3 (Vite + Router + Pinia) with WebRTC + WebSocket signaling (to be added in later milestones).

## End-to-End Flow (Target)
1. Login/Register via REST -> JWT issued.
2. Add friend by userId -> accept -> friend edge stored.
3. Client connects WebSocket `/ws` with JWT -> signaling offer/answer/ICE exchanged.
4. WebRTC PeerConnection + DataChannel established -> P2P messaging + file transfer.
5. Group chat uses mesh DataChannels (small group limit) -> Lamport clock ordering + dedupe.

## Distributed Systems Concepts
- Peer-to-peer overlay with hybrid coordination via server.
- Message passing and group communication over DataChannel.
- Logical ordering with Lamport clocks.
- Secure transport: DTLS for WebRTC, JWT for signaling.

## Milestone 1 Status
- Backend skeleton (Gin, MySQL wiring, middleware)
- Auth (register/login) with bcrypt + JWT
- DB schema migrations
- WebSocket signaling skeleton

## Run (Backend)
- Configure MySQL and set `DB_DSN` (see `backend/.env.example`).
- Apply SQL schema from `backend/migrations/001_init.sql`.
- Start server: `go run ./backend/cmd/server`

Frontend will be added in a later milestone.
