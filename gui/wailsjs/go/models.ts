export namespace bindings {
	
	export class AnalyzeRequest {
	    request_id: string;
	    session_id: string;
	    query: string;
	    model_id: string;
	
	    static createFrom(source: any = {}) {
	        return new AnalyzeRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.request_id = source["request_id"];
	        this.session_id = source["session_id"];
	        this.query = source["query"];
	        this.model_id = source["model_id"];
	    }
	}
	export class AnalyzeResponse {
	    code: number;
	    message?: string;
	    analysis_id?: string;
	    model_id?: string;
	
	    static createFrom(source: any = {}) {
	        return new AnalyzeResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.analysis_id = source["analysis_id"];
	        this.model_id = source["model_id"];
	    }
	}
	export class ComposeRequestOptions {
	    method: string;
	    url: string;
	    headers: Record<string, Array<string>>;
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new ComposeRequestOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method = source["method"];
	        this.url = source["url"];
	        this.headers = source["headers"];
	        this.body = source["body"];
	    }
	}
	export class CreateSessionRequest {
	    name: string;
	    description: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateSessionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class DeleteAgentProviderRequest {
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteAgentProviderRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	    }
	}
	export class DeleteModelRequest {
	    provider: string;
	    id: string;
	
	    static createFrom(source: any = {}) {
	        return new DeleteModelRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.id = source["id"];
	    }
	}
	export class ReplayRequestOptions {
	    headers: Record<string, Array<string>>;
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new ReplayRequestOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.headers = source["headers"];
	        this.body = source["body"];
	    }
	}
	export class RequestListOptions {
	    session_id: string;
	    host: string;
	    method: string;
	    search: string;
	    sort_by: string;
	    sort_order: string;
	    status_code: number;
	
	    static createFrom(source: any = {}) {
	        return new RequestListOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	        this.host = source["host"];
	        this.method = source["method"];
	        this.search = source["search"];
	        this.sort_by = source["sort_by"];
	        this.sort_order = source["sort_order"];
	        this.status_code = source["status_code"];
	    }
	}
	export class SessionContextStats {
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
	
	    static createFrom(source: any = {}) {
	        return new SessionContextStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.session_id = source["session_id"];
	        this.has_history = source["has_history"];
	        this.message_count = source["message_count"];
	        this.estimated_tokens = source["estimated_tokens"];
	        this.active_model = source["active_model"];
	        this.active_provider = source["active_provider"];
	        this.active_max_tokens = source["active_max_tokens"];
	        this.active_temperature = source["active_temperature"];
	        this.available_models = source["available_models"];
	    }
	}
	export class SessionResponse {
	    code: number;
	    message?: string;
	    data?: any;
	
	    static createFrom(source: any = {}) {
	        return new SessionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.data = source["data"];
	    }
	}
	export class UpdateAgentConfigRequest {
	    provider?: string;
	    api_type?: string;
	    api_key?: string;
	    base_url?: string;
	    model?: string;
	    max_tokens?: number;
	    temperature?: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateAgentConfigRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.api_type = source["api_type"];
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	        this.model = source["model"];
	        this.max_tokens = source["max_tokens"];
	        this.temperature = source["temperature"];
	    }
	}
	export class UpdateSessionRequest {
	    name?: string;
	    description?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSessionRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class UpsertAgentProviderRequest {
	    id: string;
	    name?: string;
	    api_type: string;
	    api_key?: string;
	    base_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpsertAgentProviderRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.api_type = source["api_type"];
	        this.api_key = source["api_key"];
	        this.base_url = source["base_url"];
	    }
	}
	export class UpsertModelRequest {
	    provider: string;
	    id: string;
	    name: string;
	    context: number;
	    output: number;
	
	    static createFrom(source: any = {}) {
	        return new UpsertModelRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.context = source["context"];
	        this.output = source["output"];
	    }
	}

}

export namespace config {
	
	export class AccessControlSettings {
	    enabled: boolean;
	    allowed_clients: string[];
	
