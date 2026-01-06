export namespace main {
	
	export class MediaAttachment {
	    name: string;
	    mime_type: string;
	    size: number;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new MediaAttachment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.mime_type = source["mime_type"];
	        this.size = source["size"];
	        this.url = source["url"];
	    }
	}
	export class MediaResult {
	    body: string;
	    subject?: string;
	    from?: string;
	    attachments?: MediaAttachment[];
	
	    static createFrom(source: any = {}) {
	        return new MediaResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.body = source["body"];
	        this.subject = source["subject"];
	        this.from = source["from"];
	        this.attachments = this.convertValues(source["attachments"], MediaAttachment);
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

export namespace player {
	
	export class TrackInfo {
	    title: string;
	    start: number;
	    end?: number;
	    type?: string;
	    track_num?: number;
	
	    static createFrom(source: any = {}) {
	        return new TrackInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.type = source["type"];
	        this.track_num = source["track_num"];
	    }
	}
	export class ManifestInfo {
	    title?: string;
	    artist?: string;
	    album?: string;
	    genre?: string;
	    year?: number;
	    release_type?: string;
	    duration?: number;
	    format?: string;
	    expires_at?: number;
	    issued_at?: number;
	    license_type?: string;
	    tracks?: TrackInfo[];
	    is_expired: boolean;
	    time_remaining?: string;
	
	    static createFrom(source: any = {}) {
	        return new ManifestInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.artist = source["artist"];
	        this.album = source["album"];
	        this.genre = source["genre"];
	        this.year = source["year"];
	        this.release_type = source["release_type"];
	        this.duration = source["duration"];
	        this.format = source["format"];
	        this.expires_at = source["expires_at"];
	        this.issued_at = source["issued_at"];
	        this.license_type = source["license_type"];
	        this.tracks = this.convertValues(source["tracks"], TrackInfo);
	        this.is_expired = source["is_expired"];
	        this.time_remaining = source["time_remaining"];
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

