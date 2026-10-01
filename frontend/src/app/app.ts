import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import { AppHeader } from './shared/components/app-header/app-header';
import { ConnectionStatusBar } from './shared/components/connection-status/connection-status';
import { Toast } from './shared/components/toast/toast';

@Component({
  selector: 'lu-root',
  imports: [RouterOutlet, AppHeader, ConnectionStatusBar, Toast],
  templateUrl: './app.html',
  styleUrl: './app.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class App {}
