import { ComponentFixture, TestBed } from '@angular/core/testing';

import { Pager, pageItems, pageSlice, rangeText } from './pager';

describe('pageItems', () => {
  it('shows every page when there are few', () => {
    expect(pageItems(1, 1)).toEqual([1]);
    expect(pageItems(2, 4)).toEqual([1, 2, 3, 4]);
  });

  it('folds far pages into gaps around the current one', () => {
    expect(pageItems(6, 12)).toEqual([1, 'gap', 5, 6, 7, 'gap', 12]);
    expect(pageItems(1, 12)).toEqual([1, 2, 'gap', 12]);
    // One missing page is shown, not folded.
    expect(pageItems(4, 12)).toEqual([1, 2, 3, 4, 5, 'gap', 12]);
  });
});

describe('page helpers', () => {
  it('cuts a page and keeps it inside the page count', () => {
    expect(pageSlice([1, 2, 3, 4, 5], 2, 2)).toEqual([3, 4]);
    expect(pageSlice([1, 2, 3, 4, 5], 9, 2)).toEqual([5]);
    expect(rangeText(2, 57, 20)).toBe('21–40 / 57');
    expect(rangeText(1, 0, 20)).toBe('0');
  });
});

describe('Pager', () => {
  let fixture: ComponentFixture<Pager>;
  let el: HTMLElement;

  const render = async (page: number, pageCount: number) => {
    fixture.componentRef.setInput('page', page);
    fixture.componentRef.setInput('pageCount', pageCount);
    await fixture.whenStable();
  };
  const button = (text: string) => Array.from(el.querySelectorAll('button')).find((b) => b.textContent?.trim() === text);

  beforeEach(() => {
    fixture = TestBed.createComponent(Pager);
    el = fixture.nativeElement as HTMLElement;
  });

  it('shows nothing for a single page', async () => {
    await render(1, 1);
    expect(el.querySelector('nav')).toBeNull();
  });

  it('marks the current page and asks for another', async () => {
    await render(2, 3);
    const asked: number[] = [];
    fixture.componentInstance.pageChange.subscribe((p) => asked.push(p));
    expect(el.querySelector('[aria-current="page"]')?.textContent?.trim()).toBe('2');
    button('3')!.click();
    button('← Trước')!.click();
    button('2')!.click();
    expect(asked).toEqual([3, 1]);
  });
});
