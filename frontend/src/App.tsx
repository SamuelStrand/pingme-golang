import { FormEvent, ReactNode, useCallback, useEffect, useMemo, useState } from "react";
import { ApiError, apiRequest, clearTokens, loadTokens, saveTokens } from "./api";
import {
  AlertChannel,
  AlertChannelType,
  Health,
  Target,
  TargetListResponse,
  TargetLogListResponse,
  TargetStatsResponse,
  TelegramLinkTokenResponse,
  TokenPair,
  User
} from "./types";
import {
  CheckIcon,
  EditIcon,
  ExternalIcon,
  LinkIcon,
  LogsIcon,
  LogoutIcon,
  PlusIcon,
  RefreshIcon,
  TrashIcon
} from "./icons";

type View = "overview" | "targets" | "alerts" | "telegram";

type AuthMode = "login" | "register";

type TargetDraft = {
  name: string;
  url: string;
  interval: string;
  enabled: boolean;
};

type AlertDraft = {
  type: AlertChannelType;
  address: string;
  enabled: boolean;
};

type StatsRange = "24h" | "7d";

type Notice = {
  type: "success" | "error";
  text: string;
} | null;

const emptyTargetDraft: TargetDraft = {
  name: "",
  url: "",
  interval: "60",
  enabled: true
};

const emptyAlertDraft: AlertDraft = {
  type: "webhook",
  address: "",
  enabled: true
};

const initialTargetPage: TargetListResponse = {
  items: [],
  page: 1,
  page_size: 20,
  total: 0
};

const initialLogsPage: TargetLogListResponse = {
  items: [],
  page: 1,
  page_size: 20,
  total: 0
};

