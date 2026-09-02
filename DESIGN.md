---
name: Wlog
description: A numbered, metadata-rich personal work index on cool-gray paper.
colors:
  cool-paper: "#f2f4f7"
  raised-paper: "#fafbfc"
  carbon-ink: "#1b1d20"
  slate-metadata: "#626974"
  quiet-rule: "#c6cbd2"
  strong-rule: "#7a828e"
  editorial-sky: "#117696"
  state-vermilion: "#d83a28"
  field-white: "#fff"
  selection-sky: "#bfe8f4"
  night-paper: "#171a1f"
  night-surface: "#1e2229"
  night-ink: "#f0f2f5"
  night-muted: "#aeb5c0"
  night-rule: "#454b55"
  night-strong-rule: "#89919d"
  night-link: "#82d0e8"
  night-signal: "#ff7868"
  night-field: "#20242b"
  night-selection: "#225365"
typography:
  masthead:
    fontFamily: '"Maru Buri", "Nanum Myeongjo", "AppleMyungjo", serif'
    fontSize: "34px"
    fontWeight: 700
    lineHeight: 1
    letterSpacing: "-.025em"
  headline:
    fontFamily: '"Maru Buri", "Nanum Myeongjo", "AppleMyungjo", serif'
    fontSize: "28px"
    fontWeight: 700
    lineHeight: 1.3
    letterSpacing: "-.02em"
  index-heading:
    fontFamily: '"Maru Buri", "Nanum Myeongjo", "AppleMyungjo", serif'
    fontSize: "26px"
    fontWeight: 700
    lineHeight: 1.25
    letterSpacing: "-.02em"
  record-title:
    fontFamily: '"Maru Buri", "Nanum Myeongjo", "AppleMyungjo", serif'
    fontSize: "18px"
    fontWeight: 700
    lineHeight: 1.45
    letterSpacing: "-.015em"
  reading:
    fontFamily: '"Maru Buri", "Nanum Myeongjo", "AppleMyungjo", serif'
    fontSize: "17px"
    fontWeight: 400
    lineHeight: 1.85
  interface:
    fontFamily: '"SUIT", system-ui, sans-serif'
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.6
  navigation:
    fontFamily: '"SUIT", system-ui, sans-serif'
    fontSize: "13px"
    fontWeight: 600
    lineHeight: 1.6
  metadata:
    fontFamily: '"SUIT", system-ui, sans-serif'
    fontSize: "11px"
    fontWeight: 400
    lineHeight: 1.6
rounded:
  square: "0"
spacing:
  micro: "4px"
  compact: "8px"
  field-x: "11px"
  control-x: "15px"
  row-y: "12px"
  section: "24px"
  page-x: "24px"
  page-bottom: "56px"
components:
  primary-button:
    backgroundColor: "{colors.editorial-sky}"
    textColor: "{colors.field-white}"
    typography: "{typography.interface}"
    rounded: "{rounded.square}"
    padding: "8px 15px"
    height: "42px"
  primary-button-hover:
    backgroundColor: "{colors.state-vermilion}"
    textColor: "{colors.field-white}"
    typography: "{typography.interface}"
    rounded: "{rounded.square}"
    padding: "8px 15px"
    height: "42px"
  text-input:
    backgroundColor: "{colors.field-white}"
    textColor: "{colors.carbon-ink}"
    typography: "{typography.interface}"
    rounded: "{rounded.square}"
    padding: "8px 11px"
    width: "100%"
    height: "42px"
  navigation-link:
    backgroundColor: "{colors.cool-paper}"
    textColor: "{colors.editorial-sky}"
    typography: "{typography.navigation}"
    rounded: "{rounded.square}"
    padding: "0"
  record-row:
    backgroundColor: "{colors.cool-paper}"
    textColor: "{colors.carbon-ink}"
    typography: "{typography.record-title}"
    rounded: "{rounded.square}"
    padding: "12px 0"
    width: "100%"
---

# Design System: Wlog

## Overview

**Creative North Star: "The Annotated Working Index"**

Wlog is a personal publication shaped as a working index rather than a stream, dashboard, or replica of another plain-text blog. Cool-gray paper, aligned record rows, and terse annotations give it the authority of an actively maintained register. The publication begins with writing and trusts sequence, title, topic, and date to explain the collection.

The visual system deliberately splits voices. Self-hosted Maru Buri gives the masthead, titles, and prose a literary editorial character; self-hosted SUIT keeps navigation, counts, dates, controls, and administrator forms exact and operational. A muted editorial sky tone helps the reader move, while vermilion marks freshness, errors, and decisive feedback.

**Key Characteristics:**

