export interface NotificationRule {
  id: number;
  name: string;
  enabled: boolean;
  containerExpression: string;
  logExpression: string;
  metricExpression?: string;
  eventExpression?: string;
  cooldown?: number;
  sampleWindow?: number;
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
  headers?: Record<string, string>;
  prefix?: string;
  expiresAt?: string;
}

export interface NotificationRuleInput {
  name: string;
  enabled: boolean;
  dispatcherId: number;
  logExpression: string;
  containerExpression: string;
  metricExpression?: string;
  eventExpression?: string;
  cooldown?: number;
  sampleWindow?: number;
}

export interface PreviewResult {
  containerError?: string;
  logError?: string;
  metricError?: string;
  eventError?: string;
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
  /** How many matched containers were actually read for logs (the scan is capped). */
  scannedContainers: number;
  /** How far back the log preview looked, in seconds. */
  logWindowSeconds: number;
  metricSamples?: MetricSample[];
  eventSamples?: EventSample[];
}

/** A dry run of the metric expression against one container's buffered stats. */
export interface MetricSample {
  containerId: string;
  name: string;
  host: string;
  cpu: number;
  memory: number;
  memoryUsage: number;
  /** Whether the most recent sample matches. */
  matches: boolean;
  matchedSamples: number;
  totalSamples: number;
  /** Whether a full sample window is matched well enough to fire. */
  wouldTrigger: boolean;
}

/** A representative container event, tagged with whether the event expression matches it. */
export interface EventSample {
  name: string;
  attributes?: Record<string, string>;
  matches: boolean;
}

export interface TestWebhookResult {
  success: boolean;
  statusCode?: number;
  error?: string;
}

export interface CloudConfig {
  prefix: string;
  expiresAt?: string;
  linked: boolean;
  streamLogs: boolean;
}

export interface CloudStatus {
  user: { email: string; name: string };
  plan: {
    name: string;
    events_per_month: number;
    agent_messages_per_month?: number;
    log_bytes_per_month?: number;
    retention_days: number;
  };
  // Agent chats and indexed log bytes are metered too, but a cloud older than they are
  // does not send them, so everything past events is optional and rendered only when present.
  usage: {
    events_used: number;
    events_limit: number;
    agent_messages_used?: number;
    agent_messages_limit?: number;
    log_bytes_used?: number;
    log_bytes_limit?: number;
    period: string;
  };
}
