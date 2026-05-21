"""
Minimal health server for the ContextOS AI worker.
Uses only stdlib — no extra dependencies required.
Listens on WORKER_PORT (default 8081) and serves GET /health.
"""

import http.server  # stdlib HTTP server used to listen for health check requests
import json  # used to serialise the health response body
import os  # used to read the WORKER_PORT environment variable
import sys  # used to write log lines to stderr

_PORT = int(
    os.environ.get("WORKER_PORT", "8081")
)  # bind port, overridable via environment variable


class _HealthHandler(http.server.BaseHTTPRequestHandler):
    # handles incoming GET requests and routes them by path
    def do_GET(self) -> None:
        if self.path != "/health":  # reject any path that is not /health
            self.send_response(404)  # send a 404 so callers know the path is wrong
            self.end_headers()
            return

        body = json.dumps(
            {"status": "ok", "service": "context-os-ai-worker"}
        ).encode()  # build the JSON body and encode to bytes
        self.send_response(200)  # signal a successful response
        self.send_header(
            "Content-Type", "application/json"
        )  # tell the client to parse the body as JSON
        self.send_header(
            "Content-Length", str(len(body))
        )  # set the exact byte length so the client knows when the body ends
        self.end_headers()  # flush headers before writing the body
        self.wfile.write(body)  # write the JSON bytes to the response stream

    def log_message(
        self, format: str, *args: object
    ) -> None:  # noqa: A002 — override the default log format
        print(
            f"[worker] {self.address_string()} - {format % args}",  # prefix log lines with [worker] for easy filtering
            file=sys.stderr,  # write to stderr so it doesn't mix with stdout output
            flush=True,  # flush immediately so logs appear in real time
        )


# main starts the HTTP server and blocks until the process is killed
def main() -> None:
    server = http.server.HTTPServer(
        ("", _PORT), _HealthHandler
    )  # bind to all interfaces on the configured port
    print(
        f"context-os ai-worker health listening on :{_PORT}", flush=True
    )  # log the startup address for visibility
    server.serve_forever()  # block and handle requests until the process receives a signal


if (
    __name__ == "__main__"
):  # only run when executed directly, not when imported as a module
    main()
