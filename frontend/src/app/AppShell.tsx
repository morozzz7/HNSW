import { ChartColumn, Database, Menu, Moon, Network, Search, Sun, X } from 'lucide-react'
import { useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { cn } from '@/lib/cn'
import { LayerMark } from './LayerMark'
import { useTheme } from './theme'

const NAV_GROUPS = [
  {
    label: 'Данные',
    items: [{ to: '/datasets', label: 'Датасеты', icon: Database }],
  },
  {
    label: 'Индекс',
    items: [{ to: '/index', label: 'Построение', icon: Network }],
  },
  {
    label: 'Анализ',
    items: [
      { to: '/search', label: 'Поиск', icon: Search },
      { to: '/benchmark', label: 'Бенчмарк', icon: ChartColumn },
    ],
  },
]

function SidebarContent({ onNavigate }: { onNavigate?: () => void }) {
  const { theme, toggleTheme } = useTheme()

  return (
    <div className="flex h-full flex-col px-3 py-4">
      <NavLink to="/datasets" className="flex items-center gap-2.5 rounded-lg px-2 py-1.5">
        <span className="text-accent">
          <LayerMark className="size-6" />
        </span>
        <span className="flex min-w-0 flex-col">
          <span className="text-sm font-medium leading-tight text-ink">HNSW Lab</span>
          <span className="text-[11px] leading-tight text-ink-3">лаборатория графа</span>
        </span>
      </NavLink>

      <nav aria-label="Основная навигация" className="mt-2 flex flex-col">
        {NAV_GROUPS.map((group) => (
          <div key={group.label} className="mt-5 first:mt-2">
            <p className="mb-1.5 px-2.5 text-[11px] font-medium text-ink-3">{group.label}</p>
            <div className="flex flex-col gap-0.5">
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  onClick={onNavigate}
                  className={({ isActive }) =>
                    cn(
                      'flex h-9 items-center gap-2.5 rounded-lg pl-1.5 pr-2.5 text-[13px] transition-colors',
                      isActive
                        ? 'bg-surface-2 text-ink'
                        : 'text-ink-2 hover:bg-surface-2 hover:text-ink',
                    )
                  }
                >
                  {({ isActive }) => (
                    <>
                      <span
                        aria-hidden
                        className={cn(
                          'h-4 w-0.5 rounded-full',
                          isActive ? 'bg-accent' : 'bg-transparent',
                        )}
                      />
                      <item.icon aria-hidden className="size-4 shrink-0" />
                      <span>{item.label}</span>
                    </>
                  )}
                </NavLink>
              ))}
            </div>
          </div>
        ))}
      </nav>

      <div className="mt-auto flex flex-col gap-1 pt-6">
        <button
          type="button"
          onClick={toggleTheme}
          className="flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-[13px] text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink"
        >
          {theme === 'dark' ? (
            <Sun aria-hidden className="size-4" />
          ) : (
            <Moon aria-hidden className="size-4" />
          )}
          {theme === 'dark' ? 'Светлая тема' : 'Тёмная тема'}
        </button>
        <p className="px-2.5 pt-1 font-mono text-[11px] text-ink-3">v0.1 · этап 9</p>
      </div>
    </div>
  )
}

export function AppShell() {
  const [menuOpen, setMenuOpen] = useState(false)

  return (
    <div className="min-h-dvh lg:grid lg:grid-cols-[248px_minmax(0,1fr)]">
      <aside className="hidden border-r border-line bg-surface lg:sticky lg:top-0 lg:block lg:h-dvh lg:overflow-y-auto">
        <SidebarContent />
      </aside>

      <div className="flex min-w-0 flex-col">
        <header className="flex h-14 items-center gap-3 border-b border-line bg-surface px-4 lg:hidden">
          <button
            type="button"
            onClick={() => setMenuOpen(true)}
            aria-label="Открыть меню"
            className="inline-flex size-9 items-center justify-center rounded-lg text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink"
          >
            <Menu aria-hidden className="size-5" />
          </button>
          <span className="flex items-center gap-2 text-accent">
            <LayerMark className="size-5" />
            <span className="text-sm font-medium text-ink">HNSW Lab</span>
          </span>
        </header>

        <main className="min-w-0 flex-1 px-5 py-6 sm:px-8 sm:py-8">
          <div className="mx-auto flex w-full max-w-[1440px] flex-col gap-6">
            <Outlet />
          </div>
        </main>
      </div>

      {menuOpen ? (
        <div className="fixed inset-0 z-40 lg:hidden">
          <button
            type="button"
            aria-label="Закрыть меню"
            onClick={() => setMenuOpen(false)}
            className="absolute inset-0 bg-black/50"
          />
          <div className="absolute inset-y-0 left-0 w-[264px] border-r border-line bg-surface">
            <button
              type="button"
              onClick={() => setMenuOpen(false)}
              aria-label="Закрыть меню"
              className="absolute right-2 top-3 inline-flex size-8 items-center justify-center rounded-lg text-ink-2 transition-colors hover:bg-surface-2 hover:text-ink"
            >
              <X aria-hidden className="size-4" />
            </button>
            <SidebarContent onNavigate={() => setMenuOpen(false)} />
          </div>
        </div>
      ) : null}
    </div>
  )
}
