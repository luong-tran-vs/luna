import { TestBed } from '@angular/core/testing';
import { ActivatedRouteSnapshot, RouterStateSnapshot } from '@angular/router';

import { CanLeave, unsavedChangesGuard } from './unsaved-changes.guard';

describe('unsavedChangesGuard', () => {
  const run = (page: CanLeave) =>
    TestBed.runInInjectionContext(() =>
      unsavedChangesGuard(page, {} as ActivatedRouteSnapshot, {} as RouterStateSnapshot, {} as RouterStateSnapshot),
    );

  it('passes the page answer through', async () => {
    expect(run({ canLeave: () => true })).toBe(true);
    expect(run({ canLeave: () => false })).toBe(false);
    await expect(run({ canLeave: () => Promise.resolve(false) })).resolves.toBe(false);
    await expect(run({ canLeave: () => Promise.resolve(true) })).resolves.toBe(true);
  });
});
