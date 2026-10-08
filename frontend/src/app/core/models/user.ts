/** member: every lesson; guest: only the first lesson of each roadmap and no grammar section. */
export type Role = 'admin' | 'member' | 'guest';

/** Vietnamese name of each role. */
export const ROLE_NAMES: Record<Role, string> = {
  admin: 'Quản trị',
  member: 'Thành viên',
  guest: 'Khách',
};

/** Account returned by /api/auth/* (specs/002-user-accounts/contracts/auth-api.md). */
export interface User {
  id: string;
  email: string;
  role: Role;
  timezone: string;
}

/** An account in the admin list (GET /api/admin/users). */
export interface Account {
  id: string;
  email: string;
  role: Role;
  createdAt: string;
}

/** What an admin enters for an account; on update an empty password keeps the current one. */
export interface AccountInput {
  email: string;
  password: string;
  role: Role;
}
