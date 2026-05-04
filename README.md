our-space
===

`our-space` is a management software for maker spaces or similar open workshops. It has a management component for staff
and an operating system image running on a terminal for members to check in and out. 

Current features:
 - Manage members and their management cards.

Features under development:
 - Checkin/checkout - presence management
 - Safety briefing management

Planned features:
 - Workshop/Event management
 - Self service data update
 - Hardware lending
 - Member profile/knowledge management
 - GDPR data export and deletion
 - Hardware terminal management and provisioning

See the issue tab for more ideas.

## Getting started
### Components and Layout
`ourspace-backend` is the data management part of our-space. It offers a REST and gRPC API to manage various entities.
The tech stack of the backend is Go, Protobuf/gRPC, gRPC-Gateway and PostgreSQL.

`ourspace-firmware` is the embedded linux image and the software running on the terminal devices that allow member interaction.

`ourspace-frontend` is the frontend component that allows staff to manage Maker Space related data. It is also deployed onto the terminal devices and is the point of interaction with the users.

`pkg` contains common packages, without any business logic.

`scripts` contains helper scripts, e.g. to generate 

### Required software
You need the following software installed to work with the backend:
 - Go
 - Docker
 - Docker Compose
 - Make

Anything else will be installed by the Makefile when running the targets that require additional tools, such as `generate`.

You need the following software installed to work with the frontend:
 - NodeJS (preferably via [fnm](https://github.com/Schniz/fnm))
 - pnpm

Further required tools will be installed via pnpm.

### Development
backend, frontend and the Go parts of the firmware can be executed locally. The backend requires a PostgreSQL database, 
which is available as a Docker container defined in the project root docker-compose.yml. It can be started with 
`docker compose up -d`. The command `make setup` sets the repository up for initial use: It creates a signing key, 
installs dependencies and re-generates the generated files. The backend can be started with `go run` as usual or 
`make run` in the ourspace-backend folder. After running the backend and initializing the database, you can use 
`make create-user` to create an initial user.

### Testing backend changes
To test gRPC requests, you can use [grpcui](https://github.com/fullstorydev/grpcui). The tool can be installed with Go 
(run in the home directory), and can be started with `grpcui -plaintext localhost:50051`. It then offers a UI via 
browser that can be used to send gRPC requests to the server. If `grpcui` can't be executed, then the Go `bin/` folder 
is probably not in the `PATH`.

HTTP requests can be tested with `curl` or other HTTP clients (e.g. Bruno). However, you should probably focus more on 
the request mapping between HTTP and gRPC, as the functionality can be easier tested with grpcui.

### Making changes
We work with trunk based development, in the variant with short-lived feature branches and pull request reviews. 
This means, to change anything, create a (short-lived) branch from the latest main, implement the changes, create a 
pull request to merge back into main. There are no other long-lived branches besides main, like develop or similar.

It's better to make small changes and regularly open PRs, even if this leaves the feature in an unfinished state. 
This software is not yet used in production (to our knowledge), so this is a perfectly fine approach for now. In the 
future, this can be done with feature flags to hide unfinished features.

