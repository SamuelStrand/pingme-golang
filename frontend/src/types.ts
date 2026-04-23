export type TokenPair = {
  access_token: string;
  refresh_token: string;
};

export type ApiErrorPayload = {
  error?: string;
  message?: string;
  fields?: Record<string, string>;
};

export type User = {
  id: string;
  email: string;
  user_tg?: string | null;
  created_at: string;
};

export type Health = {
  status?: string;
  db?: string;
};

export type TargetStatus = "unknown" | "up" | "down";

export type Target = {
  id: string;
  url: string;
  name: string;
  interval: number;
  enabled: boolean;
  status: TargetStatus;
  last_checked_at?: string | null;
  created_at: string;
};

export type TargetListResponse = {
  items: Target[];
  page: number;
  page_size: number;
  total: number;
};

export type TargetLog = {
  id: string;
  status_code: number;
  response_time_ms: number;
  success: boolean;
  error_message: string;
  checked_at: string;
};

export type TargetLogListResponse = {
  items: TargetLog[];
  page: number;
  page_size: number;
  total: number;
};

export type TargetTimelinePoint = {
  timestamp: string;
  success: boolean;
  response_time_ms: number;
};

export type TargetStatsResponse = {
  target_id: string;
  from: string;
  to: string;
  uptime_percent: number;
  avg_response_ms: number;
  total_checks: number;
  failed_checks: number;
  timeline: TargetTimelinePoint[];
};

export type AlertChannelType = "telegram" | "webhook";

export type AlertChannel = {
  id: string;
  user_id: string;
  type: AlertChannelType;
  address: string;
  enabled: boolean;
  created_at: string;
};

export type TelegramLinkTokenResponse = {
  link_url: string;
  expires_at: string;
};
