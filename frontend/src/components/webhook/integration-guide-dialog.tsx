import { Badge } from '@appica/ui-react/badge'
import { Dialog, DialogBody, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@appica/ui-react/dialog'
import { Copy, Key, Link as LinkIcon } from '@appica/icons-react'
import type { ReactNode } from 'react'
import type { AppDefinition, Integration } from '@/api/types'
import { AppButton } from '@/components/common/app-button'

export function IntegrationGuideDialog({ integration, app, secret, secretLoading, open, onOpenChange, onRevealSecret, onCopy }: {
  integration: Integration | null
  app?: AppDefinition
  secret?: string
  secretLoading?: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
  onRevealSecret: () => void
  onCopy: (value: string) => void
}) {
  if (!integration) return null
  const webhookUrl = `${window.location.origin}${integration.webhook_path}`
  const needsSecret = app?.auth_mode ? app.auth_mode !== 'endpoint_url' : integration.app_code !== 'emby'
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{app?.name ?? integration.app_code} 接入说明</DialogTitle>
          <DialogDescription>将“{integration.name}”配置到对应 App，保存后可在消息流和日志页面检查接收结果。</DialogDescription>
        </DialogHeader>
        <DialogBody className="max-h-[70vh] space-y-5 overflow-y-auto">
          <GuideValue icon={<LinkIcon size={16} />} label="Webhook 地址" value={webhookUrl} onCopy={() => onCopy(webhookUrl)} />
          {needsSecret ? (
            secret ? <GuideValue icon={<Key size={16} />} label="Webhook Secret" value={secret} onCopy={() => onCopy(secret)} sensitive /> : (
              <div className="rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-900 dark:bg-amber-950/40">
                <p className="text-sm font-medium text-amber-900 dark:text-amber-200">配置时还需要当前 Secret</p>
                <AppButton className="mt-3" size="sm" variant="outline" disabled={secretLoading} onClick={onRevealSecret}>{secretLoading ? '正在读取…' : '查看 Secret'}</AppButton>
              </div>
            )
          ) : <p className="rounded-xl bg-sky-50 p-4 text-sm text-sky-800 dark:bg-sky-950/50 dark:text-sky-200">Emby 不支持配置自定义请求头，因此随机 Webhook 地址本身就是接入凭据。请勿公开该地址。</p>}
          <GuideSteps appCode={integration.app_code} webhookUrl={webhookUrl} secret={secret} />
        </DialogBody>
        <DialogFooter><DialogClose render={<AppButton>完成</AppButton>} /></DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function GuideValue({ icon, label, value, sensitive, onCopy }: { icon: ReactNode; label: string; value: string; sensitive?: boolean; onCopy: () => void }) {
  return (
    <div>
      <p className="mb-2 flex items-center gap-2 text-sm font-medium">{icon}{label}{sensitive && <Badge variant="warning" size="sm">敏感信息</Badge>}</p>
      <div className="flex items-start gap-2"><code className="min-w-0 flex-1 break-all rounded-xl bg-neutral-950 p-3 text-xs leading-5 text-emerald-100">{value}</code><AppButton size="sm" variant="outline" onClick={onCopy}><Copy size={15} />复制</AppButton></div>
    </div>
  )
}

function GuideSteps({ appCode, webhookUrl, secret }: { appCode: string; webhookUrl: string; secret?: string }) {
  if (appCode === 'jellyfin') return <Instruction title="Jellyfin Webhook 插件配置" steps={[
    '控制台 → 插件 → 目录 → Notifications → Webhook，安装并重启 Jellyfin。',
    '控制台 → 插件 → My Plugins → Webhook，点击 Add Generic Destination。',
    `Webhook URL 填写 ${webhookUrl}，勾选需要的 Notification Type 和 Send All Properties (ignores template)。`,
    '添加请求头 Content-Type: application/json。',
    `添加请求头 X-Webhook-Secret: ${secret || '上方显示的 Webhook Secret'}，不要添加引号。`,
    '保存后触发一次播放或媒体库事件；返回 202 Accepted 表示接收成功。',
  ]} />
  if (appCode === 'emby') return <Instruction title="Emby Notifications 配置" steps={[
    'Server Dashboard → Plugin Catalog → Notifications，安装 Webhooks 插件并重启 Emby。',
    '管理员用户偏好设置 → Notifications → + Add Notification → Webhooks。',
    `Webhook URL 填写 ${webhookUrl}。`,
    '选择需要发送的事件，并按需限制用户、媒体库和设备。',
    '保存并测试；Emby 无需填写额外 Secret，请将完整 Webhook 地址视为敏感凭据。',
  ]} />
  if (appCode === 'github') return <Instruction title="GitHub Webhook 配置" steps={[
    '进入仓库 Settings → Webhooks → Add webhook。',
    `Payload URL 填写 ${webhookUrl}。`,
    'Content type 选择 application/json。',
    `Secret 填写 ${secret || '上方显示的 Webhook Secret'}。`,
    '选择需要接收的事件并启用 Webhook，检查 Recent Deliveries 是否返回 202。',
  ]} />
  return <Instruction title="通用 Webhook 配置" steps={[
    `向 ${webhookUrl} 发送 POST 请求。`,
    '请求体使用 JSON，并设置 Content-Type: application/json。',
    `请求头添加 X-Webhook-Secret: ${secret || '上方显示的 Webhook Secret'}。`,
    '可通过 X-Webhook-Event 或 JSON 中的 event_type、event、type 指定原始消息类型。',
  ]} />
}

function Instruction({ title, steps }: { title: string; steps: string[] }) {
  return <section className="rounded-xl border border-neutral-200 p-4 dark:border-neutral-800"><h3 className="font-semibold">{title}</h3><ol className="mt-3 space-y-3">{steps.map((step, index) => <li key={step} className="flex gap-3 text-sm leading-6 text-neutral-600 dark:text-neutral-300"><span className="grid size-6 shrink-0 place-items-center rounded-full bg-emerald-50 text-xs font-bold text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300">{index + 1}</span><span className="min-w-0 break-words">{step}</span></li>)}</ol></section>
}
