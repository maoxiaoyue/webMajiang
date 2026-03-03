import { sys } from 'cc';

export class ErrorReporter {
    private static isInitialized = false;

    public static init() {
        if (this.isInitialized) return;
        this.isInitialized = true;

        if (typeof window !== 'undefined') {
            // Global Error Handler
            window.addEventListener('error', (event) => {
                this.sendReport({
                    message: event.message,
                    source: event.filename,
                    line: event.lineno,
                    column: event.colno,
                    error: event.error ? event.error.stack : "",
                    userAgent: navigator.userAgent,
                    url: window.location.href
                });
            });

            // Unhandled Promise Rejection Handler
            window.addEventListener('unhandledrejection', (event) => {
                let reasonStr = "Unknown Promise Rejection";
                let stack = "";
                if (event.reason instanceof Error) {
                    reasonStr = event.reason.message;
                    stack = event.reason.stack || "";
                } else {
                    try {
                        reasonStr = JSON.stringify(event.reason);
                    } catch (e) {
                        reasonStr = String(event.reason);
                    }
                }

                this.sendReport({
                    message: `Unhandled Rejection: ${reasonStr}`,
                    source: 'Promise',
                    line: 0,
                    column: 0,
                    error: stack,
                    userAgent: navigator.userAgent,
                    url: window.location.href
                });
            });
            console.log("[ErrorReporter] Global error handlers attached.");
        }
    }

    private static sendReport(data: any) {
        // Try avoiding spam
        if (!data.message) return;

        // Ensure we send to the current origin's API, assuming we play on the same served origin.
        // Or configure a hardcoded backend if needed.
        const apiUrl = window.location.origin + "/api/client/error";

        fetch(apiUrl, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(data)
        }).catch((err) => {
            console.log("[ErrorReporter] Failed to send report: ", err);
        });
    }
}
