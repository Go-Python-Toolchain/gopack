"""A custom management command: `./notes seed`.

Custom management commands are a normal part of a Django project, and they work
from a gopack bundle exactly as they do from a checkout. This one resets the
notes table and inserts a handful of example rows.
"""

from django.core.management.base import BaseCommand

from notes.models import Note

EXAMPLES = [
    ("release gopack", "tag the version and push the binaries", True),
    ("write docs", "protocol, benchmarks, limitations, roadmap", True),
    ("buy milk", "two liters, semi skimmed", False),
    ("book flights", "for the conference in March", False),
    ("water the plants", "the fern is looking sad", False),
]


class Command(BaseCommand):
    help = "Reset the notes table and insert example notes"

    def handle(self, *args, **options):
        Note.objects.all().delete()
        for title, body, pinned in EXAMPLES:
            Note.objects.create(title=title, body=body, pinned=pinned)
        count = Note.objects.count()
        self.stdout.write(self.style.SUCCESS(f"seeded {count} notes"))
