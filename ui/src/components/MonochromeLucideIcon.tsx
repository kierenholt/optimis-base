import React from 'react';
import {
  Activity,
  AlertTriangle,
  ArrowDown,
  ArrowRight,
  ArrowUp,
  Award,
  BookOpen,
  Box,
  Bus,
  Calculator,
  Calendar,
  Check,
  ClipboardCheck,
  Download,
  FileCheck,
  GraduationCap,
  HeartPulse,
  Home,
  House,
  Inbox,
  KeyRound,
  LayoutGrid,
  LockOpen,
  LogOut,
  LucideIcon,
  Pin,
  Plus,
  RefreshCw,
  Search,
  Settings,
  ShieldAlert,
  Stethoscope,
  Trash2,
  User,
  UserCheck,
  Users,
  X,
} from 'lucide-react';

const ICON_MAP: Record<string, LucideIcon> = {
  activity: Activity,
  'alert-triangle': AlertTriangle,
  'arrow-down': ArrowDown,
  'arrow-right': ArrowRight,
  'arrow-up': ArrowUp,
  award: Award,
  'book-open': BookOpen,
  box: Box,
  bus: Bus,
  calculator: Calculator,
  calendar: Calendar,
  check: Check,
  'clipboard-check': ClipboardCheck,
  download: Download,
  'file-check': FileCheck,
  'graduation-cap': GraduationCap,
  'heart-pulse': HeartPulse,
  home: Home,
  house: House,
  inbox: Inbox,
  'key-round': KeyRound,
  'layout-grid': LayoutGrid,
  'lock-open': LockOpen,
  'log-out': LogOut,
  pin: Pin,
  plus: Plus,
  'refresh-cw': RefreshCw,
  search: Search,
  settings: Settings,
  'shield-alert': ShieldAlert,
  stethoscope: Stethoscope,
  'trash-2': Trash2,
  user: User,
  'user-check': UserCheck,
  users: Users,
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
