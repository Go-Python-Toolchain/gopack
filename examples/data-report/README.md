# data-report: a pandas and NumPy pipeline in one executable

A small data pipeline, the shape of a typical internal tool, split into the
stages you would actually separate:

```
data-report/
  run.py                 command line entry
  salesreport/
    generate.py          deterministic sample data
    transform.py         derive revenue and aggregate by region
    report.py            assemble the printable report
  requirements.txt       pandas, numpy
```

It leans on pandas and NumPy, both of which carry compiled extensions, so it is
awkward to hand to a colleague who does not have the scientific stack set up.
gopack makes it one executable.

## Bundle it

```
gopack build ./data-report -r ./data-report/requirements.txt --entry run.py -o report
```

## Run it, with no Python installed

```
./report
```

```
regional sales report
python 3.12.13
pandas 2.2.2
numpy 1.26.4
rows 10000

        orders   units      revenue
region
north     2534  125567 3,524,249.19
south     2527  124644 3,483,459.05
west      2453  123228 3,401,482.80
east      2486  123175 3,373,477.60

total revenue 13,782,668.64
```

Pass a row count to change the size of the synthetic dataset, for example
`./report 50000`. The data is generated from a fixed seed, so the report is
deterministic.
