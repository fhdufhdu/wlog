# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Stack

Static HTML and CSS with minimal vanilla JavaScript. HTMX is deliberately deferred until server-rendered partial interactions exist.

## Users

The primary user is the blog owner, who publishes and manages a personal blog. Readers visit to scan recent writing and read individual posts without interface noise.

## Product Purpose

A personal blog that makes writing easy to browse and comfortable to read. The current deliverable is a functional frontend mockup rather than a connected publishing backend.

## Positioning

The blog is a small, independent web publication: direct, text-led, and influenced by the plain-document discipline of bzcf.io without copying its composition or becoming a news aggregator or RSS reader.

## Operating Context

Readers browse a chronological post index, search or filter writing, open an article, and view basic author information. The owner can open a settings surface and configure a login ID and password as a frontend demonstration.

## Capabilities and Constraints

- Includes post index, article reading, search/filter, archive/about navigation, and an owner settings surface.
- Includes frontend-only ID/password configuration behavior; real authentication, secure password storage, publishing, and persistence require a backend and are out of scope.
- RSS and feed-reader behavior are explicitly out of scope.
- The product name, author identity, real article content, deployment target, and backend are open decisions. Mock content must be identifiable as demonstration content.

## Brand Commitments

The interface takes only content immediacy and low decorative chrome from bzcf.io. Wlog owns a cool-gray editorial index system with serif writing, SUIT interface text, numbered post rows, aligned category/date metadata, cobalt navigation, and vermilion status cues. The first viewport starts with writing rather than an introduction or promotional message.

## Evidence on Hand

No real posts, author biography, logo, photography, claims, or production credentials were supplied. The frontend uses clearly marked demonstration copy.

## Product Principles

- Writing is always more prominent than interface chrome.
- Dense does not mean cramped: scanning and reading get distinct rhythms.
- Navigation and settings remain obvious without dominating the page.
- The frontend stays understandable without a framework.
- Security-sensitive behavior is never implied to be production-ready when it is only mocked.
