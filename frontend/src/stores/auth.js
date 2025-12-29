import { defineStore } from "pinia";
import api from "../services/api";
import signaling from "../services/signaling";
import webrtc from "../services/webrtcService";

export const useAuthStore = defineStore("auth", {
  state: () => ({
    token: "",
    userId: "",
    bootstrapped: false
  }),
  getters: {
    isAuthenticated: (state) => Boolean(state.token)
  },
  actions: {
    restore() {
      if (this.bootstrapped) return;
      this.token = localStorage.getItem("token") || "";
      this.userId = localStorage.getItem("userId") || "";
      if (this.token) {
        signaling.connect(this.token);
      }
      this.bootstrapped = true;
    },
    async login(payload) {
      const res = await api.post("/auth/login", payload);
      this.setSession(res.data);
    },
    async register(payload) {
      const res = await api.post("/auth/register", payload);
      this.setSession(res.data);
    },
    setSession({ accessToken, userId }) {
      this.token = accessToken;
      this.userId = userId;
      localStorage.setItem("token", accessToken);
      localStorage.setItem("userId", userId);
      signaling.connect(accessToken);
    },
    logout() {
      this.token = "";
      this.userId = "";
      localStorage.removeItem("token");
      localStorage.removeItem("userId");
      webrtc.disconnectAll();
      signaling.disconnect();
    }
  }
});
