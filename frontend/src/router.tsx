import { createBrowserRouter, Navigate } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ErrorPage, NotFoundPage } from '@/components/common/ErrorPage'
import Login from '@/pages/Login'
import SetupPassword from '@/pages/SetupPassword'
import Dashboard from '@/pages/Dashboard'
import Reviews from '@/pages/Reviews'
import ReviewDetail from '@/pages/ReviewDetail'
import Reports from '@/pages/Reports'
import ReportDetail from '@/pages/ReportDetail'
import Statistics from '@/pages/Statistics'
import Findings from '@/pages/Findings'
import Repositories from '@/pages/Repositories'
import Rules from '@/pages/Rules'
import ReportTypes from '@/pages/ReportTypes'
import Settings from '@/pages/Settings'

/**
 * Application router configuration
 */
export const router = createBrowserRouter([
  {
    path: '/admin/setup-password',
    element: <SetupPassword />,
    errorElement: <ErrorPage />,
  },
  {
    path: '/admin/login',
    element: <Login />,
    errorElement: <ErrorPage />,
  },
  {
    path: '/admin',
    element: <AppLayout />,
    errorElement: <ErrorPage />,
    children: [
      {
        index: true,
        element: <Dashboard />,
      },
      {
        path: 'reviews',
        element: <Reviews />,
      },
      {
        path: 'reviews/:id',
        element: <ReviewDetail />,
      },
      {
        path: 'reports',
        element: <Reports />,
      },
      {
        path: 'reports/:id',
        element: <ReportDetail />,
      },
      {
        path: 'statistics',
        element: <Statistics />,
      },
      {
        path: 'findings',
        element: <Findings />,
      },
      {
        path: 'repositories',
        element: <Repositories />,
      },
      {
        path: 'rules',
        element: <Rules />,
      },
      {
        path: 'report-types',
        element: <ReportTypes />,
      },
      {
        path: 'settings',
        element: <Settings />,
      },
    ],
  },
  {
    path: '/',
    element: <Navigate to="/admin" replace />,
  },
  {
    path: '*',
    element: <NotFoundPage />,
  },
])
