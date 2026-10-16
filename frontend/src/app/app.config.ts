import { provideHttpClient, withInterceptors } from '@angular/common/http';
import {
  ApplicationConfig,
  inject,
  isDevMode,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
} from '@angular/core';
import { provideRouter, withViewTransitions } from '@angular/router';
import { provideServiceWorker } from '@angular/service-worker';

import { routes } from './app.routes';
import { apiUrlInterceptor } from './core/interceptors/api-url-interceptor';
import { errorInterceptor } from './core/interceptors/error-interceptor';
import { AppUpdateService } from './core/services/app-update.service';
import { AuthService } from './core/services/auth.service';
import { InstallService } from './core/services/install.service';

// Angular 21 is zoneless by default, so no change detection provider is needed.
export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    // Pages cross-fade where the browser supports view transitions (styles/motion.css).
    provideRouter(routes, withViewTransitions({ skipInitialTransition: true })),
    provideHttpClient(withInterceptors([errorInterceptor, apiUrlInterceptor])),
    // Restore the session before the first navigation so route guards know who is logged in.
    provideAppInitializer(() => inject(AuthService).load()),
    // PWA: caches the app shell (ngsw-config.json) so it installs and opens fast; API calls always
    // go to the network. Off in `ng serve` so a stale cache never hides code changes.
    provideServiceWorker('ngsw-worker.js', {
      enabled: !isDevMode(),
      // Right away: waiting for the app to settle held back the first update check by up to 30 s.
      registrationStrategy: 'registerImmediately',
    }),
    provideAppInitializer(() => inject(AppUpdateService).start()),
    // Catch the browser's install prompt early; the account page shows it as a button.
    provideAppInitializer(() => inject(InstallService).start()),
  ],
};