	    static createFrom(source: any = {}) {
	        return new AccessControlSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.allowed_clients = source["allowed_clients"];
	    }
	}
	export class MCPServerConfig {
	    name: string;
	    command: string;
	    args: string[];
	    env?: Record<string, string>;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MCPServerConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.env = source["env"];
	        this.enabled = source["enabled"];
	    }
	}
	export class MCPSettings {
	    servers: MCPServerConfig[];
	
	    static createFrom(source: any = {}) {
	        return new MCPSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.servers = this.convertValues(source["servers"], MCPServerConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CertSettings {
	    ca_cert: string;
	    ca_key: string;
	    organization: string;
	
	    static createFrom(source: any = {}) {
	        return new CertSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ca_cert = source["ca_cert"];
	        this.ca_key = source["ca_key"];
	        this.organization = source["organization"];
	    }
	}
	export class WindowSettings {
	    structure_view: boolean;
	    use_dark_theme: boolean;
	
	    static createFrom(source: any = {}) {
	        return new WindowSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.structure_view = source["structure_view"];
	        this.use_dark_theme = source["use_dark_theme"];
	    }
	}
	export class ToolSettings {
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
	
	    static createFrom(source: any = {}) {
	        return new ToolSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.no_caching = source["no_caching"];
	        this.block_cookies = source["block_cookies"];
	        this.map_remote_enabled = source["map_remote_enabled"];
	        this.map_local_enabled = source["map_local_enabled"];
	        this.rewrite_enabled = source["rewrite_enabled"];
	        this.block_list_enabled = source["block_list_enabled"];
	        this.dns_spoofing = source["dns_spoofing"];
	        this.mirror_enabled = source["mirror_enabled"];
	        this.auto_save_enabled = source["auto_save_enabled"];
	        this.client_process = source["client_process"];
	    }
	}
	export class WebInterfaceSettings {
	    enabled: boolean;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new WebInterfaceSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.port = source["port"];
	    }
	}
	export class PortForwardRule {
	    listen_host: string;
	    listen_port: number;
	    target_host: string;
	    target_port: number;
	
	    static createFrom(source: any = {}) {
	        return new PortForwardRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listen_host = source["listen_host"];
	        this.listen_port = source["listen_port"];
	        this.target_host = source["target_host"];
	        this.target_port = source["target_port"];
	    }
	}
	export class PortForwardingSettings {
	    enabled: boolean;
	    rules: PortForwardRule[];
	
	    static createFrom(source: any = {}) {
	        return new PortForwardingSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.rules = this.convertValues(source["rules"], PortForwardRule);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReverseProxyRule {
	    source: string;
	    target: string;
	
	    static createFrom(source: any = {}) {
	        return new ReverseProxyRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.target = source["target"];
	    }
	}
	export class ReverseProxySettings {
	    enabled: boolean;
	    rules: ReverseProxyRule[];
	
	    static createFrom(source: any = {}) {
	        return new ReverseProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.rules = this.convertValues(source["rules"], ReverseProxyRule);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BreakpointSettings {
	    enabled: boolean;
	    request_matchers: string[];
	    response_matchers: string[];
	
	    static createFrom(source: any = {}) {
	        return new BreakpointSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.request_matchers = source["request_matchers"];
	        this.response_matchers = source["response_matchers"];
	    }
	}
	export class ThrottlingSettings {
	    enabled: boolean;
	    latency_ms: number;
	    downstream_kbps: number;
	    upstream_kbps: number;
	
	    static createFrom(source: any = {}) {
	        return new ThrottlingSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.latency_ms = source["latency_ms"];
	        this.downstream_kbps = source["downstream_kbps"];
	        this.upstream_kbps = source["upstream_kbps"];
	    }
	}
	export class ExternalProxySettings {
	    enabled: boolean;
	    scheme: string;
	    host: string;
	    port: number;
	    username: string;
	    password?: string;
	    bypass_hosts: string[];
	
	    static createFrom(source: any = {}) {
	        return new ExternalProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.scheme = source["scheme"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.bypass_hosts = source["bypass_hosts"];
	    }
	}
	export class SSLProxyingSettings {
	    enabled: boolean;
	    include_hosts: string[];
	    exclude_hosts: string[];
	
	    static createFrom(source: any = {}) {
	        return new SSLProxyingSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.include_hosts = source["include_hosts"];
	        this.exclude_hosts = source["exclude_hosts"];
	    }
	}
	export class RecordingSettings {
	    enabled: boolean;
	    max_capture_body_size_mb: number;
	
	    static createFrom(source: any = {}) {
	        return new RecordingSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.max_capture_body_size_mb = source["max_capture_body_size_mb"];
	    }
	}
	export class ProxyListenerSettings {
	    http_enabled: boolean;
	    port: number;
	    https_enabled: boolean;
	    mitm_enabled: boolean;
	    socks5_enabled: boolean;
	    auto_start_on_boot: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProxyListenerSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.http_enabled = source["http_enabled"];
	        this.port = source["port"];
	        this.https_enabled = source["https_enabled"];
	        this.mitm_enabled = source["mitm_enabled"];
	        this.socks5_enabled = source["socks5_enabled"];
	        this.auto_start_on_boot = source["auto_start_on_boot"];
	    }
	}
	export class ProxySettings {
	    listener: ProxyListenerSettings;
	    recording: RecordingSettings;
	    ssl_proxying: SSLProxyingSettings;
	    access_control: AccessControlSettings;
	    external_proxy: ExternalProxySettings;
	    throttling: ThrottlingSettings;
	    breakpoints: BreakpointSettings;
	    reverse_proxy: ReverseProxySettings;
	    port_forwarding: PortForwardingSettings;
	    web_interface: WebInterfaceSettings;
	
	    static createFrom(source: any = {}) {
	        return new ProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listener = this.convertValues(source["listener"], ProxyListenerSettings);
	        this.recording = this.convertValues(source["recording"], RecordingSettings);
	        this.ssl_proxying = this.convertValues(source["ssl_proxying"], SSLProxyingSettings);
	        this.access_control = this.convertValues(source["access_control"], AccessControlSettings);
	        this.external_proxy = this.convertValues(source["external_proxy"], ExternalProxySettings);
	        this.throttling = this.convertValues(source["throttling"], ThrottlingSettings);
	        this.breakpoints = this.convertValues(source["breakpoints"], BreakpointSettings);
	        this.reverse_proxy = this.convertValues(source["reverse_proxy"], ReverseProxySettings);
	        this.port_forwarding = this.convertValues(source["port_forwarding"], PortForwardingSettings);
	        this.web_interface = this.convertValues(source["web_interface"], WebInterfaceSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppSettings {
	    proxy: ProxySettings;
	    tools: ToolSettings;
	    window: WindowSettings;
	    cert: CertSettings;
	    mcp: MCPSettings;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxy = this.convertValues(source["proxy"], ProxySettings);
	        this.tools = this.convertValues(source["tools"], ToolSettings);
	        this.window = this.convertValues(source["window"], WindowSettings);
	        this.cert = this.convertValues(source["cert"], CertSettings);
	        this.mcp = this.convertValues(source["mcp"], MCPSettings);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	

}

export namespace storage {
	
	export class FindInSessionOptions {
	    session_id: string;
	    query: string;
	    is_regex: boolean;
	    is_case_sensitive: boolean;
	    is_whole_word: boolean;
	    include_req_url: boolean;
	    include_req_header: boolean;
	    include_req_body: boolean;
	    include_resp_header: boolean;
	    include_resp_body: boolean;
	    include_notes: boolean;
	    include_error: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FindInSessionOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session_id = source["session_id"];
	        this.query = source["query"];
	        this.is_regex = source["is_regex"];
	        this.is_case_sensitive = source["is_case_sensitive"];
	        this.is_whole_word = source["is_whole_word"];
	        this.include_req_url = source["include_req_url"];
	        this.include_req_header = source["include_req_header"];
	        this.include_req_body = source["include_req_body"];
	        this.include_resp_header = source["include_resp_header"];
	        this.include_resp_body = source["include_resp_body"];
	        this.include_notes = source["include_notes"];
	        this.include_error = source["include_error"];
	    }
	}

}

