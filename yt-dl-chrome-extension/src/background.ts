const goPort = 50999;

//////////////////////
// START HIGHLANDER //
//////////////////////

export const isGecko: boolean = chrome.runtime.getURL('').startsWith('moz-extension://');
export const isSafari: boolean = chrome.runtime.getURL('').startsWith('safari-web-extension://');

export function performsAsyncOperation(callback: () => any): Promise<any> {
    return new Promise((resolve) => {
        resolve(callback());
    });
}

export function execRuntimeSendMessages(obj: any): Promise<any> {
    return new Promise((resolve) => {
        chrome.runtime.sendMessage(obj, (data) => {
            if (chrome.runtime.lastError) {
                console.error(chrome.runtime.lastError);
            }
            resolve(data);
        });
    });
}

const INTERNAL_TESTALIVE_PORT = "DNA_Internal_alive_test";
const nextSeconds = 25;
const SECONDS = 1000;
const DEBUG = false;

let alivePort: chrome.runtime.Port | null = null;
let isFirstStart = true;
let isAlreadyAwake = false;

let timer: number;
let firstCall: number;
let lastCall: number;

let wakeup: NodeJS.Timeout | undefined = undefined;
let wCounter = 0;

const starter = `-------- >>> ${convertNoDate(Date.now())} UTC - Service Worker with HIGHLANDER DNA is starting <<< --------`;
//#endregion

chrome.runtime.onInstalled.addListener((details) => {

    async () => await initialize();

    switch (details.reason) {
        case "install":
            console.log("This runs when the extension is newly installed.");
            start();
            break;

        case "update":
            console.log("This runs when an extension is updated.");
            start();
            break;

        default:
            break;
    }
});

// SW is starting
console.log(starter);
start();

// Clears the Highlander interval when the browser closes.
chrome.windows.onRemoved.addListener((windowId) => {
    wCounter--;
    if (wCounter > 0) {
        return;
    }

    console.log("Browser is closing");

    if (wakeup !== undefined) {
        isAlreadyAwake = false;
        // Uncomment to shutdown Highlander
        // clearInterval(wakeup);  // # shutdown Highlander
        // wakeup = undefined;     // # shutdown Highlander
    }
});

chrome.windows.onCreated.addListener(async (window) => {
    console.log("Browser is creating a new window");
    let w = await chrome.windows.getAll();
    wCounter = w.length;
    if (wCounter === 1) {
        updateJobs();
    }
});

// Tabs listeners
chrome.tabs.onCreated.addListener(onCreatedTabListener);
chrome.tabs.onUpdated.addListener(onUpdatedTabListener);
chrome.tabs.onRemoved.addListener(onRemovedTabListener);

// START
async function start() {
    console.log("Hello world");
    startHighlander();
}

async function updateJobs() {
    console.log("In updateJobs() -> isAlreadyAwake=", isAlreadyAwake);
    if (!isAlreadyAwake) {
        startHighlander();
    }
}

function onCreatedTabListener(tab: chrome.tabs.Tab): void {
    if (DEBUG) console.log("Created TAB id=", tab.id);
}

function onUpdatedTabListener(tabId: number, changeInfo: chrome.tabs.TabChangeInfo, tab: chrome.tabs.Tab): void {
    if (DEBUG) console.log("Updated TAB id=", tabId);
}

function onRemovedTabListener(tabId: number): void {
    if (DEBUG) console.log("Removed TAB id=", tabId);
}

async function checkTabs() {
    let results = await chrome.tabs.query({});
    results.forEach(onCreatedTabListener);
}

async function initialize() {
    await checkTabs();
    updateJobs();
}

function startHighlander() {
    if (wakeup === undefined) {
        isFirstStart = true;
        isAlreadyAwake = true;
        firstCall = Date.now();
        lastCall = firstCall;
        timer = 300;

        wakeup = setInterval(Highlander, timer);
        console.log(`-------- >>> Highlander has been started at ${convertNoDate(firstCall)}`);
    }
}

// HIGHLANDER FUNCTIONS
async function Highlander() {
    const now = Date.now();
    const age = now - firstCall;
    lastCall = now;

    const str = `HIGHLANDER ------< ROUND >------ Time elapsed from first start: ${convertNoDate(age)}`;
    console.log(str);

    if (alivePort == null) {
        alivePort = chrome.runtime.connect({ name: INTERNAL_TESTALIVE_PORT });

        alivePort.onDisconnect.addListener((p) => {
            if (chrome.runtime.lastError) {
                if (DEBUG) console.log(`(DEBUG Highlander) Expected disconnect error. ServiceWorker status should be still RUNNING.`);
            } else {
                if (DEBUG) console.log(`(DEBUG Highlander): port disconnected`);
            }

            alivePort = null;
        });
    }

    if (alivePort) {
        alivePort.postMessage({ content: "ping" });

        if (chrome.runtime.lastError) {
            if (DEBUG) console.log(`(DEBUG Highlander): postMessage error: ${chrome.runtime.lastError.message}`);
        } else {
            if (DEBUG) console.log(`(DEBUG Highlander): "ping" sent through ${alivePort.name} port`);
        }
    }

    if (isFirstStart) {
        isFirstStart = false;
        setTimeout(() => {
            nextRound();
        }, 100);
    }
}

function convertNoDate(long: number): string {
    const dt = new Date(long).toISOString();
    return dt.slice(-13, -5); // HH:MM:SS only
}

function nextRound() {
    clearInterval(wakeup as NodeJS.Timeout);
    timer = nextSeconds * SECONDS;
    wakeup = setInterval(Highlander, timer);
}
////////////////////
// END HIGHLANDER //
////////////////////

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.ytlink) {
        if (message.name) {
            getTrack(message.ytlink, message.name);
        } else {
            getTrack(message.ytlink)
        }
    } else if (message.rules) {
        initRules()
    }
});

