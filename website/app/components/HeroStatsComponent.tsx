"use client";

import { ReactNode, useEffect, useState } from "react";

const CACHE_KEY = "databasus.hero-stats";
const CACHE_TTL_MS = 30 * 60 * 1000;

const GITHUB_STARS_URL = "https://api.github.com/repos/databasus/databasus";
// Docker Hub serves no CORS headers, so the pull count comes from the shields.io mirror
const DOCKER_PULLS_URL =
  "https://img.shields.io/docker/pulls/databasus/databasus.json";

type Stat =
  | { status: "loading" }
  | { status: "ready"; text: string }
  | { status: "failed" };

type CachedStats = {
  stars: string | null;
  pulls: string | null;
  fetchedAt: number;
};

function getValidCount(count: unknown): string | null {
  return typeof count === "string" &&
    /^\d+(?:,\d{3})*(?:\.\d+)?[kMBGTPE]?$/i.test(count)
    ? count
    : null;
}

function readCache(): CachedStats | null {
  try {
    const rawCache = window.localStorage.getItem(CACHE_KEY);
    if (!rawCache) return null;

    const parsedCache: unknown = JSON.parse(rawCache);
    if (typeof parsedCache !== "object" || parsedCache === null) return null;

    const { stars, pulls, fetchedAt } = parsedCache as Record<string, unknown>;
    if (typeof fetchedAt !== "number" || !Number.isFinite(fetchedAt)) {
      return null;
    }

    return {
      stars: getValidCount(stars),
      pulls: getValidCount(pulls),
      fetchedAt,
    };
  } catch {
    return null;
  }
}

function writeCache(entry: CachedStats) {
  try {
    window.localStorage.setItem(CACHE_KEY, JSON.stringify(entry));
  } catch {
    // private mode or full storage: the stats still load, just without caching
  }
}

async function fetchStars(): Promise<string | null> {
  const response = await fetch(GITHUB_STARS_URL, {
    headers: { Accept: "application/vnd.github+json" },
  });
  if (!response.ok) return null;

  const payload: unknown = await response.json();
  const count = (payload as { stargazers_count?: unknown })?.stargazers_count;

  return typeof count === "number" ? count.toLocaleString("en-US") : null;
}

async function fetchPulls(): Promise<string | null> {
  const response = await fetch(DOCKER_PULLS_URL);
  if (!response.ok) return null;

  const payload: unknown = await response.json();
  const value = (payload as { value?: unknown })?.value;

  return getValidCount(value);
}

function toStat(text: string | null): Stat {
  return text === null ? { status: "failed" } : { status: "ready", text };
}

function getSettledValue(result: PromiseSettledResult<string | null>) {
  return result.status === "fulfilled" ? result.value : null;
}

export default function HeroStatsComponent() {
  const [stars, setStars] = useState<Stat>({ status: "loading" });
  const [pulls, setPulls] = useState<Stat>({ status: "loading" });

  useEffect(() => {
    let isCancelled = false;

    const loadHeroStats = async () => {
      const cached = readCache();
      if (cached) {
        if (isCancelled) return;
        setStars(toStat(cached.stars));
        setPulls(toStat(cached.pulls));

        if (Date.now() - cached.fetchedAt < CACHE_TTL_MS) return;
      }

      const [starsResult, pullsResult] = await Promise.allSettled([
        fetchStars(),
        fetchPulls(),
      ]);
      if (isCancelled) return;

      // a provider that fails keeps whatever the cache had, instead of blanking out
      const nextStars = getSettledValue(starsResult) ?? cached?.stars ?? null;
      const nextPulls = getSettledValue(pullsResult) ?? cached?.pulls ?? null;

      setStars(toStat(nextStars));
      setPulls(toStat(nextPulls));

      if (nextStars !== null || nextPulls !== null) {
        writeCache({
          stars: nextStars,
          pulls: nextPulls,
          fetchedAt: Date.now(),
        });
      }
    };

    // deferred to idle time so the requests never compete with the hero image
    const supportsIdleCallback =
      typeof window.requestIdleCallback === "function";
    const statsLoadTaskId = supportsIdleCallback
      ? window.requestIdleCallback(loadHeroStats, { timeout: 3000 })
      : window.setTimeout(loadHeroStats, 1500);

    return () => {
      isCancelled = true;

      if (supportsIdleCallback) {
        window.cancelIdleCallback(statsLoadTaskId);
      } else {
        window.clearTimeout(statsLoadTaskId);
      }
    };
  }, []);

  return (
    // reserved height keeps the hero from reflowing once the numbers land
    <div className="flex min-h-[30px] flex-wrap items-center gap-2">
      <StatBadge icon={<GitHubMark />} label="GitHub stars" stat={stars} />
      <StatBadge icon={<DockerMark />} label="Docker pulls" stat={pulls} />
    </div>
  );
}

