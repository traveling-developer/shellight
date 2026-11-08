# shellight is a companion to the openshell CLI

shellight requires an installed and configured openshell CLI and always follows
the CLI's active gateway, reading its endpoint and credentials from the CLI
config. It has no onboarding, login or gateway switcher of its own, and
supports only the auth modes a local gateway uses (mtls, plaintext, none).
Rarely used or destructive work (creating sandboxes, managing providers,
editing policy rules) stays in the CLI; shellight only watches and offers
small, targeted actions. This keeps shellight small and avoids a second,
diverging source of gateway configuration.