function initRules() {
    const RULE = {
        id: 1,
        condition: {
            initiatorDomains: [chrome.runtime.id],
            requestDomains: ['music.youtube.com'],
            resourceTypes: [chrome.declarativeNetRequest.ResourceType.MAIN_FRAME, chrome.declarativeNetRequest.ResourceType.SUB_FRAME],
        },
        action: {
            type: chrome.declarativeNetRequest.RuleActionType.MODIFY_HEADERS,
            responseHeaders: [
                { header: 'X-Frame-Options', operation: chrome.declarativeNetRequest.HeaderOperation.REMOVE },
                { header: 'Frame-Options', operation: chrome.declarativeNetRequest.HeaderOperation.REMOVE },
                { header: 'Content-Security-Policy', operation: chrome.declarativeNetRequest.HeaderOperation.REMOVE },
                { header: 'User-Agent', operation: chrome.declarativeNetRequest.HeaderOperation.SET, value: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36" },
            ],
        },
    };
    chrome.declarativeNetRequest.updateDynamicRules({
        removeRuleIds: [RULE.id],
        addRules: [RULE],
    });
}

async function downloadTrack(ytlink: string): Promise<GetTrackResponse> {
    try {
        const response = await fetch(`http://localhost:${goPort}/download?id=${sanitizeUrl(ytlink)}`, {
            mode: "cors"
        });

        if (!response.ok) {
            throw new Error('Failed to fetch track data');
        }

        return await response.json();
    } catch (error) {
        console.error('Failed to fetch track data:', error);
        throw error;
    }
}
async function getTrack(link: string, fn?: string) {
    console.log("sending download track request for: " + link)
    if (fn == null || fn == undefined || fn == "") {
        fn = ""
    }
    downloadTrack(link).then(() => {
        var ytlink = sanitizeUrl(link)
        console.log("polling download status");
        pollStatus(ytlink).then(() => {
            getStatus(ytlink).then((gcr) => {
                console.log("got download status")
                console.log(gcr)
                if (gcr.status === "complete") {
                    updatePopup(ytlink)
                    closemessage(ytlink)
                } else {
                    sendErrorMessage(ytlink)
                }
            });

        }).catch(() => {
            console.log("caught error in dl poll")
            sendErrorMessage(ytlink)
        })
    });
}

async function closemessage(id: string) {
    try {
        const tabs = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
        const activeTab = tabs[0]; // Access the first tab directly

        if (activeTab?.id) {
            chrome.tabs.sendMessage(activeTab.id, { complete: true, id: id });
            // Handle the response here
        } else {
            console.error("No active tabs found");
        }
    } catch (error) {
        console.error("Error:", error);
    }
}

async function sendErrorMessage(id: string) {
    try {
        const tabs = await chrome.tabs.query({ active: true, lastFocusedWindow: true });
        const activeTab = tabs[0]; // Access the first tab directly

        if (activeTab?.id) {
            chrome.tabs.sendMessage(activeTab.id, { error: true, id: id });
            // Handle the response here
        } else {
            console.error("No active tabs found");
        }
    } catch (error) {
        console.error("Error:", error);
    }
}

function updatePopup(id: string) {
    getFromDB(function (db) {
        var n = db.get(id)
        if (n != null && n != undefined) {
            n.state = "dl_done"
            db.set(id, n)
            chrome.storage.local.set({ "downloaddb": Object.fromEntries(db) })
        }
    })
}

async function pollStatus(ytlink: string, st?: number): Promise<boolean> {
    return new Promise<boolean>((resolve, reject) => {
        const startTime = st === undefined ? new Date().getTime() : st;
        const w = st === undefined ? 7500 : 5000;

        setTimeout(async () => {
            if (new Date().getTime() > startTime + 120000) {
                reject(false); // Timeout reached, track not converted
            } else {
                try {
                    const gic = await getStatus(ytlink);
                    if (gic.status === "complete") {
                        resolve(true); // Track is converted
                    } else if (gic.status === "faield") {
                        reject(false); // Error occurred

                    } else {
                        const result = await pollStatus(ytlink, startTime);
                        resolve(result); // Recursively check until conversion or timeout
                    }
                } catch (error) {
                    const result = await pollStatus(ytlink, startTime);
                    reject(result); // Recursively check in case of errors
                }
            }
        }, w);
    });
}

async function getStatus(ytlink: string): Promise<DLStatusTrack> {
    try {
        const response = await fetch(`http://localhost:${goPort}/status?id=${ytlink}`, {
            mode: "cors"
        })
        if (!response.ok) {
            throw new Error('Failed to get download status');
        }
        return await response.json();
    } catch (error) {
        console.error('Failed to get get download status:', error)
        return {
            id: "",
            status: "",
            playlist_track_count: 0,
            playlist_track_done: 0,
        };
    }
}

function sanitizeUrl(ytlink: string): string {
    const reg = new RegExp('https://|www.|music.youtube.com/|youtube.com/|youtu.be/|watch\\?v=|&feature=share|playlist\\?list=', 'g');
    return ytlink.replace(reg, "").split("&")[0];
}


function getFromDB(callback: (db: Map<string, YTDetails>) => void) {
    chrome.storage.local.get("downloaddb", function (result) {
        const downloaddb = result.downloaddb;
        if (downloaddb) {
            const db = new Map<string, YTDetails>(Object.entries(downloaddb));
            callback(db);
        } else {
            const db = new Map<string, YTDetails>();
            callback(db);
        }
    });
}
