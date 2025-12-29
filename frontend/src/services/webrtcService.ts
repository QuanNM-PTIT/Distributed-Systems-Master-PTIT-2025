type SignalPayload = {
	type: string;
	to: string;
	groupId?: string;
	payload: Record<string, unknown> | null;
};

type SignalMessage = {
	type: string;
	from?: string;
	to: string;
	groupId?: string;
	payload?: any;
};

type PeerState = {
	peerId: string;
	pc: RTCPeerConnection;
	dc?: RTCDataChannel;
	pendingCandidates: RTCIceCandidateInit[];
	signalPrefix: "signal" | "group.signal";
	groupId?: string;
};

type EventHandler = (payload: any) => void;

type SignalingClient = {
  send: (message: SignalPayload) => void;
};

class WebRTCService {
  private peers = new Map<string, PeerState>();
  private signaling: SignalingClient | null = null;
  private handlers: Record<string, Set<EventHandler>> = {
    message: new Set(),
    status: new Set()
  };

  init(signaling: SignalingClient) {
    this.signaling = signaling;
  }

  on(event: "message" | "status", handler: EventHandler) {
    this.handlers[event].add(handler);
    return () => this.handlers[event].delete(handler);
  }

  private emit(event: "message" | "status", payload: any) {
    this.handlers[event].forEach((handler) => handler(payload));
  }

  private createPeer(peerId: string, initiator: boolean, options?: { signalPrefix?: "signal" | "group.signal"; groupId?: string }) {
    const pc = new RTCPeerConnection({
      iceServers: [{ urls: "stun:stun.l.google.com:19302" }]
    });

    const state: PeerState = {
      peerId,
      pc,
      pendingCandidates: [],
      signalPrefix: options?.signalPrefix || "signal",
      groupId: options?.groupId
    };

    pc.onicecandidate = (event) => {
      if (!event.candidate || !this.signaling) return;
      this.signaling.send({
        type: "signal.ice",
        to: peerId,
        payload: event.candidate.toJSON()
      });
    };

    pc.onconnectionstatechange = () => {
      this.emit("status", { peerId, state: pc.connectionState });
    };

    pc.ondatachannel = (event) => {
      state.dc = event.channel;
      this.attachDataChannel(peerId, state.dc);
    };

    if (initiator) {
      const dc = pc.createDataChannel("chat", { ordered: true });
      state.dc = dc;
      this.attachDataChannel(peerId, dc);
    }

    this.peers.set(peerId, state);
    return state;
  }

  private attachDataChannel(peerId: string, dc: RTCDataChannel) {
    dc.onopen = () => this.emit("status", { peerId, state: "channel-open" });
    dc.onclose = () => this.emit("status", { peerId, state: "channel-closed" });
    dc.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        this.emit("message", { peerId, data });
      } catch {
        this.emit("message", { peerId, data: event.data });
      }
    };
  }

  async connect(peerId: string) {
    const state = this.peers.get(peerId) || this.createPeer(peerId, true);
    const offer = await state.pc.createOffer();
    await state.pc.setLocalDescription(offer);
    if (this.signaling) {
      this.signaling.send({
        type: `${state.signalPrefix}.offer`,
        to: peerId,
        groupId: state.groupId,
        payload: offer
      });
    }
  }

  async connectGroup(peerId: string, groupId: string) {
    const state = this.peers.get(peerId) || this.createPeer(peerId, true, { signalPrefix: "group.signal", groupId });
    state.signalPrefix = "group.signal";
    state.groupId = groupId;
    const offer = await state.pc.createOffer();
    await state.pc.setLocalDescription(offer);
    if (this.signaling) {
      this.signaling.send({
        type: "group.signal.offer",
        to: peerId,
        groupId,
        payload: offer
      });
    }
  }

  async handleSignal(msg: SignalMessage) {
    if (!msg.type || !msg.to) return;
    const peerId = msg.from || msg.to;
    if (!peerId) return;

    const isGroup = msg.type.startsWith("group.signal.");
    const prefix = isGroup ? "group.signal" : "signal";

    if (msg.type === "signal.offer" || msg.type === "group.signal.offer") {
      const state = this.peers.get(peerId) || this.createPeer(peerId, false, { signalPrefix: prefix, groupId: msg.groupId });
      state.signalPrefix = prefix as "signal" | "group.signal";
      state.groupId = msg.groupId;
      await state.pc.setRemoteDescription(new RTCSessionDescription(msg.payload));
      const answer = await state.pc.createAnswer();
      await state.pc.setLocalDescription(answer);
      if (this.signaling) {
        this.signaling.send({
          type: `${state.signalPrefix}.answer`,
          to: peerId,
          groupId: state.groupId,
          payload: answer
        });
      }
      await this.flushPendingIce(state);
      return;
    }

    if (msg.type === "signal.answer" || msg.type === "group.signal.answer") {
      const state = this.peers.get(peerId);
      if (!state) return;
      await state.pc.setRemoteDescription(new RTCSessionDescription(msg.payload));
      await this.flushPendingIce(state);
      return;
    }

    if (msg.type === "signal.ice" || msg.type === "group.signal.ice") {
      const state = this.peers.get(peerId) || this.createPeer(peerId, false, { signalPrefix: prefix, groupId: msg.groupId });
      state.signalPrefix = prefix as "signal" | "group.signal";
      state.groupId = msg.groupId;
      if (state.pc.remoteDescription) {
        await state.pc.addIceCandidate(new RTCIceCandidate(msg.payload));
      } else {
        state.pendingCandidates.push(msg.payload);
      }
    }
  }

  private async flushPendingIce(state: PeerState) {
    while (state.pendingCandidates.length) {
      const candidate = state.pendingCandidates.shift();
      if (candidate) {
        await state.pc.addIceCandidate(new RTCIceCandidate(candidate));
      }
    }
  }

  send(peerId: string, payload: Record<string, unknown>) {
    const state = this.peers.get(peerId);
    if (!state?.dc || state.dc.readyState !== "open") return false;
    state.dc.send(JSON.stringify(payload));
    return true;
  }

  disconnect(peerId: string) {
    const state = this.peers.get(peerId);
    if (!state) return;
    state.dc?.close();
    state.pc.close();
    this.peers.delete(peerId);
  }

  disconnectAll() {
    for (const [peerId, state] of this.peers.entries()) {
      state.dc?.close();
      state.pc.close();
      this.peers.delete(peerId);
    }
  }
}

export default new WebRTCService();
