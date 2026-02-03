import { RouterProvider } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { router } from './router'
import { ThemeProvider } from './hooks/useTheme.tsx'
import { ExportProvider } from './hooks/useExport.tsx'
import { Toaster } from '@/components/ui/toaster'

// Create a query client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60, // 1 minute
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

/**
 * Root application component
 */
function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider>
        <ExportProvider>
          <RouterProvider router={router} />
          <Toaster />
        </ExportProvider>
      </ThemeProvider>
    </QueryClientProvider>
  )
}

export default App
