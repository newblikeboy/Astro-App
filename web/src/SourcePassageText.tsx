import { Fragment } from "react";
import type { SourcePassage } from "./types";
import formatting from "./data/guruji-numerology-layout.json";

type BlockKind = "title" | "subtitle" | "heading" | "subheading" | "paragraph" | "quote" | "formula" | "space" | "divider";
type Layout = { hash: string; length: number; blocks: [number, BlockKind][] };
const layouts = formatting.layouts as unknown as Record<string, Layout>;

/** sourceTextFingerprint matches the offline UTF-16 fingerprint before applying source-specific formatting. */
export function sourceTextFingerprint(text: string) {
  let hash = 2166136261;
  for (let index = 0; index < text.length; index++) hash = Math.imul(hash ^ text.charCodeAt(index), 16777619);
  return (hash >>> 0).toString(16).padStart(8, "0");
}

/** SourcePlain hides only line-wrap whitespace after a literal hyphen, preserving the original characters beneath the visual reflow. */
function SourcePlain({ text }: { text: string }) {
  return <>{text.split(/(?<=-)([ \t]*\r?\n[ \t]*)/g).map((part,index)=>index%2 ? <span className="source-format-marker" aria-hidden="true" key={index}>{part}</span> : <Fragment key={index}>{part}</Fragment>)}</>;
}

/** SourceInline interprets author-supplied emphasis and joins PDF hyphen wraps without changing the underlying text. */
export function SourceInline({ text, heading = false }: { text: string; heading?: boolean }) {
  const prefix = heading ? text.match(/^\s*#{1,6}[ \t]+/)?.[0] || "" : "";
  return <>{prefix && <span className="source-format-marker" aria-hidden="true">{prefix}</span>}{text.slice(prefix.length).split(/(\*\*[\s\S]+?\*\*)/g).map((part,index)=>part.startsWith("**") && part.endsWith("**") ? <Fragment key={index}><span className="source-format-marker" aria-hidden="true">**</span><strong><SourcePlain text={part.slice(2,-2)} /></strong><span className="source-format-marker" aria-hidden="true">**</span></Fragment> : <SourcePlain text={part} key={index} />)}</>;
}

/** SourcePassageText reflows verified PDF paragraphs/headings without adding, deleting or rearranging the source text. */
export default function SourcePassageText({ entry, structured = false }: { entry: SourcePassage; structured?: boolean }) {
  if (!structured) return <p className="guruji-source-text" lang="hi">{entry.text}</p>;
  const layout = layouts[`${entry.category}:${entry.key}`];
  const verified = layout && entry.sha256 === formatting.source_sha256 && layout.length === entry.text.length && layout.hash === sourceTextFingerprint(entry.text);
  let start = 0;
  return <div className="guruji-source-text guruji-formatted-text" lang="hi" data-layout={verified ? "pdf" : "flow"}>
    {verified ? layout.blocks.map(([end,kind],index)=>{
      const text = entry.text.slice(start,end);
      start = end;
      if (kind === "title") return <h4 className="source-article-title" key={index}><SourceInline text={text} heading /></h4>;
      if (kind === "heading") return <h5 className="source-section-title" key={index}><SourceInline text={text} heading /></h5>;
      if (kind === "subheading") return <h6 className="source-subsection-title" key={index}><SourceInline text={text} heading /></h6>;
      if (kind === "subtitle") return <p className="source-article-subtitle" key={index}><SourceInline text={text} /></p>;
      if (kind === "quote") return <blockquote key={index}><SourceInline text={text} /></blockquote>;
      if (kind === "space") return <span className="source-format-marker" aria-hidden="true" key={index}>{text}</span>;
      if (kind === "divider") return <div className="source-article-divider" aria-hidden="true" key={index}>{text}</div>;
      return <p className={kind === "formula" ? "source-formula" : undefined} key={index}><SourceInline text={text} /></p>;
    }) : <p>{entry.text}</p>}
  </div>;
}
