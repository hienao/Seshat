import { Badge } from "@appica/ui-react/badge";
import { Input } from "@appica/ui-react/input";
import {
  AlertCircle,
  Download,
  FileText,
  Refresh,
  Trash,
} from "@appica/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "@/api/services";
import { AppButton } from "@/components/common/app-button";
import { EmptyState, ErrorState, Message } from "@/components/common/feedback";
import { PageHeader } from "@/components/common/page-header";
import { Panel } from "@/components/common/panel";
import { errorMessage } from "@/lib/error-message";
import { formatDateTime } from "@/i18n/format";

type Filters = {
  startAt: string;
  endAt: string;
  method: string;
  route: string;
  statusGroup: string;
  requestId: string;
  keyword: string;
};

const inputClass =
  "h-10 w-full rounded-lg border border-neutral-300 bg-white px-3 text-sm dark:border-neutral-700 dark:bg-neutral-900";

function initialFilters(): Filters {
  const end = new Date();
  const start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
  return {
    startAt: toDateTimeLocal(start),
    endAt: toDateTimeLocal(end),
    method: "",
    route: "",
    statusGroup: "",
    requestId: "",
    keyword: "",
  };
}

function toDateTimeLocal(date: Date) {
  const offset = date.getTimezoneOffset() * 60000;
  return new Date(date.getTime() - offset).toISOString().slice(0, 16);
}
function toApiTime(value: string) {
  return value ? new Date(value).toISOString() : undefined;
}
function apiFilters(filters: Filters) {
  return {
    startAt: toApiTime(filters.startAt),
    endAt: toApiTime(filters.endAt),
    method: filters.method,
    route: filters.route,
    statusGroup: filters.statusGroup,
    requestId: filters.requestId,
    keyword: filters.keyword,
  };
}
function formatDate(value: string) {
  return value ? formatDateTime(value) : "-";
}
function formatHeaders(value: string) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}
function statusVariant(
  status: number,
): "soft" | "outline" | "error" | "success" {
  if (status >= 500) return "error";
  if (status >= 400) return "outline";
  if (status < 300) return "success";
  return "soft";
}

