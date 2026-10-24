import { useEffect, useState } from "react";
import { api } from "./api";
import type { VideoLink } from "./video-links";

/** useVideoLink loads one published link, rejects stale responses across navigation and exposes a retry for unavailable requests. */
export function useVideoLink(slug: string) {
  const [attempt, setAttempt] = useState(0);
  const [state, setState] = useState<{ slug: string; link?: VideoLink; error?: string }>({ slug: "" });
  useEffect(() => {
    if (!slug) return;
    let active = true;
    setState({ slug });
    api<VideoLink>("/video-links/" + encodeURIComponent(slug)).then(
      (link) => { if (active) setState({ slug, link }); },
      (error) => { if (active) setState({ slug, error: error.message }); },
    );
    return () => { active = false; };
  }, [slug, attempt]);
  return {
    link: state.slug === slug ? state.link : undefined,
    error: state.slug === slug ? state.error : undefined,
    /** retry requests the current link again after a recoverable loading error. */
    retry: () => setAttempt((value) => value + 1),
  };
}
