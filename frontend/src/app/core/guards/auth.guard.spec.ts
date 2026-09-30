import { signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import {
  ActivatedRouteSnapshot,
  CanActivateFn,
  provideRouter,
  Router,
  RouterStateSnapshot,
  UrlTree,
} from '@angular/router';

import { Role } from '../models/user';
import { AuthService } from '../services/auth.service';
import { adminGuard, authGuard, guestGuard } from './auth.guard';

describe('auth guards', () => {
  const loggedIn = signal(false);
  const admin = signal(false);

  const run = (guard: CanActivateFn, url = '/') =>
    TestBed.runInInjectionContext(() =>
      guard({} as ActivatedRouteSnapshot, { url } as RouterStateSnapshot),
    );
  const asUrl = (result: unknown) => TestBed.inject(Router).serializeUrl(result as UrlTree);

  const login = (role: Role | null) => {
    loggedIn.set(role !== null);
    admin.set(role === 'admin');
  };

  beforeEach(() => {
    login(null);
    TestBed.configureTestingModule({
      providers: [
        provideRouter([]),
        { provide: AuthService, useValue: { isLoggedIn: loggedIn, isAdmin: admin } },
      ],
    });
  });

  describe('authGuard', () => {
    it('sends a logged-out user to /login with the return URL', () => {
      expect(asUrl(run(authGuard, '/admin?tab=1'))).toBe('/login?returnUrl=%2Fadmin%3Ftab%3D1');
    });

    it('lets a logged-in user through', () => {
      login('learner');
      expect(run(authGuard)).toBe(true);
    });
  });

  describe('guestGuard', () => {
    it('lets a logged-out user through', () => {
      expect(run(guestGuard, '/login')).toBe(true);
    });

    it('sends a logged-in user home', () => {
      login('learner');
      expect(asUrl(run(guestGuard, '/login'))).toBe('/');
    });
  });

  describe('adminGuard', () => {
    it('sends a learner to /forbidden', () => {
      login('learner');
      expect(asUrl(run(adminGuard, '/admin'))).toBe('/forbidden');
    });

    it('lets an admin through', () => {
      login('admin');
      expect(run(adminGuard, '/admin')).toBe(true);
    });
  });
});
