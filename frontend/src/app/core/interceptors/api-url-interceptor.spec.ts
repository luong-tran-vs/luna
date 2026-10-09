import { HttpClient, provideHttpClient, withInterceptors } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { environment } from '../../../environments/environment';
import { serverUrl } from '../api-url';
import { apiUrlInterceptor } from './api-url-interceptor';

describe('apiUrlInterceptor', () => {
  let http: HttpClient;
  let controller: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(withInterceptors([apiUrlInterceptor])),
        provideHttpClientTesting(),
      ],
    });
    http = TestBed.inject(HttpClient);
    controller = TestBed.inject(HttpTestingController);
  });

  afterEach(() => controller.verify());

  it('sends API paths to the API base of the environment, with the session cookie', () => {
    http.get('/health').subscribe();
    const req = controller.expectOne(`${environment.apiUrl.replace(/\/+$/, '')}/health`);
    expect(req.request.withCredentials).toBe(true);
    req.flush({});
  });

  it('leaves absolute URLs alone', () => {
    http.get('https://example.com/voice.json').subscribe();
    const req = controller.expectOne('https://example.com/voice.json');
    expect(req.request.withCredentials).toBe(false);
    req.flush({});
  });
});

describe('serverUrl', () => {
  it('loads a URL returned by the backend (already with /api) from the backend', () => {
    const base = environment.apiUrl.replace(/\/+$/, '');
    expect(serverUrl('/api/lessons/l1/images/hello')).toBe(`${base}/lessons/l1/images/hello`);
  });

  it('leaves other URLs alone', () => {
    expect(serverUrl('https://loremflickr.com/1.jpg')).toBe('https://loremflickr.com/1.jpg');
  });
});