function App() {
  const [tokens, setTokens] = useState<TokenPair | null>(() => loadTokens());
  const [user, setUser] = useState<User | null>(null);
  const [health, setHealth] = useState<Health | null>(null);
  const [healthError, setHealthError] = useState(false);
  const [view, setView] = useState<View>("overview");
  const [notice, setNotice] = useState<Notice>(null);
  const [booting, setBooting] = useState(true);
  const [busy, setBusy] = useState(false);

  const [targets, setTargets] = useState<TargetListResponse>(initialTargetPage);
  const [targetPage, setTargetPage] = useState(1);
  const [targetDraft, setTargetDraft] = useState<TargetDraft>(emptyTargetDraft);
  const [editingTargetId, setEditingTargetId] = useState<string | null>(null);
  const [selectedTargetId, setSelectedTargetId] = useState<string | null>(null);

  const [logs, setLogs] = useState<TargetLogListResponse>(initialLogsPage);
  const [logPage, setLogPage] = useState(1);
  const [logFrom, setLogFrom] = useState("");
  const [logTo, setLogTo] = useState("");
  const [logsLoading, setLogsLoading] = useState(false);
  const [targetStats, setTargetStats] = useState<TargetStatsResponse | null>(null);
  const [statsRange, setStatsRange] = useState<StatsRange>("24h");
  const [statsLoading, setStatsLoading] = useState(false);

  const [channels, setChannels] = useState<AlertChannel[]>([]);
  const [alertDraft, setAlertDraft] = useState<AlertDraft>(emptyAlertDraft);
  const [editingChannelId, setEditingChannelId] = useState<string | null>(null);

  const [telegramLink, setTelegramLink] = useState<TelegramLinkTokenResponse | null>(null);
  const [authMode, setAuthMode] = useState<AuthMode>("login");
  const [authEmail, setAuthEmail] = useState("");
  const [authPassword, setAuthPassword] = useState("");
  const [authBusy, setAuthBusy] = useState(false);

  const commitTokens = useCallback((nextTokens: TokenPair) => {
    saveTokens(nextTokens);
    setTokens(nextTokens);
  }, []);

  const endSession = useCallback(() => {
    clearTokens();
    setTokens(null);
    setUser(null);
    setTargets(initialTargetPage);
    setChannels([]);
    setSelectedTargetId(null);
    setLogs(initialLogsPage);
    setTargetStats(null);
    setTelegramLink(null);
    setView("overview");
  }, []);

  const showError = useCallback((error: unknown) => {
    setNotice({ type: "error", text: messageFromError(error) });
  }, []);

  const authedRequest = useCallback(
    async <T,>(path: string, options: RequestInit = {}): Promise<T> => {
      if (!tokens) {
        throw new Error("Authentication required");
      }

      try {
        return await apiRequest<T>(path, {
          ...options,
          token: tokens.access_token
        });
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          try {
            const refreshed = await apiRequest<TokenPair>("/auth/refresh", {
              method: "POST",
              body: JSON.stringify({ refresh_token: tokens.refresh_token })
            });
            commitTokens(refreshed);
            return await apiRequest<T>(path, {
              ...options,
              token: refreshed.access_token
            });
          } catch (refreshError) {
            endSession();
            throw refreshError;
          }
        }
        throw error;
      }
    },
    [commitTokens, endSession, tokens]
  );

  const fetchHealth = useCallback(async () => {
    try {
      const nextHealth = await apiRequest<Health>("/health");
      setHealth(nextHealth);
      setHealthError(false);
    } catch {
      setHealth(null);
      setHealthError(true);
    }
  }, []);

  const fetchTargets = useCallback(
    async (page = 1) => {
      const nextTargets = await authedRequest<TargetListResponse>(
        `/targets?page=${page}&page_size=20`
      );
      setTargets(nextTargets);
      setTargetPage(nextTargets.page);
      if (selectedTargetId && !nextTargets.items.some((target) => target.id === selectedTargetId)) {
        setSelectedTargetId(null);
        setLogs(initialLogsPage);
        setTargetStats(null);
      }
    },
    [authedRequest, selectedTargetId]
  );

  const fetchChannels = useCallback(async () => {
    const nextChannels = await authedRequest<AlertChannel[]>("/alert-channels");
    setChannels(nextChannels);
  }, [authedRequest]);

  const fetchMe = useCallback(async () => {
    const nextUser = await authedRequest<User>("/me");
    setUser(nextUser);
  }, [authedRequest]);

  const fetchLogs = useCallback(
    async (targetId = selectedTargetId, page = logPage) => {
      if (!targetId) {
        return;
      }
      setLogsLoading(true);
      try {
        const params = new URLSearchParams({
          page: String(page),
          page_size: "20"
        });
        if (logFrom) {
          params.set("from", new Date(logFrom).toISOString());
        }
        if (logTo) {
          params.set("to", new Date(logTo).toISOString());
        }
        const nextLogs = await authedRequest<TargetLogListResponse>(
          `/targets/${targetId}/logs?${params.toString()}`
        );
        setLogs(nextLogs);
        setLogPage(nextLogs.page);
      } finally {
        setLogsLoading(false);
      }
    },
    [authedRequest, logFrom, logPage, logTo, selectedTargetId]
  );

  const fetchTargetStats = useCallback(
    async (targetId = selectedTargetId, range = statsRange) => {
      if (!targetId) {
        return;
      }
      setStatsLoading(true);
      try {
        const nextStats = await authedRequest<TargetStatsResponse>(
          `/targets/${targetId}/stats?range=${encodeURIComponent(range)}`
        );
        setTargetStats(nextStats);
      } finally {
        setStatsLoading(false);
      }
    },
    [authedRequest, selectedTargetId, statsRange]
  );

  const loadDashboard = useCallback(async () => {
    if (!tokens) {
      setBooting(false);
      return;
    }
    setBooting(true);
    try {
      await Promise.all([fetchMe(), fetchTargets(1), fetchChannels(), fetchHealth()]);
    } catch (error) {
      showError(error);
    } finally {
      setBooting(false);
    }
  }, [fetchChannels, fetchHealth, fetchMe, fetchTargets, showError, tokens]);

  useEffect(() => {
    void fetchHealth();
  }, [fetchHealth]);

  useEffect(() => {
    void loadDashboard();
  }, [loadDashboard]);

  useEffect(() => {
    if (!selectedTargetId) {
      setTargetStats(null);
      return;
    }
    void fetchLogs(selectedTargetId, 1).catch(showError);
    void fetchTargetStats(selectedTargetId, statsRange).catch(showError);
  }, [fetchLogs, fetchTargetStats, selectedTargetId, showError, statsRange]);

  useEffect(() => {
    if (!notice) {
      return;
    }
    const timeout = window.setTimeout(() => setNotice(null), 4500);
    return () => window.clearTimeout(timeout);
  }, [notice]);

  const stats = useMemo(() => {
    const up = targets.items.filter((target) => target.status === "up").length;
    const down = targets.items.filter((target) => target.status === "down").length;
    const unknown = targets.items.filter((target) => target.status === "unknown").length;
    const enabled = targets.items.filter((target) => target.enabled).length;
    return { up, down, unknown, enabled };
  }, [targets.items]);

  const selectedTarget = targets.items.find((target) => target.id === selectedTargetId) || null;
  const targetPageCount = Math.max(1, Math.ceil(targets.total / targets.page_size));
  const logPageCount = Math.max(1, Math.ceil(logs.total / logs.page_size));

  async function handleAuthSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setAuthBusy(true);
    try {
      const nextTokens = await apiRequest<TokenPair>(
        authMode === "login" ? "/auth/login" : "/auth/register",
        {
          method: "POST",
          body: JSON.stringify({
            email: authEmail,
            password: authPassword
          })
        }
      );
      commitTokens(nextTokens);
      setNotice({ type: "success", text: authMode === "login" ? "Signed in" : "Account created" });
      setAuthPassword("");
    } catch (error) {
      showError(error);
    } finally {
      setAuthBusy(false);
    }
  }

  async function handleLogout() {
    if (tokens?.refresh_token) {
      try {
        await apiRequest<void>("/auth/logout", {
          method: "POST",
          body: JSON.stringify({ refresh_token: tokens.refresh_token })
        });
      } catch {
        // Logout still clears local credentials if the server session is already gone.
      }
    }
    endSession();
  }

  async function handleTargetSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    try {
      const payload = {
        name: targetDraft.name,
        url: targetDraft.url,
        interval: Number(targetDraft.interval),
        enabled: targetDraft.enabled
      };

      if (editingTargetId) {
        await authedRequest<Target>(`/targets/${editingTargetId}`, {
          method: "PATCH",
          body: JSON.stringify(payload)
        });
        setNotice({ type: "success", text: "Target updated" });
      } else {
        await authedRequest<Target>("/targets", {
          method: "POST",
          body: JSON.stringify(payload)
        });
        setNotice({ type: "success", text: "Target created" });
      }

      setTargetDraft(emptyTargetDraft);
      setEditingTargetId(null);
      await fetchTargets(targetPage);
    } catch (error) {
      showError(error);
    } finally {
      setBusy(false);
    }
  }

  async function handleDeleteTarget(target: Target) {
    if (!window.confirm(`Delete ${displayTargetName(target)}?`)) {
      return;
    }
    setBusy(true);
    try {
      await authedRequest<void>(`/targets/${target.id}`, { method: "DELETE" });
      setNotice({ type: "success", text: "Target deleted" });
      if (selectedTargetId === target.id) {
        setSelectedTargetId(null);
        setLogs(initialLogsPage);
        setTargetStats(null);
      }
      await fetchTargets(targetPage);
    } catch (error) {
      showError(error);
    } finally {
      setBusy(false);
    }
  }

  function startTargetEdit(target: Target) {
    setEditingTargetId(target.id);
    setTargetDraft({
      name: target.name || "",
      url: target.url,
      interval: String(target.interval),
      enabled: target.enabled
    });
    setView("targets");
  }

  async function handleAlertSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    try {
      if (editingChannelId) {
        await authedRequest<AlertChannel>(`/alert-channels/${editingChannelId}`, {
          method: "PATCH",
          body: JSON.stringify(alertDraft)
        });
        setNotice({ type: "success", text: "Alert channel updated" });
      } else {
        await authedRequest<AlertChannel>("/alert-channels", {
          method: "POST",
          body: JSON.stringify(alertDraft)
        });
        setNotice({ type: "success", text: "Alert channel created" });
      }
      setAlertDraft(emptyAlertDraft);
      setEditingChannelId(null);
      await fetchChannels();
    } catch (error) {
      showError(error);
    } finally {
      setBusy(false);
    }
  }

  async function handleDeleteChannel(channel: AlertChannel) {
    if (!window.confirm(`Delete ${channel.type} channel?`)) {
      return;
    }
    setBusy(true);
    try {
      await authedRequest<void>(`/alert-channels/${channel.id}`, { method: "DELETE" });
      setNotice({ type: "success", text: "Alert channel deleted" });
      await fetchChannels();
    } catch (error) {
      showError(error);
    } finally {
      setBusy(false);
    }
  }

  function startChannelEdit(channel: AlertChannel) {
    setEditingChannelId(channel.id);
    setAlertDraft({
      type: channel.type,
      address: channel.address,
      enabled: channel.enabled
    });
    setView("alerts");
  }

  async function generateTelegramLink() {
    setBusy(true);
    try {
      const link = await authedRequest<TelegramLinkTokenResponse>("/telegram/link-token", {
        method: "POST"
      });
      setTelegramLink(link);
      setNotice({ type: "success", text: "Telegram link created" });
    } catch (error) {
      showError(error);
    } finally {
      setBusy(false);
    }
  }

  async function copyTelegramLink() {
    if (!telegramLink) {
      return;
    }
    try {
      await navigator.clipboard.writeText(telegramLink.link_url);
      setNotice({ type: "success", text: "Link copied" });
    } catch (error) {
      showError(error);
    }
  }

  function useConnectedTelegramChat() {
    if (!user?.user_tg) {
      return;
    }
    setAlertDraft({
      type: "telegram",
      address: user.user_tg,
      enabled: true
    });
    setEditingChannelId(null);
    setView("alerts");
  }

  if (!tokens) {
    return (
      <main className="auth-shell">
        <section className="auth-panel" aria-label="Authentication">
          <div className="brand-row">
            <div className="brand-mark">PM</div>
            <div>
              <h1>PingMe</h1>
              <p>Uptime monitoring dashboard</p>
            </div>
          </div>

          <div className="segmented" role="tablist" aria-label="Authentication mode">
            <button
              type="button"
              className={authMode === "login" ? "active" : ""}
              onClick={() => setAuthMode("login")}
            >
              Login
            </button>
            <button
              type="button"
              className={authMode === "register" ? "active" : ""}
              onClick={() => setAuthMode("register")}
            >
              Register
            </button>
          </div>

          <form className="stack-form" onSubmit={handleAuthSubmit}>
            <label>
              Email
              <input
                type="email"
                autoComplete="email"
                value={authEmail}
                onChange={(event) => setAuthEmail(event.target.value)}
                required
              />
            </label>
            <label>
              Password
              <input
                type="password"
                autoComplete={authMode === "login" ? "current-password" : "new-password"}
                value={authPassword}
                onChange={(event) => setAuthPassword(event.target.value)}
                minLength={8}
                required
              />
            </label>
            <button className="primary-button" type="submit" disabled={authBusy}>
              <CheckIcon />
              {authBusy ? "Please wait" : authMode === "login" ? "Login" : "Create account"}
            </button>
          </form>
        </section>
        {notice && <Toast notice={notice} />}
      </main>
    );
  }

  return (
    <main className="app-shell">
      <aside className="sidebar" aria-label="Main navigation">
        <div className="brand-row compact">
          <div className="brand-mark">PM</div>
          <div>
            <h1>PingMe</h1>
            <p>{user?.email || "Signed in"}</p>
          </div>
        </div>

        <nav className="nav-stack">
          <NavButton label="Overview" active={view === "overview"} onClick={() => setView("overview")} />
          <NavButton label="Targets" active={view === "targets"} onClick={() => setView("targets")} />
          <NavButton label="Alerts" active={view === "alerts"} onClick={() => setView("alerts")} />
          <NavButton label="Telegram" active={view === "telegram"} onClick={() => setView("telegram")} />
        </nav>

        <div className="sidebar-footer">
          <HealthPill health={health} failed={healthError} />
          <button className="ghost-button" type="button" onClick={handleLogout}>
            <LogoutIcon />
            Logout
          </button>
        </div>
      </aside>

      <section className="content-shell">
        <header className="topbar">
          <div>
            <p className="eyebrow">{view}</p>
            <h2>{titleForView(view)}</h2>
          </div>
          <div className="topbar-actions">
            <button
              className="icon-button"
              type="button"
              title="Refresh data"
              aria-label="Refresh data"
              onClick={() => void loadDashboard().catch(showError)}
              disabled={booting}
            >
              <RefreshIcon />
            </button>
          </div>
        </header>

        {booting ? (
          <div className="panel loading-panel">Loading dashboard</div>
        ) : (
          <>
            {view === "overview" && (
              <section className="view-grid">
                <div className="metric-grid">
                  <Metric label="Targets" value={targets.total} />
                  <Metric label="Enabled" value={stats.enabled} />
                  <Metric label="Up" value={stats.up} accent="good" />
                  <Metric label="Down" value={stats.down} accent="bad" />
                  <Metric label="Unknown" value={stats.unknown} accent="muted" />
                  <Metric label="Channels" value={channels.length} />
                </div>

                <div className="panel">
                  <PanelHeader
                    title="Recent Targets"
                    action={
                      <button className="small-button" type="button" onClick={() => setView("targets")}>
                        <PlusIcon />
                        Target
                      </button>
                    }
                  />
                  <TargetTable
                    targets={targets.items.slice(0, 6)}
                    onLogs={(target) => {
                      setSelectedTargetId(target.id);
                      setView("targets");
                    }}
                    onEdit={startTargetEdit}
                    onDelete={handleDeleteTarget}
                  />
                </div>

                <div className="panel">
                  <PanelHeader
                    title="Alert Channels"
                    action={
                      <button className="small-button" type="button" onClick={() => setView("alerts")}>
                        <PlusIcon />
                        Channel
                      </button>
                    }
                  />
                  <ChannelList channels={channels} onEdit={startChannelEdit} onDelete={handleDeleteChannel} />
                </div>
              </section>
            )}

            {view === "targets" && (
              <section className="split-view">
                <div className="panel">
                  <PanelHeader title={editingTargetId ? "Edit Target" : "New Target"} />
                  <form className="grid-form" onSubmit={handleTargetSubmit}>
                    <label>
                      Name
                      <input
                        value={targetDraft.name}
                        onChange={(event) =>
                          setTargetDraft((draft) => ({ ...draft, name: event.target.value }))
                        }
                        placeholder="Production API"
                      />
                    </label>
                    <label>
                      URL
                      <input
                        type="url"
                        value={targetDraft.url}
                        onChange={(event) =>
                          setTargetDraft((draft) => ({ ...draft, url: event.target.value }))
                        }
                        placeholder="https://example.com"
                        required
                      />
                    </label>
                    <label>
                      Interval, seconds
                      <input
                        type="number"
                        min={30}
                        max={3600}
                        value={targetDraft.interval}
                        onChange={(event) =>
                          setTargetDraft((draft) => ({ ...draft, interval: event.target.value }))
                        }
                        required
                      />
                    </label>
                    <label className="toggle-row">
                      <input
                        type="checkbox"
                        checked={targetDraft.enabled}
                        onChange={(event) =>
                          setTargetDraft((draft) => ({ ...draft, enabled: event.target.checked }))
                        }
                      />
                      Enabled
                    </label>
                    <div className="form-actions">
                      {editingTargetId && (
                        <button
                          className="ghost-button"
                          type="button"
                          onClick={() => {
                            setEditingTargetId(null);
                            setTargetDraft(emptyTargetDraft);
                          }}
                        >
                          Cancel
                        </button>
                      )}
                      <button className="primary-button" type="submit" disabled={busy}>
                        <CheckIcon />
                        {editingTargetId ? "Save target" : "Create target"}
                      </button>
                    </div>
                  </form>
                </div>

                <div className="panel wide-panel">
                  <PanelHeader
                    title="Targets"
                    action={
                      <button
                        className="icon-button"
                        type="button"
                        title="Refresh targets"
                        aria-label="Refresh targets"
                        onClick={() => void fetchTargets(targetPage).catch(showError)}
                      >
                        <RefreshIcon />
                      </button>
                    }
                  />
                  <TargetTable
                    targets={targets.items}
                    selectedId={selectedTargetId}
                    onLogs={(target) => setSelectedTargetId(target.id)}
                    onEdit={startTargetEdit}
                    onDelete={handleDeleteTarget}
                  />
                  <Pagination
                    page={targetPage}
                    pageCount={targetPageCount}
                    onPrev={() => void fetchTargets(Math.max(1, targetPage - 1)).catch(showError)}
                    onNext={() =>
                      void fetchTargets(Math.min(targetPageCount, targetPage + 1)).catch(showError)
                    }
                  />
                </div>

                <div className="panel wide-panel">
                  <TargetAnalyticsPanel
                    target={selectedTarget}
                    stats={targetStats}
                    loading={statsLoading}
                    range={statsRange}
                    onRangeChange={setStatsRange}
                    onRefresh={() => {
                      if (!selectedTarget) {
                        return;
                      }
                      void fetchTargetStats(selectedTarget.id, statsRange).catch(showError);
                    }}
                  />
                </div>

                <div className="panel wide-panel">
                  <PanelHeader
                    title={selectedTarget ? `Logs: ${displayTargetName(selectedTarget)}` : "Logs"}
                    action={
                      selectedTarget && (
                        <button
                          className="icon-button"
                          type="button"
                          title="Refresh logs"
                          aria-label="Refresh logs"
                          onClick={() => void fetchLogs(selectedTarget.id, logPage).catch(showError)}
                          disabled={logsLoading}
                        >
                          <RefreshIcon />
                        </button>
                      )
                    }
                  />
                  {selectedTarget ? (
                    <>
                      <div className="filters-row">
                        <label>
                          From
                          <input
                            type="datetime-local"
                            value={logFrom}
                            onChange={(event) => setLogFrom(event.target.value)}
                          />
                        </label>
                        <label>
                          To
                          <input
                            type="datetime-local"
                            value={logTo}
                            onChange={(event) => setLogTo(event.target.value)}
                          />
                        </label>
                        <button
                          className="small-button"
                          type="button"
                          onClick={() => void fetchLogs(selectedTarget.id, 1).catch(showError)}
                        >
                          <CheckIcon />
                          Apply
                        </button>
                      </div>
                      <LogsTable logs={logs.items} loading={logsLoading} />
                      <Pagination
                        page={logPage}
                        pageCount={logPageCount}
                        onPrev={() =>
                          void fetchLogs(selectedTarget.id, Math.max(1, logPage - 1)).catch(showError)
                        }
                        onNext={() =>
                          void fetchLogs(selectedTarget.id, Math.min(logPageCount, logPage + 1)).catch(showError)
                        }
                      />
                    </>
                  ) : (
                    <EmptyState title="No target selected" />
                  )}
                </div>
              </section>
            )}

            {view === "alerts" && (
              <section className="split-view">
                <div className="panel">
                  <PanelHeader title={editingChannelId ? "Edit Channel" : "New Channel"} />
                  <form className="grid-form" onSubmit={handleAlertSubmit}>
                    <label>
                      Type
                      <select
                        value={alertDraft.type}
                        onChange={(event) =>
                          setAlertDraft((draft) => ({
                            ...draft,
                            type: event.target.value as AlertChannelType,
                            address: event.target.value === "telegram" && user?.user_tg ? user.user_tg : draft.address
                          }))
                        }
                      >
                        <option value="webhook">Webhook</option>
                        <option value="telegram">Telegram</option>
                      </select>
                    </label>
                    <label>
                      Address
                      <input
                        value={alertDraft.address}
                        onChange={(event) =>
                          setAlertDraft((draft) => ({ ...draft, address: event.target.value }))
                        }
                        placeholder={alertDraft.type === "telegram" ? "Telegram chat ID" : "https://example.com/hook"}
                        required
                      />
                    </label>
                    <label className="toggle-row">
                      <input
                        type="checkbox"
                        checked={alertDraft.enabled}
                        onChange={(event) =>
                          setAlertDraft((draft) => ({ ...draft, enabled: event.target.checked }))
                        }
                      />
                      Enabled
                    </label>
                    <div className="form-actions">
                      {user?.user_tg && (
                        <button className="ghost-button" type="button" onClick={useConnectedTelegramChat}>
                          <LinkIcon />
                          Use chat
                        </button>
                      )}
                      {editingChannelId && (
                        <button
                          className="ghost-button"
                          type="button"
                          onClick={() => {
                            setEditingChannelId(null);
                            setAlertDraft(emptyAlertDraft);
                          }}
                        >
                          Cancel
                        </button>
                      )}
                      <button className="primary-button" type="submit" disabled={busy}>
                        <CheckIcon />
                        {editingChannelId ? "Save channel" : "Create channel"}
                      </button>
                    </div>
                  </form>
                </div>

                <div className="panel wide-panel">
                  <PanelHeader
                    title="Alert Channels"
                    action={
                      <button
                        className="icon-button"
                        type="button"
                        title="Refresh channels"
                        aria-label="Refresh channels"
                        onClick={() => void fetchChannels().catch(showError)}
                      >
                        <RefreshIcon />
                      </button>
                    }
                  />
                  <ChannelList channels={channels} onEdit={startChannelEdit} onDelete={handleDeleteChannel} />
                </div>
              </section>
            )}

            {view === "telegram" && (
              <section className="view-grid">
                <div className="panel telegram-panel">
                  <PanelHeader title="Telegram" />
                  <div className="telegram-status">
                    <div>
                      <span className="label">Connected chat</span>
                      <strong>{user?.user_tg || "Not connected"}</strong>
                    </div>
                    {user?.user_tg && (
                      <button className="small-button" type="button" onClick={useConnectedTelegramChat}>
                        <PlusIcon />
                        Channel
                      </button>
                    )}
                  </div>

                  <div className="telegram-actions">
                    <button className="primary-button" type="button" onClick={generateTelegramLink} disabled={busy}>
                      <LinkIcon />
                      Generate link
                    </button>
                    <button className="ghost-button" type="button" onClick={() => void fetchMe().catch(showError)}>
                      <RefreshIcon />
                      Refresh user
                    </button>
                  </div>

                  {telegramLink && (
                    <div className="link-box">
                      <span>{telegramLink.link_url}</span>
                      <div className="link-actions">
                        <button
                          className="icon-button"
                          type="button"
                          title="Copy Telegram link"
                          aria-label="Copy Telegram link"
                          onClick={copyTelegramLink}
                        >
                          <LinkIcon />
                        </button>
                        <a
                          className="icon-button"
                          title="Open Telegram link"
                          aria-label="Open Telegram link"
                          href={telegramLink.link_url}
                          target="_blank"
                          rel="noreferrer"
                        >
                          <ExternalIcon />
                        </a>
                      </div>
                      <small>Expires {formatDate(telegramLink.expires_at)}</small>
                    </div>
                  )}
                </div>
              </section>
            )}
          </>
        )}
      </section>

      {notice && <Toast notice={notice} />}
    </main>
  );
}

