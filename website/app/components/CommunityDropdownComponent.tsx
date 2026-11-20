"use client";

import { useEffect, useRef, useState } from "react";
import { COMMUNITY_LINKS } from "./communityLinks";

export default function CommunityDropdownComponent({
  label,
  triggerClassName,
}: {
  label: string;
  triggerClassName: string;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!isOpen) return;

    const closeOnOutsideClick = (event: MouseEvent) => {
      if (!dropdownRef.current?.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    const dismissOnEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setIsOpen(false);
      buttonRef.current?.focus();
    };

    document.addEventListener("mousedown", closeOnOutsideClick);
    document.addEventListener("keydown", dismissOnEscape);
    return () => {
      document.removeEventListener("mousedown", closeOnOutsideClick);
      document.removeEventListener("keydown", dismissOnEscape);
    };
  }, [isOpen]);

  return (
    <div
      className="relative"
      ref={dropdownRef}
      onMouseEnter={() => setIsOpen(true)}
      onMouseLeave={() => setIsOpen(false)}
    >
      <button
        ref={buttonRef}
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className={`cursor-pointer ${triggerClassName}`}
        aria-expanded={isOpen}
      >
        {label}
      </button>

      {isOpen && (
        // pt-2 instead of a margin keeps the hover area contiguous, so moving
        // the cursor from the button to the list does not close the dropdown
        <div className="absolute left-0 top-full z-50 pt-2">
          <div className="w-40 overflow-hidden rounded-lg border border-[#ffffff20] bg-[#0F1115] py-1 shadow-2xl">
            {COMMUNITY_LINKS.map((link) => (
              <a
                key={link.href}
                href={link.href}
                target="_blank"
                rel="noopener noreferrer"
                className="block px-3 py-2 text-base text-gray-300 transition-colors hover:bg-[#1f2937] hover:text-white"
              >
                {link.label}
              </a>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
