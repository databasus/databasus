export interface SiteLink {
  label: string;
  href: string;
  isExternal: boolean;
}

// The Discord link goes through the site's /discord redirect, so a new invite changes one file
export const COMMUNITY_LINKS: SiteLink[] = [
  {
    label: "Telegram",
    href: "https://t.me/databasus_community",
    isExternal: true,
  },
  { label: "Discord", href: "/discord", isExternal: true },
];

export const SOCIAL_MEDIA_LINKS: SiteLink[] = [
  {
    label: "LinkedIn",
    href: "https://www.linkedin.com/company/databasus/",
    isExternal: true,
  },
  {
    label: "Bluesky",
    href: "https://bsky.app/profile/databasus.bsky.social",
    isExternal: true,
  },
  { label: "X", href: "https://x.com/databasus", isExternal: true },
];
