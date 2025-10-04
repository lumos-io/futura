"""
ClickHouse client for Futura Engine.

This module provides the connection and query interface to ClickHouse,
replacing the Prometheus adapter functionality from the controller
with direct database queries for metrics and model data.
"""

import logging
from typing import Dict, List, Optional, Any, Tuple, Union
from datetime import datetime, timedelta
import json
import aiohttp

logger = logging.getLogger(__name__)


class EngineDataAccess:
    """High-level data access layer for engine operations."""

    def __init__(self, clickhouse_client):
        self.client = clickhouse_client

    async def store_training_event(self, training_id: str, app_key: str, job_name: str,
                                 event_type: str, status: str, metadata: Dict[str, Any]):
        """Store a training event."""
        # TODO: Implement actual ClickHouse storage
        pass

    async def store_training_result(self, training_id: str, job_name: str, success: bool,
                                  model_version: str = "", final_loss: float = 0.0,
                                  episodes_completed: int = 0, training_time_seconds: int = 0,
                                  model_uri: str = "", metrics: Optional[Dict] = None,
                                  final_reward: float = 0.0, training_metrics: Optional[Dict] = None,
                                  error_message: str = "", completed_at = None):
        """Store training results."""
        # TODO: Implement actual ClickHouse storage
        pass

    async def store_training_cleanup(self, training_id: str, job_name: str):
        """Store training cleanup event."""
        # TODO: Implement actual ClickHouse storage
        pass


class ClickHouseClient:
    """
    Async ClickHouse client for engine data operations.

    Handles connections to both:
    - Engine database: Model registry, decisions, training jobs
    - Analytics database: Kubernetes metrics and telemetry
    """

    def __init__(
        self,
        engine_dsn: str = "http://localhost:8123/engine",
        analytics_dsn: str = "http://localhost:8123/analytics",
        username: Optional[str] = "user",
        password: Optional[str] = "password",
        timeout: int = 30
    ):
        """
        Initialize ClickHouse client.

        Args:
            engine_dsn: Engine database connection string
            analytics_dsn: Analytics database connection string
            username: Database username
            password: Database password
            timeout: Query timeout in seconds
        """
        self.engine_dsn = engine_dsn
        self.analytics_dsn = analytics_dsn
        self.username = username
        self.password = password
        self.timeout = timeout
        self.session: Optional[aiohttp.ClientSession] = None

    async def __aenter__(self):
        """Async context manager entry."""
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        await self.close()

    async def connect(self):
        """Establish connection to ClickHouse."""
        connector = aiohttp.TCPConnector(limit=100, limit_per_host=30)
        timeout = aiohttp.ClientTimeout(total=self.timeout)

        auth = None
        if self.username and self.password:
            auth = aiohttp.BasicAuth(self.username, self.password)

        self.session = aiohttp.ClientSession(
            connector=connector,
            timeout=timeout,
            auth=auth
        )

        # Test connections
        try:
            await self._test_connection(self.engine_dsn)
            await self._test_connection(self.analytics_dsn)
            logger.info("ClickHouse connections established successfully")
        except Exception as e:
            logger.error(f"Failed to connect to ClickHouse: {str(e)}")
            raise

    async def close(self):
        """Close ClickHouse connection."""
        if self.session:
            await self.session.close()
            self.session = None

    async def _test_connection(self, dsn: str):
        """Test connection to specific database."""
        if not self.session:
            raise RuntimeError("Client not connected")

        async with self.session.get(f"{dsn}?query=SELECT 1") as response:
            if response.status != 200:
                raise RuntimeError(
                    f"Connection test failed: {response.status}")

    async def execute_query(
        self,
        query: str,
        params: Optional[Dict[str, Any]] = None,
        database: str = "engine"
    ) -> List[Dict[str, Any]]:
        """
        Execute SELECT query and return results.

        Args:
            query: SQL query string
            params: Query parameters
            database: Target database ("engine" or "analytics")

        Returns:
            List of result dictionaries
        """
        dsn = self.engine_dsn if database == "engine" else self.analytics_dsn

        if not self.session:
            raise RuntimeError("Client not connected")

        # Format query with parameters if provided
        if params:
            query = query.format(**params)

        logger.debug(f"Executing query on {database}: {query[:200]}...")

        try:
            async with self.session.post(
                dsn,
                data=query,
                headers={"Content-Type": "text/plain"},
                params={"default_format": "JSONEachRow"}
            ) as response:
                if response.status != 200:
                    error_text = await response.text()
                    raise RuntimeError(
                        f"Query failed: {response.status} - {error_text}")

                result_text = await response.text()
                if not result_text.strip():
                    return []

                # Parse JSONEachRow format
                results = []
                for line in result_text.strip().split('\n'):
                    if line.strip():
                        results.append(json.loads(line))

                logger.debug(f"Query returned {len(results)} rows")
                return results

        except Exception as e:
            logger.error(f"Query execution failed: {str(e)}")
            raise

    async def execute_insert(
        self,
        table: str,
        data: Union[Dict[str, Any], List[Dict[str, Any]]],
        database: str = "engine"
    ) -> bool:
        """
        Insert data into table.

        Args:
            table: Target table name
            data: Data to insert (single dict or list of dicts)
            database: Target database ("engine" or "analytics")

        Returns:
            Success status
        """
        dsn = self.engine_dsn if database == "engine" else self.analytics_dsn

        if not self.session:
            raise RuntimeError("Client not connected")

        # Normalize data to list
        if isinstance(data, dict):
            data = [data]

        # Convert to JSONEachRow format
        json_lines = []
        for row in data:
            json_lines.append(json.dumps(row, default=str))

        insert_data = '\n'.join(json_lines)
        query = f"INSERT INTO {table} FORMAT JSONEachRow"

        logger.debug(f"Inserting {len(data)} rows into {database}.{table}")

        try:
            async with self.session.post(
                dsn,
                data=f"{query}\n{insert_data}",
                headers={"Content-Type": "text/plain"}
            ) as response:
                if response.status != 200:
                    error_text = await response.text()
                    logger.error(
                        f"Insert failed: {response.status} - {error_text}")
                    return False

                logger.debug(f"Successfully inserted {len(data)} rows")
                return True

        except Exception as e:
            logger.error(f"Insert execution failed: {str(e)}")
            return False


