# developers-italia-api for the software catalog of Developers Italia
API for the developers.italia.it public software collection

## Table of Contents
#### [Requirements](#requirements)
#### [Description](#description)
#### [Documentation](#documentation)

# [Description](#description)

Developers Italia API is a RESTful API that provides information about the catalog of Free and Open Source software aimed to Public Administrations.

# [Requirements](#requirements)
Developers Italia API requires the following:
- [Docker](https://www.docker.com/)
- [Docker Compose](https://docs.docker.com/compose/install/)

Eventually, you can install a SQL Viewer / Manage tool to inspect the database.


# [Documentation](#documentation)

### Development
The application uses [https://github.com/cosmtrek/air](Air) for live-reloading of the application in development environment.
We use Docker and docker-compose to bring up the developer environment, **just clone the repo** and:

Build the container:
```bash
docker compose up
```

Wait until the Docker logs explicitly say that the website is served at 0.0.0.0:3000 and navigate to http://localhost:3000.
Docker compose will bring up API and PostgreSQL containers.
The application will automatically reload after a changes has been applied.

### Contributing

Developers Italia API exists also thanks to your contributions! Here is a list of users who already contributed to this repository:

TODO

### License

Copyright (c) 2022 - Presidenza del Consiglio dei Ministri

The source code is released under the AGPL version 3 license.

