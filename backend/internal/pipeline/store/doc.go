// The purpose of the Storer command is to fetch the messages from the enriched kafka topics
// and split each message in the appropriate store kafka topic. This is necessary because
// each table in ClickHouse is using the Apache Kafka engine to automatically ingest the
// messages so that they can appear in near real-time. In the background there, ClickHouse is
// using Materialize Views to make the query faster.

package store
