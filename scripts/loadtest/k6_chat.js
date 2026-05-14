import http from "k6/http";
import { check, sleep } from "k6";

export const options = {
  vus: 10,
  duration: "30s",
};

const base = __ENV.SUB2API_URL || "http://127.0.0.1:3000";
const key = __ENV.SUB2API_KEY || "";

export default function () {
  if (!key) {
    return;
  }
  const payload = JSON.stringify({
    model: "deepseek-chat",
    messages: [{ role: "user", content: "ping" }],
    stream: false,
  });
  const res = http.post(`${base.replace(/\/+$/, "")}/v1/chat/completions`, payload, {
    headers: {
      Authorization: `Bearer ${key}`,
      "Content-Type": "application/json",
    },
    timeout: "60s",
  });
  check(res, {
    "2xx": (r) => r.status >= 200 && r.status < 300,
  });
  sleep(0.15);
}

export function setup() {
  if (!key) {
    throw new Error("Set SUB2API_KEY to a raw API key before running load test.");
  }
}
