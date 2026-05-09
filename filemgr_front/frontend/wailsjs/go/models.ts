export namespace main {
	
	export class FileInfo {
	    Name: string;
	    IsDir: boolean;
	    Ext: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.IsDir = source["IsDir"];
	        this.Ext = source["Ext"];
	    }
	}

}

