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

## Start Project (Step-by-step)

### 1. MySQL setup
- Create database (example): `CREATE DATABASE p2p_chat;`
- Apply schema from `backend/migrations/001_init.sql`
- Update `DB_DSN` in `.env` or environment variables (example below)

Example `DB_DSN`:
```
root:password@tcp(127.0.0.1:3306)/p2p_chat?parseTime=true
```

### 2. Backend (Go + Gin)
```
cd backend
go mod download
go mod tidy
export DB_DSN="root:password@tcp(127.0.0.1:3306)/p2p_chat?parseTime=true"
export JWT_SECRET="change_me"
export ALLOWED_ORIGIN="http://localhost:5173"
go run ./cmd/server
```

### 3. Frontend (Vue 3 + Vite)
```
cd frontend
npm install
npm run dev
```

Open:
- Frontend: `http://localhost:5173`
- Backend: `http://localhost:8080`
