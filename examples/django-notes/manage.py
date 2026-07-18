#!/usr/bin/env python
"""Django's command line utility for administrative tasks.

This is the standard manage.py. It is the bundle's entry point, so a gopack
bundle of this project responds to every Django management command:
`./notes migrate`, `./notes seed`, `./notes runserver`, `./notes check`, and so
on, exactly as you would run them with a normal Django checkout.
"""

import os
import sys


def main() -> None:
    os.environ.setdefault("DJANGO_SETTINGS_MODULE", "config.settings")
    try:
        from django.core.management import execute_from_command_line
    except ImportError as exc:
        raise ImportError(
            "Couldn't import Django. Are you sure it is installed and available "
            "on your PYTHONPATH environment variable?"
        ) from exc
    execute_from_command_line(sys.argv)


if __name__ == "__main__":
    main()
