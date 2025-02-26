// src/types.ts

export interface OAuthToken {
  access_token: string
  token_type: string
  refresh_token?: string
  expiry?: string
  expires_in?: number
  scope?: string
}

export interface Email {
  id: string
  threadId?: string
  labelIds?: string[]
  snippet?: string
  historyId?: string
  internalDate?: string
  payload?: EmailPayload
  sizeEstimate?: number
  showDetails?: boolean // UI state property
}

export interface EmailPayload {
  partId?: string
  mimeType?: string
  filename?: string
  headers?: EmailHeader[]
  body?: EmailBody
  parts?: EmailPart[]
}

export interface EmailHeader {
  name: string
  value: string
}

export interface EmailBody {
  attachmentId?: string
  size?: number
  data?: string
}

export interface EmailPart {
  partId?: string
  mimeType?: string
  filename?: string
  headers?: EmailHeader[]
  body?: EmailBody
  parts?: EmailPart[]
}

export interface GmailApiResponse {
  messages?: Email[]
  nextPageToken?: string
  resultSizeEstimate?: number
}
