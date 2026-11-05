## Why

The SFTP editor offers an option to skip host key verification without explaining that it reduces security. Restore the warning before release so users understand the risk before enabling the option.

## What Changes

- Show the existing translated security warning beside the host key verification checkbox whenever its advanced settings are visible.
- Associate the warning with the checkbox so assistive technology can read it as the control's description.
- Add a regression test that fails on the current editor and covers the warning before the option is enabled, with the option already enabled, and in all six interface languages.
- Repair a pre-existing WAL statistics test exposed by the full verification run: compare minute buckets with the application's manual clock rather than wall time.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `storages`: Require an accessible, localized security warning alongside the option to skip SFTP host key verification.

## Impact

The change answers to root `AGENTS.md` and `frontend/AGENTS.md`. Implementation touches `frontend/src/features/edit-storage/ui/edit/storages/EditSFTPStorageComponent.tsx` and a new adjacent screen test. The existing `storages.edit.sftp.skipHostKeyTooltip` key supplies the text in all six dictionaries. The verification repair also touches `backend/internal/modules/backups/physicalapitests/streamstatistics/statistics_test.go` under `backend/AGENTS.md`. No dependencies or compatibility breaks are introduced.

### Out of Scope

- Changing warning wording or translations.
- Changing the option's default, persistence or advanced-settings disclosure behavior.
- Changing SFTP authentication, host key verification or backup execution.
- Changing HTTP contracts, backend production code or other storage editors.
