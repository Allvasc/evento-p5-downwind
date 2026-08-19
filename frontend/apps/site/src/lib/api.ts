import { createApiClient, ApiError } from "@p5wellness/shared";

export { ApiError };

export const api = createApiClient({
  baseUrl: "/api/v1",
  getToken: () => localStorage.getItem("p5_student_token"),
});
