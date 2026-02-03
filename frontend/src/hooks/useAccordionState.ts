import { useState, useCallback } from 'react'

/**
 * Hook to manage accordion expand/collapse state with localStorage persistence.
 * State is stored per rule_id, so the same rule type shares state across all reviews.
 */
export function useAccordionState() {
  const [, forceUpdate] = useState({})

  const getState = useCallback((ruleId: string, section: 'prompt' | 'result'): boolean => {
    const key = `review-accordion-${ruleId}-${section}`
    const saved = localStorage.getItem(key)
    if (saved !== null) {
      return saved === 'true'
    }
    // Default values: prompt collapsed, result expanded
    return section === 'result'
  }, [])

  const setState = useCallback((ruleId: string, section: 'prompt' | 'result', isOpen: boolean) => {
    const key = `review-accordion-${ruleId}-${section}`
    localStorage.setItem(key, String(isOpen))
    forceUpdate({})
  }, [])

  return { getState, setState }
}

