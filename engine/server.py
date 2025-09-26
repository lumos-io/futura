#!/usr/bin/env python3
"""
Futura Engine Server

Main gRPC server that implements all three engine services:
- RecommendationService (MPA Server): Orchestrates optimization decisions
- RLServer: ML model serving and lifecycle management
- AgentCoordinator: Training job orchestration

This server can run in different configurations:
- All-in-one: All services in a single process
- Distributed: Each service in separate processes/pods
"""

from proto.gen.engine import engine_pb2_grpc
from services.agent_coordinator import AgentCoordinator
from services.rl_server import RLServer
from services.recommendation_service import RecommendationService
import argparse
from typing import Optional
import logging
import time
import threading
import signal
from concurrent import futures
import grpc


# Import our service implementations

# Import generated gRPC code

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class FuturaEngineServer:
    """
    Main server class that orchestrates all gRPC services.
    """

    def __init__(
        self,
        port: int = 50051,
        enable_recommendation_service: bool = True,
        enable_rl_server: bool = True,
        enable_agent_coordinator: bool = True,
        rl_server_address: Optional[str] = None
    ):
        self.port = port
        self.server = None
        self.shutdown_event = threading.Event()

        # Initialize services
        self.rl_server = None
        self.rl_server_client = None
        self.agent_coordinator = None
        self.recommendation_service = None

        # Setup services based on configuration
        if enable_rl_server:
            logger.info("Initializing RL Server service")
            self.rl_server = RLServer()

        if enable_agent_coordinator:
            logger.info("Initializing Agent Coordinator service")
            self.agent_coordinator = AgentCoordinator()

        # Setup RL Server client for Recommendation Service
        if enable_recommendation_service:
            if rl_server_address and not enable_rl_server:
                # Connect to external RL Server
                logger.info(
                    f"Connecting to external RL Server at {rl_server_address}")
                rl_channel = grpc.insecure_channel(rl_server_address)
                self.rl_server_client = engine_pb2_grpc.RLServerStub(
                    rl_channel)
            elif enable_rl_server:
                # Use internal RL Server (create stub from the same process)
                logger.info("Using internal RL Server")
                # In production, you might want to use a more sophisticated approach
                # For now, we'll pass the server instance directly
                self.rl_server_client = None  # Will be set up during server creation

            logger.info("Initializing Recommendation Service")
            self.recommendation_service = RecommendationService(
                self.rl_server_client)

    def create_server(self) -> grpc.Server:
        """Create and configure the gRPC server."""

        # Create server with thread pool
        server = grpc.server(
            futures.ThreadPoolExecutor(max_workers=50),
            options=[
                ('grpc.keepalive_time_ms', 30000),
                ('grpc.keepalive_timeout_ms', 5000),
                ('grpc.keepalive_permit_without_calls', True),
                ('grpc.http2.max_pings_without_data', 0),
                ('grpc.http2.min_time_between_pings_ms', 10000),
                ('grpc.http2.min_ping_interval_without_data_ms', 300000)
            ]
        )

        # Add services to server
        if self.recommendation_service:
            engine_pb2_grpc.add_RecommendationServiceServicer_to_server(
                self.recommendation_service, server
            )
            logger.info("Recommendation Service added to server")

        if self.rl_server:
            engine_pb2_grpc.add_RLServerServicer_to_server(
                self.rl_server, server
            )
            logger.info("RL Server added to server")

        if self.agent_coordinator:
            engine_pb2_grpc.add_AgentCoordinatorServicer_to_server(
                self.agent_coordinator, server
            )
            logger.info("Agent Coordinator added to server")

        # Add listening port
        listen_addr = f'[::]:{self.port}'
        server.add_insecure_port(listen_addr)

        return server

    def start(self):
        """Start the gRPC server."""
        logger.info(f"Starting Futura Engine Server on port {self.port}")

        self.server = self.create_server()
        self.server.start()

        logger.info(f"Server started successfully on port {self.port}")
        self._log_service_status()

        # Setup graceful shutdown
        def signal_handler(signum, frame):
            logger.info(
                f"Received signal {signum}, initiating graceful shutdown...")
            self.shutdown_event.set()

        signal.signal(signal.SIGTERM, signal_handler)
        signal.signal(signal.SIGINT, signal_handler)

        # Start background tasks
        self._start_background_tasks()

    def wait_for_termination(self):
        """Wait for server termination."""
        try:
            while not self.shutdown_event.is_set():
                time.sleep(1)
        except KeyboardInterrupt:
            logger.info("Keyboard interrupt received, shutting down...")
            self.shutdown_event.set()

        self.stop()

    def stop(self):
        """Stop the gRPC server gracefully."""
        if self.server:
            logger.info("Stopping gRPC server...")
            # Give clients 30 seconds to finish their requests
            self.server.stop(grace=30)
            logger.info("Server stopped successfully")

    def _log_service_status(self):
        """Log which services are enabled."""
        enabled_services = []

        if self.recommendation_service:
            enabled_services.append("RecommendationService")
        if self.rl_server:
            enabled_services.append("RLServer")
        if self.agent_coordinator:
            enabled_services.append("AgentCoordinator")

        logger.info(f"Enabled services: {', '.join(enabled_services)}")

    def _start_background_tasks(self):
        """Start background maintenance tasks."""
        if self.agent_coordinator:
            # Start cleanup task for expired training jobs
            cleanup_thread = threading.Thread(
                target=self._cleanup_task,
                daemon=True,
                name="AgentCoordinatorCleanup"
            )
            cleanup_thread.start()
            logger.info("Started agent coordinator cleanup task")

        if self.rl_server and hasattr(self.rl_server, 'training_job_manager'):
            # Start training job monitoring task
            training_monitor_thread = threading.Thread(
                target=self._training_monitor_task,
                daemon=True,
                name="TrainingJobMonitor"
            )
            training_monitor_thread.start()
            logger.info("Started training job monitoring task")

    def _cleanup_task(self):
        """Background task to clean up expired training jobs and agents."""
        while not self.shutdown_event.is_set():
            try:
                if self.agent_coordinator:
                    self.agent_coordinator.cleanup_expired_training_jobs()
            except Exception as e:
                logger.error(f"Error in cleanup task: {str(e)}")

            # Run cleanup every hour
            self.shutdown_event.wait(3600)

    def _training_monitor_task(self):
        """Background task to monitor Kubernetes training jobs."""
        import asyncio

        async def monitor_loop():
            while not self.shutdown_event.is_set():
                try:
                    if self.rl_server and hasattr(self.rl_server, 'training_job_manager'):
                        # Run the job monitoring and collection
                        await self.rl_server.training_job_manager.monitor_and_collect_jobs()
                except Exception as e:
                    logger.error(f"Error in training job monitoring: {str(e)}")

                # Check every 2 minutes
                await asyncio.sleep(120)

        # Run the async monitoring loop
        try:
            asyncio.run(monitor_loop())
        except Exception as e:
            logger.error(f"Training monitor task failed: {str(e)}")


