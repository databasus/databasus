"use client";

import { useState } from "react";
import { VIDEO_PLAY_LABELS } from "./copyLabels";

interface LiteYouTubeEmbedProps {
  videoId: string;
  title: string;
  thumbnailSrc: string;
  readonly playLabel?: string;
}

export default function LiteYouTubeEmbed({
  videoId,
  title,
  thumbnailSrc,
  playLabel = VIDEO_PLAY_LABELS.en.replace("{title}", title),
}: LiteYouTubeEmbedProps) {
  const [isIframeLoaded, setIsIframeLoaded] = useState(false);

  const handleClick = () => {
    setIsIframeLoaded(true);
  };

  return (
    <div className="relative w-full overflow-hidden rounded-lg" style={{ paddingBottom: "56.25%" }}>
      {!isIframeLoaded ? (
        <>
          <img
            src={thumbnailSrc}
            alt={title}
            loading="lazy"
            className="absolute left-0 top-0 h-full w-full object-cover"
          />
          <button
            onClick={handleClick}
            className="absolute cursor-pointer left-0 top-0 z-10 flex h-full w-full items-center justify-center transition-all hover:bg-opacity-20"
            aria-label={playLabel}
          >
            <div className="flex items-center justify-center rounded-xl bg-blue-600 px-3 py-1 transition-transform hover:scale-105">
              <svg
                className="h-8 w-8 sm:h-12 sm:w-12"
                viewBox="0 0 48 48"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <rect width="48" height="48" rx="12" fill="#155DFC" />
                <path d="M36 24L18 34.3923V13.6077L36 24Z" fill="white" />
              </svg>
            </div>
          </button>
        </>
      ) : (
        <iframe
          src={`https://www.youtube.com/embed/${videoId}?autoplay=1`}
          title={title}
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
          className="absolute left-0 top-0 h-full w-full"
        />
      )}
    </div>
  );
}
