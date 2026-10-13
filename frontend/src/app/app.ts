import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';

import { UpdateBanner } from './shared/components/update-banner/update-banner';

/**
 * Root: each area (learner, admin, public) brings its own layout through the routes. The bar
 * offering a new version of the app shows over every area.
 */
@Component({
  selector: 'lu-root',
  imports: [RouterOutlet, UpdateBanner],
  template: '<router-outlet /><lu-update-banner />',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class App {}
