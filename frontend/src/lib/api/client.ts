import axios from "axios";

const devServer = "http://localhost:3333";
const prodServer = "https://capyplan.up.railway.app";

export const api = axios.create({
  baseURL: `${prodServer}/api/v0`,
  headers: {
    "Content-Type": "application/json",
  },
});

export function authHeaders(token: string | null) {
  return token ? { Authorization: `Bearer ${token}` } : undefined;
}
