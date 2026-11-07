// @vitest-environment happy-dom
import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import type { Workspace } from '@/entities/workspaces';
import { DICTIONARIES } from '@/shared/i18n/dictionaries';
import { renderScreen } from '@/shared/lib/test/renderScreen';
import { WorkspaceSelectionComponent } from '@/widgets/app-shell/ui/WorkspaceSelectionComponent';

const workspaces: Workspace[] = [
  {
    id: 'first-workspace',
    name: 'First team',
    currentUserRole: 'WORKSPACE_OWNER',
    createdAt: '2026-10-01T00:00:00Z',
  },
  {
    id: 'second-workspace',
    name: 'Second team',
    currentUserRole: 'WORKSPACE_OWNER',
    createdAt: '2026-10-01T00:00:00Z',
  },
];

describe('WorkspaceSelectionComponent', () => {
  it.each(['en', 'ru'] as const)(
    'opens creation from the filtered dropdown with keyboard access when the locale is %s',
    async (locale) => {
      const onCreateWorkspace = vi.fn();
      const onWorkspaceSelect = vi.fn();
      const { user } = renderScreen(
        <WorkspaceSelectionComponent
          workspaces={workspaces}
          selectedWorkspace={workspaces[0]}
          canCreateWorkspace
          onCreateWorkspace={onCreateWorkspace}
          onWorkspaceSelect={onWorkspaceSelect}
        />,
        { locale },
      );
      const dictionary = DICTIONARIES[locale].app.workspaceSelection;
      const selector = screen.getByRole('combobox', { name: dictionary.selected });

      expect(screen.queryByRole('button', { name: dictionary.createNew })).toBeNull();

      await user.click(selector);
      await user.type(
        screen.getByRole('textbox', { name: dictionary.selected }),
        'No matching team',
      );
      await user.tab();

      const create = await screen.findByRole('button', { name: dictionary.createNew });
      expect(document.activeElement).toBe(create);

      await user.keyboard('{Enter}');

      expect(onCreateWorkspace).toHaveBeenCalledTimes(1);
      expect(onWorkspaceSelect).not.toHaveBeenCalled();
      await waitFor(() => expect(selector.getAttribute('aria-expanded')).toBe('false'));
      expect(document.activeElement).toBe(selector);
    },
  );

  it('switches to a matching workspace without offering creation when permission is absent', async () => {
    const onWorkspaceSelect = vi.fn();
    const { user } = renderScreen(
      <WorkspaceSelectionComponent
        workspaces={workspaces}
        selectedWorkspace={workspaces[0]}
        canCreateWorkspace={false}
        onCreateWorkspace={vi.fn()}
        onWorkspaceSelect={onWorkspaceSelect}
      />,
    );
    const selector = screen.getByRole('combobox', { name: 'Selected workspace' });

    await user.click(selector);
    await user.type(screen.getByRole('textbox', { name: 'Selected workspace' }), 'Second');
    expect(
      screen.queryByRole('button', { name: DICTIONARIES.en.app.workspaceSelection.createNew }),
    ).toBeNull();
    await user.click(await screen.findByText('Second team'));

    expect(onWorkspaceSelect).toHaveBeenCalledWith(workspaces[1]);
  });

  it.each([true, false])(
    'offers initial workspace creation according to permission when canCreateWorkspace=%s',
    async (canCreateWorkspace) => {
      const onCreateWorkspace = vi.fn();
      const { user } = renderScreen(
        <WorkspaceSelectionComponent
          workspaces={[]}
          canCreateWorkspace={canCreateWorkspace}
          onCreateWorkspace={onCreateWorkspace}
          onWorkspaceSelect={vi.fn()}
        />,
      );

      expect(screen.queryByRole('combobox')).toBeNull();
      const create = screen.queryByRole('button', { name: 'Create workspace' });

      if (canCreateWorkspace) {
        await user.click(create!);

        expect(onCreateWorkspace).toHaveBeenCalledTimes(1);
      } else {
        expect(create).toBeNull();
      }
    },
  );
});
