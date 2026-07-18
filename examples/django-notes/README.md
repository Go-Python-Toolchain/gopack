# django-notes: a full Django project in one executable

A real Django project, with the layout and the commands you already know:

```
django-notes/
  manage.py              the standard Django entry point (the bundle's entry)
  config/
    settings.py          project settings
    urls.py              the admin site and the notes app
    wsgi.py, asgi.py     server entry points
  notes/
    models.py            the Note model
    admin.py             Note registered in the Django admin
    views.py             the notes page view
    urls.py              the app's routes
    apps.py              the app config
    migrations/          a real migration (0001_initial)
    management/commands/
      seed.py            a custom `seed` management command
    templates/notes/     base.html and note_list.html (the front end)
    static/notes/        style.css
  requirements.txt       Django
```

The bundle's entry is `manage.py`, so the single executable responds to every
Django management command, exactly as a normal checkout does.

## Bundle it

```
gopack build ./django-notes -r ./django-notes/requirements.txt --entry manage.py -o notes
```

## Run it, with no Python installed

Everything you would run with `python manage.py ...` you run on the bundle:

```
./notes migrate          # apply migrations, create the SQLite database
./notes seed             # a custom management command that inserts example notes
./notes check            # Django's system checks
./notes createsuperuser  # create an admin login (or use --noinput with env vars)
./notes runserver        # serve the site on http://127.0.0.1:8000
```

`migrate`, `seed`, and `check` from the bundle:

```
$ ./notes migrate
  Applying contenttypes.0001_initial... OK
  ...
  Applying notes.0001_initial... OK
  Applying sessions.0001_initial... OK

$ ./notes seed
seeded 5 notes

$ ./notes check
System check identified no issues (0 silenced).
```

With the server running, `http://127.0.0.1:8000/` renders the notes page from
its template with the seeded data and its stylesheet, and `http://127.0.0.1:8000/admin/`
is the Django admin site. All of it runs from the one file, on a machine with no
Python and no Django installed.

## Where data lives

The SQLite database and anything the app writes go next to the code, which
inside the bundle is the extraction cache, a writable directory. So `migrate`
creates the database on first run and later commands see it.
