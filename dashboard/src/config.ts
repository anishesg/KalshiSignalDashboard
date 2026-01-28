// API configuration
// The backend serves both API and frontend, so we use relative paths
// In development, Vite proxies /api to localhost:8080
// In production (Railway), the Go server handles everything on the same domain

export const apiFetch = async (endpoint: string, options?: RequestInit): Promise<Response> => {
  // Always use relative paths - backend serves everything from same domain
  return fetch(endpoint, options)
}