- A cool-gray, 1120px page frame with a 760px record index and 150–184px topic register.
- A 34px Maru Buri masthead, thin gray header rule, and writing immediately in the first viewport.
- Numbered rows aligning serif titles against SUIT dates, topics, and freshness states.
- Muted sky navigation and action color, with vermilion reserved for urgent state and response.
- Square, ruled fields and fieldsets that keep owner configuration inside the publication world.
- A semantic dark inversion and responsive single-axis reading path.

## Colors

The palette resembles cool archival paper marked with carbon, muted sky indexing ink, and restrained vermilion annotations.

### Primary

- **Editorial Sky** (editorial-sky): The fixed top registration bar, links, active mobile route, topic labels, input focus edge, reading movement, and primary actions.
- **State Vermilion** (state-vermilion): Freshness labels, focus outlines, validation, and interaction moments that need immediate recognition.

### Neutral

- **Cool Paper** (cool-paper): The page canvas and mobile sticky header.
- **Raised Paper** (raised-paper): Fieldsets and quiet secondary controls; it separates operational regions tonally without becoming a card.
- **Carbon Ink** (carbon-ink): Titles, reading text, strong dividers, and current selections.
- **Slate Metadata** (slate-metadata): Dates, counts, notes, footer text, sequence numbers, and supporting labels.
- **Quiet Rule** (quiet-rule): Record separators and secondary structural lines.
- **Strong Rule** (strong-rule): The header edge, control borders, footer edge, and other firm boundaries.
- **Field White** (field-white): Light-theme input surfaces and white text on saturated actions.
- **Selection Sky** (selection-sky): Text selection in the light theme.
- **Night Paper**, **Night Surface**, **Night Ink**, **Night Muted**, **Night Rule**, **Night Strong Rule**, and **Night Field**: The dark-theme neutral inversion.
- **Night Link**, **Night Signal**, and **Night Selection**: Dark-theme semantic counterparts for navigation, state, and selection.

### Named Rules

**The Sky-Leads Rule.** Editorial Sky denotes routes, filters, focus, and primary action; it is functional indexing ink, never ambient decoration.

**The Vermilion-Responds Rule.** Vermilion appears only when something is current, fresh, progressing, invalid, focused, or actively responding.

**The Semantic Inversion Rule.** Dark appearance changes token values, not hierarchy, geometry, density, or component roles.

## Typography

**Display Font:** Maru Buri, self-hosted in regular, semibold, and bold weights, with Korean serif fallbacks.

**Body Font:** Maru Buri for articles and editorial titles; SUIT Variable for interface copy and metadata.

**Label Font:** SUIT Variable with tabular numerals for sequence numbers, dates, years, and counts.

**Character:** The pairing is editorial without nostalgia and operational without app-shell sterility. Maru Buri slows the eye at titles and prose; SUIT keeps the surrounding index precise, compact, and easy to compare.

### Hierarchy

- **Masthead** (masthead): The 34px publication mark; it is the strongest typographic signature without becoming a hero.
- **Headline** (headline): Page and article headings above firm document rules.
- **Index Heading** (index-heading): The recent-post heading paired with a compact count.
- **Record Title** (record-title): The primary scan target in each numbered row.
- **Reading** (reading): Sustained 17px prose at 1.85 line height within a 70ch measure.
- **Interface** (interface): Forms, controls, and general operational language.
- **Navigation** (navigation): The compact top navigation, topic register, and settings tabs.
- **Metadata** (metadata): Sequence numbers, dates, topics, freshness labels, year marks, and small counts.

### Named Rules

**The Two-Voice Rule.** Maru Buri owns authored content and editorial hierarchy; SUIT owns navigation, metadata, controls, and system feedback.

**The Tabular Record Rule.** Dates, years, counts, and row numbers use tabular numerals so the index reads by columns rather than by decoration.

## Layout

The page is centered within a 1120px frame with 24px horizontal padding. Its desktop index uses a 760px primary column and a 150–184px sticky topic register, separated by a fluid 52–72px gutter. The header stops at 1016px, aligns the masthead and compact navigation along its baseline, and leaves 42px before the index. Article views narrow to 720px, with prose capped at 70ch.

Each desktop record is a minimum 72px three-column grid: 42px sequence, flexible title, and 190px right-aligned metadata. The index heading and count share a baseline above a two-pixel carbon rule. This repeated alignment—not containers or imagery—creates the page's visual rhythm.

Below 960px the topic register moves above the list as a horizontally scrollable filter. Below 720px the masthead becomes a sticky 27px anchor, four primary routes become a fixed safe-area-aware bottom navigation, record rows become two-column 88px structures, and metadata stacks beneath a 44px title target. Fields grow to 48px, pagination to 44px, and settings actions remain sticky above bottom navigation.

