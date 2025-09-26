#!/usr/bin/env python3
"""
Futura Engine CLI

Command-line interface for starting individual services or combinations
of services in the Futura ML-powered Kubernetes optimization engine.
"""

import argparse
import asyncio
import logging
import os
import sys
import signal
import threading
import time
from typing import Optional, List

from server import FuturaEngineServer
from config import EngineConfig

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class FuturaCLI:
    """
    CLI interface for the Futura Engine with individual service startup.
    """

    def __init__(self):
        self.server: Optional[FuturaEngineServer] = None
        self.shutdown_event = threading.Event()

    def create_parser(self) -> argparse.ArgumentParser:
        """Create the CLI argument parser."""

        parser = argparse.ArgumentParser(
            description='Futura Engine - ML-powered Kubernetes optimization',
            epilog='Examples:\n'
                   '  futura-engine recommendation-service --port 8080\n'
                   '  futura-engine rl-server --clickhouse-url http://localhost:8123\n'
                   '  futura-engine agent-coordinator --namespace futura-training\n'
                   '  futura-engine all-services --port 8080\n',
            formatter_class=argparse.RawDescriptionHelpFormatter
        )

        # Global options
        parser.add_argument('--debug', action='store_true',
                           help='Enable debug logging')
        parser.add_argument('--config-file', type=str,
                           help='Path to configuration file')

        # Service selection subcommands
        subparsers = parser.add_subparsers(dest='command', help='Service to start')

        # Recommendation Service
        rec_parser = subparsers.add_parser('recommendation-service',
                                         help='Start only the Recommendation Service')
        self._add_common_args(rec_parser)
        self._add_clickhouse_args(rec_parser)
        rec_parser.add_argument('--rl-server-address', type=str,
                               default='localhost:8081',
                               help='Address of RL Server for ML recommendations')

        # RL Server
        rl_parser = subparsers.add_parser('rl-server',
                                        help='Start only the RL Server')
        self._add_common_args(rl_parser)
        self._add_clickhouse_args(rl_parser)
        self._add_training_args(rl_parser)

        # Agent Coordinator
        agent_parser = subparsers.add_parser('agent-coordinator',
                                           help='Start only the Agent Coordinator')
        self._add_common_args(agent_parser)

        # All Services
        all_parser = subparsers.add_parser('all-services',
                                         help='Start all services together')
        self._add_common_args(all_parser)
        self._add_clickhouse_args(all_parser)
        self._add_training_args(all_parser)

        # Health check
        health_parser = subparsers.add_parser('health',
                                            help='Check service health')
        health_parser.add_argument('--endpoint', type=str,
                                 default='localhost:8080',
                                 help='Service endpoint to check')

        # Version
        subparsers.add_parser('version', help='Show version information')

        return parser

    def _add_common_args(self, parser: argparse.ArgumentParser):
        """Add common arguments to a parser."""
        parser.add_argument('--port', type=int, default=8080,
                          help='Port to listen on (default: 8080)')
        parser.add_argument('--host', type=str, default='0.0.0.0',
                          help='Host to bind to (default: 0.0.0.0)')

    def _add_clickhouse_args(self, parser: argparse.ArgumentParser):
        """Add ClickHouse-related arguments."""
        parser.add_argument('--clickhouse-url', type=str,
                          default='http://localhost:8123',
                          help='ClickHouse server URL')
        parser.add_argument('--clickhouse-engine-db', type=str,
                          default='engine',
                          help='ClickHouse engine database name')
        parser.add_argument('--clickhouse-analytics-db', type=str,
                          default='analytics',
                          help='ClickHouse analytics database name')

    def _add_training_args(self, parser: argparse.ArgumentParser):
        """Add training-related arguments."""
        parser.add_argument('--training-namespace', type=str,
                          default='futura-training',
                          help='Kubernetes namespace for training jobs')
        parser.add_argument('--training-image', type=str,
                          default='futura/rl-trainer:latest',
                          help='Container image for training jobs')
        parser.add_argument('--model-storage-uri', type=str,
                          default='s3://futura-models',
                          help='S3 URI for model storage')

    def run(self, args: List[str] = None) -> int:
        """Run the CLI with given arguments."""

        parser = self.create_parser()
        parsed_args = parser.parse_args(args)

        # Configure logging level
        if parsed_args.debug:
            logging.getLogger().setLevel(logging.DEBUG)
            logger.debug("Debug logging enabled")

        # Handle commands
        if parsed_args.command == 'version':
            return self._show_version()
        elif parsed_args.command == 'health':
            return self._check_health(parsed_args)
        elif parsed_args.command in ['recommendation-service', 'rl-server', 'agent-coordinator', 'all-services']:
            return self._start_services(parsed_args)
        else:
            parser.print_help()
            return 1

    def _show_version(self) -> int:
        """Show version information."""
        print("Futura Engine v0.1.0")
        print("ML-powered Kubernetes optimization engine")
        print("Built with Python, gRPC, ClickHouse, and Kubernetes")
        return 0

    def _check_health(self, args: argparse.Namespace) -> int:
        """Check service health."""
        try:
            import grpc
            from proto.gen.engine import engine_pb2_grpc, engine_pb2

            # Try to connect to the service
            channel = grpc.insecure_channel(args.endpoint)

            # Try recommendation service first
            stub = engine_pb2_grpc.RecommendationServiceStub(channel)

            # Create a simple health check request (list models)
            app_ref = engine_pb2.AppRef(
                api_key="health-check",
                namespace="default",
                app_name="health-check"
            )
            request = engine_pb2.ListModelsRequest(app=app_ref)

            # Set a short timeout
            response = stub.ListModels(request, timeout=5.0)

            print(f"✅ Service at {args.endpoint} is healthy")
            return 0

        except Exception as e:
            print(f"❌ Service at {args.endpoint} is unhealthy: {str(e)}")
            return 1

    def _start_services(self, args: argparse.Namespace) -> int:
        """Start the specified services."""

        try:
            # Determine which services to enable
            enable_recommendation = args.command in ['recommendation-service', 'all-services']
            enable_rl = args.command in ['rl-server', 'all-services']
            enable_coordinator = args.command in ['agent-coordinator', 'all-services']

            logger.info(f"Starting Futura Engine: {args.command}")
            logger.info(f"Services: Recommendation={enable_recommendation}, "
                       f"RL={enable_rl}, Coordinator={enable_coordinator}")

            # Create RL server address for recommendation service
            rl_server_address = None
            if enable_recommendation and enable_rl:
                rl_server_address = f"localhost:{args.port}"
            elif enable_recommendation and hasattr(args, 'rl_server_address'):
                rl_server_address = args.rl_server_address

            # Create server instance
            self.server = FuturaEngineServer(
                port=args.port,
                enable_recommendation_service=enable_recommendation,
                enable_rl_server=enable_rl,
                enable_agent_coordinator=enable_coordinator,
                rl_server_address=rl_server_address
            )

            # Set up signal handlers
            signal.signal(signal.SIGTERM, self._signal_handler)
            signal.signal(signal.SIGINT, self._signal_handler)

            # Start the server
            self.server.start()

            logger.info(f"🚀 Futura Engine started successfully on port {args.port}")
            self._print_service_info(args, enable_recommendation, enable_rl, enable_coordinator)

            # Wait for termination
            self.server.wait_for_termination()

            return 0

        except Exception as e:
            logger.error(f"Failed to start services: {str(e)}")
            return 1

    def _signal_handler(self, signum, frame):
        """Handle shutdown signals."""
        logger.info(f"Received signal {signum}, initiating graceful shutdown...")
        self.shutdown_event.set()
        if self.server:
            self.server.stop()

    def _print_service_info(self, args: argparse.Namespace, rec: bool, rl: bool, coord: bool):
        """Print information about started services."""

        print(f"\n🎯 Futura Engine Services Running on {args.host}:{args.port}")
        print("=" * 60)

        if rec:
            print("📋 Recommendation Service")
            print("   ├── Endpoint: RecommendationService")
            print("   ├── Methods: GetRecommendation, SyncSLO, SyncClusterConfig")
            print("   └── Purpose: Main optimization recommendations")

        if rl:
            print("🧠 RL Server")
            print("   ├── Endpoint: RLServer")
            print("   ├── Methods: GetAction, TriggerTrain, EnsureModel")
            print("   └── Purpose: ML-powered decision making")

        if coord:
            print("🔧 Agent Coordinator")
            print("   ├── Endpoint: AgentCoordinator")
            print("   ├── Methods: RegisterAgent, GetAssignment, ReportProgress")
            print("   └── Purpose: Training job coordination")

        print("\n🔗 External Dependencies:")
        if hasattr(args, 'clickhouse_url'):
            print(f"   ├── ClickHouse: {args.clickhouse_url}")
        if hasattr(args, 'training_namespace'):
            print(f"   ├── Training Namespace: {args.training_namespace}")
        if hasattr(args, 'model_storage_uri'):
            print(f"   └── Model Storage: {args.model_storage_uri}")

        print(f"\n💡 Health Check: futura-engine health --endpoint {args.host}:{args.port}")
        print("🛑 Stop: Ctrl+C or SIGTERM")
        print("=" * 60)


def main():
    """Main entry point for the CLI."""
    cli = FuturaCLI()
    return cli.run()


if __name__ == '__main__':
    sys.exit(main())