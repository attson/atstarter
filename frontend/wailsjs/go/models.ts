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

}

export namespace main {
	
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

