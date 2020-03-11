# Maven Feed
A small webapp which serves Maven artifact versions as an RSS and Atom feed, so you can use any feed reader to keep up
to date with new releases.


# Running
Right now nothing is pushed to Docker hub yet, so you will first have to build it yourself (check `build.sh` and 
`Dockerfile` for some hints). Then just run the Docker container (or without Docker, using `./maven-feed.bin`).

The app expects the following environment variables

| Var | Default | Description |
| --- | --- | --- |
| `ARTIFACTS` | (Required) | Pipe-separated list of artifacts to exposed in the feed, eg. <code>org.jooq:jooq&#124;org.postgresql:postgresql&#124;com.google.guava:guava</code>. |
| `SELF_URL` | (Required) | Full URL to the app, to be included feeds. |
| `BIND_HOST` | `0.0.0.0` | Which host to bind on. |
| `BIND_PORT` | `8080` | Which port to bind on. |
| `DEBUG_ENABLED` | `false` | Set to `true` to enable logging of everything the app does. |

The app exposes three paths, each for a different feed type:

- `/rss`: an RSS feed.
- `/atom`: an Atom feed.
- `/json`: A JSON feed (https://jsonfeed.org/).


# The future
Things I would like to add in the future (no promises, no time line!):

- Publish ready-to-use Docker image to Docker hub
- Being able to add multiple different feeds at runtime which each can expose different artifacts
- Some caching, which can be useful when a feed reader queries the app a lot and there are a lot of artifacts configured
