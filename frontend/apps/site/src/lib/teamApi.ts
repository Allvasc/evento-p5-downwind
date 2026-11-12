import { createApiClient, ApiError } from "@p5wellness/shared";

export { ApiError };

export const TEAM_TOKEN_KEY = "p5_team_token";

// Evento escolhido no seletor do Admin / check-in. Vai em toda chamada da equipe (header
// X-Event-Id) e o backend filtra tudo por ele — listas, relatórios, check-in e vouchers
// de um evento nunca aparecem no outro.
export const TEAM_EVENT_KEY = "p5_team_event_id";
export const EVENT_HEADER = "X-Event-Id";

export function eventHeaders(): Record<string, string> {
  const id = localStorage.getItem(TEAM_EVENT_KEY);
  return id ? { [EVENT_HEADER]: id } : {};
}

export const api = createApiClient({
  baseUrl: "/api/v1",
  getToken: () => localStorage.getItem(TEAM_TOKEN_KEY),
  getHeaders: eventHeaders,
});
