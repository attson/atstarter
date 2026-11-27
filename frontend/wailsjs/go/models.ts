export namespace docker {

	export class ComposeService {
	    name: string;
	    state: string;
	    image: string;
	    ports: string[];

	    static createFrom(source: any = {}) {
	        return new ComposeService(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.state = source["state"];
	        this.image = source["image"];
	        this.ports = source["ports"];
	    }
	}
	export class ContainerState {
	    id: string;
	    name: string;
	    image: string;
	    state: string;
	    status: string;
	    compose: string;
	    service: string;
	    composeWorkingDir: string;
	    ports: string[];

	    static createFrom(source: any = {}) {
	        return new ContainerState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.image = source["image"];
	        this.state = source["state"];
	        this.status = source["status"];
	        this.compose = source["compose"];
	        this.service = source["service"];
	        this.composeWorkingDir = source["composeWorkingDir"];
	        this.ports = source["ports"];
	    }
	}
	export class Info {
	    available: boolean;
	    version: string;
	    reason: string;

	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.version = source["version"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace filetree {

	export class Entry {
	    name: string;
	    isDir: boolean;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	    }
	}
	export class FileBytes {
	    data: number[];
	    modTime: number;
	    isBinary: boolean;
	    truncatedAt?: number;

	    static createFrom(source: any = {}) {
	        return new FileBytes(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.data = source["data"];
	        this.modTime = source["modTime"];
	        this.isBinary = source["isBinary"];
	        this.truncatedAt = source["truncatedAt"];
	    }
	}
	export class FileContent {
	    content: string;
	    size: number;
	    truncated: boolean;
	    binary: boolean;

	    static createFrom(source: any = {}) {
	        return new FileContent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	        this.size = source["size"];
	        this.truncated = source["truncated"];
	        this.binary = source["binary"];
	    }
	}
	export class FileMetaInfo {
	    size: number;
	    modTime: number;
	    isDir: boolean;
	    isBinary: boolean;

	    static createFrom(source: any = {}) {
	        return new FileMetaInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	        this.isDir = source["isDir"];
	        this.isBinary = source["isBinary"];
	    }
	}
	export class SearchMatch {
	    path: string;
	    name: string;
	    isDir: boolean;

	    static createFrom(source: any = {}) {
	        return new SearchMatch(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.isDir = source["isDir"];
	    }
	}
	export class SearchResults {
	    matches: SearchMatch[];
	    truncated: boolean;

	    static createFrom(source: any = {}) {
	        return new SearchResults(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.matches = this.convertValues(source["matches"], SearchMatch);
	        this.truncated = source["truncated"];
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

export namespace git {

	export class Branches {
	    repo: boolean;
	    branch: string;
	    detached: boolean;
	    dirty: boolean;
	    head: string;
	    local: string[];
	    remote: string[];

	    static createFrom(source: any = {}) {
	        return new Branches(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repo = source["repo"];
	        this.branch = source["branch"];
	        this.detached = source["detached"];
	        this.dirty = source["dirty"];
	        this.head = source["head"];
	        this.local = source["local"];
	        this.remote = source["remote"];
	    }
	}

}

export namespace main {

	export class AppSettings {
	    language: string;
	    shell: string;
	    autoShell: string;
	    availableShells: string[];
	    logFontFamily: string;
	    logFontSize: number;
	    logLetterSpacing: number;

	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.language = source["language"];
	        this.shell = source["shell"];
	        this.autoShell = source["autoShell"];
	        this.availableShells = source["availableShells"];
	        this.logFontFamily = source["logFontFamily"];
	        this.logFontSize = source["logFontSize"];
	        this.logLetterSpacing = source["logLetterSpacing"];
	    }
	}
	export class CommandInput {
	    id: string;
	    name: string;
	    line: string;
	    cwd: string;
	    env: Record<string, string>;
	    isDefault: boolean;

	    static createFrom(source: any = {}) {
	        return new CommandInput(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.line = source["line"];
	        this.cwd = source["cwd"];
	        this.env = source["env"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class LaunchResult {
	    projectId: string;
	    commandId: string;
	    runId: string;
	    error: string;

	    static createFrom(source: any = {}) {
	        return new LaunchResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.commandId = source["commandId"];
	        this.runId = source["runId"];
	        this.error = source["error"];
	    }
	}
	export class PackageScript {
	    name: string;
	    script: string;

	    static createFrom(source: any = {}) {
	        return new PackageScript(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.script = source["script"];
	    }
	}
	export class UpdateState {
	    current: string;
	    latest: string;
	    available: boolean;
	    notes: string;
	    checking: boolean;
	    lastCheckAt: number;
	    downloading: boolean;
	    downloadPct: number;
	    ready: boolean;
	    error: string;
	    assetUrl: string;
	    assetSize: number;
	    canInstall: boolean;

	    static createFrom(source: any = {}) {
	        return new UpdateState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.available = source["available"];
	        this.notes = source["notes"];
	        this.checking = source["checking"];
	        this.lastCheckAt = source["lastCheckAt"];
	        this.downloading = source["downloading"];
	        this.downloadPct = source["downloadPct"];
	        this.ready = source["ready"];
	        this.error = source["error"];
	        this.assetUrl = source["assetUrl"];
	        this.assetSize = source["assetSize"];
	        this.canInstall = source["canInstall"];
	    }
	}

}

export namespace projectenv {

	export class Issue {
	    code: string;
	    message: string;
	    detail?: string;
	    path?: string;
	    key?: string;
	    blocking?: boolean;

	    static createFrom(source: any = {}) {
	        return new Issue(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.message = source["message"];
	        this.detail = source["detail"];
	        this.path = source["path"];
	        this.key = source["key"];
	        this.blocking = source["blocking"];
	    }
	}
	export class AdapterState {
	    targetId?: string;
	    fingerprint?: string;
	    status?: string;
	    lastApplyAt?: string;
	    lastApplyResult?: string;
	    blockers?: Issue[];

	    static createFrom(source: any = {}) {
	        return new AdapterState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetId = source["targetId"];
	        this.fingerprint = source["fingerprint"];
	        this.status = source["status"];
	        this.lastApplyAt = source["lastApplyAt"];
	        this.lastApplyResult = source["lastApplyResult"];
	        this.blockers = this.convertValues(source["blockers"], Issue);
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
	export class Source {
	    kind: string;
	    id?: string;
	    name?: string;

	    static createFrom(source: any = {}) {
	        return new Source(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class FieldIntent {
	    mode: string;
	    value?: string;

	    static createFrom(source: any = {}) {
	        return new FieldIntent(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.value = source["value"];
	    }
	}
	export class OriginValue {
	    value: string;
	    origin: string;
	    mainConfig: boolean;

	    static createFrom(source: any = {}) {
	        return new OriginValue(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.origin = source["origin"];
	        this.mainConfig = source["mainConfig"];
	    }
	}
	export class Change {
	    key: string;
	    operation: string;
	    current: OriginValue[];
	    desired: FieldIntent;
	    source: Source;
	    issues?: Issue[];

	    static createFrom(source: any = {}) {
	        return new Change(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.operation = source["operation"];
	        this.current = this.convertValues(source["current"], OriginValue);
	        this.desired = this.convertValues(source["desired"], FieldIntent);
	        this.source = this.convertValues(source["source"], Source);
	        this.issues = this.convertValues(source["issues"], Issue);
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
	export class Target {
	    adapter: string;
	    id: string;
	    worktreePath: string;
	    commonDir: string;
	    configPath: string;

	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.adapter = source["adapter"];
	        this.id = source["id"];
	        this.worktreePath = source["worktreePath"];
	        this.commonDir = source["commonDir"];
	        this.configPath = source["configPath"];
	    }
	}
	export class ApplyResult {
	    target: Target;
	    status: string;
	    fingerprint?: string;
	    changes?: Change[];
	    issues?: Issue[];

	    static createFrom(source: any = {}) {
	        return new ApplyResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = this.convertValues(source["target"], Target);
	        this.status = source["status"];
	        this.fingerprint = source["fingerprint"];
	        this.changes = this.convertValues(source["changes"], Change);
	        this.issues = this.convertValues(source["issues"], Issue);
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

	export class TextEdit {
	    start: number;
	    end: number;
	    replacement: string;

	    static createFrom(source: any = {}) {
	        return new TextEdit(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.start = source["start"];
	        this.end = source["end"];
	        this.replacement = source["replacement"];
	    }
	}
	export class Snapshot {
	    target: Target;
	    values: Record<string, Array<OriginValue>>;
	    fingerprint: string;
	    issues: Issue[];
	    exists: boolean;

	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = this.convertValues(source["target"], Target);
	        this.values = this.convertValues(source["values"], Array<OriginValue>, true);
	        this.fingerprint = source["fingerprint"];
	        this.issues = this.convertValues(source["issues"], Issue);
	        this.exists = source["exists"];
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
	export class RepositoryAlias {
	    path: string;
	    ruleIds?: string[];

	    static createFrom(source: any = {}) {
	        return new RepositoryAlias(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.ruleIds = source["ruleIds"];
	    }
	}
	export class ChangePlan {
	    adapter: string;
	    target: Target;
	    repoPath: string;
	    aliases?: RepositoryAlias[];
	    matchingRuleIds?: string[];
	    snapshot: Snapshot;
	    fingerprint: string;
	    desired: Record<string, FieldIntent>;
	    changes: Change[];
	    warnings?: Issue[];
	    blockers?: Issue[];
	    nativeEdits?: TextEdit[];
	    nativeDiff?: string[];

	    static createFrom(source: any = {}) {
	        return new ChangePlan(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.adapter = source["adapter"];
	        this.target = this.convertValues(source["target"], Target);
	        this.repoPath = source["repoPath"];
	        this.aliases = this.convertValues(source["aliases"], RepositoryAlias);
	        this.matchingRuleIds = source["matchingRuleIds"];
	        this.snapshot = this.convertValues(source["snapshot"], Snapshot);
	        this.fingerprint = source["fingerprint"];
	        this.desired = this.convertValues(source["desired"], FieldIntent, true);
	        this.changes = this.convertValues(source["changes"], Change);
	        this.warnings = this.convertValues(source["warnings"], Issue);
	        this.blockers = this.convertValues(source["blockers"], Issue);
	        this.nativeEdits = this.convertValues(source["nativeEdits"], TextEdit);
	        this.nativeDiff = source["nativeDiff"];
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
	export class ClonePresetRequest {
	    revision: string;
	    id: string;
	    name: string;

	    static createFrom(source: any = {}) {
	        return new ClonePresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}
	export class Rule {
	    id: string;
	    name: string;
	    path: string;
	    recursive: boolean;
	    autoDiscover: boolean;
	    presetId?: string;
	    git?: Record<string, FieldIntent>;
	    codex?: Record<string, FieldIntent>;

	    static createFrom(source: any = {}) {
	        return new Rule(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.recursive = source["recursive"];
	        this.autoDiscover = source["autoDiscover"];
	        this.presetId = source["presetId"];
	        this.git = this.convertValues(source["git"], FieldIntent, true);
	        this.codex = this.convertValues(source["codex"], FieldIntent, true);
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
	export class Preset {
	    id: string;
	    name: string;
	    description?: string;
	    git?: Record<string, FieldIntent>;
	    codex?: Record<string, FieldIntent>;

	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.git = this.convertValues(source["git"], FieldIntent, true);
	        this.codex = this.convertValues(source["codex"], FieldIntent, true);
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
	export class LayerConfig {
	    presetId?: string;
	    git?: Record<string, FieldIntent>;
	    codex?: Record<string, FieldIntent>;

	    static createFrom(source: any = {}) {
	        return new LayerConfig(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.presetId = source["presetId"];
	        this.git = this.convertValues(source["git"], FieldIntent, true);
	        this.codex = this.convertValues(source["codex"], FieldIntent, true);
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
	export class Config {
	    version: number;
	    defaults: LayerConfig;
	    presets: Preset[];
	    rules: Rule[];

	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.defaults = this.convertValues(source["defaults"], LayerConfig);
	        this.presets = this.convertValues(source["presets"], Preset);
	        this.rules = this.convertValues(source["rules"], Rule);
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
	export class DeletePresetRequest {
	    revision: string;
	    id: string;

	    static createFrom(source: any = {}) {
	        return new DeletePresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.id = source["id"];
	    }
	}
	export class DeleteRuleRequest {
	    revision: string;
	    id: string;

	    static createFrom(source: any = {}) {
	        return new DeleteRuleRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.id = source["id"];
	    }
	}
	export class EnvironmentFields {
	    git?: Record<string, FieldIntent>;
	    codex?: Record<string, FieldIntent>;

	    static createFrom(source: any = {}) {
	        return new EnvironmentFields(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.git = this.convertValues(source["git"], FieldIntent, true);
	        this.codex = this.convertValues(source["codex"], FieldIntent, true);
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
	export class DoctorRequest {
	    repositoryPaths?: string[];
	    ruleIds?: string[];
	    session?: EnvironmentFields;

	    static createFrom(source: any = {}) {
	        return new DoctorRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repositoryPaths = source["repositoryPaths"];
	        this.ruleIds = source["ruleIds"];
	        this.session = this.convertValues(source["session"], EnvironmentFields);
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
	export class Document {
	    config: Config;
	    revision: string;

	    static createFrom(source: any = {}) {
	        return new Document(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], Config);
	        this.revision = source["revision"];
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





	export class PlanRequest {
	    repositoryPaths?: string[];
	    ruleIds?: string[];
	    session?: EnvironmentFields;

	    static createFrom(source: any = {}) {
	        return new PlanRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repositoryPaths = source["repositoryPaths"];
	        this.ruleIds = source["ruleIds"];
	        this.session = this.convertValues(source["session"], EnvironmentFields);
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


	export class RepositoryRecord {
	    id: string;
	    aliases: RepositoryAlias[];
	    matchingRuleIds?: string[];
	    winningSourceIds?: string[];
	    resolvedTargetHash?: string;
	    status?: string;
	    adapters: Record<string, AdapterState>;

	    static createFrom(source: any = {}) {
	        return new RepositoryRecord(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.aliases = this.convertValues(source["aliases"], RepositoryAlias);
	        this.matchingRuleIds = source["matchingRuleIds"];
	        this.winningSourceIds = source["winningSourceIds"];
	        this.resolvedTargetHash = source["resolvedTargetHash"];
	        this.status = source["status"];
	        this.adapters = this.convertValues(source["adapters"], AdapterState, true);
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
	export class ResolveRequest {
	    repoPath: string;
	    session?: EnvironmentFields;

	    static createFrom(source: any = {}) {
	        return new ResolveRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoPath = source["repoPath"];
	        this.session = this.convertValues(source["session"], EnvironmentFields);
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
	export class ResolvedField {
	    intent: FieldIntent;
	    source: Source;

	    static createFrom(source: any = {}) {
	        return new ResolvedField(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.intent = this.convertValues(source["intent"], FieldIntent);
	        this.source = this.convertValues(source["source"], Source);
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
	export class ResolvedEnvironment {
	    repoPath: string;
	    matchingRuleIds: string[];
	    git: Record<string, ResolvedField>;
	    codex: Record<string, ResolvedField>;

	    static createFrom(source: any = {}) {
	        return new ResolvedEnvironment(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.repoPath = source["repoPath"];
	        this.matchingRuleIds = source["matchingRuleIds"];
	        this.git = this.convertValues(source["git"], ResolvedField, true);
	        this.codex = this.convertValues(source["codex"], ResolvedField, true);
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


	export class RuleScanState {
	    ruleId: string;
	    scanId?: string;
	    lastScanAt?: string;
	    warnings?: Issue[];

	    static createFrom(source: any = {}) {
	        return new RuleScanState(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruleId = source["ruleId"];
	        this.scanId = source["scanId"];
	        this.lastScanAt = source["lastScanAt"];
	        this.warnings = this.convertValues(source["warnings"], Issue);
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
	export class SaveDefaultsRequest {
	    revision: string;
	    presetId?: string;
	    git?: Record<string, FieldIntent>;
	    codex?: Record<string, FieldIntent>;

	    static createFrom(source: any = {}) {
	        return new SaveDefaultsRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.presetId = source["presetId"];
	        this.git = this.convertValues(source["git"], FieldIntent, true);
	        this.codex = this.convertValues(source["codex"], FieldIntent, true);
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
	export class SavePresetRequest {
	    revision: string;
	    id?: string;
	    name?: string;
	    description?: string;
	    git?: Record<string, FieldIntent>;
	    codex?: Record<string, FieldIntent>;

	    static createFrom(source: any = {}) {
	        return new SavePresetRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.git = this.convertValues(source["git"], FieldIntent, true);
	        this.codex = this.convertValues(source["codex"], FieldIntent, true);
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
	export class SaveRuleRequest {
	    revision: string;
	    id?: string;
	    name?: string;
	    path?: string;
	    recursive?: boolean;
	    autoDiscover?: boolean;
	    presetId?: string;
	    git?: Record<string, FieldIntent>;
	    codex?: Record<string, FieldIntent>;

	    static createFrom(source: any = {}) {
	        return new SaveRuleRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.revision = source["revision"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.recursive = source["recursive"];
	        this.autoDiscover = source["autoDiscover"];
	        this.presetId = source["presetId"];
	        this.git = this.convertValues(source["git"], FieldIntent, true);
	        this.codex = this.convertValues(source["codex"], FieldIntent, true);
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


	export class StateDocument {
	    version: number;
	    ruleScans: RuleScanState[];
	    repositories: RepositoryRecord[];

	    static createFrom(source: any = {}) {
	        return new StateDocument(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.ruleScans = this.convertValues(source["ruleScans"], RuleScanState);
	        this.repositories = this.convertValues(source["repositories"], RepositoryRecord);
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

export namespace runner {

	export class Status {
	    State: string;
	    PID: number;
	    ExitCode: number;

	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.State = source["State"];
	        this.PID = source["PID"];
	        this.ExitCode = source["ExitCode"];
	    }
	}

}

export namespace store {

	export class DetectionOption {
	    type: string;
	    command: string;
	    args: string[];

	    static createFrom(source: any = {}) {
	        return new DetectionOption(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.command = source["command"];
	        this.args = source["args"];
	    }
	}
	export class GroupItem {
	    projectId: string;
	    commandId: string;

	    static createFrom(source: any = {}) {
	        return new GroupItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectId = source["projectId"];
	        this.commandId = source["commandId"];
	    }
	}
	export class LaunchCommand {
	    id: string;
	    name: string;
	    command: string;
	    args: string[];
	    cwd: string;
	    env: Record<string, string>;
	    isDefault: boolean;

	    static createFrom(source: any = {}) {
	        return new LaunchCommand(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.cwd = source["cwd"];
	        this.env = source["env"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class LaunchGroup {
	    id: string;
	    name: string;
	    items: GroupItem[];

	    static createFrom(source: any = {}) {
	        return new LaunchGroup(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.items = this.convertValues(source["items"], GroupItem);
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
	export class Project {
	    id: string;
	    name: string;
	    path: string;
	    command: string;
	    args: string[];
	    cwd: string;
	    env: Record<string, string>;
	    detectedType: string;
	    autoDetected: boolean;
	    commands?: LaunchCommand[];
	    composeFile?: string;
	    detectionOptions?: DetectionOption[];
	    pinnedOrder?: number;

	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.cwd = source["cwd"];
	        this.env = source["env"];
	        this.detectedType = source["detectedType"];
	        this.autoDetected = source["autoDetected"];
	        this.commands = this.convertValues(source["commands"], LaunchCommand);
	        this.composeFile = source["composeFile"];
	        this.detectionOptions = this.convertValues(source["detectionOptions"], DetectionOption);
	        this.pinnedOrder = source["pinnedOrder"];
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

