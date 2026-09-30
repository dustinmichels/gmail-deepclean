// frontend/src/types.ts

export interface SessionInfo {
  sessionToken: string
  userEmail: string
  userId: string
}

export interface AuthMeResponse {
  authenticated: boolean
  userEmail: string
  userId: string
  createdAt: string
}

export interface ContactStat {
  email: string
  name: string
  receivedCount: number
  sentCount: number
  totalSize: number
  firstReceived?: string
  lastReceived?: string
  firstSent?: string
  lastSent?: string
  neverEmailed: boolean
}

export interface ContactsResponse {
  contacts: ContactStat[]
  total: number
  limit: number
  offset: number
}

export interface OverallStats {
  totalEmailsProcessed: number
  totalReceived: number
  totalSent: number
  uniqueSenders: number
  neverEmailedCount: number
  totalSizeBytes: number
}

export interface ScanProgress {
  userId: string
  userEmail: string
  isProcessing: boolean
  totalEmails: number
  estimatedTotal?: number
  statusMessage: string
  lastScannedAt?: string
}

export interface EmailMetadata {
  id: string
  threadId: string
  from: string
  fromName: string
  to: string[]
  subject: string
  date: string
  snippet: string
  labelIds: string[]
  sizeEstimate: number
  isSent: boolean
  senderNeverEmailed?: boolean
  senderEmailCount?: number
}

export interface EmailsResponse {
  emails: EmailMetadata[]
  total: number
  limit: number
  offset: number
}
