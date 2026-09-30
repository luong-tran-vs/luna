import { provideHttpClient, withInterceptors } from '@angular/common/http';
import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
} from '@angular/core';
import { provideRouter } from '@angular/router';

import { routes } from './app.routes';
import { errorInterceptor } from './core/interceptors/error-interceptor';
import { AuthService } from './core/services/auth.service';

// Angular 21 is zoneless by default, so no change detection provider is needed.
export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes),
    provideHttpClient(withInterceptors([errorInterceptor])),
    // Restore the session before the first navigation so route guards know who is logged in.
    provideAppInitializer(() => inject(AuthService).load()),
  ],
};
