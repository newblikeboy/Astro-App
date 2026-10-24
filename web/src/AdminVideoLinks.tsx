import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { Copy, ExternalLink, Plus } from "lucide-react";
import { api } from "./api";
import { signs } from "./Chart";
import { t } from "./i18n";
import type { Rule } from "./types";
import { videoSharePath, videoTopics } from "./video-links";
import type { VideoLink } from "./video-links";
import "./video-journey.css";

/** newVideoLink creates an unpublished Moon-sign-based campaign with no preselected interpretation or invented prediction. */
function newVideoLink(): VideoLink {
  return { slug: "", title: "", topic: "debt", sign: 0, sign_basis: "moon", rule_ids: [], enabled: false, version: 0 };
}

/** AdminVideoLinks manages reusable topic links, approved-reading selection, publication and copying without changing service prices. */
export default function AdminVideoLinks({ csrf }: { csrf: string }) {
  const [links, setLinks] = useState<VideoLink[]>([]);
  const [rules, setRules] = useState<Rule[]>([]);
  const [form, setForm] = useState<VideoLink>(newVideoLink);
  const [busy, setBusy] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [search, setSearch] = useState("");
  useEffect(() => {
    let active = true;
    api<{ links: VideoLink[]; rules: Rule[] }>("/admin/video-links").then((data) => {
      if (active) { setLinks(data.links); setRules(data.rules); setLoaded(true); }
    }).catch((error) => { if (active) setError(error.message); });
    return () => { active = false; };
  }, []);

  /** save persists the publication choice, updates the link list and retains the stable share URL for editing. */
  async function save(event: FormEvent) {
    event.preventDefault();
    setBusy(true); setError(""); setNotice("");
    try {
      const result = await api<VideoLink>("/admin/video-links/" + encodeURIComponent(form.slug), "PUT", form, csrf);
      setForm(result);
      setLinks((old) => [result, ...old.filter((link) => link.slug !== result.slug)]);
      setNotice(t(result.enabled ? "Video link published. Copy it into your video's description or pinned comment." : "Video link saved as a draft. Publish it when you are ready."));
    } catch (error) { setError((error as Error).message); }
    finally { setBusy(false); }
  }

  /** copy writes an active public URL to the clipboard while leaving the selectable URL visible if clipboard permission fails. */
  async function copy(link: VideoLink) {
    try {
      await navigator.clipboard.writeText(location.origin + videoSharePath(link.slug));
      setNotice(t("Link copied."));
    } catch { setNotice(t("Select and copy the link shown below.")); }
  }

  const visibleRules = rules.filter((rule) => (rule.topic + " " + rule.id + " " + (rule.category || "general")).toLowerCase().includes(search.toLowerCase()));
  const unavailable = form.rule_ids.filter((id) => !rules.some((rule) => rule.id === id));
  return <div className="admin-video-links">
    {error && <p className="error" role="alert">{t(error)} <button type="button" className="text-button" onClick={() => location.reload()}>{t("Reload")}</button></p>}
    {notice && <p className="success" role="status">{notice}</p>}
    <div className="video-admin-grid">
      <section className="admin-card">
        <div className="panel-title"><h2>{t("Your video links")}</h2><button type="button" className="button secondary" disabled={busy} onClick={() => { setForm(newVideoLink()); setError(""); setNotice(""); }}><Plus size={17} />{t("New link")}</button></div>
        {!loaded && !error && <p role="status">{t("Loading video links…")}</p>}
        {loaded && !links.length && <p>{t("Create a link for a video about debt, promotion, a zodiac sign or another life topic.")}</p>}
        <div className="video-link-list">{links.map((link) => <article key={link.slug}>
          <span className="pill">{t(link.enabled ? "Published" : "Draft")}</span>
          <h3>{link.title}</h3><small>{link.slug}</small>
          <div className="video-link-actions"><button type="button" className="button secondary" disabled={busy} onClick={() => { setForm(link); setError(""); setNotice(""); }}>{t("Edit")}</button>
            {link.enabled && <><button type="button" className="text-button" onClick={() => void copy(link)}><Copy size={16} />{t("Copy link")}</button><a href={videoSharePath(link.slug)} target="_blank" rel="noopener noreferrer"><ExternalLink size={16} />{t("Open link")}</a></>}
          </div>
        </article>)}</div>
      </section>
      <section className="admin-card">
        <h2>{t(form.version ? "Edit video link" : "Create a video link")}</h2>
        <form onSubmit={(event) => void save(event)} className="video-link-form">
          <label>{t("Link name (URL)")}<input required minLength={3} maxLength={64} pattern="[a-z0-9]+(-[a-z0-9]+)*" placeholder="karj-se-rahat" value={form.slug} disabled={busy || form.version > 0} onChange={(event) => setForm({ ...form, slug: event.target.value })} /></label>
          <p className="field-note">{t("Use lowercase English letters, numbers and hyphens. The link name stays fixed after saving.")}</p>
          <label>{t("Video topic")}<select aria-label={t("Video topic")} value={form.topic} onChange={(event) => setForm({ ...form, topic: event.target.value, sign: event.target.value === "zodiac" ? 1 : 0 })}>{videoTopics.map(([value, label]) => <option key={value} value={value}>{t(label)}</option>)}</select></label>
          {form.topic === "zodiac" && <div className="form-row">
            <label>{t("Zodiac sign")}<select aria-label={t("Zodiac sign")} value={form.sign} onChange={(event) => setForm({ ...form, sign: Number(event.target.value) })}>{signs.map((sign, index) => <option key={sign} value={index + 1}>{t(sign)}</option>)}</select></label>
            <label>{t("Sign reference")}<select aria-label={t("Sign reference")} value={form.sign_basis} onChange={(event) => setForm({ ...form, sign_basis: event.target.value as VideoLink["sign_basis"] })}><option value="moon">{t("Moon sign")}</option><option value="lagna">{t("Ascendant")}</option><option value="sun">{t("Sun sign")}</option></select></label>
          </div>}
          <label>{t("Video title")}<input required minLength={3} maxLength={180} value={form.title} onChange={(event) => setForm({ ...form, title: event.target.value })} /></label>
          <fieldset><legend>{t("Readings for this video")}</legend>
            <p className="field-note">{t("Select Guruji-approved readings about this exact topic. Visitors see a reading only when its chart conditions match their own Kundli.")}</p>
            <label>{t("Find approved readings")}<input type="search" value={search} onChange={(event) => setSearch(event.target.value)} /></label>
            <div className="video-rule-options">{visibleRules.map((rule) => <label className="checkbox-label" key={rule.id}>
              <input type="checkbox" checked={form.rule_ids.includes(rule.id)} onChange={(event) => setForm({ ...form, rule_ids: event.target.checked ? [...form.rule_ids, rule.id] : form.rule_ids.filter((id) => id !== rule.id) })} />
              <span>{rule.topic}<small>{rule.id} · {t("Version")} {rule.version}</small></span>
            </label>)}</div>
            {!visibleRules.length && <p>{t("No approved readings found. Add or approve the topic's interpretations in Interpretations & remedies.")}</p>}
            {unavailable.length > 0 && <p role="alert">{t("Some selected readings are no longer available.")} <button type="button" onClick={() => setForm({ ...form, rule_ids: form.rule_ids.filter((id) => !unavailable.includes(id)) })}>{t("Remove unavailable readings")}</button></p>}
            {!form.rule_ids.length && <p className="notice">{t("No readings selected: visitors will see calculated chart factors and a consultation option, without a personal prediction.")}</p>}
          </fieldset>
          <label className="checkbox-label"><input type="checkbox" checked={form.enabled} onChange={(event) => setForm({ ...form, enabled: event.target.checked })} />{t("Publish this link")}</label>
          <p className="field-note">{t("Uncheck and save to disable a shared link. Existing paid reports and consultations are retained.")}</p>
          <button type="submit" className="button" disabled={busy || !loaded}>{t(busy ? "Saving…" : "Save video link")}</button>
          {form.version > 0 && <label>{t("Share link")}<input readOnly value={location.origin + videoSharePath(form.slug)} onFocus={(event) => event.target.select()} /></label>}
          {form.version > 0 && form.enabled && <button type="button" className="button secondary" onClick={() => void copy(form)}><Copy size={16} />{t("Copy link")}</button>}
        </form>
      </section>
    </div>
  </div>;
}
