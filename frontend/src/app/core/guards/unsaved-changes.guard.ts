import { CanDeactivateFn } from '@angular/router';

/** A page that may hold unsaved work and asks before it is left. */
export interface CanLeave {
  canLeave(): boolean | Promise<boolean>;
}

/** Lets the page decide whether navigation away may continue (F7 drafts). */
export const unsavedChangesGuard: CanDeactivateFn<CanLeave> = (component) => component.canLeave();
