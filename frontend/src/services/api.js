import axios from "axios";
import { useAuthStore } from "../stores/auth";

const api = axios.create({
  baseURL: "http://localhost:8080/api/v1"
});

api.interceptors.request.use((config) => {
  const auth = useAuthStore();
  if (auth.token) {
    config.headers.Authorization = `Bearer ${auth.token}`;
  }
  return config;
});

export default api;
