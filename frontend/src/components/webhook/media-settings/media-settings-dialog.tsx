import { Input } from '@appica/ui-react/input'
import { Dialog, DialogBody, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@appica/ui-react/dialog'
import { useMutation, useQuery } from '@tanstack/react-query'
import { useEffect, useState, type FormEvent } from 'react'
import { api } from '@/api/services'
import type { Integration } from '@/api/types'
import { AppButton } from '@/components/common/app-button'
import { FormField } from '@/components/common/form-field'
import { Message } from '@/components/common/feedback'
import { SecretInput } from '@/components/notification-channels/secret-input'
import { errorMessage } from '@/lib/error-message'
import { mediaSettingsDefinition } from './registry'

export function MediaSettingsDialog({ integration, open, onOpenChange, onSaved }: {
	integration: Integration | null
	open: boolean
	onOpenChange: (open: boolean) => void
	onSaved: () => void
}) {
	const definition = integration ? mediaSettingsDefinition(integration.app_code) : undefined
	const settings = useQuery({
		queryKey: ['webhooks', 'integrations', integration?.id, 'media-settings'],
		queryFn: () => api.integrationMediaSettings(integration!.id),
		enabled: open && Boolean(integration && definition),
	})
	const [serverUrl, setServerUrl] = useState('')
	const [apiKey, setAPIKey] = useState('')
	const [tested, setTested] = useState(false)

	useEffect(() => {
		if (!settings.data) return
		setServerUrl(settings.data.server_url)
		setAPIKey(settings.data.api_key)
		setTested(false)
	}, [settings.data])

	const body = { server_url: serverUrl.trim(), api_key: apiKey.trim() }
	const test = useMutation({ mutationFn: () => api.testIntegrationMediaSettings(integration!.id, body), onSuccess: () => setTested(true) })
	const save = useMutation({ mutationFn: () => api.updateIntegrationMediaSettings(integration!.id, body), onSuccess: () => { onSaved(); onOpenChange(false) } })

	if (!integration || !definition) return null
	function submit(event: FormEvent) {
		event.preventDefault()
		if (body.server_url && body.api_key) save.mutate()
	}
	const valid = Boolean(body.server_url && body.api_key)

	return (
		<Dialog open={open} onOpenChange={onOpenChange}>
			<DialogContent className="max-w-xl">
				<form onSubmit={submit}>
					<DialogHeader>
						<DialogTitle>{definition.appName} 媒体 API</DialogTitle>
						<DialogDescription>保存后优先使用该实例的媒体资料；请求失败时才回退到系统 TMDB API。</DialogDescription>
					</DialogHeader>
					<DialogBody className="space-y-5">
						{settings.isPending ? <p className="py-6 text-center text-sm text-neutral-500">正在读取配置…</p> : settings.error ? <Message variant="error" title={errorMessage(settings.error, '读取媒体 API 配置失败')} /> : <>
							<FormField label="服务器地址" description={definition.serverHelp}>
								<Input type="url" value={serverUrl} onChange={(event) => { setServerUrl(event.target.value); setTested(false) }} placeholder={definition.serverPlaceholder} required autoComplete="url" />
							</FormField>
							<FormField label="API Key" description={definition.apiKeyHelp}>
								<SecretInput revealLabel={`${definition.appName} API Key`} value={apiKey} onChange={(event) => { setAPIKey(event.target.value); setTested(false) }} required autoComplete="off" />
							</FormField>
						</>}
						{tested && <Message variant="success" title="连接成功" />}
						{test.error && <Message variant="error" title={errorMessage(test.error, '连接测试失败')} />}
						{save.error && <Message variant="error" title={errorMessage(save.error, '保存媒体 API 配置失败')} />}
					</DialogBody>
					<DialogFooter>
						<DialogClose render={<AppButton type="button" variant="ghost">取消</AppButton>} />
						<AppButton type="button" variant="outline" disabled={!valid || test.isPending || settings.isPending} onClick={() => test.mutate()}>{test.isPending ? '测试中…' : '测试连接'}</AppButton>
						<AppButton type="submit" disabled={!valid || save.isPending || settings.isPending}>{save.isPending ? '保存中…' : '保存'}</AppButton>
					</DialogFooter>
				</form>
			</DialogContent>
		</Dialog>
	)
}
