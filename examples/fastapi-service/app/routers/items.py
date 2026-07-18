"""The JSON items API, mounted under /api."""

from fastapi import APIRouter, HTTPException

from ..models import Item, ItemIn
from ..store import store

router = APIRouter(prefix="/api", tags=["items"])


@router.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@router.get("/items", response_model=list[Item])
def list_items() -> list[Item]:
    return store.list()


@router.post("/items", response_model=Item, status_code=201)
def create_item(data: ItemIn) -> Item:
    return store.add(data)


@router.get("/items/{item_id}", response_model=Item)
def get_item(item_id: int) -> Item:
    item = store.get(item_id)
    if item is None:
        raise HTTPException(status_code=404, detail="item not found")
    return item
