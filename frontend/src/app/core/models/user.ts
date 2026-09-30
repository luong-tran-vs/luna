export type Role = 'admin' | 'learner';

/** Account returned by /api/auth/* (specs/002-user-accounts/contracts/auth-api.md). */
export interface User {
  id: string;
  email: string;
  role: Role;
  timezone: string;
}
