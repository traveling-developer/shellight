# Design

Mockups for shellight's look ("Native · Chartreuse"). Take their values
(colours, sizes, spacing, radii) into shellight's own Svelte components and
CSS variables; do not copy the markup. They are meant to be read, not
rendered.

## Where the app differs from the mockups

- **Menu, notifications, restart dialog stay native.** A sandbox line in the
  menu is a status-shape icon plus "name — Phase"; "Update available" is a
  plain item. Notifications show the app icon only. The restart dialog uses
  the mockup's texts with the system button colour.
- **Tray icon:** the glyph (frame with ">_", dot bottom right) stays; only
  its colours change:

  | | white glyph (dark menu bar) | black glyph (light menu bar) |
  |---|---|---|
  | glyph | #FFFFFF | #111111 |
  | ok | #B8E62A | #8DC21A |
  | busy | #F2B84B | #B4560B |
  | error | #FF7A66 | #C4321F |
  | no result yet | #9A9E95 | #9A9E95 |

- **Dashboard sidebar** keeps the phase: status shape, name, phase
  right-aligned (Error in the error colour), then the policy proposal badge.
- **Port forwards** in the details are always shown, "none" when empty;
  "Policy proposals" shows "none pending".
- **Views without a mockup** (gateway unreachable, no sandboxes, workspace
  view with providers, version banner) reuse the same building blocks:
  section titles, tiles, framed lists, the red-tinted error box.
