import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router'
import './index.css'
import { Shell } from './Shell'
import { DashboardPage } from './pages/Dashboard'
import { LoginPage } from './pages/Login'
import { TransactionsPage } from './pages/Transactions'
import { AccountsPage } from './pages/Accounts'
import { AccountDetailPage } from './pages/AccountDetail'
import { ReportsPage } from './pages/Reports'
import { ContributionsPage } from './pages/Contributions'
import { ImportPage } from './pages/Import'
import { SettingsPage } from './pages/Settings'

const rootRoute = createRootRoute({ component: Shell })

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: DashboardPage,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginPage,
})

const transactionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/transactions',
  component: TransactionsPage,
})

const accountsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/accounts',
  component: AccountsPage,
})

const accountDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/accounts/$accountId',
  component: AccountDetailPage,
})

const reportsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/reports',
  component: ReportsPage,
})

const contributionsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/contributions',
  component: ContributionsPage,
})

const importRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/import',
  component: ImportPage,
})

const settingsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings',
  component: SettingsPage,
})

const routeTree = rootRoute.addChildren([
  indexRoute,
  loginRoute,
  transactionsRoute,
  accountsRoute,
  accountDetailRoute,
  reportsRoute,
  contributionsRoute,
  importRoute,
  settingsRoute,
])

const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: 1, staleTime: 30_000 },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
)