class EngineDataAccess:
    """
    Data access layer for engine-specific operations.

    Handles model registry, decisions, training jobs, etc.
    """

    def __init__(self, clickhouse_client: ClickHouseClient):
        self.client = clickhouse_client

    async def store_recommendation_decision(
        self,
        cluster_id: str,
        namespace: str,
        app_name: str,
        workload_kind: str,
        decision_id: str,
        model_version: str,
        confidence: float,
        audit_reasons: List[str],
        plan_vertical: List[Dict[str, str]],
        plan_replicas: int,
        effective_policy: Dict[str, str]
    ) -> bool:
        """Store recommendation decision for audit trail."""

        decision_data = {
            "cluster_id": cluster_id,
            "namespace": namespace,
            "app_name": app_name,
            "workload_kind": workload_kind,
            "decision_id": decision_id,
            "model_version": model_version,
            "confidence": confidence,
            "audit_reasons": audit_reasons,
            "plan_vertical": plan_vertical,
            "plan_replicas": plan_replicas,
            "effective_policy": effective_policy,
            "ts": datetime.utcnow()
        }

        return await self.client.execute_insert(
            "recommendation_decisions",
            decision_data,
            database="engine"
        )

    async def store_execution_outcome(
        self,
        cluster_id: str,
        namespace: str,
        app_name: str,
        workload_kind: str,
        decision_id: str,
        success: bool,
        note: str,
        post_action_metrics: Dict[str, float],
        reported_at: datetime
    ) -> bool:
        """Store execution outcome for learning."""

        outcome_data = {
            "cluster_id": cluster_id,
            "namespace": namespace,
            "app_name": app_name,
            "workload_kind": workload_kind,
            "decision_id": decision_id,
            "success": 1 if success else 0,
            "note": note,
            "post_action_metrics": post_action_metrics,
            "reported_at": reported_at,
            "ts": datetime.utcnow()
        }

        return await self.client.execute_insert(
            "execution_outcomes",
            outcome_data,
            database="engine"
        )

    async def register_model(
        self,
        cluster_id: str,
        namespace: str,
        app_name: str,
        workload_kind: str,
        model_version: str,
        policy_name: str,
        checkpoint_uri: str,
        labels: Dict[str, str],
        compatible_feature_schema: List[str]
    ) -> bool:
        """Register model in the registry."""

        model_data = {
            "cluster_id": cluster_id,
            "namespace": namespace,
            "app_name": app_name,
            "workload_kind": workload_kind,
            "model_version": model_version,
            "policy_name": policy_name,
            "checkpoint_uri": checkpoint_uri,
            "labels": labels,
            "compatible_feature_schema": compatible_feature_schema,
            "updated_at": datetime.utcnow(),
            "ts": datetime.utcnow()
        }

        return await self.client.execute_insert(
            "model_registry",
            model_data,
            database="engine"
        )

    async def get_latest_model(
        self,
        cluster_id: str,
        namespace: str,
        app_name: str
    ) -> Optional[Dict[str, Any]]:
        """Get latest model for application."""

        query = """
        SELECT * FROM model_registry
        WHERE cluster_id = '{cluster_id}'
        AND namespace = '{namespace}'
        AND app_name = '{app_name}'
        ORDER BY updated_at DESC
        LIMIT 1
        """

        results = await self.client.execute_query(
            query,
            params={
                "cluster_id": cluster_id,
                "namespace": namespace,
                "app_name": app_name
            },
            database="engine"
        )

        return results[0] if results else None

    async def create_training_job(
        self,
        training_id: str,
        cluster_id: str,
        namespace: str,
        app_name: str,
        workload_kind: str,
        reason: str,
        horizon_hours: int,
        base_version: str,
        hparams: Dict[str, str],
        job_name: str
    ) -> bool:
        """Create training job record."""

        job_data = {
            "training_id": training_id,
            "cluster_id": cluster_id,
            "namespace": namespace,
            "app_name": app_name,
            "workload_kind": workload_kind,
            "reason": reason,
            "horizon_hours": horizon_hours,
            "base_version": base_version,
            "hparams": hparams,
            "job_name": job_name,
            "status": "PENDING",
            "error_message": "",
            "created_at": datetime.utcnow(),
            "updated_at": datetime.utcnow()
        }

        return await self.client.execute_insert(
            "training_jobs",
            job_data,
            database="engine"
        )

    async def update_training_job_status(
        self,
        training_id: str,
        status: str,
        error_message: str = ""
    ) -> bool:
        """Update training job status."""

        # Note: ClickHouse doesn't support direct UPDATE, so we insert a new record
        # The ReplacingMergeTree will handle deduplication based on updated_at

        update_data = {
            "training_id": training_id,
            "status": status,
            "error_message": error_message,
            "updated_at": datetime.utcnow()
        }

        # TODO: Implement proper UPDATE mechanism or use mutations
        logger.warning(
            f"Training job status update needs proper UPDATE implementation: {training_id} -> {status}")
        return True

    async def store_training_progress(
        self,
        training_id: str,
        agent_id: str,
        step: int,
        train_loss: float,
        eval_reward: float,
        progress_pct: float,
        scalars: Dict[str, float]
    ) -> bool:
        """Store training progress."""

        progress_data = {
            "training_id": training_id,
            "agent_id": agent_id,
            "step": step,
            "train_loss": train_loss,
            "eval_reward": eval_reward,
            "progress_pct": progress_pct,
            "scalars": scalars,
            "ts": datetime.utcnow()
        }

        return await self.client.execute_insert(
            "training_progress",
            progress_data,
            database="engine"
        )

    async def store_training_result(
        self,
        training_id: str,
        agent_id: str,
        success: bool,
        error_message: str,
        model_version: str,
        checkpoint_uri: str,
        eval_reward: float,
        metrics: Dict[str, float]
    ) -> bool:
        """Store final training result."""

        result_data = {
            "training_id": training_id,
            "agent_id": agent_id,
            "success": 1 if success else 0,
            "error_message": error_message,
            "model_version": model_version,
            "checkpoint_uri": checkpoint_uri,
            "eval_reward": eval_reward,
            "metrics": metrics,
            "ts": datetime.utcnow()
        }

        return await self.client.execute_insert(
            "training_results",
            result_data,
            database="engine"
        )

    async def get_recent_decisions(
        self,
        cluster_id: str,
        namespace: str,
        app_name: str,
        hours: int = 24
    ) -> List[Dict[str, Any]]:
        """Get recent recommendation decisions for analysis."""

        since_time = datetime.utcnow() - timedelta(hours=hours)

        query = """
        SELECT * FROM recommendation_decisions
        WHERE cluster_id = '{cluster_id}'
        AND namespace = '{namespace}'
        AND app_name = '{app_name}'
        AND ts >= '{since_time}'
        ORDER BY ts DESC
        """

        return await self.client.execute_query(
            query,
            params={
                "cluster_id": cluster_id,
                "namespace": namespace,
                "app_name": app_name,
                "since_time": since_time.isoformat()
            },
            database="engine"
        )

    async def get_outcome_success_rate(
        self,
        cluster_id: str,
        namespace: str,
        app_name: str,
        hours: int = 24
    ) -> float:
        """Calculate recent execution success rate for drift detection."""

        since_time = datetime.utcnow() - timedelta(hours=hours)

        query = """
        SELECT
            countIf(success = 1) as successes,
            count() as total
        FROM execution_outcomes
        WHERE cluster_id = '{cluster_id}'
        AND namespace = '{namespace}'
        AND app_name = '{app_name}'
        AND reported_at >= '{since_time}'
        """

        results = await self.client.execute_query(
            query,
            params={
                "cluster_id": cluster_id,
                "namespace": namespace,
                "app_name": app_name,
                "since_time": since_time.isoformat()
            },
            database="engine"
        )

        if not results or results[0]["total"] == 0:
            return 1.0  # Default to success if no data

        return results[0]["successes"] / results[0]["total"]
