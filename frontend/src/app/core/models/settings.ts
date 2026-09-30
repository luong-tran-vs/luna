import { ThemePreference } from '../services/theme.service';

/** Types for the settings API (specs/010-user-settings/contracts/settings-api.md). */

export interface Settings {
  theme: ThemePreference;
  dailyReviewLimit: number;
  timezone: string;
}

export type SettingsPatch = Partial<Settings>;

export const MIN_REVIEW_LIMIT = 5;
export const MAX_REVIEW_LIMIT = 200;