function NavButton({
  label,
  active,
  onClick
}: {
  label: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button className={active ? "nav-button active" : "nav-button"} type="button" onClick={onClick}>
      {label}
    </button>
  );
}

function PanelHeader({
  title,
  action
}: {
  title: string;
  action?: ReactNode;
}) {
  return (
    <div className="panel-header">
      <h3>{title}</h3>
      {action && <div className="panel-action">{action}</div>}
    </div>
  );
}

function Metric({
  label,
  value,
  accent
}: {
  label: string;
  value: number | string;
  accent?: "good" | "bad" | "muted";
}) {
  return (
    <div className={`metric ${accent || ""}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function TargetAnalyticsPanel({
  target,
  stats,
  loading,
  range,
  onRangeChange,
  onRefresh
}: {
  target: Target | null;
  stats: TargetStatsResponse | null;
  loading: boolean;
  range: StatsRange;
  onRangeChange: (range: StatsRange) => void;
  onRefresh: () => void;
}) {
  const action =
    target &&
    (
      <div className="analytics-toolbar">
        <div className="range-switch" role="tablist" aria-label="Analytics range">
          <button
            type="button"
            className={range === "24h" ? "range-button active" : "range-button"}
            onClick={() => onRangeChange("24h")}
          >
            24h
          </button>
          <button
            type="button"
            className={range === "7d" ? "range-button active" : "range-button"}
            onClick={() => onRangeChange("7d")}
          >
            7d
          </button>
        </div>
        <button
          className="icon-button"
          type="button"
          title="Refresh analytics"
          aria-label="Refresh analytics"
          onClick={onRefresh}
          disabled={loading}
        >
          <RefreshIcon />
        </button>
      </div>
    );

  return (
    <>
      <PanelHeader
        title={target ? `Analytics: ${displayTargetName(target)}` : "Analytics"}
        action={action}
      />
      {!target ? (
        <EmptyState title="Select a target to see analytics" />
      ) : !stats ? (
        <EmptyState title={loading ? "Loading analytics" : "No analytics yet"} />
      ) : (
        <div className="analytics-shell">
          <div className="analytics-metrics">
            <Metric
              label="Uptime"
              value={`${formatPercent(stats.uptime_percent)}%`}
              accent={accentForUptime(stats.uptime_percent)}
            />
            <Metric label="Avg latency" value={`${Math.round(stats.avg_response_ms)} ms`} />
            <Metric label="Checks" value={stats.total_checks} />
            <Metric label="Failures" value={stats.failed_checks} accent={stats.failed_checks > 0 ? "bad" : "good"} />
          </div>
          <StatsTimeline stats={stats} />
          <div className="analytics-footnote">
            Range {formatDate(stats.from)} to {formatDate(stats.to)}
          </div>
        </div>
      )}
    </>
  );
}

function TargetTable({
  targets,
  selectedId,
  onLogs,
  onEdit,
  onDelete
}: {
  targets: Target[];
  selectedId?: string | null;
  onLogs: (target: Target) => void;
  onEdit: (target: Target) => void;
  onDelete: (target: Target) => void;
}) {
  if (targets.length === 0) {
    return <EmptyState title="No targets" />;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Status</th>
            <th>Interval</th>
            <th>Last Check</th>
            <th className="actions-cell">Actions</th>
          </tr>
        </thead>
        <tbody>
          {targets.map((target) => (
            <tr key={target.id} className={target.id === selectedId ? "selected-row" : ""}>
              <td>
                <div className="target-cell">
                  <strong>{displayTargetName(target)}</strong>
                  <a href={target.url} target="_blank" rel="noreferrer">
                    {target.url}
                  </a>
                </div>
              </td>
              <td>
                <StatusBadge status={target.status} enabled={target.enabled} />
              </td>
              <td>{target.interval}s</td>
              <td>{target.last_checked_at ? formatDate(target.last_checked_at) : "Never"}</td>
              <td className="actions-cell">
                <button
                  className="icon-button"
                  type="button"
                  title="Show logs"
                  aria-label="Show logs"
                  onClick={() => onLogs(target)}
                >
                  <LogsIcon />
                </button>
                <button
                  className="icon-button"
                  type="button"
                  title="Edit target"
                  aria-label="Edit target"
                  onClick={() => onEdit(target)}
                >
                  <EditIcon />
                </button>
                <button
                  className="icon-button danger"
                  type="button"
                  title="Delete target"
                  aria-label="Delete target"
                  onClick={() => onDelete(target)}
                >
                  <TrashIcon />
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function LogsTable({ logs, loading }: { logs: TargetLogListResponse["items"]; loading: boolean }) {
  if (loading) {
    return <EmptyState title="Loading logs" />;
  }

  if (logs.length === 0) {
    return <EmptyState title="No logs" />;
  }

  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Checked At</th>
            <th>Result</th>
            <th>Status</th>
            <th>Latency</th>
            <th>Error</th>
          </tr>
        </thead>
        <tbody>
          {logs.map((log) => (
            <tr key={log.id}>
              <td>{formatDate(log.checked_at)}</td>
              <td>
                <span className={log.success ? "log-result success" : "log-result failure"}>
                  {log.success ? "Success" : "Failure"}
                </span>
              </td>
              <td>{log.status_code || "-"}</td>
              <td>{log.response_time_ms} ms</td>
              <td>{log.error_message || "-"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function StatsTimeline({ stats }: { stats: TargetStatsResponse }) {
  if (stats.timeline.length === 0) {
    return <EmptyState title="No checks recorded for this range" />;
  }

  const peakResponseTime = Math.max(
    ...stats.timeline.map((point) => Math.max(point.response_time_ms, 1)),
    1
  );

  return (
    <div className="timeline-card">
      <div className="timeline-grid" role="img" aria-label="Target response timeline">
        {stats.timeline.map((point) => {
          const height = Math.max(18, Math.round((Math.max(point.response_time_ms, 1) / peakResponseTime) * 92));
          const label = `${formatDate(point.timestamp)} - ${point.success ? "Success" : "Failure"} - ${point.response_time_ms} ms`;
          return (
            <span
              key={`${point.timestamp}-${point.response_time_ms}-${point.success}`}
              className={point.success ? "timeline-bar success" : "timeline-bar failure"}
              style={{ height: `${height}px` }}
              title={label}
            />
          );
        })}
      </div>
      <div className="timeline-meta">
        <span>{stats.timeline.length} checks shown</span>
        <span>{stats.failed_checks === 0 ? "No failures in range" : `${stats.failed_checks} failed checks`}</span>
      </div>
    </div>
  );
}

function ChannelList({
  channels,
  onEdit,
  onDelete
}: {
  channels: AlertChannel[];
  onEdit: (channel: AlertChannel) => void;
  onDelete: (channel: AlertChannel) => void;
}) {
  if (channels.length === 0) {
    return <EmptyState title="No alert channels" />;
  }

  return (
    <div className="channel-list">
      {channels.map((channel) => (
        <article className="channel-item" key={channel.id}>
          <div>
            <div className="channel-title">
              <span className="type-pill">{channel.type}</span>
              <StatusDot enabled={channel.enabled} />
            </div>
            <p>{channel.address}</p>
            <small>{formatDate(channel.created_at)}</small>
          </div>
          <div className="row-actions">
            <button
              className="icon-button"
              type="button"
              title="Edit channel"
              aria-label="Edit channel"
              onClick={() => onEdit(channel)}
            >
              <EditIcon />
            </button>
            <button
              className="icon-button danger"
              type="button"
              title="Delete channel"
              aria-label="Delete channel"
              onClick={() => onDelete(channel)}
            >
              <TrashIcon />
            </button>
          </div>
        </article>
      ))}
    </div>
  );
}

function Pagination({
  page,
  pageCount,
  onPrev,
  onNext
}: {
  page: number;
  pageCount: number;
  onPrev: () => void;
  onNext: () => void;
}) {
  return (
    <div className="pagination">
      <button className="ghost-button" type="button" onClick={onPrev} disabled={page <= 1}>
        Prev
      </button>
      <span>
        {page} / {pageCount}
      </span>
      <button className="ghost-button" type="button" onClick={onNext} disabled={page >= pageCount}>
        Next
      </button>
    </div>
  );
}

function StatusBadge({ status, enabled }: { status: Target["status"]; enabled: boolean }) {
  if (!enabled) {
    return <span className="status-badge disabled">disabled</span>;
  }
  return <span className={`status-badge ${status}`}>{status}</span>;
}

function StatusDot({ enabled }: { enabled: boolean }) {
  return <span className={enabled ? "status-dot enabled" : "status-dot disabled"} />;
}

function HealthPill({ health, failed }: { health: Health | null; failed: boolean }) {
  const ok = !failed && health?.status === "ok";
  return <span className={ok ? "health-pill ok" : "health-pill bad"}>{ok ? "API online" : "API offline"}</span>;
}

function EmptyState({ title }: { title: string }) {
  return <div className="empty-state">{title}</div>;
}

function Toast({ notice }: { notice: Exclude<Notice, null> }) {
  return <div className={`toast ${notice.type}`}>{notice.text}</div>;
}

function titleForView(view: View) {
  switch (view) {
    case "overview":
      return "Dashboard";
    case "targets":
      return "Targets";
    case "alerts":
      return "Alert Channels";
    case "telegram":
      return "Telegram";
  }
}

function displayTargetName(target: Target) {
  return target.name || target.url;
}

function accentForUptime(value: number): "good" | "bad" | "muted" {
  if (value >= 99) {
    return "good";
  }
  if (value >= 95) {
    return "muted";
  }
  return "bad";
}

function formatPercent(value: number) {
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: value % 1 === 0 ? 0 : 1,
    maximumFractionDigits: 1
  }).format(value);
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit"
  }).format(new Date(value));
}

function messageFromError(error: unknown) {
  if (error instanceof ApiError) {
    const fields = error.payload?.fields ? Object.values(error.payload.fields).join(", ") : "";
    return fields || error.payload?.message || error.payload?.error || error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "Unexpected error";
}

export default App;
