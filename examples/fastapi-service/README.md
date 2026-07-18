# fastapi-service: a FastAPI app with a front end, in one executable

A real FastAPI project with the structure you would actually use:

```
fastapi-service/
  run.py                 entry: `serve` (uvicorn) or `check` (in-process test)
  app/
    main.py              application factory, mounts static and routers
    models.py            Pydantic schemas
    store.py             in-memory item store
    routers/
      items.py           the JSON API under /api
      pages.py           the HTML front end, rendered with Jinja2
    templates/
      base.html          the page layout
      index.html         the items page
    static/
      style.css          the front-end stylesheet
  requirements.txt       fastapi, uvicorn, jinja2, httpx
```

It has a JSON API and an HTML front end built from Jinja templates and a static
stylesheet, so it exercises the whole stack: Starlette, Pydantic, the template
engine, and static file serving. Shipping this normally means installing Python
and a pile of dependencies on the target, or building a container. gopack turns
it into one file.

## Bundle it

```
gopack build ./fastapi-service -r ./fastapi-service/requirements.txt --entry run.py -o items
```

## Run it, with no Python installed

Serve it as a real HTTP server:

```
./items serve
```

```
serving on http://127.0.0.1:8000 (Ctrl-C to stop)
```

Open `http://127.0.0.1:8000/` for the rendered items page, `/api/items` for the
JSON, and `/docs` for the automatic OpenAPI docs.

Or run the in-process self-test, which drives every route and prints the
results without opening a port:

```
./items check
```

```
fastapi items service
python 3.12.13
GET /api/health 200 {'status': 'ok'}
GET /             200 html ok
GET /static/style.css 200 text/css; charset=utf-8
POST /api/items   201 {'name': 'lamp', 'price': 12.5, 'id': 4}
GET /api/items    200 4 items
POST invalid      422 (validation rejected)
```
