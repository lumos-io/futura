# Pipeline

This is the events pipeline with 3 stages: collect, validate and enrich. Each step uses Kafka as main
backbone to persist the messages so that they can be forwarded to the next step or, in case of failure,
they will pushe to the Dead Letter Queue.

## TODO

- [ ] Deploy Kafka Connect to sink the Dead Letter Queue topics to S3 for long term retention
- [ ] Deal with Invalid messages in S3 for post-processing
