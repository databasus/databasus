import { Button, Input } from 'antd';
import { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { type Workspace } from '../../entities/workspaces';
import { useIsMobile } from '../../shared/hooks';

interface Props {
  workspaces: Workspace[];
  selectedWorkspace?: Workspace;
  onCreateWorkspace: () => void;
  onWorkspaceSelect: (workspace: Workspace) => void;
}

export const WorkspaceSelectionComponent = ({
  workspaces,
  selectedWorkspace,
  onCreateWorkspace,
  onWorkspaceSelect,
}: Props) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const dropdownRef = useRef<HTMLDivElement>(null);

  const filteredWorkspaces = useMemo(() => {
    if (!searchValue.trim()) return workspaces;
    const searchTerm = searchValue.toLowerCase();
    return workspaces.filter((workspace) => workspace.name.toLowerCase().includes(searchTerm));
  }, [workspaces, searchValue]);

  const openWorkspace = (workspace: Workspace) => {
    setIsDropdownOpen(false);
    setSearchValue('');
    onWorkspaceSelect?.(workspace);
  };

  // Handle click outside dropdown
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
        setSearchValue('');
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  if (workspaces.length === 0) {
    return (
      <Button
        type="primary"
        onClick={onCreateWorkspace}
        size={isMobile ? 'small' : 'middle'}
        className="border-blue-600 bg-blue-600 hover:border-blue-700 hover:bg-blue-700"
      >
        {isMobile ? t('app.workspaceSelection.createShort') : t('app.workspaceSelection.create')}
      </Button>
    );
  }

  return (
    <div
      className="my-1 flex-1 select-none md:ml-2 md:w-[250px] md:max-w-[242px]"
      ref={dropdownRef}
    >
      <div className="mb-1 hidden text-xs text-gray-400 md:block" style={{ lineHeight: 0.7 }}>
        {t('app.workspaceSelection.selected')}
      </div>

      <div className="relative">
        <div
          className="cursor-pointer rounded bg-gray-100 p-1 px-2 hover:bg-gray-200 dark:bg-gray-700 dark:hover:bg-gray-600"
          onClick={() => setIsDropdownOpen(!isDropdownOpen)}
        >
          <div className="flex items-center justify-between text-sm dark:text-gray-200">
            <div className="flex-1 truncate pr-1">
              {selectedWorkspace?.name || t('app.workspaceSelection.placeholder')}
            </div>
            <img
              src="/icons/menu/arrow-down-gray.svg"
              alt=""
              className={`ml-1 flex-shrink-0 transition-transform duration-200 ${isDropdownOpen ? 'rotate-180' : ''}`}
              width={isMobile ? 14 : 15}
              height={isMobile ? 14 : 15}
            />
          </div>
        </div>

        {isDropdownOpen && (
          <div className="absolute top-full right-0 left-0 z-50 mt-1 min-w-[250px] rounded-md border border-gray-200 bg-white shadow-lg md:right-auto md:left-0 md:min-w-full dark:border-gray-600 dark:bg-gray-800">
            <div className="border-b border-gray-100 p-2 dark:border-gray-700">
              <Input
                placeholder={t('app.workspaceSelection.search')}
                value={searchValue}
                onChange={(e) => setSearchValue(e.target.value)}
                className="border-0 shadow-none"
                size={isMobile ? 'small' : 'middle'}
                autoFocus
              />
            </div>

            <div className="max-h-[250px] overflow-y-auto md:max-h-[400px]">
              {filteredWorkspaces.map((workspace) => (
                <div
                  key={workspace.id}
                  className="cursor-pointer truncate px-3 py-2 text-sm hover:bg-gray-50 dark:text-gray-200 dark:hover:bg-gray-700"
                  onClick={() => openWorkspace(workspace)}
                >
                  {workspace.name}
                </div>
              ))}

              {filteredWorkspaces.length === 0 && searchValue && (
                <div className="px-3 py-2 text-sm text-gray-500 dark:text-gray-400">
                  {t('app.workspaceSelection.notFound')}
                </div>
              )}
            </div>

            <div className="border-t border-gray-100 dark:border-gray-700">
              <div
                className="cursor-pointer px-3 py-2 text-sm text-blue-600 hover:bg-gray-50 hover:text-blue-700 dark:hover:bg-gray-700"
                onClick={() => {
                  onCreateWorkspace();
                  setIsDropdownOpen(false);
                  setSearchValue('');
                }}
              >
                {t('app.workspaceSelection.createNew')}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
