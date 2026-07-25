import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { Navigate } from '@tanstack/react-router'
import { fetchMe, logout } from './api'
// Side effect: applies the stored theme and follows OS changes app-wide.
import './theme'

export function Shell() {
  const { data: me, isLoading } = useQuery({ queryKey: ['me'], queryFn: fetchMe })
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  if (!isLoading && me === null && pathname !== '/login') {
    return <Navigate to="/login" />
  }

  return (
    <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
      <header className="border-b border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900 print:hidden">
        <div className="mx-auto flex h-14 w-full max-w-6xl items-center justify-between gap-2 px-3 sm:px-4">
          <div className="flex min-w-0 items-center gap-3 sm:gap-5">
            <Link
              to="/"
              className="flex shrink-0 items-center gap-2 text-lg font-semibold tracking-tight text-slate-900 dark:text-slate-100"
            >
              <BudLogo size={28} /> <span className="hidden sm:inline">getbud</span>
            </Link>
            <NavLinks />
          </div>
          <div className="flex shrink-0 items-center gap-3">
            <UserMenu />
          </div>
        </div>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  )
}

function BudLogo({ size = 28 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" aria-hidden="true">
      <circle cx="16" cy="16" r="14" className="fill-teal-600" />
      <text
        x="16"
        y="22"
        fontSize="17"
        fontWeight="700"
        textAnchor="middle"
        className="fill-white"
        style={{ fontFamily: 'ui-sans-serif, system-ui' }}
      >
        $
      </text>
    </svg>
  )
}

function NavLinks() {
  const { data: me } = useQuery({ queryKey: ['me'], queryFn: fetchMe })
  if (!me) return null
  const link =
    'text-sm text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-100 [&.active]:font-medium [&.active]:text-slate-900 dark:[&.active]:text-slate-100'
  return (
    <div className="flex items-center gap-3 overflow-x-auto sm:gap-5">
      <Link to="/" activeOptions={{ exact: true }} className={link}>
        Dashboard
      </Link>
      <Link to="/transactions" className={link}>
        Transactions
      </Link>
      <Link to="/accounts" className={link}>
        Accounts
      </Link>
      <Link to="/reports" className={link}>
        Reports
      </Link>
      <Link to="/contributions" className={link}>
        Contributions
      </Link>
      <Link to="/import" className={link}>
        Import
      </Link>
      <Link to="/settings" className={link}>
        Settings
      </Link>
    </div>
  )
}

function UserMenu() {
  const { data: me } = useQuery({ queryKey: ['me'], queryFn: fetchMe })
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const signOut = useMutation({
    mutationFn: logout,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['me'] })
      navigate({ to: '/login' })
    },
  })

  if (!me) return null

  return (
    <div className="flex items-center gap-3">
      {me.avatarUrl ? (
        <img src={me.avatarUrl} alt="" className="h-8 w-8 rounded-full" referrerPolicy="no-referrer" />
      ) : (
        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-slate-200 dark:bg-slate-700 text-sm font-medium text-slate-600 dark:text-slate-400">
          {(me.displayName || me.email).charAt(0).toUpperCase()}
        </div>
      )}
      <span className="hidden text-sm text-slate-700 dark:text-slate-300 md:inline">
        {me.displayName || me.email}
      </span>
      <button
        type="button"
        onClick={() => signOut.mutate()}
        disabled={signOut.isPending}
        className="rounded-lg border border-slate-300 dark:border-slate-600 px-3 py-1.5 text-sm text-slate-600 dark:text-slate-400 hover:bg-slate-50 dark:hover:bg-slate-800 disabled:opacity-50"
      >
        Sign out
      </button>
    </div>
  )
}
