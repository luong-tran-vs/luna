---
name: Vibrant Guided EdTech
colors:
  surface: '#f8f9ff'
  surface-dim: '#cbdbf5'
  surface-bright: '#f8f9ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#eff4ff'
  surface-container: '#e5eeff'
  surface-container-high: '#dce9ff'
  surface-container-highest: '#d3e4fe'
  on-surface: '#0b1c30'
  on-surface-variant: '#434655'
  inverse-surface: '#213145'
  inverse-on-surface: '#eaf1ff'
  outline: '#737686'
  outline-variant: '#c3c6d7'
  surface-tint: '#0053db'
  primary: '#004ac6'
  on-primary: '#ffffff'
  primary-container: '#2563eb'
  on-primary-container: '#eeefff'
  inverse-primary: '#b4c5ff'
  secondary: '#006c49'
  on-secondary: '#ffffff'
  secondary-container: '#6cf8bb'
  on-secondary-container: '#00714d'
  tertiary: '#784b00'
  on-tertiary: '#ffffff'
  tertiary-container: '#996100'
  on-tertiary-container: '#ffeedd'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dbe1ff'
  primary-fixed-dim: '#b4c5ff'
  on-primary-fixed: '#00174b'
  on-primary-fixed-variant: '#003ea8'
  secondary-fixed: '#6ffbbe'
  secondary-fixed-dim: '#4edea3'
  on-secondary-fixed: '#002113'
  on-secondary-fixed-variant: '#005236'
  tertiary-fixed: '#ffddb8'
  tertiary-fixed-dim: '#ffb95f'
  on-tertiary-fixed: '#2a1700'
  on-tertiary-fixed-variant: '#653e00'
  background: '#f8f9ff'
  on-background: '#0b1c30'
  surface-variant: '#d3e4fe'
typography:
  headline-xl:
    fontFamily: Plus Jakarta Sans
    fontSize: 30px
    fontWeight: '700'
    lineHeight: 38px
  headline-xl-mobile:
    fontFamily: Plus Jakarta Sans
    fontSize: 24px
    fontWeight: '700'
    lineHeight: 32px
  headline-lg:
    fontFamily: Plus Jakarta Sans
    fontSize: 22px
    fontWeight: '700'
    lineHeight: 28px
  headline-md:
    fontFamily: Plus Jakarta Sans
    fontSize: 18px
    fontWeight: '600'
    lineHeight: 24px
  headline-sm:
    fontFamily: Plus Jakarta Sans
    fontSize: 16px
    fontWeight: '600'
    lineHeight: 22px
  body-lg:
    fontFamily: Plus Jakarta Sans
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-md:
    fontFamily: Plus Jakarta Sans
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  body-sm:
    fontFamily: Plus Jakarta Sans
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 18px
  label-lg:
    fontFamily: Plus Jakarta Sans
    fontSize: 14px
    fontWeight: '600'
    lineHeight: 20px
  label-md:
    fontFamily: Plus Jakarta Sans
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
  label-sm:
    fontFamily: Plus Jakarta Sans
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 14px
rounded:
  sm: 0.25rem
  DEFAULT: 0.5rem
  md: 0.75rem
  lg: 1rem
  xl: 1.5rem
  full: 9999px
spacing:
  gutter: 1rem
  gutter-tablet: 1.5rem
  gutter-desktop: 2rem
  margin: 1rem
  margin-tablet: 2rem
  margin-desktop: 3rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 1rem
  space-lg: 1.5rem
  space-xl: 2rem
---

## Brand & Style

This design system embodies an encouraging, accessible, and structured visual language tailored for mobile-first language acquisition. Designed for continuous self-improvement, the aesthetic merges modern corporate clarity with tactile, celebratory gamification. The visual mood evokes confidence, daily momentum, and clarity without cognitive overload.

The design movement is **Modern Friendly Functionalism**:
- **Structured simplicity**: High legibility, ample breathing room, and soft boundaries build an inviting container for dense pedagogical content (phonetics, scripts, translations).
- **Gamified affirmation**: High-contrast, purposeful accent hues (emerald achievement, amber streaks, bright cognitive blue) reinforce micro-habits and rewarding milestones.
- **Approachable geometry**: Generously rounded containers, squircle badges, and smooth interactive surfaces soften the intimidating friction of language learning.

## Colors

The palette leverages a crisp, daylight-first system built around cognitive focus and emotional reinforcement. 

- **Primary (`#2563EB`)**: Anchor blue used for critical interactive calls-to-action, active progress bars, current lesson indicators, and selected assessment choices.
- **Secondary (`#10B981`)**: Emerald green signaling success states, completed units, high accuracy scores, and affirmative validation feedback.
- **Tertiary (`#F59E0B`)**: Warm energetic amber dedicated strictly to habit metrics—daily streaks, flame icons, trophy milestones, and active commitment tallies.
- **Neutral (`#64748B`)**: Slate neutral providing stable informational hierarchy: `#0F172A` for dominant headings, `#334155` for body and sentence parsing, `#64748B` for secondary annotations and audio timestamps, down to `#F8FAFC` for page backgrounds and `#F1F5F9` for inactive module containers.
- **Semantic Error (`#EF4444`)**: Used selectively for incorrect quiz evaluations and critical alert banners.

## Typography

The type scale relies entirely on **Plus Jakarta Sans** for its balanced humanist curvature, generous counter spaces, and exceptional legibility across mixed-language Latin script rendering (including Vietnamese diacritics and phonetic IPA symbols).

