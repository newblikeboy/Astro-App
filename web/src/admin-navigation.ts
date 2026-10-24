import { Activity, CalendarDays, CircleHelp, FileText, Gem, LayoutDashboard, ListChecks, Settings2, ShieldCheck, Users, Wallet, Link } from "lucide-react";

export const adminSections = [
  {id: "overview", label: "Business overview", group: "Workspace", icon: LayoutDashboard, description: "See what needs your attention and choose your next task."},
  {id: "shop", label: "Gemstone shop", group: "Business", icon: Gem, description: "Manage products, prices, stock and stone deliveries."},
  {id: "orders", label: "Payments & refunds", group: "Business", icon: Wallet, description: "Review every payment, verify its status and manage refunds."},
  {id: "calendar", label: "Consultations", group: "Business", icon: CalendarDays, description: "Review bookings, reschedule appointments and manage availability."},
  {id: "users", label: "Customers", group: "Business", icon: Users, description: "Find customer accounts and review saved-profile counts."},
  {id: "finance", label: "Finance & expenses", group: "Business", icon: Activity, description: "Reconcile monthly collections, expenses and partner distributions."},
  {id: "content", label: "Website content", group: "Manage", icon: FileText, description: "Update your introduction, videos, FAQs and business policies."},
  {id: "video-links", label: "Video topic links", group: "Manage", icon: Link, description: "Create a personal Kundli journey for each YouTube video."},
  {id: "rules", label: "Interpretations & remedies", group: "Manage", icon: CircleHelp, description: "Review and approve versioned readings and remedies."},
  {id: "jobs", label: "Delivery queue", group: "Manage", icon: ListChecks, description: "Follow report, meeting and notification delivery; retry failed jobs."},
  {id: "settings", label: "Service settings", group: "Manage", icon: Settings2, description: "Set service prices, taxes, booking hours and selling availability."},
  {id: "audit", label: "Activity log", group: "Manage", icon: ShieldCheck, description: "Review recorded administrator actions and references."},
] as const;
