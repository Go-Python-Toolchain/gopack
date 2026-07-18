"""Entry point for the items service.

Usage:
  run.py serve [host] [port]   run a real HTTP server with uvicorn (default)
  run.py check                 exercise every route in process and print results

The bundle's entry is this file, so `./items serve` runs the server and
`./items check` runs the self-test. `check` is deterministic, which makes it the
subject the benchmark uses.
"""

import sys

from app.main import app


def check() -> int:
    """Drive the whole stack in process with a test client."""
    from starlette.testclient import TestClient

    client = TestClient(app)
    print("fastapi items service")
    print("python", sys.version.split()[0])

    r = client.get("/api/health")
    print("GET /api/health", r.status_code, r.json())

    r = client.get("/")
    ok_html = r.status_code == 200 and "<title>Items" in r.text
    print("GET /            ", r.status_code, "html ok" if ok_html else "html MISSING")

    r = client.get("/static/style.css")
    print("GET /static/style.css", r.status_code, r.headers.get("content-type"))

    r = client.post("/api/items", json={"name": "lamp", "price": 12.5})
    print("POST /api/items  ", r.status_code, r.json())

    r = client.get("/api/items")
    items = r.json()
    print("GET /api/items   ", r.status_code, f"{len(items)} items")

    r = client.post("/api/items", json={"name": "", "price": -1})
    print("POST invalid     ", r.status_code, "(validation rejected)")

    ok = ok_html and len(items) >= 4
    return 0 if ok else 1


def serve() -> int:
    import uvicorn

    host = sys.argv[2] if len(sys.argv) > 2 else "127.0.0.1"
    port = int(sys.argv[3]) if len(sys.argv) > 3 else 8000
    print(f"serving on http://{host}:{port} (Ctrl-C to stop)")
    uvicorn.run(app, host=host, port=port, log_level="info")
    return 0


def main() -> int:
    command = sys.argv[1] if len(sys.argv) > 1 else "serve"
    if command == "check":
        return check()
    if command == "serve":
        return serve()
    print(f"unknown command {command!r}; use 'serve' or 'check'", file=sys.stderr)
    return 2


if __name__ == "__main__":
    raise SystemExit(main())