export function AdminLogsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<Filters>(initialFilters);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [clearText, setClearText] = useState("");
  const current = apiFilters(filters);
  const logs = useQuery({
    queryKey: ["admin", "api-logs", current],
    queryFn: () => api.apiLogs(current),
  });
  const summary = useQuery({
    queryKey: ["admin", "api-log-summary", current],
    queryFn: () => api.apiLogSummary(current),
  });
  const detail = useQuery({
    queryKey: ["admin", "api-log", selectedId],
    queryFn: () => api.apiLog(selectedId as number),
    enabled: selectedId !== null,
  });
  const clear = useMutation({
    mutationFn: () =>
      api.clearApiLogs({ ...current, confirmation: "CLEAR_LOGS" }),
    onSuccess: () => {
      setClearText("");
      void queryClient.invalidateQueries({ queryKey: ["admin", "api-logs"] });
      void queryClient.invalidateQueries({
        queryKey: ["admin", "api-log-summary"],
      });
    },
  });
  const exportLogs = useMutation({
    mutationFn: (format: "csv" | "jsonl") =>
      api.exportApiLogs({ ...current, format }),
    onSuccess: ({ blob, filename }) => {
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = filename.replaceAll('"', "");
      anchor.click();
      URL.revokeObjectURL(url);
    },
  });

  function update<K extends keyof Filters>(key: K, value: Filters[K]) {
    setFilters((previous) => ({ ...previous, [key]: value }));
  }
  function clearLogs() {
    if (clearText !== "CLEAR_LOGS") return;
    if (window.confirm(t("admin.logs.clearApplicationDescription")))
      clear.mutate();
  }

  return (
    <div className="mx-auto max-w-7xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader
        eyebrow={t("admin.logs.eyebrow")}
        title={t("admin.logs.apiTitle")}
        description={t("admin.logs.apiDescription")}
        action={
          <AppButton
            size="sm"
            variant="outline"
            disabled={logs.isFetching}
            onClick={() => void logs.refetch()}
          >
            <Refresh size={16} />
            {t("common.actions.refresh")}
          </AppButton>
        }
      />
      {(clear.error || exportLogs.error) && (
        <Message
          variant="error"
          title={errorMessage(
            clear.error || exportLogs.error,
            t("admin.logs.operationFailed"),
          )}
        />
      )}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
        {[
          [t("admin.logs.totalRequests"), summary.data?.total ?? 0],
          [
            t("admin.logs.successfulRequests"),
            summary.data?.success_count ?? 0,
          ],
          [t("admin.logs.clientErrors"), summary.data?.client_errors ?? 0],
          [t("admin.logs.serverErrors"), summary.data?.server_errors ?? 0],
          [
            t("admin.logs.averageDuration"),
            `${Math.round(summary.data?.average_ms ?? 0)} ms`,
          ],
        ].map(([label, value]) => (
          <div key={label} className="app-panel p-5">
            <p className="text-xs font-medium text-neutral-500">{label}</p>
            <p className="mt-2 text-2xl font-bold tracking-tight">{value}</p>
          </div>
        ))}
      </div>
      <Panel title={t("admin.logs.filters")} icon={<FileText size={20} />}>
        <div className="grid gap-4 md:grid-cols-3 lg:grid-cols-4">
          <label className="space-y-2 text-sm font-medium">
            <span>{t("admin.logs.startAt")}</span>
            <input
              className={inputClass}
              type="datetime-local"
              value={filters.startAt}
              onChange={(event) => update("startAt", event.target.value)}
            />
          </label>
          <label className="space-y-2 text-sm font-medium">
            <span>{t("admin.logs.endAt")}</span>
            <input
              className={inputClass}
              type="datetime-local"
              value={filters.endAt}
              onChange={(event) => update("endAt", event.target.value)}
            />
          </label>
          <label className="space-y-2 text-sm font-medium">
            <span>Method</span>
            <select
              className={inputClass}
              value={filters.method}
              onChange={(event) => update("method", event.target.value)}
            >
              <option value="">{t("admin.logs.all")}</option>
              {["GET", "POST", "PUT", "DELETE", "PATCH"].map((method) => (
                <option key={method}>{method}</option>
              ))}
            </select>
          </label>
          <label className="space-y-2 text-sm font-medium">
            <span>{t("admin.logs.status")}</span>
            <select
              className={inputClass}
              value={filters.statusGroup}
              onChange={(event) => update("statusGroup", event.target.value)}
            >
              <option value="">{t("admin.logs.all")}</option>
              <option value="2xx">2xx</option>
              <option value="3xx">3xx</option>
              <option value="4xx">4xx</option>
              <option value="5xx">5xx</option>
            </select>
          </label>
          <label className="space-y-2 text-sm font-medium md:col-span-2">
            <span>{t("admin.logs.route")}</span>
            <Input
              value={filters.route}
              onChange={(event) => update("route", event.target.value)}
              placeholder="/api/webhooks/events"
            />
          </label>
          <label className="space-y-2 text-sm font-medium">
            <span>Request ID</span>
            <Input
              value={filters.requestId}
              onChange={(event) => update("requestId", event.target.value)}
              placeholder={t('admin.logs.requestId')}
            />
          </label>
          <label className="space-y-2 text-sm font-medium">
            <span>{t("admin.logs.keyword")}</span>
            <Input
              value={filters.keyword}
              onChange={(event) => update("keyword", event.target.value)}
            />
          </label>
        </div>
        <div className="mt-5 flex flex-wrap gap-2">
          <AppButton
            size="sm"
            variant="outline"
            disabled={exportLogs.isPending}
            onClick={() => exportLogs.mutate("csv")}
          >
            <Download size={16} />
            {t("admin.logs.exportCsv")}
          </AppButton>
          <AppButton
            size="sm"
            variant="outline"
            disabled={exportLogs.isPending}
            onClick={() => exportLogs.mutate("jsonl")}
          >
            <Download size={16} />
            {t("admin.logs.exportJsonl")}
          </AppButton>
        </div>
      </Panel>
      <Panel
        title={t("admin.logs.records")}
        description={
          logs.data
            ? t("admin.logs.total", { count: logs.data.total })
            : t("common.states.loading")
        }
      >
        {logs.isPending ? (
          <div className="py-12 text-center text-sm text-neutral-500">
            {t("admin.logs.loading")}
          </div>
        ) : logs.error ? (
          <ErrorState
            message={errorMessage(logs.error)}
            onRetry={() => void logs.refetch()}
          />
        ) : !logs.data?.items.length ? (
          <EmptyState
            title={t("admin.logs.empty")}
            description={t("admin.logs.emptyDescription")}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[920px] text-left text-sm">
              <thead>
                <tr className="border-b border-neutral-200 text-xs text-neutral-500 dark:border-neutral-800">
                  <th className="px-2 py-3">{t("admin.logs.time")}</th>
                  <th className="px-2 py-3">{t("admin.logs.status")}</th>
                  <th className="px-2 py-3">{t("admin.logs.method")}</th>
                  <th className="px-2 py-3">{t("admin.logs.route")}</th>
                  <th className="px-2 py-3">{t("admin.logs.duration")}</th>
                  <th className="px-2 py-3">{t("admin.logs.userApp")}</th>
                  <th className="px-2 py-3">Request ID</th>
                </tr>
              </thead>
              <tbody>
                {logs.data.items.map((item) => (
                  <tr
                    key={item.id}
                    className="cursor-pointer border-b border-neutral-100 transition hover:bg-neutral-50 dark:border-neutral-900 dark:hover:bg-neutral-900"
                    onClick={() => setSelectedId(item.id)}
                  >
                    <td className="whitespace-nowrap px-2 py-3 text-xs text-neutral-500">
                      {formatDate(item.occurred_at)}
                    </td>
                    <td className="px-2 py-3">
                      <Badge
                        variant={statusVariant(item.status_code)}
                        size="sm"
                      >
                        {item.status_code}
                      </Badge>
                    </td>
                    <td className="px-2 py-3 font-mono text-xs font-semibold">
                      {item.method}
                    </td>
                    <td className="max-w-[280px] truncate px-2 py-3 font-mono text-xs">
                      {item.route}
                    </td>
                    <td className="whitespace-nowrap px-2 py-3 text-xs">
                      {item.latency_ms} ms
                    </td>
                    <td className="px-2 py-3 text-xs">
                      {item.app_code
                        ? `${item.app_code}${item.integration_id ? ` / #${item.integration_id}` : ""}`
                        : item.username || item.client_ip || "-"}
                    </td>
                    <td className="max-w-[160px] truncate px-2 py-3 font-mono text-xs text-neutral-500">
                      {item.request_id}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {logs.data?.has_more && (
          <div className="mt-4 text-center text-xs text-neutral-500">
            {t("admin.logs.more")}
          </div>
        )}
      </Panel>
      <Panel
        title={t("admin.logs.clear")}
        description={t("admin.logs.clearDescription")}
        icon={<Trash size={20} />}
      >
        <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
          <label className="max-w-md flex-1 space-y-2 text-sm font-medium">
            <span>CLEAR_LOGS</span>
            <Input
              value={clearText}
              onChange={(event) => setClearText(event.target.value)}
              placeholder="CLEAR_LOGS"
            />
          </label>
          <AppButton
            variant="destructive"
            disabled={clearText !== "CLEAR_LOGS" || clear.isPending}
            onClick={clearLogs}
          >
            <Trash size={16} />
            {t("admin.logs.clearRange")}
          </AppButton>
        </div>
      </Panel>
      {selectedId !== null && (
        <div
          className="fixed inset-0 z-50 flex justify-end bg-black/30"
          onClick={() => setSelectedId(null)}
        >
          <aside
            className="h-full w-full max-w-2xl overflow-y-auto bg-white p-6 shadow-2xl dark:bg-neutral-950"
            onClick={(event) => event.stopPropagation()}
          >
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs font-semibold uppercase tracking-widest text-emerald-700">
                  Log Detail
                </p>
                <h2 className="mt-1 text-xl font-bold">{t('admin.logs.detail')}</h2>
              </div>
              <AppButton
                size="sm"
                variant="outline"
                onClick={() => setSelectedId(null)}
              >
                {t('common.actions.close')}
              </AppButton>
            </div>
            {detail.isPending ? (
              <p className="mt-8 text-sm text-neutral-500">{t('admin.logs.loadingDetail')}</p>
            ) : detail.error ? (
              <div className="mt-8">
                <Message variant="error" title={errorMessage(detail.error)} />
              </div>
            ) : detail.data ? (
              <div className="mt-6 space-y-5">
                <div className="grid grid-cols-2 gap-3 text-sm">
                  {[
                    [t("admin.logs.time"), formatDate(detail.data.occurred_at)],
                    [t("admin.logs.status"), String(detail.data.status_code)],
                    [t("admin.logs.duration"), `${detail.data.latency_ms} ms`],
                    ["Request ID", detail.data.request_id],
                    [t("admin.logs.user"), detail.data.username || "-"],
                    ["App", detail.data.app_code || "-"],
                    [
                      t("admin.logs.integration"),
                      detail.data.integration_id
                        ? `#${detail.data.integration_id}`
                        : "-",
                    ],
                    [
                      t("admin.logs.eventId"),
                      detail.data.event_id ? `#${detail.data.event_id}` : "-",
                    ],
                    [t("admin.logs.requestSize"), `${detail.data.request_bytes} bytes`],
                    [t("admin.logs.responseSize"), `${detail.data.response_bytes} bytes`],
                    [t("admin.logs.errorCode"), detail.data.error_code || "-"],
                    ["IP", detail.data.client_ip || "-"],
                    ["User-Agent", detail.data.user_agent || "-"],
                  ].map(([label, value]) => (
                    <div
                      key={label}
                      className="rounded-lg bg-neutral-50 p-3 dark:bg-neutral-900"
                    >
                      <p className="text-xs text-neutral-500">{label}</p>
                      <p className="mt-1 break-all text-sm font-medium">
                        {value}
                      </p>
                    </div>
                  ))}
                </div>
                {detail.data.error_message && (
                  <div className="rounded-lg bg-red-50 p-4 text-sm text-red-800 dark:bg-red-950/30 dark:text-red-200">
                    <div className="flex gap-2">
                      <AlertCircle size={17} />
                      {detail.data.error_message}
                    </div>
                  </div>
                )}
                {detail.data.request_headers && (
                  <section>
                    <h3 className="mb-2 text-sm font-semibold">
                      {t('admin.logs.requestHeaders')}
                    </h3>
                    <pre className="max-h-64 overflow-auto rounded-xl bg-neutral-950 p-4 text-xs leading-6 text-emerald-100">
                      {formatHeaders(detail.data.request_headers)}
                    </pre>
                  </section>
                )}
                {detail.data.request_body && (
                  <section>
                    <h3 className="mb-2 text-sm font-semibold">
                      {t('admin.logs.requestBody')}
                      {detail.data.request_body.includes("...[truncated")
                        ? t("admin.logs.truncated")
                        : ""}
                    </h3>
                    <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-all rounded-xl bg-neutral-950 p-4 text-xs leading-6 text-emerald-100">
                      {detail.data.request_body}
                    </pre>
                  </section>
                )}
                <details>
                  <summary className="cursor-pointer text-sm font-medium text-neutral-500">
                    {t('admin.logs.fullStructured')}
                  </summary>
                  <pre className="mt-2 overflow-auto rounded-xl bg-neutral-950 p-4 text-xs leading-6 text-emerald-100">
                    {JSON.stringify(detail.data, null, 2)}
                  </pre>
                </details>
              </div>
            ) : null}
          </aside>
        </div>
      )}
    </div>
  );
}
