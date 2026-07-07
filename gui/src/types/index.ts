export interface Session {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
  is_active: boolean;
  description: string;
}

export interface Request {
  id: string;
  session_id: string;
  created_at: string;
  updated_at: string;
  
  method: string;
  scheme: string;
  host: string;
  port: number;
  path: string;
  url: string;
  query_string: string;
  http_version: string;
  
  headers: Record<string, string[]> | null;
  cookies: Record<string, string> | null;
  is_websocket: boolean;
  content_type: string;
  body: string;
  body_size: number;
  
  status_code: number;
  status_reason: string;
  resp_headers: Record<string, string[]> | null;
  resp_content_type: string;
  resp_body: string;
  resp_body_size: number;
  
  duration: number;
  remote_addr: string;
  client_addr: string;
  server_addr: string;
  notes: string;
  error: string;
  request_start_time: string;
  request_end_time: string;
  response_start_time: string;
  response_end_time: string;
  dns_duration: number;
  connect_duration: number;
  tls_handshake_duration: number;
  request_duration: number;
  response_duration: number;
  latency_duration: number;
  keep_alive: boolean;
  tls_version: string;
  tls_cipher_suite: string;
  tls_server_name: string;
  tls_did_resume: boolean;
  tls_alpn: string;
  tls_curve_id: string;
  tls_ocsp_stapled: boolean;
  tls_sct_count: number;
  tls_server_certificates: TLSCertificate[] | null;
  tls_server_extensions: TLSExtension[] | null;
  websocket_frames: WebSocketFrame[] | null;
}

export interface WebSocketFrame {
  id: string;
  direction: string;
  opcode: number;
  frame_type: string;
  payload: string;
  payload_size: number;
  created_at: string;
  fin: boolean;
  masked: boolean;
}

export interface TLSExtension {
  id: number;
  name: string;
  value: string;
}

export interface TLSCertificate {
  subject_common_name: string;
  subject: string;
  issuer_common_name: string;
  issuer: string;
  serial_number: string;
  dns_names: string[];
  email_addresses: string[];
  ip_addresses: string[];
  version: number;
  is_ca: boolean;
  signature_algorithm: string;
  public_key_algorithm: string;
  not_before: string;
  not_after: string;
  ocsp_servers: string[];
  issuing_certificate_url: string[];
  extensions: TLSExtension[];
}

export interface AgentAnalysis {
  id: string;
  request_id: string;
  created_at: string;
  target_field: string;
  target_location: string;
  target_value: string;
  query: string;
  result: string;
  model: string;
  tokens_used: number;
}

export interface AnalysisRequest {
  request_id?: string;
  target_field: string;
  target_location: string;
  target_value: string;
  query?: string;
  model_id?: string;
  session_id?: string;
}

export interface AgentEvent {
  depth: number;
  type: 'start' | 'thinking' | 'thought' | 'action' | 'observation' | 'final' | 'warning' | 'decision' | 'text_delta' | 'provider_retry' | 'intervention_applied';
  content?: string;
  tool_name?: string;
  tool_call_id?: string;
  arguments?: string;
  result?: string;
  request_ids?: string[];
  tool_calls: number;
  created_at: string;
  specialist_name?: string;
  error_category?: string;
  error_tool_name?: string;
  error_timeout?: string;
  error_recovered?: boolean;
  retry_attempt?: number;
  retry_max?: number;
  intervention_id?: string;
}

export interface TokenUsage {
  input_tokens: number;
  output_tokens: number;
  cache_creation_input_tokens?: number;
  cache_read_input_tokens?: number;
  total_tokens?: number;
}

export interface SessionContextStats {
  code: number;
  session_id: string;
  has_history: boolean;
  message_count: number;
  estimated_tokens: number;
  active_model: string;
  active_provider: string;
  active_max_tokens: number;
  active_temperature: number;
  available_models: number;
}

export interface AgentResult {
  request_id: string;
  session_id?: string;
  model: string;
  final_answer: string;
  trace_chain: AgentTraceStep[];
  depth_used: number;
  tool_calls: number;
  stopped_early: boolean;
  stop_reason?: string;
  token_usage?: TokenUsage;
  provenance?: ProvenanceChain;
}

