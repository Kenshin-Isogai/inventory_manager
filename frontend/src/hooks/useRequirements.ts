import useSWR from 'swr'

import { fetchRequirements } from '../lib/mockApi'

export function useRequirements(device?: string, scope?: string) {
  return useSWR(['requirements', device ?? '', scope ?? ''], ([, currentDevice, currentScope]) =>
    fetchRequirements(currentDevice, currentScope),
  )
}
