## ADDED Requirements

### Requirement: Frontend checks reject new style bypasses

Frontend lint SHALL reject newly introduced direct color values outside approved theme and chart-palette owners, and removal of visible focus without an accessible replacement. Coverage SHALL include component styles, style sheets and utility-class colors, including named colors, functional color notation and default color-palette utilities. Layout utilities, semantic theme references and technical data SHALL remain permitted. Diagnostics SHALL identify the source and violated rule.

#### Scenario: A utility class bypasses the theme

- **WHEN** a contributor adds a direct palette color to an ordinary shared control
- **THEN** frontend lint fails even if the color contains no hexadecimal literal

#### Scenario: A theme owner defines a color

- **WHEN** an approved theme or chart-palette owner defines a color for its declared role
- **THEN** the color is accepted without exempting unrelated component styles

#### Scenario: Focus is removed without a replacement

- **WHEN** a contributor disables a control's focus indication without supplying a visible replacement
- **THEN** frontend lint fails

### Requirement: Style migration exceptions are finite and shrinking

Existing style violations SHALL be represented by individual findings with their rule, source, normalized offending value or construct, occurrence count, owner and removal milestone. A new occurrence, an increased count or an unused exception SHALL fail lint. A finding SHALL remain recognizable after unrelated line movement. Wildcard exemptions for whole files, folders or utility palettes SHALL not authorize future violations.

#### Scenario: An existing color is copied

- **WHEN** an accepted existing style violation is copied into another occurrence
- **THEN** lint fails for the added occurrence

#### Scenario: A violation is repaired

- **WHEN** a recorded violation disappears from the source
- **THEN** lint fails until its stale exception is removed
