---
adr: 0002
id: adr-0002-shadcn-tailwind
title: Use shadcn/ui + Tailwind (Radix) for UI Primitives
status: Accepted
date: 2025-11-02
deciders: [core, frontend]
tags: [frontend, ui, design-system]
relatedDocs:
  - docs/web-app.md
---

## Context
We want a consistent, accessible UI kit that works with React 19, supports theming, and is easy to customize without locking into a heavy design system.

## Decision
Adopt shadcn/ui (built on Radix primitives) with Tailwind CSS as the styling system for shared UI components.

## Consequences
- Consistent primitives (Button, Input, Dialog, etc.) with accessible defaults.
- Theming via CSS variables/Tailwind tokens; dark/light supported at shell level.
- Minimal runtime overhead; components are local code we can adapt.

## Alternatives Considered
- MUI/Chakra: larger abstraction, theming overhead, harder to match bespoke admin look.
- Headless UI only: more composition work to reach parity.

## Links
- docs/web-app.md

