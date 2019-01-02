# productreview

## Set up the database.
Run migration.sql to modify your Postgres database.

## Commands
The system is composed of three packages. Dependencies are vendorized with go mod.

* cmd/server
* cmd/processor
* cmd/notifier

To run directly you can use: `make server`, `make processor`, and `make notifier`.

`-help` flag shows what flags are accepted. [AdventureWorks-for-Postgres](https://github.com/lorint/AdventureWorks-for-Postgres) and unprotected redis are assumed as default.

## Execution
Logs are currently generated with logrus and printed on stderr.

Test with something like:

```bash
curl -XPOST http://localhost:8888/api/reviews --data '{"productid": 43, "name": "abcd", "email": "foo@example.com", "review": "the banking fee is too high", "rating": 2}'
```

## TODO
No failover measure is assumed. A cmd/fixreviews program might list all reviews not processed yet and enqueue them once again. This should be run while the processor program is not running to avoid duplicated workloads.
