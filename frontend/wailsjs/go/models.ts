export namespace main {
	
	export class Attachment {
	    File_Name: string;
	    File_Type: string;
	    File_Path: string;
	
	    static createFrom(source: any = {}) {
	        return new Attachment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.File_Name = source["File_Name"];
	        this.File_Type = source["File_Type"];
	        this.File_Path = source["File_Path"];
	    }
	}
	export class CompanyResult {
	    code: string;
	    attachments: Attachment[];
	
	    static createFrom(source: any = {}) {
	        return new CompanyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.attachments = this.convertValues(source["attachments"], Attachment);
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

