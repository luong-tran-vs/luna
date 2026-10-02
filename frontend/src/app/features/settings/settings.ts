import { DOCUMENT } from '@angular/common';
import { ChangeDetectionStrategy, Component, computed, DestroyRef, inject, signal } from '@angular/core';
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { RouterLink } from '@angular/router';

import { ApiError } from '../../core/interceptors/error-interceptor';
import { MAX_REVIEW_LIMIT, MIN_REVIEW_LIMIT, Settings as SettingsData } from '../../core/models/settings';
import { ExportApiService } from '../../core/services/export-api.service';
import { SettingsApiService } from '../../core/services/settings-api.service';
import { Icon } from '../../shared/components/icon/icon';

const LIMIT_ERROR = 'Số thẻ từ 5 đến 200';

interface ZoneOption {
  name: string;
  label: string;
  /** Lower-case name with "/" and "_" as spaces, for the search box. */
  key: string;
}

/** "GMT+7" for a timezone now; "" when the browser cannot tell. */
function offsetOf(zone: string, now: Date): string {
  try {
    const parts = new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'shortOffset' }).formatToParts(now);
    return parts.find((p) => p.type === 'timeZoneName')?.value ?? '';
  } catch {
    return '';
  }
}

/** Every IANA timezone the browser knows; the current one only when it cannot list them. */
function browserZones(): string[] {
  const intl = Intl as typeof Intl & { supportedValuesOf?: (key: string) => string[] };
  try {
    const zones = intl.supportedValuesOf?.('timeZone');
    if (zones?.length) {
      return zones;
    }
  } catch {
    // Fall through to the current zone.
  }
  return [Intl.DateTimeFormat().resolvedOptions().timeZone];
}

function searchKey(text: string): string {
  return text.toLowerCase().replace(/[/_]/g, ' ').trim();
}

/** Settings page (F12): daily card limit, timezone, data export. Light/dark lives on the account page. */
@Component({
  selector: 'lu-settings',
  imports: [Icon, ReactiveFormsModule, RouterLink],
  templateUrl: './settings.html',
  styleUrl: './settings.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Settings {
  private readonly api = inject(SettingsApiService);
  private readonly destroyRef = inject(DestroyRef);
  private readonly exporter = inject(ExportApiService);
  private readonly document = inject(DOCUMENT);

  protected readonly exporting = signal(false);
  protected readonly exportError = signal(false);

  protected readonly minLimit = MIN_REVIEW_LIMIT;
  protected readonly maxLimit = MAX_REVIEW_LIMIT;

  protected readonly loaded = signal<SettingsData | null>(null);
  protected readonly loadError = signal(false);
  protected readonly saveState = signal<'idle' | 'saving' | 'saved' | 'error'>('idle');
  protected readonly serverFields = signal<Record<string, string>>({});
  private readonly submitted = signal(false);

  protected readonly form = new FormGroup({
    dailyReviewLimit: new FormControl<number | null>(null, [
      Validators.required,
      Validators.min(MIN_REVIEW_LIMIT),
      Validators.max(MAX_REVIEW_LIMIT),
      Validators.pattern(/^\d+$/),
    ]),
    timezone: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
  });

  /** Search box text for the timezone list. */
  protected readonly zoneQuery = signal('');
  private readonly zones: readonly ZoneOption[] = (() => {
    const now = new Date();
    return browserZones().map((name) => {
      const offset = offsetOf(name, now);
      return { name, label: offset ? `${name} (${offset})` : name, key: searchKey(name) };
    });
  })();
  private readonly selectedZone = toSignal(this.form.controls.timezone.valueChanges, { initialValue: '' });

  /** Zones matching the search, always including the selected one (so the select keeps its value). */
  protected readonly zoneOptions = computed(() => {
    const query = searchKey(this.zoneQuery());
    const selected = this.selectedZone();
    const list = this.zones.filter((z) => !query || z.key.includes(query) || z.name === selected);
    // The stored zone may be an alias the browser does not list (Asia/Ho_Chi_Minh vs Asia/Saigon).
    if (selected && !list.some((z) => z.name === selected)) {
      const offset = offsetOf(selected, new Date());
      list.unshift({ name: selected, label: offset ? `${selected} (${offset})` : selected, key: searchKey(selected) });
    }
    return list;
  });

  private readonly limitStatus = toSignal(this.form.controls.dailyReviewLimit.statusChanges, { initialValue: 'VALID' });
  protected readonly limitError = computed(() => {
    const server = this.serverFields()['dailyReviewLimit'];
    if (server) {
      return server;
    }
    return this.submitted() && this.limitStatus() === 'INVALID' ? LIMIT_ERROR : null;
  });
  protected readonly timezoneError = computed(() => this.serverFields()['timezone'] ?? null);

  constructor() {
    this.load();
  }

  protected load(): void {
    this.loadError.set(false);
    this.api
      .get()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (s) => {
          this.loaded.set(s);
          this.form.setValue({ dailyReviewLimit: s.dailyReviewLimit, timezone: s.timezone });
        },
        error: () => this.loadError.set(true),
      });
  }


  protected onZoneQuery(event: Event): void {
    this.zoneQuery.set((event.target as HTMLInputElement).value);
  }

  protected submit(): void {
    this.submitted.set(true);
    this.serverFields.set({});
    this.saveState.set('idle');
    this.form.controls.dailyReviewLimit.updateValueAndValidity();
    if (this.form.invalid) {
      return;
    }
    const { dailyReviewLimit, timezone } = this.form.getRawValue();
    this.saveState.set('saving');
    this.api
      .update({ dailyReviewLimit: Number(dailyReviewLimit), timezone })
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (s) => {
          this.loaded.set(s);
          this.saveState.set('saved');
        },
        error: (err: unknown) => {
          const body = err instanceof ApiError ? (err.body as { fields?: Record<string, string> } | null) : null;
          if (body?.fields) {
            this.serverFields.set(body.fields);
            this.saveState.set('idle');
          } else {
            this.saveState.set('error');
          }
        },
      });
  }

  /** Downloads the learner's data as a JSON file (F13). */
  protected exportData(): void {
    this.exporting.set(true);
    this.exportError.set(false);
    this.exporter
      .download()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: ({ blob, filename }) => {
          const url = URL.createObjectURL(blob);
          const link = this.document.createElement('a');
          link.href = url;
          link.download = filename;
          link.click();
          // Revoked after the click so every browser has started the download.
          setTimeout(() => URL.revokeObjectURL(url));
          this.exporting.set(false);
        },
        error: () => {
          this.exportError.set(true);
          this.exporting.set(false);
        },
      });
  }
}
