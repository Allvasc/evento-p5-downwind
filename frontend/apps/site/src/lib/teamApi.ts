import { createApiClient, ApiError } from "@p5wellness/shared";

export { ApiError };

export const TEAM_TOKEN_KEY = "p5_team_token";

export const api = createApiClient({
  baseUrl: "/api/v1",
  getToken: () => localStorage.getItem(TEAM_TOKEN_KEY),
});
