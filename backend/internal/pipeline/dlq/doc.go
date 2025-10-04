// This package represents the Dead Letter Queue for messages that fail to pass
// either the validate or enrichment step. Each of the invalid message is pushed
// to the appropriate Kafka topic and there is a Kafka Connect service that consumes
// those messages and push them to S3. The Kafka Connect is external to the code
// and it is deployed separately

package dlq
