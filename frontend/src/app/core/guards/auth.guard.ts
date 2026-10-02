import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';

import { AuthService } from '../services/auth.service';

/** Logged-out users go to /login, keeping the requested page as returnUrl. */
export const authGuard: CanActivateFn = (_route, state) =>
  inject(AuthService).isLoggedIn() ||
  inject(Router).createUrlTree(['/login'], { queryParams: { returnUrl: state.url } });

/** Login and register pages are only for logged-out users; others go to their own area. */
export const guestGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  return !auth.isLoggedIn() || inject(Router).createUrlTree([auth.isAdmin() ? '/admin' : '/']);
};

/** Admin pages; learners see the "no permission" page. Use after authGuard. */
export const adminGuard: CanActivateFn = () =>
  inject(AuthService).isAdmin() || inject(Router).createUrlTree(['/forbidden']);

/** Learning pages; admins only manage content, so they go to the admin area. Use after authGuard. */
export const learnerGuard: CanActivateFn = () =>
  !inject(AuthService).isAdmin() || inject(Router).createUrlTree(['/admin']);
