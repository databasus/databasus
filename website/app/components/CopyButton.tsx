"use client";

import { useState } from "react";
import type { Locale } from "@/app/i18n";
import { COPY_LABELS } from "./copyLabels";

interface CopyButtonProps {
  text: string;
  className?: string;
  lang?: Locale | "en";
}

export function CopyButton({
  text,
  className = "",
  lang = "en",
}: CopyButtonProps) {
  const [isCopied, setIsCopied] = useState(false);

  const copyToClipboard = () => {
    if (!navigator.clipboard) {
      const textArea = document.createElement("textarea");
      textArea.value = text;
      document.body.appendChild(textArea);
      textArea.select();
      document.execCommand("copy");
      document.body.removeChild(textArea);
      showCopyFeedback();
      return;
    }

    navigator.clipboard
      .writeText(text)
      .then(() => {
        showCopyFeedback();
      })
      .catch((err) => {
        console.error("Failed to copy text: ", err);
        const textArea = document.createElement("textarea");
        textArea.value = text;
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand("copy");
        document.body.removeChild(textArea);
        showCopyFeedback();
      });
  };

  const showCopyFeedback = () => {
    setIsCopied(true);
    setTimeout(() => {
      setIsCopied(false);
    }, 2000);
  };

  return (
    <button
      onClick={copyToClipboard}
      className={`w-max whitespace-nowrap rounded px-2 py-1 text-xs text-white transition-colors ${
        isCopied ? "bg-green-500" : "bg-blue-600 hover:bg-blue-700"
      } ${className}`}
    >
      {isCopied ? COPY_LABELS[lang].copied : COPY_LABELS[lang].copy}
    </button>
  );
}
