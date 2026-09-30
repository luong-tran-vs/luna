import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { Component } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';

import { errorInterceptor } from '../../../core/interceptors/error-interceptor';
import { ConnectionStatusBar } from './connection-status';

@Component({ template: '' })
class Blank {}

describe('ConnectionStatusBar', () => {
  let fixture: ComponentFixture<ConnectionStatusBar>;
  let controller: HttpTestingController;
  let router: Router;

  const region = () => (fixture.nativeElement as HTMLElement).querySelector('[role="status"]')!;
  const text = () => region().textContent?.trim() ?? '';

  /** Navigates, which triggers a health check, and answers it. */
  const check = async (url: string, flush: (req: ReturnType<HttpTestingController['expectOne']>) => void) => {
    await router.navigateByUrl(url);
    flush(controller.expectOne('/api/health'));
    await fixture.whenStable();
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConnectionStatusBar],
      providers: [
        provideRouter([
          { path: '', component: Blank },
          { path: 'other', component: Blank },
        ]),
        provideHttpClient(withInterceptors([errorInterceptor])),
        provideHttpClientTesting(),
      ],
    }).compileComponents();
    controller = TestBed.inject(HttpTestingController);
    router = TestBed.inject(Router);
    fixture = TestBed.createComponent(ConnectionStatusBar);
    await fixture.whenStable();
  });

  afterEach(() => controller.verify());

  it('shows nothing while checking or when connected, in a polite live region', async () => {
    expect(region().getAttribute('aria-live')).toBe('polite');
    expect(text()).toBe('');
    await check('/', (req) => req.flush({ status: 'ok', database: 'up' }));
    expect(text()).toBe('');
  });

  it('shows "Mất kết nối cơ sở dữ liệu" on 503 with database down', async () => {
    await check('/', (req) => req.flush({ status: 'degraded', database: 'down' }, { status: 503, statusText: 'Unavailable' }));
    expect(text()).toBe('Mất kết nối cơ sở dữ liệu');
  });

  it('shows "Không kết nối được máy chủ" on a network error, 502, or a broken body', async () => {
    await check('/', (req) => req.error(new ProgressEvent('error')));
    expect(text()).toBe('Không kết nối được máy chủ');
    await check('/other', (req) => req.flush('Bad Gateway', { status: 502, statusText: 'Bad Gateway' }));
    expect(text()).toBe('Không kết nối được máy chủ');
    await check('/', (req) => req.flush('busy', { status: 503, statusText: 'Unavailable' }));
    expect(text()).toBe('Không kết nối được máy chủ');
    await check('/other', (req) => req.flush({ hello: 'world' }));
    expect(text()).toBe('Không kết nối được máy chủ');
  });

  it('checks again after each navigation and hides once healthy', async () => {
    await check('/', (req) => req.error(new ProgressEvent('error')));
    expect(text()).toBe('Không kết nối được máy chủ');
    await check('/other', (req) => req.flush({ status: 'ok', database: 'up' }));
    expect(text()).toBe('');
  });
});
