export namespace config {
	
	export class AdvancedSettings {
	    UseNightly: boolean;
	    OutputTemplate: string;
	    ExtraArgs: string;
	    JSRuntime: string;
	
	    static createFrom(source: any = {}) {
	        return new AdvancedSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.UseNightly = source["UseNightly"];
	        this.OutputTemplate = source["OutputTemplate"];
	        this.ExtraArgs = source["ExtraArgs"];
	        this.JSRuntime = source["JSRuntime"];
	    }
	}
	export class AuthSettings {
	    CookiesBrowser: string;
	    CookiesFile: string;
	    POToken: string;
	    PlayerClient: string;
	
	    static createFrom(source: any = {}) {
	        return new AuthSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.CookiesBrowser = source["CookiesBrowser"];
	        this.CookiesFile = source["CookiesFile"];
	        this.POToken = source["POToken"];
	        this.PlayerClient = source["PlayerClient"];
	    }
	}
	export class DownloadSettings {
	    Quality: string;
	    Format: string;
	    AudioFormat: string;
	    AudioQuality: string;
	    EmbedThumbnail: boolean;
	    EmbedMetadata: boolean;
	    EmbedChapters: boolean;
	    Sponsorblock: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DownloadSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Quality = source["Quality"];
	        this.Format = source["Format"];
	        this.AudioFormat = source["AudioFormat"];
	        this.AudioQuality = source["AudioQuality"];
	        this.EmbedThumbnail = source["EmbedThumbnail"];
	        this.EmbedMetadata = source["EmbedMetadata"];
	        this.EmbedChapters = source["EmbedChapters"];
	        this.Sponsorblock = source["Sponsorblock"];
	    }
	}
	export class GeneralSettings {
	    SaveFolder: string;
	    Theme: string;
	    MaxConcurrentDownloads: number;
	    ClipboardMonitoring: boolean;
	    NotificationsEnabled: boolean;
	    CheckUpdatesOnStart: boolean;
	    VerboseLogging: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GeneralSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SaveFolder = source["SaveFolder"];
	        this.Theme = source["Theme"];
	        this.MaxConcurrentDownloads = source["MaxConcurrentDownloads"];
	        this.ClipboardMonitoring = source["ClipboardMonitoring"];
	        this.NotificationsEnabled = source["NotificationsEnabled"];
	        this.CheckUpdatesOnStart = source["CheckUpdatesOnStart"];
	        this.VerboseLogging = source["VerboseLogging"];
	    }
	}
	export class NetworkSettings {
	    RateLimit: string;
	    Proxy: string;
	    Retries: number;
	
	    static createFrom(source: any = {}) {
	        return new NetworkSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.RateLimit = source["RateLimit"];
	        this.Proxy = source["Proxy"];
	        this.Retries = source["Retries"];
	    }
	}
	export class Settings {
	    SchemaVersion: number;
	    General: GeneralSettings;
	    Download: DownloadSettings;
	    Network: NetworkSettings;
	    Auth: AuthSettings;
	    Advanced: AdvancedSettings;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.SchemaVersion = source["SchemaVersion"];
	        this.General = this.convertValues(source["General"], GeneralSettings);
	        this.Download = this.convertValues(source["Download"], DownloadSettings);
	        this.Network = this.convertValues(source["Network"], NetworkSettings);
	        this.Auth = this.convertValues(source["Auth"], AuthSettings);
	        this.Advanced = this.convertValues(source["Advanced"], AdvancedSettings);
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

export namespace converter {
	
	export class ConversionJob {
	    id: string;
	    input_file: string;
	    output_file: string;
	    status: string;
	    progress: number;
	    duration: string;
	    speed: string;
	    error: string;
	    // Go type: time
	    started_at: any;
	    // Go type: time
	    finished_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new ConversionJob(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.input_file = source["input_file"];
	        this.output_file = source["output_file"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.duration = source["duration"];
	        this.speed = source["speed"];
	        this.error = source["error"];
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.finished_at = this.convertValues(source["finished_at"], null);
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
	export class ConversionOptions {
	    input_file: string;
	    output_file: string;
	    output_format: string;
	    video_codec: string;
	    audio_codec: string;
	    preset: string;
	    video_bitrate: string;
	    audio_bitrate: string;
	    resolution: string;
	    crf: number;
	    custom_args: string;
	
	    static createFrom(source: any = {}) {
	        return new ConversionOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.input_file = source["input_file"];
	        this.output_file = source["output_file"];
	        this.output_format = source["output_format"];
	        this.video_codec = source["video_codec"];
	        this.audio_codec = source["audio_codec"];
	        this.preset = source["preset"];
	        this.video_bitrate = source["video_bitrate"];
	        this.audio_bitrate = source["audio_bitrate"];
	        this.resolution = source["resolution"];
	        this.crf = source["crf"];
	        this.custom_args = source["custom_args"];
	    }
	}
	export class MediaInfo {
	    duration: string;
	    duration_sec: number;
	    width: number;
	    height: number;
	    video_codec: string;
	    audio_codec: string;
	    bitrate: string;
	    file_size: string;
	    has_video: boolean;
	    has_audio: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MediaInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.duration = source["duration"];
	        this.duration_sec = source["duration_sec"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.video_codec = source["video_codec"];
	        this.audio_codec = source["audio_codec"];
	        this.bitrate = source["bitrate"];
	        this.file_size = source["file_size"];
	        this.has_video = source["has_video"];
	        this.has_audio = source["has_audio"];
	    }
	}
	export class Preset {
	    id: string;
	    name: string;
	    description: string;
	    output_format: string;
	    video_codec: string;
	    audio_codec: string;
	    preset: string;
	    audio_bitrate: string;
	    audio_only: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Preset(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.output_format = source["output_format"];
	        this.video_codec = source["video_codec"];
	        this.audio_codec = source["audio_codec"];
	        this.preset = source["preset"];
	        this.audio_bitrate = source["audio_bitrate"];
	        this.audio_only = source["audio_only"];
	    }
	}

}

export namespace downloader {
	
	export class Solution {
	    id: string;
	    title: string;
	    description: string;
	    action: string;
	    action_data: string;
	    priority: number;
	
	    static createFrom(source: any = {}) {
	        return new Solution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.action = source["action"];
	        this.action_data = source["action_data"];
	        this.priority = source["priority"];
	    }
	}
	export class Item {
	    id: string;
	    url: string;
	    title: string;
	    status: string;
	    progress: number;
	    speed: string;
	    eta: string;
	    file_path: string;
	    file_size: number;
	    error: string;
	    error_type: string;
	    suggestions?: Solution[];
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    started_at?: any;
	    // Go type: time
	    completed_at?: any;
	    is_audio_only: boolean;
	    quality: string;
	    format: string;
	    current_item: number;
	    total_items: number;
	    file_exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.speed = source["speed"];
	        this.eta = source["eta"];
	        this.file_path = source["file_path"];
	        this.file_size = source["file_size"];
	        this.error = source["error"];
	        this.error_type = source["error_type"];
	        this.suggestions = this.convertValues(source["suggestions"], Solution);
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.started_at = this.convertValues(source["started_at"], null);
	        this.completed_at = this.convertValues(source["completed_at"], null);
	        this.is_audio_only = source["is_audio_only"];
	        this.quality = source["quality"];
	        this.format = source["format"];
	        this.current_item = source["current_item"];
	        this.total_items = source["total_items"];
	        this.file_exists = source["file_exists"];
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

export namespace history {
	
	export class Entry {
	    id: string;
	    url: string;
	    title: string;
	    status: string;
	    file_path: string;
	    file_size: number;
	    error?: string;
	    // Go type: time
	    date: any;
	    is_audio_only: boolean;
	    quality: string;
	    format: string;
	    file_ext?: string;
	    hide_in_queue: boolean;
	    file_exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.status = source["status"];
	        this.file_path = source["file_path"];
	        this.file_size = source["file_size"];
	        this.error = source["error"];
	        this.date = this.convertValues(source["date"], null);
	        this.is_audio_only = source["is_audio_only"];
	        this.quality = source["quality"];
	        this.format = source["format"];
	        this.file_ext = source["file_ext"];
	        this.hide_in_queue = source["hide_in_queue"];
	        this.file_exists = source["file_exists"];
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

export namespace jsruntime {
	
	export class Runtime {
	    name: string;
	    path: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new Runtime(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.version = source["version"];
	    }
	}
	export class RuntimeInfo {
	    detected?: Runtime;
	    available: Runtime[];
	    recommended: string;
	    deno_path: string;
	    needs_install: boolean;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.detected = this.convertValues(source["detected"], Runtime);
	        this.available = this.convertValues(source["available"], Runtime);
	        this.recommended = source["recommended"];
	        this.deno_path = source["deno_path"];
	        this.needs_install = source["needs_install"];
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

export namespace main {
	
	export class DownloadOptions {
	    is_audio_only: boolean;
	    quality: string;
	    format: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.is_audio_only = source["is_audio_only"];
	        this.quality = source["quality"];
	        this.format = source["format"];
	    }
	}

}

export namespace updater {
	
	export class UpdateInfo {
	    current_version: string;
	    latest_version: string;
	    update_available: boolean;
	    is_nightly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current_version = source["current_version"];
	        this.latest_version = source["latest_version"];
	        this.update_available = source["update_available"];
	        this.is_nightly = source["is_nightly"];
	    }
	}

}

