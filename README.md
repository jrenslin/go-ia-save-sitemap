# Sitemap archiver

This small tool takes a remote sitemap, fetches it, and requests the Internet Archive's Wayback Machine to archive each URL listed in the sitemap.

## Design decisions

- This tool deliberately does not parallelize requests to IA to reduce load on their systems.

## To-Do

- Currently, paginated sitemaps are not supported
- Maybe return prettier outputs or a summary after completion

## Usage

After running `go build` to compile the tool, you can run e.g. `./go-ia-save-sitemap https://www.jrenslin.de/sitemap.xml`, where the first command line argument points to the XML sitemap from which you want to archive).

