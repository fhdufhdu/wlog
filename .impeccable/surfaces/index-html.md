---
version: 1
slug: "index-html"
primary_target: "templates/index.html"
related_targets: ["templates/base.html","templates/post.html","styles.css"]
---

## Scope and mode

Server-rendered personal blog; public Read mode with a separate authenticated authoring surface.

## Audience, job, and task

Readers scan recent posts by category and open articles. The owner signs in to write and manage posts.

## Content and constraints

Rust/Axum renders HTML from PostgreSQL content. Credentials come from environment secrets; RSS remains out of scope.

## Chosen direction

Use bzcf.io only as an atmospheric reference for content immediacy and low chrome, not as a composition to reproduce (`seed-key: user-steer:similar-not-copy`). Wlog is an annotated working index: cool-gray paper, self-hosted Maru Buri writing, SUIT interface text, numbered chronology, cobalt navigation, and vermilion status cues. It starts directly with writing and has no manifesto, hero, marketing copy, or card layout.

## Composition and component grammar

- Desktop uses a 760px numbered record index with a sticky 184px topic register; articles remain inside a narrower reading measure.
- A 34px serif masthead and small SUIT navigation sit above a thin gray rule. On phones, three primary routes become a safe-area-aware bottom navigation.
- Recent posts are aligned editorial rows: sequence number, serif title, tabular date, topic, and description.
- The category index is vertical on desktop, then becomes a horizontally scrollable, 44px-touch-target filter above the mobile list.
- Phone post items retain their sequence numbers but stack the metadata under a full-width 44px title target.
- Article headings, blockquotes, code blocks, tables, and images use the same document grammar rather than cards.
- Login and authoring forms use square gray-bordered fields and cobalt primary actions.

## Interaction and states

Public navigation uses ordinary links and works without JavaScript. The authoring script only generates slugs and inserts uploaded-image Markdown. Mobile save actions remain sticky.

## Unresolved decisions

Author identity, real post content, deployment target, and production social image.
