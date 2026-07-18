from django.contrib import admin

from .models import Note


@admin.register(Note)
class NoteAdmin(admin.ModelAdmin):
    list_display = ("title", "pinned", "created")
    list_filter = ("pinned",)
    search_fields = ("title", "body")
    ordering = ("-pinned", "title")
