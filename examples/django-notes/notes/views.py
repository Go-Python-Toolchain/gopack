from django.db.models import Count, Q
from django.shortcuts import render

from .models import Note


def note_list(request):
    notes = Note.objects.all()
    stats = Note.objects.aggregate(
        total=Count("id"),
        pinned=Count("id", filter=Q(pinned=True)),
    )
    return render(request, "notes/note_list.html", {"notes": notes, "stats": stats})
