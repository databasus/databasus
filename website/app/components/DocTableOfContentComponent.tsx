"use client";

import { useEffect, useState, useRef } from "react";
import { usePathname } from "next/navigation";
import { splitLocaleFromPathname } from "@/app/i18n";
import { TABLE_OF_CONTENTS_LABELS } from "./landingLabels";

interface Heading {
  id: string;
  text: string;
  level: number;
}

export default function DocTableOfContentComponent() {
  const { locale } = splitLocaleFromPathname(usePathname());
  const [headings, setHeadings] = useState<Heading[]>([]);
  const [activeId, setActiveId] = useState<string>("");
  const observerRef = useRef<IntersectionObserver | null>(null);

  useEffect(() => {
    // Use a timeout to avoid cascading renders
    const timeoutId = setTimeout(() => {
      // Get all h1, h2, h3 headings from the page
      const elements = Array.from(document.querySelectorAll("article h1, article h2, article h3"));

      if (elements.length === 0) return;

      const headingData: Heading[] = elements.map((element) => ({
        id: element.id,
        text: element.textContent || "",
        level: parseInt(element.tagName.substring(1)),
      }));

      setHeadings(headingData);

      // Set up intersection observer to track active heading
      observerRef.current = new IntersectionObserver(
        (entries) => {
          entries.forEach((entry) => {
            if (entry.isIntersecting) {
              setActiveId(entry.target.id);
            }
          });
        },
        { rootMargin: "-100px 0px -80% 0px" },
      );

      elements.forEach((element) => {
        observerRef.current?.observe(element);
      });
    }, 0);

    return () => {
      clearTimeout(timeoutId);
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, []);

  const handleClick = (id: string) => {
    const element = document.getElementById(id);
    if (element) {
      element.scrollIntoView({ behavior: "smooth", block: "start" });
      // Update URL hash without triggering a scroll
      window.history.pushState(null, "", `#${id}`);
    }
  };

  if (headings.length === 0) {
    return null;
  }

  return (
    <aside className="hidden w-64 border-l border-[#ffffff20] bg-[#0F1115] xl:block">
      <div className="sticky top-0 h-screen overflow-y-auto p-6">
        <h3 className="mb-3 text-xs font-semibold uppercase tracking-wide text-gray-500">
          {TABLE_OF_CONTENTS_LABELS[locale]}
        </h3>
        <nav>
          <ul className="space-y-2 text-sm">
            {headings.map((heading) => (
              <li
                key={heading.id}
                style={{ paddingLeft: `${(heading.level - 1) * 0.75}rem` }}
                className="relative"
              >
                <button
                  onClick={() => handleClick(heading.id)}
                  className={`block w-full text-left transition-all duration-200 cursor-pointer relative pl-3 py-0.5 ${
                    activeId === heading.id
                      ? "text-white font-normal"
                      : "text-gray-500 hover:text-white"
                  }`}
                >
                  <span
                    className={`absolute left-0 top-0 bottom-0 w-0.5 transition-all duration-200 ${
                      activeId === heading.id ? "bg-blue-600 opacity-100" : "bg-gray-600 opacity-0"
                    }`}
                  />
                  {heading.text}
                </button>
              </li>
            ))}
          </ul>
        </nav>
      </div>
    </aside>
  );
}
