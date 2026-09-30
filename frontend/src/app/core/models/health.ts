/** Body of GET /api/health (specs/001-project-skeleton/contracts/health-api.md). */
export interface HealthResponse {
  status: 'ok' | 'degraded';
  database: 'up' | 'down';
}

/** Connection state shown on the home page. */
export type ConnectionStatus = 'checking' | 'connected' | 'database-down' | 'server-unreachable';
