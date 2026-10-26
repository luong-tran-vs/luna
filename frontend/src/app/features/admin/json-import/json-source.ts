import { ChangeDetectionStrategy, Component, input, model, signal } from '@angular/core';

/**
 * Where an import dialog gets its JSON (FR-030): pasted in a text area, or read from a .json file
 * into it, so the admin always sees and can fix what is sent.
 */
@Component({
  selector: 'lu-json-source',
  templateUrl: './json-source.html',
  styleUrl: './json-source.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class JsonSource {
  readonly text = model('');
  /** Prefix of the ids, unique on the page. */
  readonly idPrefix = input.required<string>();
  readonly invalid = input(false);
  /** Id of the element that describes an error of the text. */
  readonly describedBy = input<string | null>(null);

  protected readonly mode = signal<'paste' | 'file'>('paste');
  protected readonly fileNote = signal('');

  protected onInput(event: Event): void {
    this.text.set((event.target as HTMLTextAreaElement).value);
  }

  protected async pick(event: Event): Promise<void> {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) {
      return;
    }
    try {
      this.text.set(await file.text());
      this.fileNote.set(
        `Đã đọc file ${file.name}. Nội dung ở ô bên dưới, có thể sửa trước khi kiểm tra.`,
      );
      this.mode.set('paste');
    } catch {
      this.fileNote.set(`Không đọc được file ${file.name}.`);
    }
    input.value = '';
  }
}
