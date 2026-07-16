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
	    }
	}

}

