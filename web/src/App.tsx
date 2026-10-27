import { t, useLanguage } from "./i18n";
import { lazy, Suspense, useEffect, useRef, useState } from "react";
import type { FormEvent, ReactNode } from "react";
import {
  ArrowRight,
  ArrowUpRight,
  BookOpen,
  CalendarDays,
  CheckCircle2,
  ChevronDown,
  Download,
  FileText,
  Flower2,
  Globe2,
  IndianRupee,
  Info,
  LoaderCircle,
  LockKeyhole,
  LogOut,
  Menu,
  Play,
  Plus,
  ShieldCheck,
  Sun,
  UserRound,
  Video,
  X,
} from "lucide-react";
import { api, APIError, localTime, money, track, zone } from "./api";
import { clockMinutes } from "./time-display";
import TimeRange from "./TimeRange";
import type {
  Account,
  Birth,
  Booking,
  Config,
  Order,
  Place,
  Report,
  ChannelVideo,
} from "./types";
import Chart from "./Chart";
import PaymentView from "./PaymentView";
import RemedyPDFDownload from "./RemedyPDFDownload";
import LanguageSelector from "./LanguageSelector";
import Brand from "./Brand";
import CartLink from "./CartLink";
import ProfileMenu from "./ProfileMenu";
import { VideoIntroduction, VideoReading } from "./VideoJourney";
import { useVideoLink } from "./useVideoLink";
import Panchang from "./Panchang";
import Hero from "./Hero";
import GemstoneCollection from "./GemstoneCollection";
import { getLanguage } from "./i18n";
import { Button, ErrorBox } from "./UI";
import {
  navigate,
  useAppLocation,
  customerReturn,
  loginPath,
  followRoute,
} from "./routing";
import { PortalHeader, PortalFooter } from "./PortalShell";
import { MIN_CALCULATION_DATE } from "./calculation-dates";
import channelVideoCatalog from "../../internal/platform/data/channel-videos.json";