function StatBadge({
  icon,
  label,
  stat,
}: {
  icon: ReactNode;
  label: string;
  stat: Stat;
}) {
  return (
    <span
      // basis + grow: two per row while they fit the button width, stacked below that
      className={`flex grow basis-[164px] items-stretch overflow-hidden rounded-lg border border-[#ffffff1a] bg-[#0C0E13] text-[12px] leading-none transition-opacity duration-500 ${
        stat.status === "loading" ? "opacity-60" : "opacity-100"
      }`}
    >
      <span className="flex flex-1 items-center gap-1.5 whitespace-nowrap px-2.5 py-2 text-gray-400">
        {icon}
        {label}
      </span>

      <span className="flex min-w-[3.5rem] items-center justify-center border-l border-[#ffffff1a] bg-[#ffffff08] px-2.5 py-2 font-medium tabular-nums text-white">
        {stat.status === "loading" ? (
          <span className="h-[8px] w-8 animate-pulse rounded-full bg-[#ffffff26]" />
        ) : stat.status === "ready" ? (
          stat.text
        ) : (
          "n/a"
        )}
      </span>
    </span>
  );
}

function GitHubMark() {
  return (
    <svg
      aria-hidden={true}
      width="13"
      height="13"
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path d="M12 .3a12 12 0 0 0-3.8 23.4c.6.1.8-.3.8-.6v-2c-3.3.7-4-1.6-4-1.6-.6-1.4-1.4-1.8-1.4-1.8-1-.7.1-.7.1-.7 1.2.1 1.8 1.2 1.8 1.2 1 1.8 2.8 1.3 3.5 1 0-.8.4-1.3.7-1.6-2.7-.3-5.5-1.3-5.5-5.9 0-1.3.5-2.4 1.2-3.2 0-.4-.5-1.6.2-3.2 0 0 1-.3 3.3 1.2a11.5 11.5 0 0 1 6 0C17.3 4.9 18.3 5.2 18.3 5.2c.7 1.6.2 2.8.1 3.2.8.8 1.2 1.9 1.2 3.2 0 4.6-2.8 5.6-5.5 5.9.4.4.8 1.1.8 2.2v3.3c0 .3.2.7.8.6A12 12 0 0 0 12 .3Z" />
    </svg>
  );
}

function DockerMark() {
  return (
    <svg
      aria-hidden={true}
      width="14"
      height="14"
      viewBox="0 0 24 24"
      fill="currentColor"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path d="M13.983 11.078h2.119a.186.186 0 00.186-.185V9.006a.186.186 0 00-.186-.186h-2.119a.185.185 0 00-.185.185v1.888c0 .102.083.185.185.185m-2.954-5.43h2.118a.186.186 0 00.186-.186V3.574a.186.186 0 00-.186-.185h-2.118a.185.185 0 00-.185.185v1.888c0 .102.082.185.185.185m0 2.716h2.118a.187.187 0 00.186-.186V6.29a.186.186 0 00-.186-.185h-2.118a.185.185 0 00-.185.185v1.887c0 .102.082.185.185.186m-2.93 0h2.12a.186.186 0 00.184-.186V6.29a.185.185 0 00-.185-.185H8.1a.185.185 0 00-.185.185v1.887c0 .102.083.185.185.186m-2.964 0h2.119a.186.186 0 00.185-.186V6.29a.185.185 0 00-.185-.185H5.136a.186.186 0 00-.186.185v1.887c0 .102.084.185.186.186m5.893 2.715h2.118a.186.186 0 00.186-.185V9.006a.186.186 0 00-.186-.186h-2.118a.185.185 0 00-.185.185v1.888c0 .102.082.185.185.185m-2.93 0h2.12a.185.185 0 00.184-.185V9.006a.185.185 0 00-.184-.186h-2.12a.185.185 0 00-.184.185v1.888c0 .102.083.185.185.185m-2.964 0h2.119a.185.185 0 00.185-.185V9.006a.185.185 0 00-.184-.186h-2.12a.186.186 0 00-.186.186v1.887c0 .102.084.185.186.185m-2.92 0h2.12a.185.185 0 00.184-.185V9.006a.185.185 0 00-.184-.186h-2.12a.185.185 0 00-.184.185v1.888c0 .102.082.185.185.185M23.763 9.89c-.065-.051-.672-.51-1.954-.51-.338.001-.676.03-1.01.087-.248-1.7-1.653-2.53-1.716-2.566l-.344-.199-.226.327c-.284.438-.49.922-.612 1.43-.23.97-.09 1.882.403 2.661-.595.332-1.55.413-1.744.42H.751a.751.751 0 00-.75.748 11.376 11.376 0 00.692 4.062c.545 1.428 1.355 2.48 2.41 3.124 1.18.723 3.1 1.137 5.275 1.137.983.003 1.963-.086 2.93-.266a12.248 12.248 0 003.823-1.389c.98-.567 1.86-1.288 2.61-2.136 1.252-1.418 1.998-2.997 2.553-4.4h.221c1.372 0 2.215-.549 2.68-1.009.309-.293.55-.65.707-1.046l.098-.288Z" />
    </svg>
  );
}