export interface AgentTraceStep {
  depth: number;
  type: string;
  content?: string;
  tool_name?: string;
  tool_arguments?: string;
  tool_result?: string;
  request_ids?: string[];
  created_at: string;
}

export interface ParamArtifact {
  location: string;
  name: string;
  value: string;
  path?: string;
  raw_type?: string;
}

export interface ProvenanceLink {
  source_request_id: string;
  source_artifact: ParamArtifact;
  target_request_id: string;
  target_artifact: ParamArtifact;
  confidence: number;
  same_session: boolean;
  same_host: boolean;
  time_delta_ms: number;
  transform_type?: string;
  semantic_similarity: number;
}

export interface ProvenanceChain {
  target_request_id: string;
  target_artifact: ParamArtifact;
  links: ProvenanceLink[];
  confidence: number;
  evidence: string[];
}

export interface AgentModel {
  id: string;
  name: string;
  provider: string;
  description: string;
  max_tokens: number;
  supports_streaming: boolean;
}

export interface ProviderInfo {
  id: string;
  name: string;
  api_type: string;
  icon?: string;
  has_api_key: boolean;
  has_base_url: boolean;
  model_count: number;
  is_active: boolean;
  deletable: boolean;
}

export interface ModelsByProvider {
  provider: string;
  provider_name: string;
  provider_icon?: string;
  models: AgentModel[];
}

export interface ModelListResponse {
  models: AgentModel[];
  grouped: ModelsByProvider[];
  default_model: string;
  active_model?: string;
  active_provider?: string;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data?: T;
}

export interface PaginatedData<T> {
  total: number;
  page: number;
  page_size: number;
  items: T[];
}

export interface AppSettings {
  cert: {
    ca_cert: string;
    ca_key: string;
    organization: string;
  };
  proxy: {
    listener: {
      http_enabled: boolean;
      port: number;
      https_enabled: boolean;
      mitm_enabled: boolean;
      socks5_enabled: boolean;
      socks5_username: string;
      auto_start_on_boot: boolean;
    };
    recording: {
      enabled: boolean;
      max_capture_body_size_mb: number;
    };
    ssl_proxying: {
      enabled: boolean;
      include_hosts: string[];
      exclude_hosts: string[];
    };
    access_control: {
      enabled: boolean;
      allowed_clients: string[];
    };
    external_proxy: {
      enabled: boolean;
      scheme: string;
      host: string;
      port: number;
      username: string;
      password?: string;
      password_configured?: boolean;
      bypass_hosts: string[];
    };
    throttling: {
      enabled: boolean;
      latency_ms: number;
      downstream_kbps: number;
      upstream_kbps: number;
    };
    breakpoints: {
      enabled: boolean;
      request_matchers: string[];
      response_matchers: string[];
    };
    reverse_proxy: {
      enabled: boolean;
      rules: Array<Record<string, unknown>>;
    };
    port_forwarding: {
      enabled: boolean;
      rules: Array<Record<string, unknown>>;
    };
    web_interface: {
      enabled: boolean;
      port: number;
    };
  };
  tools: {
    no_caching: boolean;
    block_cookies: boolean;
    map_remote_enabled: boolean;
    map_local_enabled: boolean;
    rewrite_enabled: boolean;
    block_list_enabled: boolean;
    dns_spoofing: boolean;
    mirror_enabled: boolean;
    auto_save_enabled: boolean;
    client_process: boolean;
  };
  window: {
    structure_view: boolean;
    use_dark_theme: boolean;
  };
  mcp_server: {
    enabled: boolean;
    host: string;
    port: number;
  };
}

export interface UpdateInfo {
  current_version: string;
  latest_version: string;
  release_notes: string;
  published_at: string;
  download_url: string;
  asset_size: number;
  has_update: boolean;
}

export interface UpdateProgress {
  downloaded: number;
  total: number;
  percent: number;
}
