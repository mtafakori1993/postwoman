export namespace model {
	
	export class APIKeyAuth {
	    key: string;
	    value: string;
	    addTo: string;
	
	    static createFrom(source: any = {}) {
	        return new APIKeyAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.addTo = source["addTo"];
	    }
	}
	export class AWSV4Auth {
	    accessKey: string;
	    secretKey: string;
	    sessionToken?: string;
	    region: string;
	    service: string;
	
	    static createFrom(source: any = {}) {
	        return new AWSV4Auth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.accessKey = source["accessKey"];
	        this.secretKey = source["secretKey"];
	        this.sessionToken = source["sessionToken"];
	        this.region = source["region"];
	        this.service = source["service"];
	    }
	}
	export class KeyValue {
	    key: string;
	    value: string;
	    description?: string;
	    enabled: boolean;
	    type?: string;
	    src?: string;
	    fileName?: string;
	    contentType?: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyValue(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.description = source["description"];
	        this.enabled = source["enabled"];
	        this.type = source["type"];
	        this.src = source["src"];
	        this.fileName = source["fileName"];
	        this.contentType = source["contentType"];
	    }
	}
	export class Proxy {
	    host: string;
	    port: number;
	    protocol: string;
	    username?: string;
	    password?: string;
	    bypass?: string[];
	
	    static createFrom(source: any = {}) {
	        return new Proxy(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.protocol = source["protocol"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.bypass = source["bypass"];
	    }
	}
	export class AppSettings {
	    theme: string;
	    fontSize: number;
	    editorFontSize: number;
	    editorTabSize: number;
	    requestTimeout: number;
	    sslVerification: boolean;
	    proxy?: Proxy;
	    defaultHeaders: KeyValue[];
	    historyRetentionDays: number;
	    scriptExecutionEnabled: boolean;
	    maxResponseSize: number;
	    autoSave: boolean;
	    showRequestHeaders: boolean;
	    showResponseHeaders: boolean;
	    keyboardShortcuts: Record<string, string>;
	    followRedirects: boolean;
	    sendNoCacheHeader: boolean;
    accentColor: string;
    editorFontFamily: string;
    uiFont: string;
    compactMode: boolean;
    layoutPreset: string;
    aiEnabled: boolean;
    aiProvider: string;
    aiBaseUrl: string;
    aiApiKey: string;
    aiModel: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.fontSize = source["fontSize"];
	        this.editorFontSize = source["editorFontSize"];
	        this.editorTabSize = source["editorTabSize"];
	        this.requestTimeout = source["requestTimeout"];
	        this.sslVerification = source["sslVerification"];
	        this.proxy = this.convertValues(source["proxy"], Proxy);
	        this.defaultHeaders = this.convertValues(source["defaultHeaders"], KeyValue);
	        this.historyRetentionDays = source["historyRetentionDays"];
	        this.scriptExecutionEnabled = source["scriptExecutionEnabled"];
	        this.maxResponseSize = source["maxResponseSize"];
	        this.autoSave = source["autoSave"];
	        this.showRequestHeaders = source["showRequestHeaders"];
	        this.showResponseHeaders = source["showResponseHeaders"];
	        this.keyboardShortcuts = source["keyboardShortcuts"];
	        this.followRedirects = source["followRedirects"];
	        this.sendNoCacheHeader = source["sendNoCacheHeader"];
        this.accentColor = source["accentColor"];
        this.editorFontFamily = source["editorFontFamily"];
        this.uiFont = source["uiFont"];
        this.compactMode = source["compactMode"];
        this.layoutPreset = source["layoutPreset"];
        this.aiEnabled = source["aiEnabled"];
        this.aiProvider = source["aiProvider"];
        this.aiBaseUrl = source["aiBaseUrl"];
        this.aiApiKey = source["aiApiKey"];
        this.aiModel = source["aiModel"];
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
	export class MockRoute {
	    id: string;
	    method: string;
	    path: string;
	    statusCode: number;
	    headers: KeyValue[];
	    body: string;
	    delay: number;
	    matchQuery: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MockRoute(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.method = source["method"];
	        this.path = source["path"];
	        this.statusCode = source["statusCode"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.body = source["body"];
	        this.delay = source["delay"];
	        this.matchQuery = source["matchQuery"];
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
	export class MockServer {
	    id: string;
	    name: string;
	    port: number;
	    routes: MockRoute[];
	    isRunning: boolean;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new MockServer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.port = source["port"];
	        this.routes = this.convertValues(source["routes"], MockRoute);
	        this.isRunning = source["isRunning"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class Tab {
	    id: string;
	    requestId?: string;
	    title: string;
	    isDirty: boolean;
	    isActive: boolean;
	    protocol: string;
	    request?: Request;
	    response?: Response;
	    sidebarWidth?: number;
	    responsePanelHeight?: number;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Tab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.requestId = source["requestId"];
	        this.title = source["title"];
	        this.isDirty = source["isDirty"];
	        this.isActive = source["isActive"];
	        this.protocol = source["protocol"];
	        this.request = this.convertValues(source["request"], Request);
	        this.response = this.convertValues(source["response"], Response);
	        this.sidebarWidth = source["sidebarWidth"];
	        this.responsePanelHeight = source["responsePanelHeight"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class ConsoleEntry {
	    level: string;
	    args: string[];
	
	    static createFrom(source: any = {}) {
	        return new ConsoleEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.level = source["level"];
	        this.args = source["args"];
	    }
	}
	export class TestResult {
	    name: string;
	    passed: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.passed = source["passed"];
	        this.error = source["error"];
	    }
	}
	export class Cookie {
	    name: string;
	    value: string;
	    domain?: string;
	    path?: string;
	    expires?: number;
	    maxAge?: number;
	    secure: boolean;
	    httpOnly: boolean;
	    sameSite?: string;
	    hostOnly?: boolean;
	    session?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Cookie(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.value = source["value"];
	        this.domain = source["domain"];
	        this.path = source["path"];
	        this.expires = source["expires"];
	        this.maxAge = source["maxAge"];
	        this.secure = source["secure"];
	        this.httpOnly = source["httpOnly"];
	        this.sameSite = source["sameSite"];
	        this.hostOnly = source["hostOnly"];
	        this.session = source["session"];
	    }
	}
	export class Response {
	    status: number;
	    statusText: string;
	    headers: KeyValue[];
	    cookies: Cookie[];
	    body: string;
	    bodyBase64?: string;
	    isBinary?: boolean;
	    contentType?: string;
	    bodySize: number;
	    headerSize: number;
	    responseTime: number;
	    timestamp: number;
	    redirected: boolean;
	    url?: string;
	    error?: string;
	    tests: TestResult[];
	    console: ConsoleEntry[];
	
	    static createFrom(source: any = {}) {
	        return new Response(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.statusText = source["statusText"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.cookies = this.convertValues(source["cookies"], Cookie);
	        this.body = source["body"];
	        this.bodyBase64 = source["bodyBase64"];
	        this.isBinary = source["isBinary"];
	        this.contentType = source["contentType"];
	        this.bodySize = source["bodySize"];
	        this.headerSize = source["headerSize"];
	        this.responseTime = source["responseTime"];
	        this.timestamp = source["timestamp"];
	        this.redirected = source["redirected"];
	        this.url = source["url"];
	        this.error = source["error"];
	        this.tests = this.convertValues(source["tests"], TestResult);
	        this.console = this.convertValues(source["console"], ConsoleEntry);
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
	export class HistoryEntry {
	    id: string;
	    request: Request;
	    response?: Response;
	    timestamp: number;
	    duration: number;
	    environment?: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.request = this.convertValues(source["request"], Request);
	        this.response = this.convertValues(source["response"], Response);
	        this.timestamp = source["timestamp"];
	        this.duration = source["duration"];
	        this.environment = source["environment"];
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
	export class GlobalVariable {
	    key: string;
	    value: string;
	    type: string;
	    description?: string;
	    enabled: boolean;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new GlobalVariable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.enabled = source["enabled"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class Environment {
	    id: string;
	    workspaceId: string;
	    name: string;
	    values: Variable[];
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Environment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspaceId = source["workspaceId"];
	        this.name = source["name"];
	        this.values = this.convertValues(source["values"], Variable);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class TLS {
	    cert?: string;
	    key?: string;
	    ca?: string;
	    passphrase?: string;
	    rejectUnauthorized: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TLS(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cert = source["cert"];
	        this.key = source["key"];
	        this.ca = source["ca"];
	        this.passphrase = source["passphrase"];
	        this.rejectUnauthorized = source["rejectUnauthorized"];
	    }
	}
	export class Settings {
	    followRedirects: boolean;
	    followOriginalMethod: boolean;
	    maxRedirects: number;
	    timeout: number;
	    encoding: string;
	    proxy?: Proxy;
	    tls?: TLS;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.followRedirects = source["followRedirects"];
	        this.followOriginalMethod = source["followOriginalMethod"];
	        this.maxRedirects = source["maxRedirects"];
	        this.timeout = source["timeout"];
	        this.encoding = source["encoding"];
	        this.proxy = this.convertValues(source["proxy"], Proxy);
	        this.tls = this.convertValues(source["tls"], TLS);
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
	export class Script {
	    type: string;
	    script: string;
	    disabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Script(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.script = source["script"];
	        this.disabled = source["disabled"];
	    }
	}
	export class GraphQLBody {
	    query: string;
	    variables: string;
	
	    static createFrom(source: any = {}) {
	        return new GraphQLBody(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.variables = source["variables"];
	    }
	}
	export class Body {
	    mode: string;
	    raw?: string;
	    formdata?: KeyValue[];
	    urlencoded?: KeyValue[];
	    graphql?: GraphQLBody;
	    binary?: string;
	    fileName?: string;
	
	    static createFrom(source: any = {}) {
	        return new Body(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.raw = source["raw"];
	        this.formdata = this.convertValues(source["formdata"], KeyValue);
	        this.urlencoded = this.convertValues(source["urlencoded"], KeyValue);
	        this.graphql = this.convertValues(source["graphql"], GraphQLBody);
	        this.binary = source["binary"];
	        this.fileName = source["fileName"];
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
	export class Request {
	    id: string;
	    collectionId: string;
	    folderId?: string;
	    name: string;
	    description?: string;
	    method: string;
	    url: string;
	    headers: KeyValue[];
	    queryParams: KeyValue[];
	    pathVariables: KeyValue[];
	    body: Body;
	    auth: Auth;
	    scripts: Script[];
	    settings: Settings;
	    order: number;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Request(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.collectionId = source["collectionId"];
	        this.folderId = source["folderId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.method = source["method"];
	        this.url = source["url"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.queryParams = this.convertValues(source["queryParams"], KeyValue);
	        this.pathVariables = this.convertValues(source["pathVariables"], KeyValue);
	        this.body = this.convertValues(source["body"], Body);
	        this.auth = this.convertValues(source["auth"], Auth);
	        this.scripts = this.convertValues(source["scripts"], Script);
	        this.settings = this.convertValues(source["settings"], Settings);
	        this.order = source["order"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class Folder {
	    id: string;
	    collectionId: string;
	    parentId?: string;
	    name: string;
	    description?: string;
	    order: number;
	    auth?: Auth;
	    events?: CollectionEvent[];
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Folder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.collectionId = source["collectionId"];
	        this.parentId = source["parentId"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.order = source["order"];
	        this.auth = this.convertValues(source["auth"], Auth);
	        this.events = this.convertValues(source["events"], CollectionEvent);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class CollectionScript {
	    type: string;
	    exec: string[];
	
	    static createFrom(source: any = {}) {
	        return new CollectionScript(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.exec = source["exec"];
	    }
	}
	export class CollectionEvent {
	    listen: string;
	    script: CollectionScript;
	
	    static createFrom(source: any = {}) {
	        return new CollectionEvent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.listen = source["listen"];
	        this.script = this.convertValues(source["script"], CollectionScript);
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
	export class JWTAuth {
	    token: string;
	    algorithm?: string;
	    secret?: string;
	    payload?: string;
	    headerPrefix?: string;
	
	    static createFrom(source: any = {}) {
	        return new JWTAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	        this.algorithm = source["algorithm"];
	        this.secret = source["secret"];
	        this.payload = source["payload"];
	        this.headerPrefix = source["headerPrefix"];
	    }
	}
	export class NTLMAuth {
	    username: string;
	    password: string;
	    domain?: string;
	    workstation?: string;
	
	    static createFrom(source: any = {}) {
	        return new NTLMAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	        this.domain = source["domain"];
	        this.workstation = source["workstation"];
	    }
	}
	export class HawkAuth {
	    id: string;
	    key: string;
	    algorithm: string;
	    user?: string;
	
	    static createFrom(source: any = {}) {
	        return new HawkAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.key = source["key"];
	        this.algorithm = source["algorithm"];
	        this.user = source["user"];
	    }
	}
	export class OAuth2Auth {
	    grantType: string;
	    accessToken: string;
	    refreshToken: string;
	    clientId: string;
	    clientSecret: string;
	    authUrl: string;
	    tokenUrl: string;
	    scope: string;
	    redirectUrl: string;
    expiresAt?: number;
    tokenType?: string;
    pkce?: boolean;
    codeVerifier?: string;
    state?: string;
	    username?: string;
	    password?: string;
	
	    static createFrom(source: any = {}) {
	        return new OAuth2Auth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.grantType = source["grantType"];
	        this.accessToken = source["accessToken"];
	        this.refreshToken = source["refreshToken"];
	        this.clientId = source["clientId"];
	        this.clientSecret = source["clientSecret"];
	        this.authUrl = source["authUrl"];
	        this.tokenUrl = source["tokenUrl"];
	        this.scope = source["scope"];
	        this.redirectUrl = source["redirectUrl"];
        this.expiresAt = source["expiresAt"];
        this.tokenType = source["tokenType"];
        this.pkce = source["pkce"];
        this.codeVerifier = source["codeVerifier"];
        this.state = source["state"];
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class OAuth1Auth {
	    consumerKey: string;
	    consumerSecret: string;
	    token: string;
	    tokenSecret: string;
	    signatureMethod: string;
	    version: string;
	    realm?: string;
	
	    static createFrom(source: any = {}) {
	        return new OAuth1Auth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.consumerKey = source["consumerKey"];
	        this.consumerSecret = source["consumerSecret"];
	        this.token = source["token"];
	        this.tokenSecret = source["tokenSecret"];
	        this.signatureMethod = source["signatureMethod"];
	        this.version = source["version"];
	        this.realm = source["realm"];
	    }
	}
	export class DigestAuth {
	    username: string;
	    password: string;
	    realm?: string;
	    nonce?: string;
	    algorithm?: string;
	    qop?: string;
	
	    static createFrom(source: any = {}) {
	        return new DigestAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	        this.realm = source["realm"];
	        this.nonce = source["nonce"];
	        this.algorithm = source["algorithm"];
	        this.qop = source["qop"];
	    }
	}
	export class BasicAuth {
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new BasicAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class BearerAuth {
	    token: string;
	
	    static createFrom(source: any = {}) {
	        return new BearerAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	    }
	}
	export class Auth {
	    type: string;
	    apikey?: APIKeyAuth;
	    bearer?: BearerAuth;
	    basic?: BasicAuth;
	    digest?: DigestAuth;
	    oauth1?: OAuth1Auth;
	    oauth2?: OAuth2Auth;
	    awsv4?: AWSV4Auth;
	    hawk?: HawkAuth;
	    ntlm?: NTLMAuth;
	    jwt?: JWTAuth;
	
	    static createFrom(source: any = {}) {
	        return new Auth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.apikey = this.convertValues(source["apikey"], APIKeyAuth);
	        this.bearer = this.convertValues(source["bearer"], BearerAuth);
	        this.basic = this.convertValues(source["basic"], BasicAuth);
	        this.digest = this.convertValues(source["digest"], DigestAuth);
	        this.oauth1 = this.convertValues(source["oauth1"], OAuth1Auth);
	        this.oauth2 = this.convertValues(source["oauth2"], OAuth2Auth);
	        this.awsv4 = this.convertValues(source["awsv4"], AWSV4Auth);
	        this.hawk = this.convertValues(source["hawk"], HawkAuth);
	        this.ntlm = this.convertValues(source["ntlm"], NTLMAuth);
	        this.jwt = this.convertValues(source["jwt"], JWTAuth);
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
	export class Variable {
	    key: string;
	    value: string;
	    type: string;
	    description?: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Variable(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.value = source["value"];
	        this.type = source["type"];
	        this.description = source["description"];
	        this.enabled = source["enabled"];
	    }
	}
	export class CollectionInfo {
	    _postman_id?: string;
	    name: string;
	    description?: string;
	    schema: string;
	    version?: string;
	
	    static createFrom(source: any = {}) {
	        return new CollectionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this._postman_id = source["_postman_id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.schema = source["schema"];
	        this.version = source["version"];
	    }
	}
	export class Collection {
	    id: string;
	    workspaceId: string;
	    info: CollectionInfo;
	    variables: Variable[];
	    auth: Auth;
	    events: CollectionEvent[];
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Collection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.workspaceId = source["workspaceId"];
	        this.info = this.convertValues(source["info"], CollectionInfo);
	        this.variables = this.convertValues(source["variables"], Variable);
	        this.auth = this.convertValues(source["auth"], Auth);
	        this.events = this.convertValues(source["events"], CollectionEvent);
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
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
	export class Workspace {
	    id: string;
	    name: string;
	    description?: string;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Workspace(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class AppState {
	    workspace: Workspace;
	    workspaces: Workspace[];
	    collections: Collection[];
	    folders: Folder[];
	    requests: Request[];
	    environments: Environment[];
	    activeEnvironmentId?: string;
	    globalVariables: GlobalVariable[];
	    history: HistoryEntry[];
	    settings: AppSettings;
	    tabs: Tab[];
	    mockServers: MockServer[];
	    version: string;
	    platform: string;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.workspace = this.convertValues(source["workspace"], Workspace);
	        this.workspaces = this.convertValues(source["workspaces"], Workspace);
	        this.collections = this.convertValues(source["collections"], Collection);
	        this.folders = this.convertValues(source["folders"], Folder);
	        this.requests = this.convertValues(source["requests"], Request);
	        this.environments = this.convertValues(source["environments"], Environment);
	        this.activeEnvironmentId = source["activeEnvironmentId"];
	        this.globalVariables = this.convertValues(source["globalVariables"], GlobalVariable);
	        this.history = this.convertValues(source["history"], HistoryEntry);
	        this.settings = this.convertValues(source["settings"], AppSettings);
	        this.tabs = this.convertValues(source["tabs"], Tab);
	        this.mockServers = this.convertValues(source["mockServers"], MockServer);
	        this.version = source["version"];
	        this.platform = source["platform"];
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
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	
	export class ImportSummary {
	    format: string;
	    collections: string[];
	    environments: string[];
	    warnings: string[];
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.format = source["format"];
	        this.collections = source["collections"];
	        this.environments = source["environments"];
	        this.warnings = source["warnings"];
	        this.message = source["message"];
	    }
	}
	export class RequestResult {
	    requestId: string;
	    requestName: string;
	    status: string;
	    response?: Response;
	    tests: TestResult[];
	    duration: number;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RequestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.requestId = source["requestId"];
	        this.requestName = source["requestName"];
	        this.status = source["status"];
	        this.response = this.convertValues(source["response"], Response);
	        this.tests = this.convertValues(source["tests"], TestResult);
	        this.duration = source["duration"];
	        this.error = source["error"];
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
	export class IterationResult {
	    iteration: number;
	    requests: RequestResult[];
	    startedAt: number;
	    completedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new IterationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.iteration = source["iteration"];
	        this.requests = this.convertValues(source["requests"], RequestResult);
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
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
	
	
	export class MockRequestLog {
	    id: string;
	    method: string;
	    path: string;
	    headers: KeyValue[];
	    body: string;
	    status: number;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new MockRequestLog(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.method = source["method"];
	        this.path = source["path"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.body = source["body"];
	        this.status = source["status"];
	        this.timestamp = source["timestamp"];
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
	
	
	
	
	
	
	
	
	
	export class RunOptions {
	    collectionId: string;
	    folderId?: string;
	    requestIds?: string[];
	    environmentId?: string;
	    iterations: number;
	    delay: number;
	    dataFile?: string;
	    stopOnError: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RunOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.collectionId = source["collectionId"];
	        this.folderId = source["folderId"];
	        this.requestIds = source["requestIds"];
	        this.environmentId = source["environmentId"];
	        this.iterations = source["iterations"];
	        this.delay = source["delay"];
	        this.dataFile = source["dataFile"];
	        this.stopOnError = source["stopOnError"];
	    }
	}
	export class RunResult {
	    id: string;
	    collectionId: string;
	    startedAt: number;
	    completedAt?: number;
	    status: string;
	    totalRequests: number;
	    passedTests: number;
	    failedTests: number;
	    totalTests: number;
	    iterations: IterationResult[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new RunResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.collectionId = source["collectionId"];
	        this.startedAt = source["startedAt"];
	        this.completedAt = source["completedAt"];
	        this.status = source["status"];
	        this.totalRequests = source["totalRequests"];
	        this.passedTests = source["passedTests"];
	        this.failedTests = source["failedTests"];
	        this.totalTests = source["totalTests"];
	        this.iterations = this.convertValues(source["iterations"], IterationResult);
	        this.error = source["error"];
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
	
	export class SearchResult {
	    type: string;
	    id: string;
	    name: string;
	    parent?: string;
	    match: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.parent = source["parent"];
	        this.match = source["match"];
	        this.score = source["score"];
	    }
	}
	export class SendRequestInput {
	    request: Request;
	    environment: Record<string, string>;
	    environmentId?: string;
	    globals: Record<string, string>;
	    collectionVariables: Record<string, string>;
	    runScripts: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SendRequestInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.request = this.convertValues(source["request"], Request);
	        this.environment = source["environment"];
	        this.environmentId = source["environmentId"];
	        this.globals = source["globals"];
	        this.collectionVariables = source["collectionVariables"];
	        this.runScripts = source["runScripts"];
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
	
	
	
	
	
	export class WSMessage {
	    id: string;
	    type: string;
	    data: string;
	    isBinary: boolean;
	    timestamp: number;
	
	    static createFrom(source: any = {}) {
	        return new WSMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.data = source["data"];
	        this.isBinary = source["isBinary"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class WSConnection {
	    id: string;
	    url: string;
	    protocols: string[];
	    headers: KeyValue[];
	    isConnected: boolean;
	    messages: WSMessage[];
	    createdAt: number;
	
	    static createFrom(source: any = {}) {
	        return new WSConnection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.protocols = source["protocols"];
	        this.headers = this.convertValues(source["headers"], KeyValue);
	        this.isConnected = source["isConnected"];
	        this.messages = this.convertValues(source["messages"], WSMessage);
	        this.createdAt = source["createdAt"];
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
	


	export class ExtractionResult {
	    path: string;
	    type: string;
	    value: string;
	    display: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtractionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.type = source["type"];
	        this.value = source["value"];
	        this.display = source["display"];
	    }
	}
	export class JSONDiffEntry {
	    path: string;
	    kind: string;
	    left?: string;
	    right?: string;
	    leftType?: string;
	    rightType?: string;
	
	    static createFrom(source: any = {}) {
	        return new JSONDiffEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.left = source["left"];
	        this.right = source["right"];
	        this.leftType = source["leftType"];
	        this.rightType = source["rightType"];
	    }
	}
	export class JSONDiff {
	    equal: boolean;
	    added: number;
	    removed: number;
	    changed: number;
	    truncated: boolean;
	    entries: JSONDiffEntry[];
	
	    static createFrom(source: any = {}) {
	        return new JSONDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.equal = source["equal"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.changed = source["changed"];
	        this.truncated = source["truncated"];
	        this.entries = this.convertValues(source["entries"], JSONDiffEntry);
	    }
	
		convertValues(value: any, classs: any, asMap: boolean = false): any {
		    if (!value) return value;
		    if (asMap) {
		        for (const key of Object.keys(value)) {
		            value[key] = new classs(value[key]);
		        }
		        return value;
		    }
		    if (Array.isArray(value)) {
		        if (!Array.isArray(classs)) return value;
		        const a = [];
		        for (const item of value) {
		            a.push(new classs[0](item));
		        }
		        return a;
		    }
		    if ("object" === typeof value) {
		        return new classs(value);
		    }
		    return value;
		}
	}
	export class StoredCookie {
	    url: string;
	    cookie: Cookie;
	
	    static createFrom(source: any = {}) {
	        return new StoredCookie(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.cookie = this.convertValues(source["cookie"], Cookie);
	    }
	
		convertValues(value: any, classs: any, asMap: boolean = false): any {
		    if (!value) return value;
		    if (asMap) {
		        for (const key of Object.keys(value)) {
		            value[key] = new classs(value[key]);
		        }
		        return value;
		    }
		    if (Array.isArray(value)) {
		        if (!Array.isArray(classs)) return value;
		        const a = [];
		        for (const item of value) {
		            a.push(new classs[0](item));
		        }
		        return a;
		    }
		    if ("object" === typeof value) {
		        return new classs(value);
		    }
		    return value;
		}
	}

	export class AIRequest {
	    prompt: string;
	    requestJson?: string;
	    responseBody?: string;
	    systemPrompt?: string;
	
	    static createFrom(source: any = {}) {
	        return new AIRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.prompt = source["prompt"];
	        this.requestJson = source["requestJson"];
	        this.responseBody = source["responseBody"];
	        this.systemPrompt = source["systemPrompt"];
	    }
	}
	export class OAuth2StartResult {
	    auth: OAuth2Auth;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new OAuth2StartResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.auth = this.convertValues(source["auth"], OAuth2Auth);
	        this.url = source["url"];
	    }
	
		convertValues(value: any, classs: any, asMap: boolean = false): any {
		    if (!value) return value;
		    if (asMap) {
		        for (const key of Object.keys(value)) {
		            value[key] = new classs(value[key]);
		        }
		        return value;
		    }
		    if (Array.isArray(value)) {
		        if (!Array.isArray(classs)) return value;
		        const a = [];
		        for (const item of value) {
		            a.push(new classs[0](item));
		        }
		        return a;
		    }
		    if ("object" === typeof value) {
		        return new classs(value);
		    }
		    return value;
		}
	}

}

