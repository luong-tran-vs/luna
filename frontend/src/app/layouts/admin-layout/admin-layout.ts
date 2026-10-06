import { ChangeDetectionStrategy, Component, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterOutlet } from '@angular/router';
import { filter, map } from 'rxjs';

import { AppHeader } from '../../shared/components/app-header/app-header';
import { ConnectionStatusBar } from '../../shared/components/connection-status/connection-status';

interface AdminLink {
  path: string;
  label: string;
  /** Whether a URL path (no query) belongs to this section. */
  matches: (path: string) => boolean;
}

const ADMIN_LINKS: readonly AdminLink[] = [
  { path: '/admin', label: 'Bài học', matches: (p) => p === '/admin' || p.startsWith('/admin/lessons') },
  { path: '/admin/topics', label: 'Chủ đề', matches: (p) => p.startsWith('/admin/topics') },
  { path: '/admin/grammar', label: 'Ngữ pháp', matches: (p) => p.startsWith('/admin/grammar') },
  { path: '/admin/roadmap', label: 'Lộ trình', matches: (p) => p.startsWith('/admin/roadmap') },
  { path: '/admin/appearance', label: 'Giao diện', matches: (p) => p.startsWith('/admin/appearance') },
  { path: '/admin/tts-lab', label: 'Thử giọng đọc', matches: (p) => p.startsWith('/admin/tts-lab') },
];

/**
 * Admin area: only content management (lessons, topics, roadmap). The menu is a sidebar on the
 * left from 768px; on narrow screens it becomes a scrolling row under the header.
 */
@Component({
  selector: 'lu-admin-layout',
  imports: [RouterOutlet, RouterLink, AppHeader, ConnectionStatusBar],
  templateUrl: './admin-layout.html',
  styleUrl: './admin-layout.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class AdminLayout {
  private readonly router = inject(Router);

  protected readonly links = ADMIN_LINKS;
  protected readonly path = toSignal(
    this.router.events.pipe(
      filter((e) => e instanceof NavigationEnd),
      map(() => this.currentPath()),
    ),
    { initialValue: this.currentPath() },
  );

  private currentPath(): string {
    return this.router.url.split(/[?#]/)[0];
  }
}
