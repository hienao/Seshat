import { Badge } from '@appica/ui-react/badge'
import { BrandGolang, BrandReact, Check, Database, FileText, Lock, Rocket, Server } from '@appica/icons-react'
import { Link } from 'react-router-dom'
import { AppButton } from '@/components/common/app-button'
import { useAuthStore } from '@/stores/auth'

const stacks = [
  { icon: BrandReact, title: 'React 前端', color: 'text-sky-600', items: ['React 19 + Vite', 'Appica UI + Tailwind CSS 4', 'TanStack Query / Table', 'TypeScript 类型检查'] },
  { icon: BrandGolang, title: 'Go 后端', color: 'text-cyan-700', items: ['Go + Gin', 'GORM 数据访问', 'Swagger API 文档', '统一响应结构'] },
  { icon: Database, title: '部署与数据', color: 'text-emerald-700', items: ['默认 SQLite 持久化', '可选 PostgreSQL', 'HttpOnly Cookie 认证', 'Docker 一键部署'] },
]

export function HomePage() {
  const user = useAuthStore((state) => state.user)
  return (
    <div className="overflow-hidden px-4 pb-20 pt-14 sm:px-6 sm:pt-20">
      <div className="app-grid pointer-events-none absolute inset-x-0 top-0 -z-10 h-[680px]" />
      <section className="rise-in mx-auto max-w-6xl text-center">
        <Badge variant="soft" size="md"><Rocket size={15} />全栈模板工程</Badge>
        <h1 className="mx-auto mt-7 max-w-4xl text-5xl font-black tracking-[-0.055em] text-neutral-950 dark:text-white sm:text-7xl">
          从可靠的基础出发，<span className="text-emerald-700 dark:text-emerald-400">更快交付产品</span>
        </h1>
        <p className="mx-auto mt-6 max-w-2xl text-lg leading-8 text-neutral-600 dark:text-neutral-400">BaseGoApp 已准备好认证、用户管理、数据库持久化和容器部署，让业务代码成为项目的第一优先级。</p>
        <div className="mt-9 flex flex-wrap justify-center gap-3">
          {user ? <AppButton render={<Link to="/profile" />} size="lg">进入个人中心</AppButton> : <AppButton render={<Link to="/register" />} size="lg"><Rocket size={18} />立即开始</AppButton>}
          <AppButton render={<a href="/swagger/index.html" target="_blank" rel="noreferrer" />} variant="outline" size="lg"><FileText size={18} />API 文档</AppButton>
        </div>
        <div className="mx-auto mt-14 grid max-w-4xl grid-cols-1 gap-px overflow-hidden rounded-2xl border border-neutral-200 bg-neutral-200 text-left shadow-xl shadow-neutral-900/5 dark:border-neutral-800 dark:bg-neutral-800 sm:grid-cols-3">
          {[['前端', 'React 19'], ['后端', 'Go + Gin'], ['数据库', 'SQLite / PostgreSQL']].map(([label, value]) => <div key={label} className="bg-white/90 px-6 py-5 dark:bg-neutral-950/90"><p className="text-xs font-bold uppercase tracking-widest text-neutral-400">{label}</p><p className="mt-1 font-semibold">{value}</p></div>)}
        </div>
      </section>

      <section className="mx-auto mt-20 grid max-w-6xl gap-5 md:grid-cols-3">
        {stacks.map(({ icon: Icon, title, color, items }, index) => (
          <article key={title} className="app-panel rise-in p-6" style={{ animationDelay: `${120 + index * 90}ms` }}>
            <div className={`grid size-12 place-items-center rounded-2xl bg-neutral-100 dark:bg-neutral-800 ${color}`}><Icon size={25} /></div>
            <h2 className="mt-5 text-xl font-bold tracking-tight">{title}</h2>
            <ul className="mt-5 space-y-3">
              {items.map((item) => <li key={item} className="flex items-center gap-2.5 text-sm text-neutral-600 dark:text-neutral-400"><Check size={17} className="shrink-0 text-emerald-600" />{item}</li>)}
            </ul>
          </article>
        ))}
      </section>

      <section className="mx-auto mt-16 flex max-w-6xl flex-col items-start justify-between gap-5 rounded-3xl bg-neutral-950 px-7 py-8 text-white shadow-2xl sm:flex-row sm:items-center sm:px-10 dark:bg-emerald-950">
        <div className="flex items-start gap-4"><span className="grid size-11 shrink-0 place-items-center rounded-xl bg-emerald-500/15 text-emerald-300"><Lock size={22} /></span><div><h2 className="text-xl font-bold">安全默认值已经就位</h2><p className="mt-1 text-sm text-neutral-300">CORS 白名单、强 JWT 密钥、HttpOnly Cookie 与管理员初始化检查。</p></div></div>
        <a href="/swagger/index.html" className="inline-flex items-center gap-2 text-sm font-semibold text-emerald-300 hover:text-emerald-200"><Server size={17} />查看接口</a>
      </section>
    </div>
  )
}