def main():
    """Main entry point for the server."""
    parser = argparse.ArgumentParser(description='Futura Engine Server')

    parser.add_argument(
        '--port',
        type=int,
        default=50051,
        help='Server port (default: 50051)'
    )

    parser.add_argument(
        '--services',
        type=str,
        default='all',
        choices=['all', 'recommendation', 'rl',
                 'coordinator', 'recommendation+rl'],
        help='Which services to enable (default: all)'
    )

    parser.add_argument(
        '--rl-server-address',
        type=str,
        help='Address of external RL Server (e.g., localhost:50052)'
    )

    parser.add_argument(
        '--log-level',
        type=str,
        default='INFO',
        choices=['DEBUG', 'INFO', 'WARNING', 'ERROR'],
        help='Log level (default: INFO)'
    )

    args = parser.parse_args()

    # Configure logging level
    logging.getLogger().setLevel(getattr(logging, args.log_level))

    # Determine which services to enable
    enable_recommendation = args.services in [
        'all', 'recommendation', 'recommendation+rl']
    enable_rl = args.services in ['all', 'rl', 'recommendation+rl']
    enable_coordinator = args.services in ['all', 'coordinator']

    # Create and start server
    server = FuturaEngineServer(
        port=args.port,
        enable_recommendation_service=enable_recommendation,
        enable_rl_server=enable_rl,
        enable_agent_coordinator=enable_coordinator,
        rl_server_address=args.rl_server_address
    )

    try:
        server.start()
        server.wait_for_termination()
    except Exception as e:
        logger.error(f"Server error: {str(e)}")
        return 1

    return 0


if __name__ == '__main__':
    exit(main())
