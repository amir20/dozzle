export interface NotificationRule {
  id: number;
  name: string;
  enabled: boolean;
  containerExpression: string;
  logExpression: string;
  triggerCount: number;
  triggeredContainers: number;
  lastTriggeredAt: string | null;
  dispatcher: Dispatcher | null;
}

export interface Dispatcher {
  id: number;
  name: string;
  type: string;
  url?: string;
  template?: string;
  prefix?: string;
  expiresAt?: string;
}

export interface NotificationRuleInput {
  name: string;
  enabled: boolean;
  dispatcherId: number;
  logExpression: string;
  containerExpression: string;
}

export interface PreviewResult {
  containerError?: string;
  logError?: string;
  matchedContainers: {
    id: string;
    name: string;
    image: string;
    host: string;
  }[];
  matchedLogs: {
    id: number;
    t: string;
    m: unknown;
    rm: string;
    ts: number;
    l: string;
    s: string;
  }[];
  totalLogs: number;
  messageKeys?: string[];
}

export interface TestWebhookResult {
  success: boolean;
  statusCode?: number;
  error?: string;
}
