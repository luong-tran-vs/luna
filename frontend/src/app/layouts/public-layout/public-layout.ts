import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import { ConnectionStatusBar } from '../../shared/components/connection-status/connection-status';

/** Pages outside both areas: login, register and "no permission". No header or menu (client sketch). */
@Component({
  selector: 'lu-public-layout',
  imports: [RouterOutlet, ConnectionStatusBar],
  template: `
    <main>
      <router-outlet />
    </main>
    <footer class="app-footer">
      <lu-connection-status />
    </footer>
  `,
  styles: ':host { display: block; }',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PublicLayout {}
