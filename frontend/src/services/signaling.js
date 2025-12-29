class SignalingClient {
  constructor() {
    this.socket = null;
    this.connected = false;
    this.listeners = new Set();
    this.reconnectTimer = null;
    this.token = "";
  }

  connect(token) {
    if (this.socket && this.connected) return;
    this.token = token;
    const url = `ws://localhost:8080/ws?token=${token}`;
    this.socket = new WebSocket(url);

    this.socket.onopen = () => {
      this.connected = true;
      this.emit({ type: "ws.connected" });
    };

    this.socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.emit(data);
      } catch {
        this.emit({ type: "ws.invalid", raw: event.data });
      }
    };

    this.socket.onclose = () => {
      this.connected = false;
      this.emit({ type: "ws.disconnected" });
      this.scheduleReconnect();
    };
  }

  scheduleReconnect() {
    if (this.reconnectTimer || !this.token) return;
    this.reconnectTimer = window.setTimeout(() => {
      this.reconnectTimer = null;
      this.connect(this.token);
    }, 2000);
  }

  send(message) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) return;
    this.socket.send(JSON.stringify(message));
  }

  onMessage(handler) {
    this.listeners.add(handler);
    return () => this.listeners.delete(handler);
  }

  emit(message) {
    this.listeners.forEach((handler) => handler(message));
  }

  disconnect() {
    this.token = "";
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.socket) {
      this.socket.close();
    }
    this.socket = null;
    this.connected = false;
  }
}

export default new SignalingClient();
