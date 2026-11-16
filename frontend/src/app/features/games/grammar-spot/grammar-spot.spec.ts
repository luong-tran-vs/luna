import { ComponentFixture, TestBed } from '@angular/core/testing';
import { provideRouter } from '@angular/router';
import { of, throwError } from 'rxjs';

import { ApiError } from '../../../core/interceptors/error-interceptor';
import { GameResultBody, GrammarRound, SpotQuestion } from '../../../core/models/games';
import { GamesApiService } from '../../../core/services/games-api.service';
import { GrammarSpot } from './grammar-spot';

const wrong: SpotQuestion = {
  tokens: ['She', 'go', 'to', 'school', 'every', 'day', '.'],
  wrong: { from: 1, to: 2 },
  options: ['going', 'goes', 'went'],
  answer: 1,
  ruleVi: 'Chủ ngữ số ít: động từ thêm -s.',
  topicVi: 'Hiện tại đơn',
  pointId: 'a1-present-simple',
  source: 'fix',
};
const right: SpotQuestion = {
  tokens: ['They', 'are', 'playing', 'football', 'now', '.'],
  wrong: null,
  options: [],
  answer: 0,
  ruleVi: 'Now: hiện tại tiếp diễn.',
  topicVi: 'Hiện tại tiếp diễn',
  pointId: 'a1-present-continuous',
  source: 'fix',
};

describe('GrammarSpot', () => {
  let fixture: ComponentFixture<GrammarSpot>;
  let el: HTMLElement;
  let finished: GameResultBody[];

  const settle = async () => {
    await new Promise((resolve) => setTimeout(resolve));
    await fixture.whenStable();
  };
  const text = (node: Element | null | undefined) =>
    node?.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  const button = (label: string) =>
    Array.from(el.querySelectorAll<HTMLButtonElement>('button')).find((b) => text(b) === label)!;

  const setup = async (round: GrammarRound | ApiError) => {
    finished = [];
    TestBed.configureTestingModule({
      imports: [GrammarSpot],
      providers: [
        provideRouter([]),
        {
          provide: GamesApiService,
          useValue: {
            grammarRound: () => (round instanceof ApiError ? throwError(() => round) : of(round)),
            finish: (_: string, body: GameResultBody) => {
              finished.push(body);
              return of({ best: body.score, newBest: true, rescheduled: 0 });
            },
          },
        },
      ],
    });
    fixture = TestBed.createComponent(GrammarSpot);
    el = fixture.nativeElement as HTMLElement;
    await settle();
  };

  afterEach(() => fixture.destroy());

  it('plays a round: fix the mistake, call a right sentence right, then the result', async () => {
    await setup({ questions: [wrong, right] });
    expect(text(el.querySelector('.words'))).toBe('She go to school every day.');
    button('go').click();
    await settle();
    expect(text(el.querySelector('.sheet-title'))).toBe('Sửa “go” thành:');
    expect(el.querySelector('.token.struck')).not.toBeNull();
    button('goes').click();
    await settle();
    expect(text(el.querySelector('.words'))).toContain('She ✓ sửa thành goes to school');
    expect(text(el.querySelector('.verdict'))).toMatch(/^✓ Đúng! \+\d+$/);
    expect(text(el.querySelector('.topic'))).toBe('Hiện tại đơn');
    button('Tiếp').click();
    await settle();
    button('Không có lỗi').click();
    await settle();
    expect(text(el.querySelector('.verdict'))).toMatch(/^✓ Câu đúng! \+\d+$/);
    button('Xem kết quả').click();
    await settle();
    expect(finished.length).toBe(1);
    expect(finished[0].completed).toBe(true);
    expect(finished[0].score).toBeGreaterThan(200);
    expect(finished[0].wrongWords).toEqual([]);
  });

  it('loses a life and shows the fix when a right word is tapped', async () => {
    await setup({ questions: [wrong, right] });
    button('school').click();
    await settle();
    expect(el.querySelector('.lives')?.getAttribute('aria-label')).toBe('Còn 2 mạng');
    expect(text(el.querySelector('.answer'))).toBe('Sửa “go” thành “goes”.');
  });

  it('says when there are not enough sentences', async () => {
    await setup(new ApiError('http', 409, { error: 'not_enough_sentences' }));
    expect(text(el)).toContain('Chưa đủ câu ngữ pháp cho trò này');
  });
});