**The Immediate-Index Rule.** The first viewport contains the masthead, thin rule, recent-post heading and count, numbered records, and—when width permits—the topic register. No hero or introductory panel precedes them.

**The Aligned-Record Rule.** Preserve the relationship between sequence, title, date, topic, and state. When space tightens, stack metadata beneath the title rather than discarding the sequence or turning records into cards.

## Elevation & Depth

The system is flat and paper-like. Hierarchy comes from cool tonal steps, one- and two-pixel rules, type contrast, spacing, and the fixed sky registration bar. Focused fields use a crisp doubled sky edge rather than a glow.

### Shadow Vocabulary

- **Field Focus Edge** (`0 0 0 1px var(--link)`): Reinforces the sky input border during focus.

### Named Rules

**The Flat-Register Rule.** Persistent information stays in the document plane; rules and tonal paper changes carry structure, and shadow is reserved for transient feedback.

## Shapes

Wlog is rectilinear. Buttons, inputs, selects, fieldsets, tabs, pagination, and navigation states use zero radius. Horizontal rules and the four-pixel registration/progress bar are the dominant geometry; there are no pills, floating cards, or ornamental silhouettes.

**The Square-Annotation Rule.** Operational elements may be bordered, filled, or underlined, but their corners remain square so they read as annotations on a working document.

## Components

Components are recognizable web controls translated into the same aligned editorial grammar.

### Buttons

- **Shape:** Square, minimum 42px high, with an exact one-pixel Strong Rule border and 8px by 15px padding.
- **Primary:** Editorial Sky fill, white text, and bold SUIT; hover swaps the fill and border to State Vermilion.
- **Secondary:** Raised Paper with Carbon Ink and a Strong Rule border; hover uses a sky edge or sky text according to context.
- **Focus / Active:** Every interactive element receives a two-pixel vermilion focus outline with a three-pixel offset. Record-title activation changes to vermilion.

### Inputs / Fields

- **Style:** Full-width square Field White controls, minimum 42px high, with a Strong Rule border and 8px by 11px padding.
- **Focus:** The border and doubled edge become Editorial Sky while the caret and keyboard outline remain State Vermilion.
- **Groups:** Owner settings use a Raised Paper fieldset with a serif legend, a firm gray border, and 17px field rhythm.
- **Error / Disabled:** Error copy sits directly below its field in vermilion. Disabled pagination keeps the same geometry with muted color and reduced opacity.

### Navigation

The 34px masthead is Carbon Ink and shifts to sky on hover. Desktop routes are compact SUIT links; the current route returns to Carbon Ink and gains a two-pixel vermilion underline offset seven pixels below the text. The sticky topic register uses ruled, count-bearing rows and prefixes its current label with a vermilion dash. On phones, primary routes become safe-area-aware bottom navigation targets and the current route gains a three-pixel sky top edge.

### Numbered Record Index

The signature component is an ordered visual system implemented with counters rather than native markers. Each 72px row aligns a two-digit sequence, bold 18px Maru Buri title, tabular date, sky topic, and optional bold vermilion `새 글` state. Titles underline in sky on hover; rows remain transparent and separated only by Quiet Rules. On phones, the sequence persists while title and metadata stack.

### Article, Archive, and Feedback

Articles preserve the same paper and rules, with 17px Maru Buri prose. Blockquotes are tonal Raised Paper annotations enclosed by a Quiet Rule border.

### Named Rules

**The Shared-Publication Rule.** Reading and administrator forms use the same paper, rules, type roles, and semantic colors; administration never splits into a dashboard world.

## Do's and Don'ts

### Do:

- **Do** begin with the 34px masthead, thin header rule, recent-post heading and count, and numbered writing records.
- **Do** keep desktop records aligned as sequence, title, date, topic, and state within the 760px index.
- **Do** use Maru Buri for editorial hierarchy and prose, and SUIT for interface language and metadata.
- **Do** use muted sky for navigation, focus, and action, and vermilion for urgent, fresh, error, and active response.
- **Do** preserve square controls, visible keyboard focus, dark-token inversion, safe areas, and reduced-motion behavior.
- **Do** keep owner configuration honest and visibly part of the same publication grammar.

### Don't:

- **Don't** copy the composition of the atmospheric reference or regress to a generic plain-text blog list.
- **Don't** introduce a hero, manifesto, promotional introduction, or marketing call to action before the writing.
- **Don't** turn records, topics, articles, or settings into rounded cards, pills, dashboard tiles, or thumbnail feeds.
- **Don't** collapse the two-voice type system into all-serif or all-interface typography.
- **Don't** use sky or vermilion as general decoration, or replace aligned metadata with loose badges.
- **Don't** imply RSS behavior, production authentication, secure password persistence, or backend publishing through visual affordances.
