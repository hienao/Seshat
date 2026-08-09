import { Input } from '@appica/ui-react/input'
import type { NotificationChannelAdapter } from '../types'
import { channelInputClass, channelPayload, secretLabel, textField } from '../types'

type SMTPEncryption = 'starttls' | 'tls' | 'none'

function smtpEncryption(value: unknown): SMTPEncryption {
  return value === 'tls' || value === 'none' ? value : 'starttls'
}

export const emailChannelAdapter: NotificationChannelAdapter = {
  type: 'email',
  label: '邮箱通知',
  defaultFields: () => ({ host: '', port: '587', encryption: 'starttls', from: '', to: '', username: '', password: '' }),
  fieldsFromChannel: (channel) => ({
    host: String(channel.config.smtp_host ?? ''),
    port: String(channel.config.smtp_port ?? '587'),
    encryption: smtpEncryption(channel.config.encryption),
    from: String(channel.config.from ?? ''),
    to: String(channel.config.to ?? ''),
    username: '',
    password: '',
  }),
  toPayload: (common, fields) => {
    const username = textField(fields, 'username').trim()
    const password = textField(fields, 'password')
    return channelPayload(
      common,
      { smtp_host: textField(fields, 'host').trim(), smtp_port: Number(textField(fields, 'port')), encryption: smtpEncryption(fields.encryption), from: textField(fields, 'from').trim(), to: textField(fields, 'to').trim() },
      username || password ? { ...(username ? { username } : {}), ...(password ? { password } : {}) } : undefined,
    )
  },
  Form: ({ fields, keepsExistingCredential, update }) => (
    <>
      <div className="grid gap-4 sm:grid-cols-[1fr_8rem]">
        <label className="block space-y-2 text-sm font-medium">
          <span>SMTP 主机</span>
          <Input value={textField(fields, 'host')} onChange={(event) => update('host', event.target.value)} required placeholder="smtp.example.com" />
        </label>
        <label className="block space-y-2 text-sm font-medium">
          <span>端口</span>
          <Input type="number" min="1" max="65535" value={textField(fields, 'port')} onChange={(event) => update('port', event.target.value)} required />
        </label>
      </div>
      <label className="block space-y-2 text-sm font-medium">
        <span>连接加密</span>
        <select className={channelInputClass} value={smtpEncryption(fields.encryption)} onChange={(event) => update('encryption', event.target.value)}>
          <option value="starttls">STARTTLS（常用端口 587）</option>
          <option value="tls">TLS（常用端口 465）</option>
          <option value="none">不加密（仅可信网络）</option>
        </select>
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>发件人</span>
        <Input type="email" value={textField(fields, 'from')} onChange={(event) => update('from', event.target.value)} required placeholder="notice@example.com" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>收件人</span>
        <Input value={textField(fields, 'to')} onChange={(event) => update('to', event.target.value)} required placeholder="a@example.com, b@example.com" />
        <span className="block text-xs font-normal text-neutral-500">多个地址使用逗号或分号分隔。</span>
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>SMTP 用户名（可选，与密码同时填写）</span>
        <Input value={textField(fields, 'username')} onChange={(event) => update('username', event.target.value)} autoComplete="off" />
      </label>
      <label className="block space-y-2 text-sm font-medium">
        <span>{secretLabel('SMTP 密码（可选，与用户名同时填写）', keepsExistingCredential)}</span>
        <Input type="password" value={textField(fields, 'password')} onChange={(event) => update('password', event.target.value)} autoComplete="new-password" />
      </label>
    </>
  ),
}