const ShopApp = lazy(() => import("./ShopApp"));
const KundliReport = lazy(() => import("./KundliReport"));
const AdminApp = lazy(() => import("./AdminApp"));
const LoginPage = lazy(() => import("./LoginPage"));
const MatchingApp = lazy(() => import("./MatchingApp"));
const DevotionalLibrary = lazy(() => import("./DevotionalLibrary"));
const PujaLibrary = lazy(() => import("./PujaLibrary"));
const VivahMuhurat = lazy(() => import("./VivahMuhurat"));
const initialBirth: Birth = {
  name: "",
  date: "",
  time: "",
  place_id: "",
  unknown_time: false,
};
/** Modal renders a labelled dialog with focus management and the supplied close action. */
function Modal({
  title,
  children,
  onClose,
  wide = false,
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  wide?: boolean;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const d = ref.current!;
    d.showModal();
    return () => d.close();
  }, []);
  return (
    <dialog
      ref={ref}
      className={"modal " + (wide ? "modal-wide" : "")}
      onCancel={onClose}
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="modal-heading">
        <h2>{title}</h2>
        <button
          className="icon-button"
          onClick={onClose}
          aria-label={t("बंद करें")}
        >
          <X size={21} />
        </button>
      </div>
      <div className="modal-language">
        <LanguageSelector />
      </div>
      {children}
    </dialog>
  );
}
/** App routes between the public site, customer pages, matching, shop and separate administrator workspace. */
export default function App() {
  const { language } = useLanguage();
  const route = useAppLocation();
  useEffect(() => {
    const title = route.pathname.startsWith("/admin")
      ? t("Administration")
      : route.pathname === "/login"
        ? t("Log in to your account")
        : route.pathname.startsWith("/account")
          ? t("My account")
          : route.pathname === "/vivah-muhurat"
            ? t("Vivah Muhurat")
          : route.pathname === "/kundli-matching"
            ? t("Kundli matching")
            : route.pathname === "/kundli"
              ? t("Your Kundli")
              : route.pathname.startsWith("/shop")
                ? t("Gemstone shop")
                : "Mishra 4 You";
    // Public reading libraries own their individual guide and prayer titles.
    const readingLibrary = /^\/(?:bhakti|puja-vidhi)(?:\/|$)/.test(route.pathname);
    if (!readingLibrary)
      document.title = title === "Mishra 4 You" ? title : `${title} | Mishra 4 You`;
    const existing = document.head.querySelector("meta[data-portal-robots]");
    existing?.remove();
    if (
      route.pathname !== "/" &&
      route.pathname !== "/shop" &&
      route.pathname !== "/vivah-muhurat" &&
      !readingLibrary &&
      !route.pathname.startsWith("/shop/stones/")
    ) {
      const meta = document.createElement("meta");
      meta.name = "robots";
      meta.content = "noindex, nofollow";
      meta.dataset.portalRobots = "true";
      document.head.append(meta);
    }
  }, [route.pathname, language]);
  if (route.pathname === "/vivah-muhurat")
    return <Suspense fallback={<main className="boot" role="status">{t("Opening your workspace…")}</main>}><VivahMuhurat /></Suspense>;
  if (route.pathname === "/puja-vidhi" || route.pathname.startsWith("/puja-vidhi/"))
    return <Suspense fallback={<main className="boot" role="status">{t("Opening the collection...")}</main>}><PujaLibrary /></Suspense>;
  if (route.pathname === "/bhakti" || route.pathname.startsWith("/bhakti/"))
    return <Suspense fallback={<main className="boot" role="status">{t("Opening the collection...")}</main>}><DevotionalLibrary /></Suspense>;
  if (route.pathname === "/admin" || route.pathname === "/admin/login")
    return (
      <Suspense
        fallback={
          <main className="boot" role="status">
            {t("Opening your workspace\u2026")}
          </main>
        }
      >
        <AdminApp />
      </Suspense>
    );
  if (route.pathname === "/shop" || route.pathname.startsWith("/shop/"))
    return (
      <Suspense
        fallback={
          <main className="boot" role="status">
            {t("Opening the collection...")}
          </main>
        }
      >
        <ShopApp />
      </Suspense>
    );
  if (route.pathname === "/kundli-matching")
    return (
      <Suspense
        fallback={
          <main className="boot" role="status">
            {t("Opening your workspace…")}
          </main>
        }
      >
        <MatchingApp />
      </Suspense>
    );
  if (
    !["/", "/login", "/account", "/kundli"].includes(route.pathname) &&
    !/^\/account\/reports\/[A-Za-z0-9_-]+$/.test(route.pathname)
  )
    return (
      <>
        <PortalHeader />
        <main id="main" className="portal-status">
          <h1>{t("Page not found")}</h1>
          <p>{t("This page is not available.")}</p>
          <a
            className="button"
            href="/"
            onClick={(event) => followRoute(event, "/")}
          >
            {t("Back to website")}
          </a>
        </main>
        <PortalFooter />
      </>
    );
  return <CustomerApp />;
}
/** CustomerApp coordinates guest Kundli creation, customer sessions, offers, payments and the public landing page. */
function CustomerApp() {
  const { language } = useLanguage();
  const [config, setConfig] = useState<Config | null>(null),
    [bootError, setBootError] = useState("");
  const route = useAppLocation();
  const savedReportID = /^\/account\/reports\/[A-Za-z0-9_-]+$/.test(
    route.pathname,
  )
    ? route.pathname.slice("/account/reports/".length)
    : "";
  const view =
    route.pathname === "/login"
      ? "login"
      : route.pathname === "/account"
        ? "account"
        : route.pathname === "/kundli" || savedReportID
          ? "report"
          : "home";
  const publicLayout = view === "home" || (view === "report" && !savedReportID);
  const videoSlug = (view === "home" || view === "report") ? route.searchParams.get("video") || "" : "";
  const video = useVideoLink(videoSlug);
  const returnVideoSlug = view === "login"
    ? new URL(customerReturn(route.searchParams.get("next")), location.origin).searchParams.get("video") || ""
    : videoSlug;
  const videoSuffix = returnVideoSlug ? "?video=" + encodeURIComponent(returnVideoSlug) : "";
  const headerLoginPath = videoSlug ? loginPath(route.pathname + videoSuffix) : "/login";
  /** setView navigates to the public, customer-account or current guest-report route. */
  const setView = (next: "home" | "account" | "report") =>
    navigate(
      next === "home" ? "/" : next === "account" ? "/account" : "/kundli" + videoSuffix,
    );
  const [account, setAccount] = useState<Account | null>(null);
  const [sessionResolved, setSessionResolved] = useState(false);
  const [authError, setAuthError] = useState("");
  const [reportError, setReportError] = useState("");
  const [reportLoading, setReportLoading] = useState(false);
  const [birth, setBirth] = useState<Birth>(() => {
    try {
      return (
        JSON.parse(sessionStorage.getItem("m4y-birth") || "null") ||
        initialBirth
      );
    } catch {
      return initialBirth;
    }
  });
  const [report, setReport] = useState<Report | null>(() => {
      try {
        const draft = JSON.parse(
          sessionStorage.getItem("m4y-guest-report") || "null",
        );
        return draft?.birth && !draft.user_id ? draft : null;
      } catch {
        return null;
      }
    }),
    [offer, setOffer] = useState<"pdf" | "consultation" | null>(null),
    [payment, setPayment] = useState<Order | null>(null),
    [policy, setPolicy] = useState<string | null>(null),
    [sample, setSample] = useState(false),
    [menu, setMenu] = useState(false),
    [toast, setToast] = useState("");
  const [consent, setConsent] = useState(
      () => localStorage.getItem("m4y-analytics") || "",
    ),
    [saving, setSaving] = useState(false),
    [loading, setLoading] = useState(false);
  const afterLogin = useRef<null | (() => void)>(null);
  const finishingLogin = useRef(false);
  const previousView = useRef(view);
  const [skySelection, setSkySelection] = useState({ index: 0, id: 0 });
  const videoRail = useRef<HTMLDivElement>(null);
  const [activeAnchor, setActiveAnchor] = useState("");
  const [videoScroll, setVideoScroll] = useState({
    previous: false,
    next: true,
  });
  const accountRef = useRef<Account | null>(null);
  const accountEpoch = useRef(0);
  useEffect(() => {
    if (view !== "home" || !config) return;
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries)
          if (entry.isIntersecting) setActiveAnchor(entry.target.id);
      },
      { rootMargin: "-15% 0px -65% 0px" },
    );
    for (const id of [
      "kundli",
      "panchang",
      "gemstones",
      "videos",
    ]) {
      const node = document.getElementById(id);
      if (node) observer.observe(node);
    }
    return () => observer.disconnect();
  }, [view, !!config]);
  useEffect(() => {
    if (!menu) return;
    /** close closes the mobile navigation on Escape and restores focus to its toggle. */
    const close = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setMenu(false);
        document.querySelector<HTMLButtonElement>(".menu-toggle")?.focus();
      }
    };
    /** outside closes mobile navigation when a pointer action occurs outside the site header. */
    const outside = (event: PointerEvent) => {
      if (!(event.target as Element).closest(".site-header")) setMenu(false);
    };
    document.addEventListener("keydown", close);
    document.addEventListener("pointerdown", outside);
    return () => {
      document.removeEventListener("keydown", close);
      document.removeEventListener("pointerdown", outside);
    };
  }, [menu]);
  /** refreshAccount reloads customer-owned data while rejecting stale session responses. */
  const refreshAccount = async () => {
    const epoch = accountEpoch.current;
    setAuthError("");
    try {
      const a = await api<Account>("/me");
      if (epoch !== accountEpoch.current) return null;
      setAccount(a);
      accountRef.current = a;
      return a;
    } catch (error) {
      if (epoch !== accountEpoch.current) return null;
      if (error instanceof APIError && error.status === 401) {
        setAccount(null);
        accountRef.current = null;
      } else setAuthError((error as Error).message);
      return accountRef.current;
    } finally {
      if (epoch === accountEpoch.current) setSessionResolved(true);
    }
  };
  useEffect(() => {
    if (!sessionResolved || authError) return;
    if ((view === "account" || savedReportID) && !account)
      navigate(loginPath(route.pathname + route.search), true);
    if (view === "login" && account && !finishingLogin.current)
      resumeAfterLogin();
  }, [sessionResolved, authError, account, view, route.pathname, route.search]);
  useEffect(() => {
    if (previousView.current === "login" && view !== "login") {
      afterLogin.current = null;
      sessionStorage.removeItem("m4y-login-intent");
    }
    previousView.current = view;
    setMenu(false);
  }, [view]);
  useEffect(() => {
    try {
      if (report && !report.user_id)
        sessionStorage.setItem("m4y-guest-report", JSON.stringify(report));
      else if (report?.user_id) sessionStorage.removeItem("m4y-guest-report");
    } catch {
      /* A full browser store must not interrupt the report. */
    }
  }, [report]);
  useEffect(() => {
    if (!savedReportID || !account) return;
    if (report?.id === savedReportID && report.user_id === account.user.id)
      return;
    let active = true;
    setReportLoading(true);
    setReportError("");
    api<Report>("/reports/" + encodeURIComponent(savedReportID))
      .then((value) => {
        if (active) setReport(value);
      })
      .catch((error) => {
        if (active) setReportError(error.message);
      })
      .finally(() => {
        if (active) setReportLoading(false);
      });
    return () => {
      active = false;
    };
  }, [savedReportID, account?.user.id]);
  /** refreshConfig reloads public merchant/content configuration and exposes a recoverable boot error. */
  const refreshConfig = () =>
    api<Config>("/config")
      .then(setConfig)
      .catch((e) => setBootError(e.message));
  useEffect(() => {
    void refreshConfig();
    void refreshAccount();
    const params = new URLSearchParams(location.search);
    /** clean accepts only bounded campaign identifiers before retaining optional attribution metadata. */
    const clean = (v: string | null) =>
      v && /^[a-zA-Z0-9_.-]{0,80}$/.test(v) ? v : "";
    if (!sessionStorage.getItem("m4y-utm"))
      sessionStorage.setItem(
        "m4y-utm",
        JSON.stringify({
          source: clean(params.get("utm_source")),
          medium: clean(params.get("utm_medium")),
          campaign: clean(params.get("utm_campaign")),
        }),
      );
    if (location.pathname === "/") track("landing_visit");
  }, []);
  useEffect(() => {
    sessionStorage.setItem("m4y-birth", JSON.stringify(birth));
  }, [birth]);
  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(""), 6500);
    return () => clearTimeout(t);
  }, [toast]);
  useEffect(() => {
    if (view !== "account" || !account) return;
    const timer = setInterval(() => {
      void refreshAccount();
    }, 5000);
    return () => clearInterval(timer);
  }, [view, !!account]);
  useEffect(() => {
    if (offer) track("offer_viewed", offer);
  }, [offer]);
  /** setAnalytics persists the visitor's analytics choice and synchronizes it with an existing customer account. */
  const setAnalytics = (choice: string) => {
    localStorage.setItem("m4y-analytics", choice);
    setConsent(choice);
    const a = accountRef.current;
    if (a)
      void api(
        "/privacy",
        "PUT",
        { marketing: a.user.marketing, analytics: choice === "yes" },
        a.csrf,
      ).catch(() => {});
  };
  /** home closes navigation, respects reduced motion, and brings the selected section or first birth field into reach. */
  const home = (anchor?: string) => {
    navigate("/" + (anchor === "kundli" ? videoSuffix : "") + (anchor ? "#" + anchor : ""));
    setMenu(false);
    setTimeout(() => {
      const behavior = window.matchMedia("(prefers-reduced-motion: reduce)")
        .matches
        ? "auto"
        : "smooth";
      if (anchor) {
        const section = document.getElementById(anchor);
        section?.scrollIntoView({ behavior, block: "start" });
        if (anchor === "kundli")
          section?.querySelector("input")?.focus({ preventScroll: true });
      } else window.scrollTo({ top: 0, behavior });
    }, 20);
  };
  /** needAuth runs an authenticated action or preserves its intent while directing the visitor to login. */
  const needAuth = (then: () => void, intent = "account") => {
    if (account) {
      then();
      return;
    }
    afterLogin.current = then;
    sessionStorage.setItem("m4y-login-intent", intent);
    navigate(loginPath(view === "report" ? route.pathname + videoSuffix : "/account"));
  };
  /** openTools selects the requested daily/sky tool and scrolls to its landing-page panel. */
  const openTools = (index: number) => {
    setSkySelection((value) => ({ index, id: value.id + 1 }));
    home("panchang");
  };
  useEffect(() => {
    const rail = videoRail.current;
    if (!rail) return;
    /** update recomputes which video-rail scroll controls are currently available. */
    const update = () =>
      setVideoScroll((old) => {
        const next = {
          previous: rail.scrollLeft > 2,
          next: rail.scrollLeft + rail.clientWidth < rail.scrollWidth - 2,
        };
        return next.previous === old.previous && next.next === old.next
          ? old
          : next;
      });
    const observer = new ResizeObserver(update);
    observer.observe(rail);
    rail.addEventListener("scroll", update, { passive: true });
    update();
    return () => {
      observer.disconnect();
      rail.removeEventListener("scroll", update);
    };
  }, [view, config?.settings.content.videos.length]);
  /** scrollVideos advances the video rail by a measured card width in the requested direction. */
  const scrollVideos = (direction: number) => {
    const rail = videoRail.current;
    const card = rail?.querySelector<HTMLElement>(".video-card");
    if (!rail || !card) return;
    const step =
      card.getBoundingClientRect().width +
      parseFloat(getComputedStyle(rail).columnGap);
    rail.scrollBy({
      left: direction * step * Math.max(1, Math.round(rail.clientWidth / step)),
      behavior: matchMedia("(prefers-reduced-motion: reduce)").matches
        ? "auto"
        : "smooth",
    });
  };
  /** resumeAfterLogin restores the saved customer action or destination after successful authentication. */
  function resumeAfterLogin() {
    const next = afterLogin.current;
    const intent = sessionStorage.getItem("m4y-login-intent");
    afterLogin.current = null;
    sessionStorage.removeItem("m4y-login-intent");
    navigate(customerReturn(route.searchParams.get("next") ?? "/"), true);
    if (next) next();
    else if (intent === "save" && report) void saveCurrent();
    else if (intent === "pdf" && report?.chart) void saveCurrent(true);
    else if (intent === "consultation") setOffer("consultation");
  }
  /** logout revokes the customer session and clears private report state before returning to the requested public screen. */
  async function logout(destination = "/login") {
    if (!account) return;
    await api("/logout", "POST", {}, account.csrf);
    accountEpoch.current++;
    setAccount(null);
    accountRef.current = null;
    setReport(null);
    sessionStorage.removeItem("m4y-guest-report");
    sessionStorage.removeItem("m4y-login-intent");
    afterLogin.current = null;
    navigate(destination, true);
  }
  /** saveCurrent saves the current Kundli under the customer account before continuing an optional offer. */
  async function saveCurrent(openOffer = false) {
    if (!report) return;
    setSaving(true);
    try {
      const a = accountRef.current || (await refreshAccount());
      if (!a) {
        afterLogin.current = () => {
          void saveCurrent(openOffer);
        };
        sessionStorage.setItem("m4y-login-intent", openOffer ? "pdf" : "save");
        navigate(loginPath(view === "report" ? route.pathname + videoSuffix : "/account"));
        return;
      }
      let saved = report;
      if (!saved.user_id) {
        saved = await api<Report>("/reports", "POST", report.birth, a.csrf);
        setReport(saved);
        navigate("/account/reports/" + encodeURIComponent(saved.id) + videoSuffix, true);
      }
      await refreshAccount();
      if (openOffer) setOffer("pdf");
      else setToast(t("आपकी कुंडली खाते में सुरक्षित हो गई है।"));
    } catch (e) {
      setToast((e as Error).message);
    } finally {
      setSaving(false);
    }
  }
  /** buy opens the selected paid offer after ensuring the required chart and customer session exist. */
  const buy = (product: "pdf" | "consultation") => {
    if (product === "pdf" && !report?.chart) {
      home("kundli");
      setToast(t("व्यक्तिगत PDF के लिए पहले जन्म विवरण से अपनी कुंडली बनाएँ।"));
      return;
    }
    if (product === "pdf") {
      needAuth(() => {
        void saveCurrent(true);
      }, "pdf");
    } else needAuth(() => setOffer("consultation"), "consultation");
  };
  /** generate submits birth details for a guest Kundli and presents its result or validation error. */
  async function generate() {
    setLoading(true);
    try {
      // Recheck publication before calculating from a previously opened video page.
      if (videoSlug) await api("/video-links/" + encodeURIComponent(videoSlug));
      const result = await api<Report>("/kundli", "POST", birth);
      setReport(result);
      setView("report");
      window.scrollTo(0, 0);
      track("kundli_generated");
    } catch (e) {
      throw e;
    } finally {
      setLoading(false);
    }
  }
  /** checkout creates a server-priced PDF/consultation order using the selected offer and optional time slot. */
  async function checkout(slot: string) {
    if (!account || !offer) return;
    const o = await api<Order>(
      "/orders",
      "POST",
      {
        product: offer,
        report_id: report?.id || "",
        slot,
        idempotency_key: crypto.randomUUID(),
      },
      account.csrf,
    );
    track("checkout_started", offer);
    setOffer(null);
    if (o.status === "paid") {
      setView("account");
      await refreshAccount();
      return;
    }
    setPayment(o);
  }
  /** paid closes checkout after verified payment, refreshes purchases and opens the customer account. */
  const paid = async () => {
    setPayment(null);
    await refreshAccount();
    setView("account");
    window.scrollTo(0, 0);
    setToast(
      t(
        "भुगतान सत्यापित है। आपकी रिपोर्ट या बुकिंग की स्थिति खाते में दिखेगी।",
      ),
    );
  };
  if (!config)
    return (
      <main className="boot">
        <Sun size={48} />
        <h1>Mishra 4 You</h1>
        {bootError ? (
          <>
            <ErrorBox error={t("सेवा से संपर्क नहीं हुआ। ") + bootError} />
            <p>{t("Go API को पोर्ट 8080 पर शुरू करें।")}</p>
            <Button onClick={() => void refreshConfig()}>
              {t("पुनः प्रयास करें")}
            </Button>
          </>
        ) : (
          <p>{t("आपके लिए तैयार हो रहा है\u2026")}</p>
        )}
      </main>
    );
  if (videoSlug && !video.link) return <>
    <PortalHeader />
    <main id="main" className="portal-status">
      {video.error ? <><h1>{t("This video link is unavailable.")}</h1><ErrorBox error={t(video.error)} /><Button onClick={video.retry}>{t("Try again")}</Button><a className="button secondary" href="/" onClick={(event) => followRoute(event, "/")}>{t("Back to website")}</a></> : <p role="status">{t("Opening your video topic…")}</p>}
    </main><PortalFooter />
  </>;
  return (
    <>
      <a href="#main" className="skip">
        {t("मुख्य सामग्री पर जाएँ")}
      </a>
      {publicLayout ? (
        <header className="site-header">
          <div className="nav-wrap">
            <a
              href="/"
              className="brand"
              onClick={(e) => {
                e.preventDefault();
                home();
              }}
            >
              <Brand />
            </a>
            <nav
              id="primary-navigation"
              className={menu ? "nav-open" : ""}
              aria-label={t("मुख्य नेविगेशन")}
            >
              <button
                className="nav-primary"
                aria-current={
                  view === "home" && activeAnchor === "kundli"
                    ? "location"
                    : undefined
                }
                onClick={() => home("kundli")}
              >
                {t("निःशुल्क कुंडली")}
              </button>
              <button
                aria-current={
                  view === "home" && activeAnchor === "panchang"
                    ? "location"
                    : undefined
                }
                onClick={() => openTools(0)}
              >
                {t("Panchang")}
              </button>
              <button onClick={() => { setMenu(false); navigate("/bhakti"); }}>
                {t("भक्ति संग्रह")}
              </button>
              <button onClick={() => { setMenu(false); navigate("/puja-vidhi"); }}>
                {t("Puja Vidhi")}
              </button>
              <button onClick={() => { setMenu(false); navigate("/vivah-muhurat"); }}>
                {t("Vivah Muhurat")}
              </button>
              <button
                onClick={() => {
                  setMenu(false);
                  navigate("/kundli-matching");
                }}
              >
                {t("Kundli matching")}
              </button>
              <button
                aria-current={
                  view === "home" && activeAnchor === "gemstones"
                    ? "location"
                    : undefined
                }
                onClick={() => home("gemstones")}
              >
                {t("Gemstones")}
              </button>
              <button
                aria-current={
                  view === "home" && activeAnchor === "videos"
                    ? "location"
                    : undefined
                }
                onClick={() => home("videos")}
              >
                {t("वीडियो")}
              </button>
            </nav>
            <LanguageSelector />
            <CartLink />
            {account ? <ProfileMenu onLogout={() => logout("/")} onNavigate={() => setMenu(false)} /> : <a
              className="account-button"
              href={headerLoginPath}
              aria-label={t("Log in")}
              onClick={(event) => {
                if (
                  event.button ||
                  event.metaKey ||
                  event.ctrlKey ||
                  event.shiftKey ||
                  event.altKey
                )
                  return;
                event.preventDefault();
                setMenu(false);
                navigate(headerLoginPath);
              }}
            >
              <UserRound size={17} />
              <span>{t("Log in")}</span>
              <ArrowUpRight size={15} />
            </a>}
            <button
              className="menu-toggle icon-button"
              aria-controls="primary-navigation"
              onClick={() => setMenu(!menu)}
              aria-expanded={menu}
              aria-label={t("नेविगेशन खोलें")}
            >
              {menu ? <X /> : <Menu />}
            </button>
          </div>
        </header>
      ) : (
        <PortalHeader />
      )}
      <main
        id="main"
        className={
          publicLayout
            ? undefined
            : view === "login"
              ? "auth-page"
              : "account-page"
        }
      >
        <Suspense
          fallback={
            <div className="container workspace" role="status">
              <LoaderCircle className="spin" size={25} />
              <p>{t("Opening your workspace…")}</p>
            </div>
          }
        >
          {view === "home" && (
            <>
              {video.link ? <VideoIntroduction link={video.link} /> : <Hero
                onKundli={() => home("kundli")}
                onTools={openTools}
                onConsultation={() => buy("consultation")}
              />}
              <section className="birth-section section-space" id="kundli">
                <div className="container birth-layout">
                  <div className="section-intro">
                    <h2>
                      {t("आपकी कुंडली की")} <br />
                      {t("शुरुआत यहाँ से।")}
                    </h2>
                    <p>
                      {t(
                        "कुछ छोटे विवरण भरें और अपनी जन्म कुंडली को करीब से जानें। कोई शुल्क नहीं।",
                      )}
                    </p>
                  </div>
                  <BirthForm
                    birth={birth}
                    setBirth={setBirth}
                    loading={loading}
                    generate={generate}
                  />
                  <div className="birth-reassurance">
                    <div className="subtle-note">
                      <ShieldCheck size={23} />
                      <span>
                        {t(
                          "आप बिना लॉग इन किए कुंडली देख सकते हैं। सहेजने के लिए मोबाइल सत्यापन करें।",
                        )}
                      </span>
                    </div>
                    <ul className="birth-benefits">
                      <li>
                        <CheckCircle2 size={16} />
                        {t("D1, Navamsha & Moon charts")}
                      </li>
                      <li>
                        <CheckCircle2 size={16} />
                        {t("Dasha timelines & planet positions")}
                      </li>
                      <li>
                        <CheckCircle2 size={16} />
                        {t("Clear calculations, in your language")}
                      </li>
                    </ul>
                  </div>
                </div>
              </section>
              {!videoSlug && <><section className="steps-section">
                <h2 className="sr-only">{t("How it works")}</h2>
                <div className="container steps">
                  <div className="step">
                    <span>01</span>
                    <div>
                      <h3>{t("जन्म विवरण दें")}</h3>
                      <p>{t("तारीख, समय और जन्म स्थान")}</p>
                    </div>
                  </div>
                  <div className="step-divider" />
                  <div className="step">
                    <span>02</span>
                    <div>
                      <h3>{t("अपनी कुंडली देखें")}</h3>
                      <p>{t("ग्रह, भाव और जन्म कुंडली")}</p>
                    </div>
                  </div>
                  <div className="step-divider" />
                  <div className="step">
                    <span>03</span>
                    <div>
                      <h3>{t("मार्गदर्शन चुनें")}</h3>
                      <p>{t("उपाय रिपोर्ट या व्यक्तिगत परामर्श")}</p>
                    </div>
                  </div>
                </div>
              </section>

              <section className="guru-section">
                <div className="container guru-layout">
                  <div className="guru-identity" aria-hidden="true">
                    <div className="guru-photo-frame">
                      <img
                        src="/images/brand/mark.svg"
                        alt=""
                        width="112"
                        height="112"
                      />
                      <span className="guru-photo-caption">{t("पं. राजा मिश्रा")}</span>
                    </div>
                    <span>Mishra 4 You</span>
                  </div>
                  <div className="guru-copy">
                    <h2>
                      {t("परंपरा से जुड़ा ज्ञान,")} <br />
                      {t("आपके जीवन के लिए।")}
                    </h2>
                    <div className="guru-name">
                      {t("पं. राजा मिश्रा ")}
                      <span>· Mishra 4 You</span>
                    </div>
                    <p>{t(config.settings.content.intro)}</p>
                    <div className="editorial-note">
                      <Info size={16} />
                      {t(
                        "परिचय और सामग्री गुरुजी की स्वीकृति के बाद प्रकाशित होंगे।",
                      )}
                    </div>
                    <button
                      className="text-button"
                      onClick={() => buy("consultation")}
                    >
                      {t("Explore consultations")}
                      <ArrowUpRight size={18} />
                    </button>
                  </div>
                </div>
              </section>
              <GemstoneCollection />
              <Panchang selection={skySelection} />
              <section className="videos-section section-space" id="videos">
                <div className="container">
                  <div className="section-heading">
                    <div>
                      <h2>{t("ज्ञान की बात, गुरुजी के साथ।")}</h2>
                    </div>
                    <div className="video-heading-meta">
                      <span className="quiet-label">
                        <Play size={16} />
                        {config.settings.content.videos.length === 10 &&
                        config.settings.content.videos.every(
                          (id, index) =>
                            id === channelVideoCatalog.videos[index]?.id,
                        )
                          ? t("Top 10 by views")
                          : t("चुनिंदा वीडियो")}
                      </span>
                      <div className="video-navigation">
                        <small>
                          {config.settings.content.videos.length} {t("Videos")}
                        </small>
                        <button
                          aria-label={t("Previous videos")}
                          disabled={!videoScroll.previous}
                          onClick={() => scrollVideos(-1)}
                        >
                          <ArrowRight
                            size={16}
                            style={{ transform: "rotate(180deg)" }}
                          />
                        </button>
                        <button
                          aria-label={t("Next videos")}
                          disabled={!videoScroll.next}
                          onClick={() => scrollVideos(1)}
                        >
                          <ArrowRight size={16} />
                        </button>
                      </div>
                    </div>
                  </div>
                  {config.settings.content.videos.length ? (
                    <div
                      className="videos-grid"
                      ref={videoRail}
                      role="region"
                      aria-label={t("Video library")}
                      tabIndex={0}
                    >
                      {config.settings.content.videos.map((id, i) => (
                        <VideoCard
                          key={id}
                          id={id}
                          index={i}
                          video={(
                            config.video_catalog ?? channelVideoCatalog
                          ).videos.find((video) => video.id === id)}
                          checkedAt={
                            (config.video_catalog ?? channelVideoCatalog)
                              .checked_at
                          }
                        />
                      ))}
                    </div>
                  ) : (
                    <div className="video-empty">
                      <span>
                        <Play size={23} />
                      </span>
                      <div>
                        <h3>
                          {t("गुरुजी के चुनिंदा वीडियो जल्द यहाँ होंगे।")}
                        </h3>
                        <p>
                          {t(
                            "केवल स्वीकृत YouTube वीडियो प्रकाशित किए जाएँगे।",
                          )}
                        </p>
                      </div>
                    </div>
                  )}
                </div>
              </section>
              <section className="faq-section section-space">
                <div className="container faq-layout">
                  <div>
                    <h2>
                      {t("कुछ बातें,")} <br />
                      {t("शुरू करने से पहले।")}
                    </h2>
                    <p>{t("और मदद चाहिए?")}</p>
                    <button
                      className="text-button"
                      onClick={() => setPolicy("contact")}
                    >
                      {t("सहायता की जानकारी ")}
                      <ArrowUpRight size={16} />
                    </button>
                  </div>
                  <div className="faq-list">
                    {config.settings.content.faq.map((faq, i) => (
                      <details key={i}>
                        <summary>
                          {t(faq.question)}
                          <Plus size={20} />
                        </summary>
                        <p>{t(faq.answer)}</p>
                      </details>
                    ))}
                  </div>
                </div>
              </section>
              <section className="closing-strip">
                <div className="container">
                  <Flower2 size={35} strokeWidth={1} />
                  <div>
                    <h2>{t("खुद को समझने की शुरुआत करें।")}</h2>
                    <p>{t("आपकी जन्म कुंडली, अब सरल हिंदी में।")}</p>
                  </div>
                  <Button onClick={() => home("kundli")}>
                    {t("निःशुल्क कुंडली बनाएँ ")}
                    <ArrowRight size={17} />
                  </Button>
                </div>
              </section>
              </>}
            </>
          )}
          {view === "report" &&
            report &&
            (!savedReportID ||
              (account &&
                report.id === savedReportID &&
                report.user_id === account.user.id)) && (
              video.link ? <VideoReading link={video.link} report={report}
                onFull={() => navigate(route.pathname)}
                onEdit={() => { setBirth(report.birth); home("kundli"); }}
                buy={buy}
              /> : <KundliReport
                report={report}
                onBack={() =>
                  savedReportID ? navigate("/account") : home("kundli")
                }
                backLabel={savedReportID ? t("Back to account") : undefined}
                onEditBirth={() => {
                  setBirth(report.birth);
                  home("kundli");
                }}
                onSave={() =>
                  needAuth(() => {
                    void saveCurrent();
                  }, "save")
                }
                saving={saving}
                buy={buy}
              />
            )}
          {view === "account" &&
            (account ? (
              <AccountView
                account={account}
                setReport={(r) => {
                  setReport(r);
                  navigate("/account/reports/" + encodeURIComponent(r.id));
                  window.scrollTo(0, 0);
                }}
                onNew={() => home("kundli")}
                onBuy={buy}
                refresh={refreshAccount}
                toast={setToast}
                logout={() => logout()}
              />
            ) : (
              <div className="container workspace empty-state">
                {authError ? (
                  <>
                    <h1>{t("Your account")}</h1>
                    <ErrorBox error={authError} />
                    <Button onClick={() => void refreshAccount()}>
                      {t("Try again")}
                    </Button>
                  </>
                ) : (
                  <p role="status">{t("Checking your session\u2026")}</p>
                )}
              </div>
            ))}
          {view === "login" && (!account || finishingLogin.current) && (
            <LoginPage
              config={config}
              onPolicy={setPolicy}
              onSuccess={async () => {
                finishingLogin.current = true;
                try {
                  const a = await refreshAccount();
                  if (!a)
                    throw new Error(
                      "Unable to open your account. Please try again.",
                    );
                  await api(
                    "/privacy",
                    "PUT",
                    {
                      marketing: a.user.marketing,
                      analytics:
                        localStorage.getItem("m4y-analytics") === "yes",
                    },
                    a.csrf,
                  ).catch((error) => setToast(error.message));
                  resumeAfterLogin();
                } finally {
                  finishingLogin.current = false;
                }
              }}
            />
          )}
          {view === "report" &&
            (!report ||
              (savedReportID &&
                (!account ||
                  report.id !== savedReportID ||
                  report.user_id !== account.user.id))) && (
              <div className="container workspace empty-state">
                {savedReportID &&
                (!sessionResolved ||
                  reportLoading ||
                  (!reportError && !authError)) ? (
                  <p role="status">{t("Opening your workspace\u2026")}</p>
                ) : (
                  <>
                    <h1>{t("Your Kundli")}</h1>
                    <p role={reportError || authError ? "alert" : undefined}>
                      {t(
                        reportError ||
                          authError ||
                          "Create your birth chart to begin.",
                      )}
                    </p>
                    <Button onClick={() => home("kundli")}>
                      {t("Create your free Kundli")}
                    </Button>
                  </>
                )}
              </div>
            )}
        </Suspense>
      </main>
      {publicLayout ? (
        <footer className="site-footer">
          <div className="container">
            <div className="footer-top">
              <a
                className="brand"
                href="/"
                onClick={(e) => {
                  e.preventDefault();
                  home();
                }}
              >
                <Brand inverse />
              </a>
              <p>{t("परंपरा से प्रेरित। समझ की ओर समर्पित।")}</p>
              <button className="text-button" onClick={() => home("kundli")}>
                {t("अपनी कुंडली बनाएँ ")}
                <ArrowUpRight size={17} />
              </button>
            </div>
            <div className="footer-bottom">
              <span>© {new Date().getFullYear()} Mishra 4 You</span>
              <div>
                <button onClick={() => setPolicy("privacy")}>
                  {t("गोपनीयता")}
                </button>
                <button onClick={() => setPolicy("terms")}>
                  {t("नियम व शर्तें")}
                </button>
                <button onClick={() => setPolicy("refund")}>
                  {t("रद्दीकरण / धनवापसी")}
                </button>
                <button onClick={() => setPolicy("contact")}>
                  {t("संपर्क")}
                </button>
                <button onClick={() => setConsent("")}>
                  {t("कुकी विकल्प")}
                </button>
              </div>
            </div>
            <p className="footer-note">
              {t(
                "ज्योतिषीय मार्गदर्शन व्यक्तिगत चिंतन के लिए है। यह चिकित्सा, कानूनी या वित्तीय सलाह नहीं है। किसी परिणाम की गारंटी नहीं है।",
              )}
            </p>
          </div>
        </footer>
      ) : (
        <PortalFooter />
      )}
      {toast && (
        <div className="toast" role="status">
          <Info size={18} />
          <span>{t(toast)}</span>
          <button onClick={() => setToast("")} aria-label={t("संदेश बंद करें")}>
            <X size={16} />
          </button>
        </div>
      )}
      {publicLayout && !consent && (
        <div className="consent">
          <div>
            <strong>{t("आपकी पसंद, आपकी गोपनीयता।")}</strong>
            <p>
              {t(
                "वैकल्पिक, बिना निजी विवरण वाले आँकड़ों से अनुभव सुधारने में मदद करें।",
              )}
            </p>
          </div>
          <button
            className="small-link"
            onClick={() => {
              setAnalytics("no");
            }}
          >
            {t("केवल आवश्यक")}
          </button>
          <Button
            onClick={() => {
              setAnalytics("yes");
              track("landing_visit");
            }}
          >
            {t("आँकड़ों की अनुमति")}
          </Button>
        </div>
      )}
      {offer && (
        <Modal
          title={
            offer === "pdf"
              ? t("अपनी उपाय रिपोर्ट पाएँ")
              : t("परामर्श का समय चुनें")
          }
          onClose={() => setOffer(null)}
          wide={offer === "consultation"}
        >
          <OfferCheckout
            product={offer}
            config={config}
            checkout={checkout}
            onSample={() => setSample(true)}
          />
        </Modal>
      )}
      {payment && account && (
        <Modal
          title={
            config.simulation
              ? t("स्थानीय भुगतान सिमुलेशन")
              : t("सुरक्षित भुगतान")
          }
          onClose={() => {
            setPayment(null);
            setView("account");
            void refreshAccount();
          }}
        >
          <PaymentView
            order={payment}
            config={config}
            csrf={account.csrf}
            onPaid={paid}
          />
        </Modal>
      )}
      {policy && (
        <Modal
          title={
            {
              privacy: t("गोपनीयता नीति"),
              terms: t("नियम व शर्तें"),
              refund: t("रद्दीकरण और धनवापसी"),
              contact: t("संपर्क और सहायता"),
            }[policy] || t("जानकारी")
          }
          onClose={() => setPolicy(null)}
        >
          <div className="policy-content">
            {!config.settings.content.approved && (
              <span className="pill">
                {t("समीक्षा हेतु प्रारूप \u00B7 स्वीकृति बाकी")}
              </span>
            )}
            <p>{t(config.settings.content.policies[policy])}</p>
            <p>{t(config.settings.content.support)}</p>
          </div>
        </Modal>
      )}
      {sample && (
        <Modal
          title={t("उदाहरण उपाय रिपोर्ट")}
          onClose={() => setSample(false)}
        >
          <span className="pill">
            {t("सांकेतिक नमूना \u00B7 व्यक्तिगत सलाह नहीं")}
          </span>
          <div className="sample-modal-chart">
            <Chart sample />
          </div>
          <h3>{t("रिपोर्ट में क्या मिलेगा?")}</h3>
          <p>
            {t(
              "जन्म विवरण, गणना संदर्भ, कुंडली और ग्रह-सारणी। लागू स्वीकृत व्याख्या, मंत्र और रत्न संबंधी मार्गदर्शन खरीद के समय सुरक्षित किया जाएगा।",
            )}
          </p>
          <p className="muted">
            {t(
              "अभी व्यक्तिगत उपाय का नमूना स्वीकृत नहीं है। रत्न इस खरीद में शामिल नहीं हैं।",
            )}
          </p>
        </Modal>
      )}
    </>
  );
}
/** BirthForm collects known or unknown birth-time details and requires a trusted birthplace selection. */
function BirthForm({
  birth,
  setBirth,
  loading,
  generate,
}: {
  birth: Birth;
  setBirth: (b: Birth) => void;
  loading: boolean;
  generate: () => Promise<void>;
}) {
  const { language } = useLanguage();
  const [places, setPlaces] = useState<Place[]>([]),
    [selectedPlace, setSelectedPlace] = useState<Place>(),
    [searching, setSearching] = useState(true),
    [searchError, setSearchError] = useState(""),
    [query, setQuery] = useState(""),
    [open, setOpen] = useState(false),
    [error, setError] = useState(""),
    [fold, setFold] = useState(false);
  const started = useRef(false);
  const root = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!birth.place_id) return;
    let active = true;
    void api<Place>("/places/" + encodeURIComponent(birth.place_id))
      .then((p) => {
        if (active) setSelectedPlace(p);
      })
      .catch(() => {
        if (active) setError("स्थान सूची उपलब्ध नहीं है। पृष्ठ फिर खोलें।");
      });
    return () => {
      active = false;
    };
  }, [birth.place_id]);
  useEffect(() => {
    if (selectedPlace?.id === birth.place_id) {
      setQuery(
        language === "hi"
          ? selectedPlace.hindi + " · " + selectedPlace.name
          : selectedPlace.name,
      );
    }
  }, [selectedPlace, birth.place_id, language]);
  useEffect(() => {
    if (birth.place_id) {
      setPlaces(selectedPlace ? [selectedPlace] : []);
      setSearching(!selectedPlace);
      setSearchError("");
      return;
    }
    let active = true;
    setSearching(true);
    setSearchError("");
    const timer = window.setTimeout(() => {
      void api<Place[]>("/places?q=" + encodeURIComponent(query))
        .then((results) => {
          if (active) setPlaces(results);
        })
        .catch(() => {
          if (active) {
            setPlaces([]);
            setSearchError(
              "स्थान खोज उपलब्ध नहीं है। दोबारा लिखकर कोशिश करें।",
            );
          }
        })
        .finally(() => {
          if (active) setSearching(false);
        });
    }, 200);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [query, birth.place_id, selectedPlace]);
  useEffect(() => {
    /** fn closes the birthplace suggestions after a click outside the search control. */
    const fn = (e: MouseEvent) => {
      if (!root.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", fn);
    return () => document.removeEventListener("mousedown", fn);
  }, []);
  /** update changes one birth field and invalidates any result that belongs to the previous inputs. */
  const update = (
    key: keyof Birth,
    v: string | boolean | number | undefined,
  ) => {
    if (!started.current) {
      track("form_started");
      started.current = true;
    }
    setBirth({ ...birth, [key]: v });
  };
  /** submit validates the birth form and starts the supplied Kundli-generation action. */
  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    if (!birth.place_id) {
      setError(t("खोज सूची से अपना जन्म स्थान चुनें।"));
      return;
    }
    try {
      await generate();
    } catch (e) {
      const message = (e as Error).message;
      setError(message);
      setFold(message.includes("UTC"));
    }
  }
  return (
    <form className="birth-form" onSubmit={submit}>
      <div className="form-heading">
        <div>
          <h3>{t("जन्म विवरण")}</h3>
          <p>{t("अपनी सही जानकारी भरें")}</p>
        </div>
        <span className="free-label">{t("निःशुल्क")}</span>
      </div>
      <label htmlFor="name">{t("आपका नाम")}</label>
      <input
        id="name"
        autoComplete="name"
        value={birth.name}
        onChange={(e) => update("name", e.target.value)}
        placeholder={t("अपना पूरा नाम लिखें")}
        minLength={2}
        maxLength={100}
        required
      />
      <div className="form-row">
        <div>
          <label htmlFor="birth-date">{t("जन्म तारीख")}</label>
          <input
            id="birth-date"
            type="date"
            value={birth.date}
            min={MIN_CALCULATION_DATE}
            max={
              selectedPlace?.id === birth.place_id
                ? new Intl.DateTimeFormat("en-CA", {
                    timeZone: selectedPlace.zone,
                    year: "numeric",
                    month: "2-digit",
                    day: "2-digit",
                  }).format(new Date())
                : undefined
            }
            onChange={(e) => update("date", e.target.value)}
            required
          />
        </div>
        <div>
          <label htmlFor="birth-time">
            {t("जन्म समय ")}
            <span>{t("स्थानीय")}</span>
          </label>
          <input
            id="birth-time"
            type="time"
            step="60"
            value={clockMinutes(birth.time)}
            onChange={(e) => update("time", e.target.value)}
            required={!birth.unknown_time}
            disabled={birth.unknown_time}
          />
        </div>
      </div>
      <label className="checkbox-label">
        <input
          type="checkbox"
          checked={birth.unknown_time}
          onChange={(e) => update("unknown_time", e.target.checked)}
        />
        {t("जन्म समय नहीं पता")}
      </label>
      {birth.unknown_time && (
        <p className="field-note">
          {t(
            "सही समय के बिना लग्न, भाव और ग्रहों की निश्चित स्थिति नहीं दी जाएगी।",
          )}
        </p>
      )}
      <div ref={root} className="place-field">
        <label htmlFor="birthplace">{t("जन्म स्थान")}</label>
        <div className="input-icon">
          <Globe2 size={17} />
          <input
            id="birthplace"
            role="combobox"
            aria-expanded={open}
            aria-controls="place-options"
            aria-autocomplete="list"
            aria-describedby="place-help"
            maxLength={100}
            autoComplete="off"
            placeholder={t("शहर या गाँव खोजें")}
            value={query}
            onFocus={() => setOpen(true)}
            onChange={(e) => {
              setQuery(e.target.value);
              setSelectedPlace(undefined);
              setPlaces([]);
              setSearching(true);
              setBirth({ ...birth, place_id: "" });
              setOpen(true);
            }}
            onKeyDown={(e) => {
              if (e.key === "Escape") setOpen(false);
              if (e.key === "ArrowDown") {
                e.preventDefault();
                document
                  .getElementById("place-options")
                  ?.querySelector("button")
                  ?.focus();
              }
            }}
          />
          <ChevronDown size={16} />
        </div>
        {open && (
          <ul
            id="place-options"
            className="place-options"
            role="listbox"
            aria-busy={searching}
          >
            {searching || searchError ? (
              <li className="empty-option" role="presentation">
                <span role="status">
                  {t(searching ? "स्थान खोजे जा रहे हैं…" : searchError)}
                </span>
              </li>
            ) : places.length ? (
              places.map((p) => (
                <li
                  key={p.id}
                  role="option"
                  aria-selected={p.id === birth.place_id}
                >
                  <button
                    type="button"
                    onClick={() => {
                      setSelectedPlace(p);
                      update("place_id", p.id);
                      setQuery(
                        language === "hi" ? p.hindi + " · " + p.name : p.name,
                      );
                      setOpen(false);
                      document.getElementById("birthplace")?.focus();
                    }}
                  >
                    <b>{language === "hi" ? p.hindi : p.name}</b>
                    <small>
                      {p.name} · {p.zone}
                    </small>
                  </button>
                </li>
              ))
            ) : (
              <li className="empty-option" role="presentation">
                {t(
                  "यह स्थान नहीं मिला। दूसरा नाम आज़माएँ या ज़िला / राज्य जोड़ें। सटीक गणना के लिए अपना ही जन्म स्थान चुनें।",
                )}
              </li>
            )}
          </ul>
        )}
      </div>
      <p className="field-note" id="place-help">
        {t(
          "पूरे भारत के शहर और गाँव खोजें। एक जैसे नामों के लिए ज़िला / राज्य जोड़ें। चुने हुए स्थान का समय-क्षेत्र अपने-आप लागू होगा।",
        )}{" "}
        <a href="https://www.geonames.org/" target="_blank" rel="noreferrer">
          GeoNames
        </a>
        {" · "}
        <a
          href="https://creativecommons.org/licenses/by/4.0/"
          target="_blank"
          rel="noreferrer"
        >
          CC BY 4.0
        </a>
      </p>
      {fold && (
        <>
          <label htmlFor="offset">
            {t("दोहराए गए समय का UTC offset (मिनट)")}
          </label>
          <input
            id="offset"
            type="number"
            placeholder={t("जैसे New York: -240 या -300")}
            value={birth.utc_offset_minutes ?? ""}
            onChange={(e) =>
              update(
                "utc_offset_minutes",
                e.target.value === "" ? undefined : Number(e.target.value),
              )
            }
          />
        </>
      )}
      <ErrorBox error={t(error)} />
      <Button type="submit" className="full" busy={loading}>
        {t("मेरी निःशुल्क कुंडली बनाएँ ")}
        <ArrowRight size={18} />
      </Button>
      <p className="form-privacy">
        <LockKeyhole size={12} />
        {t("आपके विवरण सार्वजनिक नहीं होंगे। सहेजना आपकी पसंद है।")}
      </p>
    </form>
  );
}
/** OfferCheckout shows the selected product's server price, availability, consultation slots and optional illustrative PDF sample. */
function OfferCheckout({
  product,
  config,
  checkout,
  onSample,
}: {
  product: "pdf" | "consultation";
  config: Config;
  checkout: (slot: string) => Promise<void>;
  onSample: () => void;
}) {
  const [slots, setSlots] = useState<string[]>([]),
    [slotDuration, setSlotDuration] = useState(config.duration),
    [slot, setSlot] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState(""),
    [loaded, setLoaded] = useState(false);
  useEffect(() => {
    if (product === "consultation")
      void api<{
        slots: string[];
        duration: number;
      }>("/slots")
        .then((v) => {
          setSlots(v.slots);
          setSlotDuration(v.duration);
          setLoaded(true);
        })
        .catch((e) => {
          setError(e.message);
          setLoaded(true);
        });
  }, [product]);
  const bs = product === "pdf" ? config.pdf_blockers : config.booking_blockers;
  /** submit runs the selected offer checkout while displaying its pending and failure states. */
  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await checkout(slot);
    } catch (e) {
      setError((e as Error).message);
      if (product === "consultation")
        void api<{
          slots: string[];
          duration: number;
        }>("/slots").then((v) => {
          setSlots(v.slots);
          setSlotDuration(v.duration);
        });
    } finally {
      setBusy(false);
    }
  }
  return (
    <form onSubmit={submit}>
      <div className="checkout-summary">
        <span>
          {product === "pdf" ? <FileText size={31} /> : <Video size={31} />}
        </span>
        <div>
          <h3>
            {product === "pdf"
              ? t("व्यक्तिगत उपाय PDF")
              : t("एक-से-एक Zoom परामर्श")}
          </h3>
          <p>
            {product === "pdf"
              ? t("तैयार होने पर खाते से निजी डाउनलोड")
              : t("{v0} मिनट {v1}", {
                  v0: config.duration,
                  v1: config.duration_demo ? t("\u00B7 उदाहरण अवधि") : "",
                })}
          </p>
        </div>
        <strong>
          {money(
            product === "pdf"
              ? config.settings.pdf_price
              : config.settings.consultation_price,
          )}
        </strong>
      </div>
      {config.simulation && (
        <div className="notice">
          <Info size={18} />
          <span>
            {t("स्थानीय सिमुलेशन \u00B7 कोई शुल्क नहीं लगेगा।")}{" "}
            {product === "pdf"
              ? t("डेमो सामग्री वाली रिपोर्ट मिलेगी।")
              : t("वास्तविक Zoom बैठक नहीं बनेगी।")}
          </span>
        </div>
      )}
      {product === "consultation" && (
        <>
          <h3>{t("उपलब्ध समय")}</h3>
          <p className="muted">
            {t("समय आपके क्षेत्र ")}
            {zone}
            {t(" में है। चेकआउट पर 10 मिनट के लिए सुरक्षित होगा।")}
          </p>
          <div className="slot-list">
            {!loaded ? (
              <p>
                <LoaderCircle size={17} className="spin" />
                {t("समय देख रहे हैं\u2026")}
              </p>
            ) : slots.length ? (
              slots.slice(0, 60).map((v) => (
                <label
                  className={"slot " + (slot === v ? "selected" : "")}
                  key={v}
                >
                  <input
                    type="radio"
                    name="slot"
                    value={v}
                    checked={slot === v}
                    onChange={() => setSlot(v)}
                    required
                  />
                  <CalendarDays size={16} />
                  <TimeRange
                    start={v}
                    end={new Date(
                      Date.parse(v) + slotDuration * 60000,
                    ).toISOString()}
                    format={localTime}
                  />
                </label>
              ))
            ) : (
              <p>{t("अभी कोई समय उपलब्ध नहीं है। बाद में फिर देखें।")}</p>
            )}
          </div>
          <p className="field-note">
            {t("पुनर्निर्धारण / रद्दीकरण: ")}
            {config.settings.change_hours}
            {t(" घंटे पहले तक। अंतिम धनवापसी शर्तें अभी स्वीकृति के अधीन हैं।")}
          </p>
        </>
      )}
      {product === "pdf" && (
        <>
          <p>
            {t(
              "रिपोर्ट में कुंडली और लागू सामग्री का सुरक्षित संस्करण शामिल होगा। रत्न शामिल नहीं हैं। सही जन्म समय आवश्यक है।",
            )}
          </p>
          <button type="button" className="small-link" onClick={onSample}>
            {t("View a sample")} <ArrowUpRight size={14} />
          </button>
        </>
      )}
      <p className="field-note">
        {t("कर व्यवस्था:")}{" "}
        {config.settings.tax_mode === "unconfigured"
          ? t("निर्णय बाकी; केवल स्थानीय डेमो")
          : config.settings.tax_mode}
        {config.settings.tax_mode === "exclusive"
          ? t(" \u00B7 {v0}% अतिरिक्त कर अंतिम ऑर्डर में", {
              v0: config.settings.tax_bps / 100,
            })
          : ""}
      </p>
      {bs.length > 0 && <ErrorBox error={bs.join(" · ")} />}
      <ErrorBox error={t(error)} />
      <Button
        className="full"
        busy={busy}
        disabled={!!bs.length || (product === "consultation" && !slot)}
      >
        {t("ऑर्डर बनाएँ और भुगतान देखें ")}
        <ArrowRight size={17} />
      </Button>
    </form>
  );
}
/** AccountView renders owned Kundlis, purchases, consultation bookings and customer privacy controls. */
function AccountView({
  account,
  setReport,
  onNew,
  onBuy,
  refresh,
  toast,
  logout,
}: {
  account: Account;
  setReport: (r: Report) => void;
  onNew: () => void;
  onBuy: (p: "pdf" | "consultation") => void;
  refresh: () => Promise<unknown>;
  toast: (s: string) => void;
  logout: () => Promise<void>;
}) {
  const route = useAppLocation();
  const selectedTab = route.searchParams.get("tab");
  const tab = ["bookings", "orders", "privacy"].includes(selectedTab || "") ? selectedTab : "reports";
  const [reschedule, setReschedule] = useState<Booking | null>(null),
    [slots, setSlots] = useState<string[]>([]),
    [slotDuration, setSlotDuration] = useState(0),
    [slot, setSlot] = useState(""),
    [busy, setBusy] = useState(false),
    [confirmDelete, setConfirmDelete] = useState(false);
  /** action submits a customer booking change and refreshes the owned booking list on success. */
  async function action(b: Booking, action: string) {
    try {
      await api(`/bookings/${b.id}`, "POST", { action, slot }, account.csrf);
      setReschedule(null);
      await refresh();
      toast(t("अनुरोध सुरक्षित है। बैठक सेवा की स्थिति खाते में देखें।"));
    } catch (e) {
      toast((e as Error).message);
    }
  }
  return (
    <div className="container workspace">
      <div className="workspace-heading">
        <div>
          <div className="eyebrow">{t("आपकी अपनी जगह")}</div>
          <h1>{t("मेरा खाता")}</h1>
          <p>
            {account.user.phone}
            {t(" \u00B7 सुरक्षित मोबाइल सत्यापन")}
          </p>
        </div>
        <button className="text-button" onClick={() => void logout()}>
          <LogOut size={17} />
          {t("लॉग आउट")}
        </button>
      </div>
      <p>
        <a
          className="button secondary"
          href="/shop/orders"
          onClick={(e) => followRoute(e, "/shop/orders")}
        >
          {t("My stone orders")} <ArrowRight size={16} />
        </a>
      </p>
      <div className="tabs" role="tablist">
        {[
          ["reports", t("कुंडली और PDF")],
          ["bookings", t("मेरे परामर्श")],
          ["orders", t("भुगतान")],
          ["privacy", t("गोपनीयता")],
        ].map(([id, label]) => (
          <button
            role="tab"
            aria-selected={tab === id}
            key={id}
            onClick={() => navigate(id === "reports" ? "/account" : "/account?tab=" + id)}
          >
            {label}
          </button>
        ))}
      </div>
      <div role="tabpanel">
        {tab === "reports" && (
          <>
            <div className="panel-title">
              <h2>{t("सहेजी गई कुंडलियाँ")}</h2>
              <Button className="secondary" onClick={onNew}>
                <Plus size={17} />
                {t("नई कुंडली")}
              </Button>
            </div>
            {account.reports.length ? (
              <div className="saved-reports">
                {account.reports.map((r) => (
                  <article className="saved-report" key={r.id}>
                    <div className="report-mini-icon">
                      <Sun size={31} strokeWidth={1} />
                    </div>
                    <div>
                      <h3>{r.birth.name}</h3>
                      <p>
                        {r.birth.date} ·{" "}
                        {getLanguage() === "hi" ? r.place.hindi : r.place.name}
                      </p>
                      <small>
                        {r.birth.unknown_time
                          ? t("जन्म समय अज्ञात")
                          : t("जन्म कुंडली \u00B7 D1")}
                      </small>
                    </div>
                    <button
                      className="text-button"
                      onClick={() => setReport(r)}
                    >
                      {t("रिपोर्ट खोलें ")}
                      <ArrowUpRight size={17} />
                    </button>
                    <button
                      className="icon-button"
                      aria-label={t("{v0} की कुंडली मिटाएँ", {
                        v0: r.birth.name,
                      })}
                      onClick={async () => {
                        try {
                          await api(
                            `/reports/${r.id}`,
                            "DELETE",
                            undefined,
                            account.csrf,
                          );
                          await refresh();
                        } catch (e) {
                          toast((e as Error).message);
                        }
                      }}
                    >
                      <X size={16} />
                    </button>
                  </article>
                ))}
              </div>
            ) : (
              <div className="empty-state">
                <BookOpen size={35} />
                <h3>{t("आपकी पहली कुंडली का इंतज़ार है।")}</h3>
                <p>{t("जन्म विवरण भरें और रिपोर्ट को खाते में सहेजें।")}</p>
                <Button onClick={onNew}>{t("कुंडली बनाएँ")}</Button>
              </div>
            )}
            <h2 className="space-top">{t("आपकी खरीदी हुई PDF")}</h2>
            {account.orders
              .filter((o) => o.product === "pdf" && o.status === "paid")
              .map((o) => {
                const j = account.pdf_jobs.find((j) => j.ref === o.id);
                return (
                  <article className="saved-report" key={o.id}>
                    <FileText size={28} />
                    <div>
                      <h3>
                        {t(o.title)}{" "}
                        {o.demo && <span className="pill">{t("डेमो")}</span>}
                      </h3>
                      <p>
                        {j?.status === "done"
                          ? t("डाउनलोड के लिए तैयार")
                          : j?.status === "retry"
                            ? t(
                                "तैयार होने में देरी है; अपने-आप फिर प्रयास होगा",
                              )
                            : t("आपकी रिपोर्ट तैयार हो रही है\u2026")}
                      </p>
                    </div>
                    {j?.status === "done" && (
                      <RemedyPDFDownload orderId={o.id} />
                    )}
                  </article>
                );
              })}
            {!account.orders.some(
              (o) => o.product === "pdf" && o.status === "paid",
            ) && (
              <p className="muted">
                {t(
                  "कोई खरीदी हुई PDF नहीं। पहले अपनी कुंडली खोलकर रिपोर्ट चुनें।",
                )}
              </p>
            )}
          </>
        )}
        {tab === "bookings" && (
          <>
            <div className="panel-title">
              <h2>{t("मेरे परामर्श")}</h2>
              <Button onClick={() => onBuy("consultation")}>
                {t("नया परामर्श ")}
                <Plus size={16} />
              </Button>
            </div>
            <p className="muted">
              {t("सभी समय ")}
              {zone}
              {t(" में हैं।")}
            </p>
            {account.bookings.length ? (
              account.bookings.map((b) => (
                <article className="booking-row" key={b.id}>
                  <div className="booking-date">
                    <CalendarDays size={28} />
                  </div>
                  <div className="booking-info">
                    <TimeRange start={b.start} end={b.end} format={localTime} />
                    <p>
                      {Math.round(
                        (new Date(b.end).getTime() -
                          new Date(b.start).getTime()) /
                          60000,
                      )}{" "}
                      {t("मिनट \u00B7 Zoom परामर्श")}{" "}
                      {b.demo && <span className="pill">{t("डेमो")}</span>}
                    </p>
                    <span className={"status status-" + b.status}>
                      {t(statusText[b.status] || b.status)}
                    </span>
                    {b.status === "meeting_pending" && (
                      <p className="field-note">
                        {t(
                          "भुगतान सुरक्षित है। बैठक लिंक के लिए अपने-आप पुनः प्रयास हो रहा है।",
                        )}
                      </p>
                    )}
                    {b.status === "needs_resolution" && (
                      <p className="error">
                        {t(
                          "भुगतान मिला, लेकिन समय उपलब्ध नहीं रहा। नया समय चुनें या धनवापसी के लिए सहायता लें।",
                        )}
                      </p>
                    )}
                    {b.demo && b.status === "confirmed" && (
                      <p className="field-note">
                        {t(
                          "केवल स्थानीय सिमुलेशन; वास्तविक बैठक लिंक नहीं बनाया गया।",
                        )}
                      </p>
                    )}
                  </div>
                  <div className="booking-actions">
                    {b.join_url && b.status === "confirmed" && (
                      <a
                        className="button"
                        href={b.join_url}
                        target="_blank"
                        rel="noreferrer"
                      >
                        {t("Zoom से जुड़ें ")}
                        <Video size={16} />
                      </a>
                    )}
                    {[
                      "confirmed",
                      "meeting_pending",
                      "needs_resolution",
                    ].includes(b.status) && (
                      <>
                        <button
                          className="small-link"
                          onClick={async () => {
                            try {
                              const v = await api<{
                                slots: string[];
                                duration: number;
                              }>("/slots");
                              setSlots(v.slots);
                              setSlotDuration(v.duration);
                              setReschedule(b);
                              setSlot("");
                            } catch (e) {
                              toast((e as Error).message);
                            }
                          }}
                        >
                          {t("समय बदलें")}
                        </button>
                        <button
                          className="small-link danger-text"
                          onClick={() => void action(b, "cancel")}
                        >
                          {t("रद्द करने का अनुरोध")}
                        </button>
                      </>
                    )}
                  </div>
                </article>
              ))
            ) : (
              <div className="empty-state">
                <CalendarDays size={35} />
                <h3>{t("अभी कोई परामर्श बुक नहीं है।")}</h3>
                <p>
                  {t("PDF खरीदे बिना सीधे गुरुजी से परामर्श चुन सकते हैं।")}
                </p>
              </div>
            )}
          </>
        )}
        {tab === "orders" && (
          <>
            <h2>{t("ऑर्डर और भुगतान")}</h2>
            <div className="table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>{t("ऑर्डर")}</th>
                    <th>{t("विकल्प")}</th>
                    <th>{t("राशि")}</th>
                    <th>{t("स्थिति")}</th>
                  </tr>
                </thead>
                <tbody>
                  {account.orders.map((o) => (
                    <tr key={o.id}>
                      <td>
                        <small className="reference">{o.id}</small>
                        {localTime(o.created)}
                      </td>
                      <td>
                        {t(o.title)}
                        {o.demo && <span className="pill">{t("डेमो")}</span>}
                      </td>
                      <td>
                        {money(o.amount)}
                        {o.refunded > 0 && (
                          <small>
                            {t("वापसी ")}
                            {money(o.refunded)}
                          </small>
                        )}
                      </td>
                      <td>
                        <span className={"status status-" + o.status}>
                          {t(statusText[o.status] || o.status)}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {!account.orders.length && (
              <p className="muted">{t("अभी कोई ऑर्डर नहीं।")}</p>
            )}
            <p className="field-note">
              {t(
                "लंबित भुगतान को बिक्री नहीं माना जाता। प्रदाता सूचना देर से आने पर सर्वर फिर जाँचता है।",
              )}
            </p>
          </>
        )}
        {tab === "privacy" && (
          <section className="reading-panel">
            <h2>{t("आपके विवरण पर आपका नियंत्रण")}</h2>
            <p>
              {t(
                "जन्म विवरण और रिपोर्ट केवल आपके खाते में उपलब्ध हैं। वैकल्पिक प्रचार की सहमति अलग है।",
              )}
            </p>
            <label className="checkbox-label">
              <input
                type="checkbox"
                checked={account.user.marketing}
                onChange={async (e) => {
                  try {
                    await api(
                      "/privacy",
                      "PUT",
                      { marketing: e.target.checked },
                      account.csrf,
                    );
                    await refresh();
                    toast(
                      e.target.checked
                        ? t("WhatsApp विपणन की अलग सहमति दर्ज हुई।")
                        : t("प्रचार की सहमति वापस ले ली गई।"),
                    );
                  } catch (e) {
                    toast((e as Error).message);
                  }
                }}
              />
              {t(
                "मैं वैकल्पिक WhatsApp प्रचार के लिए अलग से सहमति देता/देती हूँ।",
              )}
            </label>
            <p className="muted">
              {t(
                "बुकिंग की आवश्यक सूचनाएँ इस विकल्प से अलग हैं। वर्तमान संस्करण कोई विपणन संदेश नहीं भेजता।",
              )}
            </p>
            <a
              className="button secondary space-top"
              href="/api/account/export"
              download="mishra4you-account.json"
            >
              <Download size={16} />
              {t("अपना डेटा डाउनलोड करें")}
            </a>
            <div className="privacy-delete">
              <h3>{t("खाता और निजी रिपोर्ट मिटाएँ")}</h3>
              <p>
                {t(
                  "यह कार्रवाई निजी जन्म विवरण और डाउनलोड हटाती है। सक्रिय बुकिंग/भुगतान का निपटारा पहले आवश्यक है। न्यूनतम लेखा अभिलेख अलग रखे जाते हैं।",
                )}
              </p>
              <Button className="danger" onClick={() => setConfirmDelete(true)}>
                {t("खाता मिटाएँ")}
              </Button>
            </div>
          </section>
        )}
      </div>
      {reschedule && (
        <Modal
          title={t("परामर्श का नया समय")}
          onClose={() => setReschedule(null)}
        >
          <div className="slot-list">
            {slots.map((v) => (
              <label className="slot" key={v}>
                <input
                  type="radio"
                  name="new-slot"
                  checked={slot === v}
                  onChange={() => setSlot(v)}
                />
                <TimeRange
                  start={v}
                  end={new Date(
                    Date.parse(v) + slotDuration * 60000,
                  ).toISOString()}
                  format={localTime}
                />
              </label>
            ))}
            {!slots.length && <p>{t("अभी कोई नया समय उपलब्ध नहीं।")}</p>}
          </div>
          <Button
            className="full"
            disabled={!slot}
            onClick={() => void action(reschedule, "reschedule")}
          >
            {t("नया समय सुरक्षित करें")}
          </Button>
        </Modal>
      )}
      {confirmDelete && (
        <Modal
          title={t("खाता स्थायी रूप से मिटाएँ?")}
          onClose={() => setConfirmDelete(false)}
        >
          <p>
            {t(
              "निजी रिपोर्ट वापस नहीं मिल पाएँगी। आवश्यक लेखा अभिलेखों की अवधि स्वीकृत नीति पर निर्भर होगी।",
            )}
          </p>
          <Button
            className="danger full"
            busy={busy}
            onClick={async () => {
              setBusy(true);
              try {
                await api("/account", "DELETE", undefined, account.csrf);
                sessionStorage.removeItem("m4y-birth");
                sessionStorage.removeItem("m4y-guest-report");
                sessionStorage.removeItem("m4y-login-intent");
                location.href = "/";
              } catch (e) {
                toast((e as Error).message);
                setBusy(false);
                setConfirmDelete(false);
              }
            }}
          >
            {t("हाँ, मेरा खाता मिटाएँ")}
          </Button>
        </Modal>
      )}
    </div>
  );
}
const statusText: Record<string, string> = {
  paid: "भुगतान सत्यापित",
  pending: "भुगतान लंबित",
  failed: "असफल",
  refunded: "धनवापसी पूर्ण",
  creating: "ऑर्डर बन रहा है",
  held: "चेकआउट के लिए सुरक्षित",
  expired: "समय-सीमा समाप्त",
  meeting_pending: "बैठक लिंक तैयार हो रहा है",
  confirmed: "परामर्श की पुष्टि",
  needs_resolution: "नया समय / समाधान आवश्यक",
  cancel_pending: "रद्दीकरण चल रहा है",
  cancelled: "रद्द · धनवापसी नीति अनुसार",
};
/** VideoCard renders a selected channel video and loads the external player only after interaction. */
function VideoCard({
  id,
  index,
  video,
  checkedAt,
}: {
  id: string;
  index: number;
  video?: ChannelVideo;
  checkedAt?: string;
}) {
  const [play, setPlay] = useState(false);
  const { language } = useLanguage();
  const title = video
    ? language === "en"
      ? video.english_title
      : video.title
    : t("गुरुजी का स्वीकृत वीडियो {v0}", { v0: index + 1 });
  const videoURL = `https://www.youtube.com/watch?v=${id}`;
  const views =
    video &&
    new Intl.NumberFormat(language === "hi" ? "hi-IN" : "en-US", {
      notation: "compact",
      maximumFractionDigits: 1,
    }).format(video.views);
  return (
    <article className="video-card" data-video-id={id}>
      {play ? (
        <iframe
          src={`https://www.youtube-nocookie.com/embed/${id}?autoplay=1`}
          title={title}
          allow="autoplay; encrypted-media; picture-in-picture"
          allowFullScreen
          loading="lazy"
          referrerPolicy="strict-origin-when-cross-origin"
        />
      ) : (
        <button
          onClick={() => setPlay(true)}
          aria-label={`${t("Play video")}: ${title}. ${video?.duration ? video.duration + ". " : ""}${t("Connects to YouTube")}`}
        >
          <img
            src={`https://i.ytimg.com/vi/${id}/hqdefault.jpg`}
            loading="lazy"
            alt=""
            width="480"
            height="360"
          />
          <span className="video-play" aria-hidden="true">
            <Play size={26} fill="currentColor" />
          </span>
          {video?.duration && (
            <small className="video-duration">{video.duration}</small>
          )}
        </button>
      )}
      <div className="video-meta">
        {video && (
          <span
            title={
              checkedAt
                ? `${t("Views recorded on")} ${new Intl.DateTimeFormat(language === "hi" ? "hi-IN" : "en-IN", { day: "numeric", month: "short", year: "numeric" }).format(new Date(checkedAt))}`
                : undefined
            }
          >
            {views} {t("views")}
          </span>
        )}
        {video && <span>{new Date(video.published_at).getFullYear()}</span>}
      </div>
      <h3 lang={language} title={title}>
        {title}
      </h3>
      <a
        href={videoURL}
        target="_blank"
        rel="noopener noreferrer"
        className="video-youtube-link"
        aria-label={`${t("Watch on YouTube")}: ${title}`}
      >
        {t("Watch on YouTube")}
        <ArrowUpRight size={14} aria-hidden="true" />
      </a>
    </article>
  );
}
