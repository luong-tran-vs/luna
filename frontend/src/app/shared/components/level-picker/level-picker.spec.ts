import { Component, signal } from '@angular/core';
import { ComponentFixture, TestBed } from '@angular/core/testing';
import { FormControl, ReactiveFormsModule } from '@angular/forms';

import { LevelPicker } from './level-picker';

@Component({
  imports: [LevelPicker, ReactiveFormsModule],
  template: `
    <lu-level-picker id="in-form" [formControl]="control" labelledBy="label" />
    <lu-level-picker id="alone" [value]="value()" [compact]="true" (changed)="value.set($event)" />
  `,
})
class Host {
  readonly control = new FormControl<string>('', { nonNullable: true });
  readonly value = signal<string | null>('B1');
}

describe('LevelPicker', () => {
  let fixture: ComponentFixture<Host>;
  let el: HTMLElement;

  const radios = (id: string) => Array.from(el.querySelectorAll<HTMLButtonElement>(`#${id} [role="radio"]`));
  const checked = (id: string) => radios(id).filter((r) => r.getAttribute('aria-checked') === 'true').map((r) => r.querySelector('.code')!.textContent!.trim());

  beforeEach(async () => {
    await TestBed.configureTestingModule({ imports: [Host] }).compileComponents();
    fixture = TestBed.createComponent(Host);
    el = fixture.nativeElement;
    await fixture.whenStable();
  });

  it('shows A1 → C2 with their names as a radio group named by its label', () => {
    expect(el.querySelector('#in-form [role="radiogroup"]')!.getAttribute('aria-labelledby')).toBe('label');
    expect(radios('in-form').map((r) => r.querySelector('.code')!.textContent!.trim())).toEqual(['A1', 'A2', 'B1', 'B2', 'C1', 'C2']);
    expect(radios('in-form')[0].querySelector('.hint')!.textContent!.trim()).toBe('Mới bắt đầu');
    // Nothing chosen yet: Tab lands on the first card.
    expect(checked('in-form')).toEqual([]);
    expect(radios('in-form').map((r) => r.tabIndex)).toEqual([0, -1, -1, -1, -1, -1]);
  });

  it('writes the chosen level to the form control and reads it back', async () => {
    radios('in-form')[2].click();
    await fixture.whenStable();
    expect(fixture.componentInstance.control.value).toBe('B1');
    expect(checked('in-form')).toEqual(['B1']);
    fixture.componentInstance.control.setValue('C2');
    await fixture.whenStable();
    expect(checked('in-form')).toEqual(['C2']);
    fixture.componentInstance.control.disable();
    await fixture.whenStable();
    expect(radios('in-form').every((r) => r.disabled)).toBe(true);
  });

  it('moves the choice with the arrow keys, wrapping around', async () => {
    expect(checked('alone')).toEqual(['B1']);
    expect(radios('alone')[0].querySelector('.hint')).toBeNull();
    radios('alone')[2].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', cancelable: true }));
    await fixture.whenStable();
    expect(checked('alone')).toEqual(['A2']);
    expect(document.activeElement).toBe(radios('alone')[1]);
    radios('alone')[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowLeft', cancelable: true }));
    await fixture.whenStable();
    expect(fixture.componentInstance.value()).toBe('C2');
  });
});
