"""The HTML front end, rendered with Jinja2 templates."""

from pathlib import Path

from fastapi import APIRouter, Request
from fastapi.responses import HTMLResponse
from fastapi.templating import Jinja2Templates

from ..store import store

TEMPLATES_DIR = Path(__file__).resolve().parent.parent / "templates"
templates = Jinja2Templates(directory=str(TEMPLATES_DIR))

router = APIRouter(tags=["pages"])


@router.get("/", response_class=HTMLResponse)
def index(request: Request) -> HTMLResponse:
    items = store.list()
    total = sum(item.price for item in items)
    return templates.TemplateResponse(
        request,
        "index.html",
        {"items": items, "total": total},
    )