- **Display & Headings**: Rendered in Bold (`700`) and Semi-Bold (`600`) to create strong anchor points for screens requiring quick scanning (lesson headers, score summaries).
- **Phonetics & Metadata**: Micro-labels and IPA pronunciation keys (`/he'lou/`) utilize Medium (`500`) weight at `12px` to remain legible beside full-size vocabulary words.
- **Lesson Content**: Dialogue dialogues and quiz options utilize regular weights with balanced line-heights (`1.4` to `1.5`) to prevent collisions with tone marks and accents.

## Layout & Spacing

The layout model is optimized primarily for touch interaction within vertical viewport constraints, scaling up to centered card containers on larger surfaces:

- **Mobile Viewport (Base)**: Single-column fluid presentation with an outer margin of `1rem` (16px) and internal section spacing of `1.5rem` (24px). Bottom navigation bars reserve a persistent 64px dock area plus bottom safe-area insets.
- **Tablet & Desktop Expansion**: Content max-width caps at 640px for single-path flows (lessons, drills, quizzes) to maintain focused eye travel. Dashboard overviews transition into a multi-column fluid grid (2 columns on tablet, 3 on desktop) utilizing `gutter-tablet` and `gutter-desktop`.
- **Vertical Spacing Cadence**: Rhythm follows strict multiples of 4px and 8px: `space-xs` (4px) for inline badges and icon offsets; `space-sm` (8px) for card sub-items and label-to-input gaps; `space-md` (16px) for card body padding and exercise choices; `space-lg` (24px) between learning modular blocks; `space-xl` (32px) separating primary view sections.

## Elevation & Depth

Visual hierarchy uses a hybrid of **tonal layering** and **diffused ambient shadows**, prioritizing clean daylight separation over dramatic 3D effects:

- **Base Layer (Level 0)**: Page canvas washed in ultra-soft slate neutral (`#F8FAFC`).
- **Surface Layer (Level 1)**: Primary cards, lesson containers, and quiz options resting in crisp `#FFFFFF`. Elevated by an ambient shadow: `0px 2px 8px -2px rgba(15, 23, 42, 0.05)` paired with a soft perimeter line (`1px solid #E2E8F0`).
- **Interactive Elevated Layer (Level 2)**: Active floating controls, persistent bottom sheets, and sticky progression buttons. Uses a pronounced soft float: `0px 8px 24px -4px rgba(37, 99, 235, 0.12)`.
- **Selected & Feedback State**: Selected exercise options use a directional tone tint (`#EFF6FF`) with a bolded outline (`2px solid #2563EB`) rather than excessive elevation jump, keeping user focus on accuracy.

## Shapes

The design system employs a **Rounded** aesthetic (`roundedness: 2`), reflecting warmth, tactile friendliness, and high touch affordance:

- **Standard Elements (0.5rem / 8px)**: Segmented controls, chips, small badges, and text inputs.
- **Card Containers (1rem / 16px)**: Lesson nodes, metric summary cards, dialogue bubbles, and full-width list items.
- **Primary Cards & Modals (1.5rem / 24px)**: Major dashboard hero panels, completion review sheets, and lesson category blocks.
- **Full Pill (`9999px`)**: Primary action buttons ("Tiếp tục", "Bắt đầu"), audio waveform track headers, progress bar caps, and status indicator tags.

## Components

### Buttons
- **Primary CTA**: Full pill shape (`9999px`), solid `#2563EB` fill with white text (`label-lg`), minimum height of 48px to 52px for thumbs. Pressed state renders `#1D4ED8`.
- **Secondary / Outlined**: White background with a `1.5px` border in `#E2E8F0` or `#2563EB`, text tinted accordingly.
- **Social / Auth**: Full-width bordered cards (`rounded-lg`) hosting official SVG brand glyphs with centered `label-lg` copy.

### Chips & Filter Tabs
- Segmented pills used for skill switches (e.g., "Tiếng Anh", "Tiếng Nhật") and exercise filter modes.
- **Selected**: Solid `#2563EB` with `#FFFFFF` text.
- **Unselected**: Slate ghost tint (`#F1F5F9`) with `#64748B` text, shifting to `#E2E8F0` on hover/touch.

### Cards & Metric Containers
- Built on pure white surfaces with `1rem` (16px) radius, framed by `1px solid #F1F5F9`.
- **Stat Metric Cards**: Triple-column blocks (Streak, Vocabulary count, Accuracy) featuring soft pastel icon background squircles (`#FEF3C7` for streak fire, `#DCFCE7` for vocab, `#DBEAFE` for accuracy) paired with prominent bold figures.

### Assessment Choices (Quiz & Drills)
- Radio/selection alternatives styled as horizontal block cards: 16px internal padding, `rounded-lg` (12px), bordered with `#E2E8F0`.
- Includes a circular index badge (A, B, C, D) on the leading edge.
- **Selected**: Blue wash background (`#EFF6FF`), outline transitions to `#2563EB` (2px).
- **Success / Failure**: On submit, transitions instantly to `#ECFDF5` (`#10B981` border) or `#FEF2F2` (`#EF4444` border).

### Fill-in-the-Blank Word Banks
- Floating draggable/tappable tokens shaped in soft pills (`rounded-lg`, 6px–8px radius) resting on `#FFFFFF` with soft borders. Target sentence blanks match the exact dimension and height of tokens with a dashed placeholder border (`#CBD5E1`).

### Audio & Dialogue Players
- Enclosed white card housing track progress (`#2563EB` active fill over `#E2E8F0` trough), quick playback rate toggles (1x, 0.8x), and transcript blocks highlighting spoken words in real time.

### Bottom Navigation Bar
- Fixed 64px container with white glass backdrop (`backdrop-filter: blur(12px)`), displaying 4 key destinations (Home, Learn, Review, Account). Active items show colored vector icons with micro-labels in `#2563EB`; inactive items sit in `#94A3B8`.