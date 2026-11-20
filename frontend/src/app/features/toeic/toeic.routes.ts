import { Routes } from '@angular/router';

/**
 * The TOEIC area (F27, specs/022-toeic-practice/contracts/api.md "Frontend routes"). Guests come in
 * too: the server limits what they may start, and the pages lock the rest.
 */
export const toeicRoutes: Routes = [
  {
    path: '',
    title: 'Luyện thi TOEIC · Luna',
    loadComponent: () => import('./library/library').then((m) => m.Library),
  },
  {
    path: 'practice',
    title: 'Luyện theo Part · TOEIC · Luna',
    loadComponent: () => import('./practice/practice').then((m) => m.Practice),
  },
  {
    path: 'mistakes',
    title: 'Sổ câu sai · TOEIC · Luna',
    loadComponent: () => import('./mistakes/mistakes').then((m) => m.Mistakes),
  },
  {
    path: 'tests/:key',
    title: 'Đề thi · TOEIC · Luna',
    loadComponent: () => import('./test-detail/test-detail').then((m) => m.TestDetail),
  },
  {
    path: 'attempts/:id',
    title: 'Phòng thi · TOEIC · Luna',
    loadComponent: () => import('./exam-room/exam-room').then((m) => m.ExamRoom),
  },
  {
    path: 'attempts/:id/result',
    title: 'Kết quả · TOEIC · Luna',
    loadComponent: () => import('./result/result').then((m) => m.Result),
  },
  {
    path: 'attempts/:id/q/:qid',
    title: 'Đáp án chi tiết · TOEIC · Luna',
    loadComponent: () => import('./answer-detail/answer-detail').then((m) => m.AnswerDetail),
  },
];
