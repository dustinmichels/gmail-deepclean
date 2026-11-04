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

// A sender the user has never emailed. Contacts the user has emailed are never returned.
export interface ContactStat {
  email: string
  name: string
  receivedCount: number
  totalSize: number
  firstReceived?: string
  lastReceived?: string
}

export interface ContactsResponse {
  contacts: ContactStat[]
  total: number
  limit: number
  offset: number
}

// Totals cover all scanned mail; neverEmailed* cover received mail from senders never emailed.
export interface OverallStats {
  totalEmailsProcessed: number
  totalReceived: number
  totalSent: number
  uniqueSenders: number
  totalSizeBytes: number
  neverEmailedSenders: number
  neverEmailedEmails: number
  neverEmailedSizeBytes: number
  localDbSizeBytes?: number
  candidateEmailsCount?: number
  candidateSizeBytes?: number
  candidateSendersCount?: number
}

export interface CandidateSender {
  email: string
  name: string
  candidateCount: number
  candidateSize: number
  oldestDate?: string
  newestDate?: string
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
  senderEmailCount?: number
}

export interface EmailsResponse {
  emails: EmailMetadata[]
  total: number
  limit: number
  offset: number
}
