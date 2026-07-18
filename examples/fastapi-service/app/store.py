"""A tiny in-memory item store, standing in for a database layer."""

from __future__ import annotations

from threading import Lock

from .models import Item, ItemIn


class ItemStore:
    def __init__(self) -> None:
        self._items: dict[int, Item] = {}
        self._next_id = 1
        self._lock = Lock()

    def add(self, data: ItemIn) -> Item:
        with self._lock:
            item = Item(id=self._next_id, **data.model_dump())
            self._items[item.id] = item
            self._next_id += 1
            return item

    def get(self, item_id: int) -> Item | None:
        return self._items.get(item_id)

    def list(self) -> list[Item]:
        return list(self._items.values())

    def seed(self) -> None:
        self.add(ItemIn(name="notebook", price=3.5))
        self.add(ItemIn(name="pen", price=1.2))
        self.add(ItemIn(name="stapler", price=6.0))


store = ItemStore()
