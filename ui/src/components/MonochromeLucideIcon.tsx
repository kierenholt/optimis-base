import React from 'react';
import {
  Activity,
  AlertTriangle,
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  ArrowUp,
  Award,
  BarChart3,
  Bell,
  BookOpen,
  Box,
  Briefcase,
  Bus,
  Calculator,
  Calendar,
  CalendarClock,
  Check,
  CheckCircle,
  ClipboardCheck,
  ClipboardList,
  Clock,
  CreditCard,
  Database,
  Download,
  FileCheck,
  FileText,
  Folder,
  GitBranch,
  GraduationCap,
  HeartPulse,
  Home,
  House,
  Inbox,
  KeyRound,
  Landmark,
  Layers,
  LayoutGrid,
  LockOpen,
  LogOut,
  LucideIcon,
  MessageSquare,
  Minus,
  Pin,
  Plus,
  RefreshCw,
  Search,
  Settings,
  ShieldAlert,
  ShieldCheck,
  Stethoscope,
  Trash2,
  User,
  UserCheck,
  Users,
  Utensils,
  Wallet,
  X,
} from 'lucide-react';

const ICON_MAP: Record<string, LucideIcon> = {
  activity: Activity,
  'alert-triangle': AlertTriangle,
  'arrow-down': ArrowDown,
  'arrow-left': ArrowLeft,
  'arrow-right': ArrowRight,
  'arrow-up': ArrowUp,
  award: Award,
  'bar-chart-3': BarChart3,
  bell: Bell,
  'book-open': BookOpen,
  box: Box,
  briefcase: Briefcase,
  bus: Bus,
  calculator: Calculator,
  calendar: Calendar,
  'calendar-clock': CalendarClock,
  check: Check,
  'check-circle': CheckCircle,
  'clipboard-check': ClipboardCheck,
  'clipboard-list': ClipboardList,
  clock: Clock,
  'credit-card': CreditCard,
  database: Database,
  download: Download,
  'file-check': FileCheck,
  'file-text': FileText,
  folder: Folder,
  'git-branch': GitBranch,
  'graduation-cap': GraduationCap,
  'heart-pulse': HeartPulse,
  home: Home,
  house: House,
  inbox: Inbox,
  'key-round': KeyRound,
  landmark: Landmark,
  layers: Layers,
  'layout-grid': LayoutGrid,
  'lock-open': LockOpen,
  'log-out': LogOut,
  'message-square': MessageSquare,
  minus: Minus,
  pin: Pin,
  plus: Plus,
  'refresh-cw': RefreshCw,
  search: Search,
  settings: Settings,
  'shield-alert': ShieldAlert,
  'shield-check': ShieldCheck,
  stethoscope: Stethoscope,
  'trash-2': Trash2,
  user: User,
  'user-check': UserCheck,
  users: Users,
  utensils: Utensils,
  wallet: Wallet,
  x: X,
};

interface MonochromeLucideIconProps {
  name?: string;
  fallback?: string;
  size?: number;
  style?: React.CSSProperties;
}

export function MonochromeLucideIcon({
  name,
  fallback = 'box',
  size = 16,
  style,
}: MonochromeLucideIconProps) {
  const rawKey = (name || fallback || 'box').trim().toLowerCase();
  const resolvedKey = ICON_MAP[rawKey] ? rawKey : ICON_MAP[fallback] ? fallback : 'box';
  const IconComponent = ICON_MAP[resolvedKey] || Box;

  return (
    <IconComponent
      size={size}
      stroke="currentColor"
      fill="none"
      strokeWidth={2}
      data-lucide-icon={resolvedKey}
      aria-hidden="true"
      style={{
        display: 'inline-block',
        verticalAlign: 'middle',
        flexShrink: 0,
        color: 'currentColor',
        ...style,
      }}
    />
  );
}
