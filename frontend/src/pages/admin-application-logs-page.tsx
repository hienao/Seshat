import { Badge } from "@appica/ui-react/badge";
import { Input } from "@appica/ui-react/input";
import {
  AlertCircle,
  Activity,
  Download,
  Refresh,
  Trash,
} from "@appica/icons-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api } from "@/api/services";
import type { ApplicationLogLevel } from "@/api/types";
import { AppButton } from "@/components/common/app-button";
import { ConfirmDialog } from "@/components/common/confirm-dialog";
import { EmptyState, ErrorState, Message } from "@/components/common/feedback";
import { PageHeader } from "@/components/common/page-header";
import { Panel } from "@/components/common/panel";
import { errorMessage } from "@/lib/error-message";
import { formatDateTime } from "@/i18n/format";

type Filters = {
  startAt: string;
  endAt: string;
  level: "" | ApplicationLogLevel;
  source: string;
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
    level: "",
    source: "",
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
function formatDate(value: string) {
  return value ? formatDateTime(value) : "-";
}
function apiFilters(filters: Filters) {
  return {
    startAt: toApiTime(filters.startAt),
    endAt: toApiTime(filters.endAt),
    level: filters.level,
    source: filters.source,
    requestId: filters.requestId,
    keyword: filters.keyword,
  };
}
function levelVariant(
  level: ApplicationLogLevel,
): "soft" | "outline" | "error" | "success" {
  if (level === "ERROR") return "error";
  if (level === "WARN") return "outline";
  if (level === "INFO") return "success";
  return "soft";
}
function formatFields(value?: string) {
  if (!value) return "";
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

export function AdminApplicationLogsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState<Filters>(initialFilters);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const current = apiFilters(filters);
  const logs = useQuery({
    queryKey: ["admin", "application-logs", current],
    queryFn: () => api.applicationLogs(current),
  });
  const summary = useQuery({
    queryKey: ["admin", "application-log-summary", current],
    queryFn: () => api.applicationLogSummary(current),
  });
  const detail = useQuery({
    queryKey: ["admin", "application-log", selectedId],
    queryFn: () => api.applicationLog(selectedId as number),
    enabled: selectedId !== null,
  });
  const clear = useMutation({
    mutationFn: () =>
      api.clearApplicationLogs({
        start_at: current.startAt,
        end_at: current.endAt,
        level: current.level,
        source: current.source,
        request_id: current.requestId,
        keyword: current.keyword,
        confirmation: "CLEAR_APPLICATION_LOGS",
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({
        queryKey: ["admin", "application-logs"],
      });
      void queryClient.invalidateQueries({
        queryKey: ["admin", "application-log-summary"],
      });
    },
  });
  const exportLogs = useMutation({
    mutationFn: (format: "csv" | "jsonl") =>
      api.exportApplicationLogs({ ...current, format }),
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
  return (
    <div className="mx-auto max-w-7xl space-y-7 px-4 py-10 sm:px-6">
      <PageHeader
        eyebrow={t("admin.logs.eyebrow")}
        title={t("admin.logs.applicationTitle")}
        description={t("admin.logs.applicationDescription")}
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
          [t("admin.logs.totalApplication"), summary.data?.total ?? 0],
          ["DEBUG", summary.data?.debug_count ?? 0],
          ["INFO", summary.data?.info_count ?? 0],
          ["WARN", summary.data?.warn_count ?? 0],
          ["ERROR", summary.data?.error_count ?? 0],
        ].map(([label, value]) => (
          <div key={label} className="app-panel p-5">
            <p className="text-xs font-medium text-neutral-500">{label}</p>
            <p className="mt-2 text-2xl font-bold tracking-tight">{value}</p>
          </div>
        ))}
      </div>

      <Panel title={t("admin.logs.filters")} icon={<Activity size={20} />}>
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
            <span>{t("admin.logs.level")}</span>
            <select
              className={inputClass}
              value={filters.level}
              onChange={(event) =>
                update("level", event.target.value as Filters["level"])
              }
            >
              <option value="">{t("admin.logs.allLevels")}</option>
              {(["DEBUG", "INFO", "WARN", "ERROR"] as const).map((level) => (
                <option key={level} value={level}>
                  {level}
                </option>
              ))}
            </select>
          </label>
          <label className="space-y-2 text-sm font-medium">
            <span>{t("admin.logs.source")}</span>
            <Input
              value={filters.source}
              onChange={(event) => update("source", event.target.value)}
              placeholder="webhook, server"
            />
          </label>
          <label className="space-y-2 text-sm font-medium md:col-span-2">
            <span>{t("admin.logs.keyword")}</span>
            <Input
              value={filters.keyword}
              onChange={(event) => update("keyword", event.target.value)}
            />
          </label>
          <label className="space-y-2 text-sm font-medium md:col-span-2">
            <span>Request ID</span>
            <Input
              value={filters.requestId}
              onChange={(event) => update("requestId", event.target.value)}
              placeholder={t('admin.logs.requestId')}
            />
          </label>
        </div>
        <div className="mt-5 flex flex-wrap items-center gap-2">
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
          {(summary.data?.dropped ?? 0) > 0 && (
            <span className="text-xs text-amber-700 dark:text-amber-300">
              {t("admin.logs.dropped", { count: summary.data?.dropped })}
            </span>
          )}
        </div>
      </Panel>

      <Panel
        title={t("admin.logs.applicationRecords")}
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
            title={t("admin.logs.applicationEmpty")}
            description={t("admin.logs.applicationEmptyDescription")}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-[940px] text-left text-sm">
              <thead>
                <tr className="border-b border-neutral-200 text-xs text-neutral-500 dark:border-neutral-800">
                  <th className="px-2 py-3">{t("admin.logs.time")}</th>
                  <th className="px-2 py-3">{t("admin.logs.level")}</th>
                  <th className="px-2 py-3">{t("admin.logs.source")}</th>
                  <th className="px-2 py-3">{t("admin.logs.message")}</th>
                  <th className="px-2 py-3">{t("admin.logs.related")}</th>
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
                      <Badge variant={levelVariant(item.level)} size="sm">
                        {item.level}
                      </Badge>
                    </td>
                    <td className="px-2 py-3 font-mono text-xs">
                      {item.source || "-"}
                    </td>
                    <td className="max-w-[360px] truncate px-2 py-3 font-medium">
                      {item.message}
                    </td>
                    <td className="px-2 py-3 text-xs">
                      {item.app_code
                        ? `${item.app_code}${item.integration_id ? ` / #${item.integration_id}` : ""}${item.event_id ? ` / #${item.event_id}` : ""}`
                        : "-"}
                    </td>
                    <td className="max-w-[160px] truncate px-2 py-3 font-mono text-xs text-neutral-500">
                      {item.request_id || "-"}
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
        title={t("admin.logs.clearApplication")}
        description={t("admin.logs.clearDescription")}
        icon={<Trash size={20} />}
      >
        <ConfirmDialog
          title={t("admin.logs.clearApplicationTitle")}
          description={t("admin.logs.clearApplicationDescription")}
          confirmLabel={t("admin.logs.clearConfirm")}
          destructive
          busy={clear.isPending}
          onConfirm={() => clear.mutate()}
          trigger={
            <AppButton variant="destructive" disabled={clear.isPending}>
              <Trash size={16} />
              {t("admin.logs.clearRange")}
            </AppButton>
          }
        />
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
                  Application Log
                </p>
                <h2 className="mt-1 text-xl font-bold">{t('admin.logs.applicationDetail')}</h2>
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
                <div className="flex items-center gap-3">
                  <Badge variant={levelVariant(detail.data.level)}>
                    {detail.data.level}
                  </Badge>
                  <span className="font-mono text-sm text-neutral-500">
                    {detail.data.source || t('admin.logs.unspecifiedSource')}
                  </span>
                </div>
                <div>
                  <h3 className="text-sm font-semibold">{t('admin.logs.logMessage')}</h3>
                  <p className="mt-2 whitespace-pre-wrap break-words rounded-xl bg-neutral-50 p-4 text-sm leading-7 dark:bg-neutral-900">
                    {detail.data.message}
                  </p>
                </div>
                <div className="grid grid-cols-2 gap-3 text-sm">
                  {[
                    [t("admin.logs.time"), formatDate(detail.data.occurred_at)],
                    ["Request ID", detail.data.request_id || "-"],
                    [
                      t("admin.logs.userId"),
                      detail.data.user_id ? `#${detail.data.user_id}` : "-",
                    ],
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
                {detail.data.fields && (
                  <section>
                    <h3 className="mb-2 text-sm font-semibold">{t('admin.logs.structuredFields')}</h3>
                    <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-all rounded-xl bg-neutral-950 p-4 text-xs leading-6 text-emerald-100">
                      {formatFields(detail.data.fields)}
                    </pre>
                  </section>
                )}
              </div>
            ) : null}
          </aside>
        </div>
      )}
    </div>
  );
}
