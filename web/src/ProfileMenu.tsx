import { useEffect, useRef, useState } from "react";
import { CalendarDays, LogOut, ShoppingBag, UserRound } from "lucide-react";
import { t } from "./i18n";
import { followRoute } from "./routing";
import "./profile-menu.css";

/** ProfileMenu exposes customer destinations on hover, keyboard activation or touch and reports failed logout attempts. */
export default function ProfileMenu({ onLogout, onNavigate }: {
  onLogout: () => Promise<void>;
  onNavigate: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;
    /** dismiss closes the disclosure when a pointer is pressed outside it. */
    function dismiss(event: PointerEvent) {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    }
    document.addEventListener("pointerdown", dismiss);
    return () => document.removeEventListener("pointerdown", dismiss);
  }, [open]);

  /** logout disables duplicate requests and keeps the menu available if the server rejects logout. */
  async function logout() {
    setBusy(true);
    setError("");
    try {
      await onLogout();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="profile-menu" ref={root}
      onPointerEnter={(event) => { if (event.pointerType === "mouse") setOpen(true); }}
      onPointerLeave={(event) => {
        if (!busy && event.pointerType === "mouse" && !root.current?.contains(document.activeElement)) setOpen(false);
      }}
      onBlur={(event) => { if (!busy && !event.currentTarget.contains(event.relatedTarget)) setOpen(false); }}
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          event.stopPropagation();
          setOpen(false);
          trigger.current?.focus();
        }
      }}
    >
      <button className="profile-trigger" ref={trigger} type="button"
        aria-label={t("Profile options")} aria-expanded={open} aria-controls="customer-profile-options"
        onClick={() => { onNavigate(); setOpen(!open); }}
        onKeyDown={(event) => {
          if (event.key === "ArrowDown") {
            event.preventDefault();
            setOpen(true);
            requestAnimationFrame(() => root.current?.querySelector("a")?.focus());
          }
        }}
      ><UserRound size={21} aria-hidden="true" /></button>
      {open && <div className="profile-options" id="customer-profile-options">
        {[
          { href: "/account", label: "My Profile", Icon: UserRound },
          { href: "/account?tab=bookings", label: "My Consultation", Icon: CalendarDays },
          { href: "/account?tab=orders", label: "My Orders", Icon: ShoppingBag },
        ].map(({ href, label, Icon }) => (
          <a key={href} href={href} onClick={(event) => {
            followRoute(event, href);
            onNavigate();
            setOpen(false);
          }}><Icon size={18} aria-hidden="true" />{t(label)}</a>
        ))}
        <button type="button" className="profile-logout" disabled={busy} onClick={() => void logout()}>
          <LogOut size={18} aria-hidden="true" />{t(busy ? "Logging out…" : "Logout")}
        </button>
        {error && <p role="alert">{error}</p>}
      </div>}
    </div>
  );
}
