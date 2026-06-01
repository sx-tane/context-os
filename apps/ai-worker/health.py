"""
Minimal health server for the ContextOS AI worker.
Uses only stdlib — no extra dependencies required.
Listens on WORKER_PORT (default 8081) and serves GET /health.
"""

import http.server  # stdlib HTTP server used to listen for health check requests
import json  # used to serialise the health response body
import os  # used to read the WORKER_PORT environment variable
import sys  # used to write log lines to stderr

from embed import DEFAULT_MODEL, EMBED_DIM, embed_texts  # deterministic embedding

_PORT = int(
    os.environ.get("WORKER_PORT", "8081")
)  # bind port, overridable via environment variable

_MAX_BODY = 1 << 20  # reject embed request bodies larger than 1 MiB


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

    # handles incoming POST requests and routes them by path
    def do_POST(self) -> None:
        if self.path != "/embed":  # only /embed accepts POST
            self._send_json(404, {"error": "not found"})
            return

        length = int(self.headers.get("Content-Length", "0") or "0")
        if length <= 0 or length > _MAX_BODY:  # guard against missing or oversized bodies
            self._send_json(400, {"error": "invalid content length"})
            return

        try:
            payload = json.loads(self.rfile.read(length).decode("utf-8"))
            texts = payload["texts"]
        except (ValueError, KeyError, TypeError):
            self._send_json(400, {"error": "body must be JSON with a 'texts' array"})
            return
        if not isinstance(texts, list) or not all(isinstance(t, str) for t in texts):
            self._send_json(400, {"error": "'texts' must be an array of strings"})
            return

        self._send_json(
            200,
            {
                "model": DEFAULT_MODEL,  # extension point for a future real model
                "dim": EMBED_DIM,
                "vectors": embed_texts(texts),  # one deterministic vector per input
            },
        )

    # _send_json serialises body as JSON and writes it with the given status code
    def _send_json(self, status: int, body: object) -> None:
        encoded = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)
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
