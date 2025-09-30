#!/usr/bin/env python3
"""
Futura Engine - Main Entry Point
"""

import argparse
import logging
import signal

from config import EngineConfig
from server import FuturaEngineServer

logger = logging.getLogger(__name__)


def main():
    parser = argparse.ArgumentParser(description='Futura Engine')
    parser.add_argument('--config', type=str, default='config.toml',
                        help='Path to configuration file (default: config.toml)')
    args = parser.parse_args()

    # Load configuration
    try:
        config = EngineConfig.from_toml(args.config)
    except FileNotFoundError:
        print(f"Error: Configuration file not found: {args.config}")
        return 1
    except Exception as e:
        print(f"Error: Failed to load configuration: {str(e)}")
        return 1

    # Configure logging
    logging.basicConfig(
        level=getattr(logging, config.server.log_level),
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Determine which services to enable based on config.service
    enable_recommendation = config.service in ["all", "recommendation"]
    enable_rl = config.service in ["all", "rl"]
    enable_coordinator = config.service in ["all", "agent-coordinator"]

    logger.info(f"Starting Futura Engine (service: {config.service})")
    logger.info(
        f"Recommendation={enable_recommendation}, RL={enable_rl}, Coordinator={enable_coordinator}")

    # Create and start server
    server = FuturaEngineServer(
        port=config.server.port,
        enable_recommendation_service=enable_recommendation,
        enable_rl_server=enable_rl,
        enable_agent_coordinator=enable_coordinator,
        rl_server_address=config.rl_server_address
    )

    # Handle shutdown signals
    def signal_handler(signum, frame):
        logger.info(f"Received signal {signum}, shutting down...")
        server.stop()

    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)

    try:
        server.start()
        logger.info(
            f"🚀 Futura Engine started on {config.server.host}:{config.server.port}")
        server.wait_for_termination()
    except Exception as e:
        logger.error(f"Server error: {str(e)}")
        return 1

    return 0


if __name__ == "__main__":
    exit(main())
