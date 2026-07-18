"""Application factory: build the FastAPI app, mount static files and routers."""

from pathlib import Path

from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles

from .routers import items, pages
from .store import store

STATIC_DIR = Path(__file__).resolve().parent / "static"


def create_app() -> FastAPI:
    app = FastAPI(title="items service")
    app.mount("/static", StaticFiles(directory=str(STATIC_DIR)), name="static")
    app.include_router(pages.router)
    app.include_router(items.router)

    # Start with a few items so the front page and the API have something to
    # show on first run.
    if not store.list():
        store.seed()
    return app


app = create_app()
