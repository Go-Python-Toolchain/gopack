from django.db import models


class Note(models.Model):
    title = models.CharField(max_length=100)
    body = models.TextField()
    pinned = models.BooleanField(default=False)
    created = models.DateTimeField(auto_now_add=True)

    class Meta:
        ordering = ["-pinned", "title"]

    def __str__(self) -> str:
        return self.title
