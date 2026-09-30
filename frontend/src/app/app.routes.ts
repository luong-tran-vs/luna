import { Routes } from '@angular/router';

import { adminGuard, authGuard, guestGuard } from './core/guards/auth.guard';

export const routes: Routes = [
  {
    path: '',
    canActivate: [authGuard],
    loadComponent: () => import('./features/home/home').then((m) => m.Home),
  },
  {
    path: 'login',
    canActivate: [guestGuard],
    title: 'Đăng nhập · Luna',
    loadComponent: () => import('./features/auth/login/login').then((m) => m.Login),
  },
  {
    path: 'register',
    canActivate: [guestGuard],
    title: 'Đăng ký · Luna',
    loadComponent: () => import('./features/auth/register/register').then((m) => m.Register),
  },
  {
    path: 'admin',
    canActivate: [authGuard, adminGuard],
    loadChildren: () => import('./features/admin/admin.routes').then((m) => m.adminRoutes),
  },
  {
    path: 'lessons',
    canActivate: [authGuard],
    loadChildren: () => import('./features/lesson/lesson.routes').then((m) => m.lessonRoutes),
  },
  {
    path: 'today',
    canActivate: [authGuard],
    title: 'Hôm nay · Luna',
    loadComponent: () => import('./features/lesson/today/today').then((m) => m.Today),
  },
  {
    path: 'goal',
    canActivate: [authGuard],
    title: 'Mục tiêu · Luna',
    loadComponent: () => import('./features/lesson/goal/goal').then((m) => m.Goal),
  },
  {
    path: 'settings',
    canActivate: [authGuard],
    title: 'Cài đặt · Luna',
    loadComponent: () => import('./features/settings/settings').then((m) => m.Settings),
  },
  {
    path: 'stats',
    canActivate: [authGuard],
    title: 'Thống kê · Luna',
    loadComponent: () => import('./features/stats/stats').then((m) => m.Stats),
  },
  {
    path: 'vocabulary',
    canActivate: [authGuard],
    loadChildren: () => import('./features/vocabulary/vocabulary.routes').then((m) => m.vocabularyRoutes),
  },
  {
    path: 'forbidden',
    canActivate: [authGuard],
    title: 'Không có quyền · Luna',
    loadComponent: () => import('./features/forbidden/forbidden').then((m) => m.Forbidden),
  },
  { path: '**', redirectTo: '' },
];
