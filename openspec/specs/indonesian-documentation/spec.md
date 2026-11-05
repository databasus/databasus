# Indonesian Documentation Specification

## Purpose

Lets Indonesian readers follow the same current installation, security and product guidance as readers of the English README and website.

## Requirements

### Requirement: Indonesian documentation matches published guidance

The Indonesian README and website SHALL preserve the meaning, technical instructions, links and section anchors of the current English sources. They SHALL omit content removed from the published English documentation.

#### Scenario: A reader follows installation instructions

- **WHEN** a reader opens the Indonesian installation guide or README
- **THEN** commands, configuration values and supported product behavior agree with the English source
- **AND** the surrounding explanations are in Indonesian

#### Scenario: Upstream removes a documentation page

- **WHEN** the English website no longer publishes a page
- **THEN** Indonesian navigation does not link to that page
- **AND** the Indonesian website does not retain an orphaned translation of it

### Requirement: Readers can navigate Indonesian documentation

The website SHALL offer Bahasa Indonesia in its language selection and publish Indonesian versions of its translated routes with matching language metadata. Language changes SHALL preserve the current page where a translation exists and preserve section anchors. The README language navigation SHALL link to its Indonesian translation.

#### Scenario: A reader changes documentation language

- **WHEN** a reader selects Bahasa Indonesia on a translated website page
- **THEN** the corresponding Indonesian page opens with Indonesian navigation and language metadata
- **AND** an existing section anchor is preserved

#### Scenario: A reader opens the Indonesian README

- **WHEN** a reader follows Bahasa Indonesia from the root README
- **THEN** the Indonesian README opens with working links back to the available language versions
