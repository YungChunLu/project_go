# A project built with GO

## Prerequisite

* Please add your google maps api key in the `docker-compose.yml` file
* Feel free to change the `password`, `user`, `database name`, but please use the same values in both server and postgres.

## Usage

```bash
# Setup the server and postgres
docker-compose up --build -d

# After both server and postgres are operational, run the unit tests
go test -v server_test.go
```
