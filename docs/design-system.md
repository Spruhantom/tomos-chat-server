# TomChat design system

The visual direction follows the TomOS neon interface: dark surfaces, high-contrast text, electric cyan and blue accents, restrained glow effects, clear status indicators and keyboard-accessible navigation.

## Principles
- Prioritise readability and predictable navigation over decoration.
- Use cyan for focus and primary interactive states; blue for secondary emphasis.
- Distinguish connection, delivery, encryption and trust states with text and icons, not colour alone.
- Make error, offline and unverified states explicit.
- Support compact mobile layouts, wide desktop layouts and terminal interfaces.
- Never imply a message is encrypted, delivered or trusted unless the underlying state confirms it.

The TUI uses a Ratatui implementation of this visual direction. Android-specific Compose styling is documented in the Android source project and remains to be migrated.
