export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

export interface ApiClientOptions {
  baseUrl?: string;
  getToken?: () => string | null;
}

export function createApiClient(options: ApiClientOptions = {}) {
  const baseUrl = options.baseUrl ?? "/api/v1";

  async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const token = options.getToken?.();
    const headers = new Headers(init.headers);
    headers.set("Content-Type", "application/json");
    if (token) headers.set("Authorization", `Bearer ${token}`);

    const res = await fetch(`${baseUrl}${path}`, {
      ...init,
      headers,
      credentials: "include",
    });

    if (!res.ok) {
      const body = await res.json().catch(() => ({ error: res.statusText }));
      throw new ApiError(res.status, body.error ?? "Erro inesperado");
    }
    if (res.status === 204) return undefined as T;
    return res.json() as Promise<T>;
  }

  return {
    get: <T>(path: string) => request<T>(path, { method: "GET" }),
    post: <T>(path: string, body?: unknown) =>
      request<T>(path, { method: "POST", body: body ? JSON.stringify(body) : undefined }),
    put: <T>(path: string, body?: unknown) =>
      request<T>(path, { method: "PUT", body: body ? JSON.stringify(body) : undefined }),
    patch: <T>(path: string, body?: unknown) =>
      request<T>(path, { method: "PATCH", body: body ? JSON.stringify(body) : undefined }),
    del: <T>(path: string) => request<T>(path, { method: "DELETE" }),
  };
}
