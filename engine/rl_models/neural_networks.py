"""
PyTorch neural networks for RL-based Kubernetes autoscaling.

This module implements the neural network architectures for the PPO and Meta-PPO
agents used in the Futura engine, including actor-critic networks and trajectory
encoders for meta-learning.
"""

import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.optim as optim
import numpy as np
from typing import List, Dict, Any, Optional, Tuple
import logging
import os

logger = logging.getLogger(__name__)

# Device detection
def get_device():
    """Get the best available device (CUDA if available, otherwise CPU)."""
    if torch.cuda.is_available():
        return torch.device("cuda")
    else:
        return torch.device("cpu")

DEVICE = get_device()


class ActorNetwork(nn.Module):
    """
    Actor network for PPO agent.

    Maps state observations to action probabilities for discrete actions:
    - 0: No action
    - 1: Scale out (horizontal +1)
    - 2: Scale in (horizontal -1)
    - 3: Scale up CPU (vertical CPU +1)
    - 4: Scale down CPU (vertical CPU -1)
    - 5: Scale up memory (vertical memory +1)
    - 6: Scale down memory (vertical memory -1)
    """

    def __init__(self, state_dim: int, action_dim: int, hidden_dim: int = 128):
        super(ActorNetwork, self).__init__()

        self.fc1 = nn.Linear(state_dim, hidden_dim)
        self.fc2 = nn.Linear(hidden_dim, hidden_dim)
        self.fc3 = nn.Linear(hidden_dim, action_dim)

        # Initialize weights
        self._initialize_weights()

        # Move to device
        self.to(DEVICE)

    def _initialize_weights(self):
        """Initialize network weights using Xavier initialization."""
        for layer in [self.fc1, self.fc2, self.fc3]:
            nn.init.xavier_uniform_(layer.weight)
            nn.init.zeros_(layer.bias)

    def forward(self, state: torch.Tensor) -> torch.Tensor:
        """Forward pass through the actor network."""
        x = F.relu(self.fc1(state))
        x = F.relu(self.fc2(x))
        action_logits = self.fc3(x)

        # Apply softmax to get action probabilities
        action_probs = F.softmax(action_logits, dim=-1)

        return action_probs


class CriticNetwork(nn.Module):
    """
    Critic network for PPO agent.

    Maps state observations to state value estimates.
    """

    def __init__(self, state_dim: int, hidden_dim: int = 128):
        super(CriticNetwork, self).__init__()

        self.fc1 = nn.Linear(state_dim, hidden_dim)
        self.fc2 = nn.Linear(hidden_dim, hidden_dim)
        self.fc3 = nn.Linear(hidden_dim, 1)  # Single value output

        # Initialize weights
        self._initialize_weights()

        # Move to device
        self.to(DEVICE)

    def _initialize_weights(self):
        """Initialize network weights using Xavier initialization."""
        for layer in [self.fc1, self.fc2, self.fc3]:
            nn.init.xavier_uniform_(layer.weight)
            nn.init.zeros_(layer.bias)

    def forward(self, state: torch.Tensor) -> torch.Tensor:
        """Forward pass through the critic network."""
        x = F.relu(self.fc1(state))
        x = F.relu(self.fc2(x))
        value = self.fc3(x)

        return value


class TrajectoryEncoder(nn.Module):
    """
    RNN-based trajectory encoder for meta-learning.

    Encodes sequences of (state, action, reward) tuples into
    trajectory embeddings for meta-PPO.
    """

    def __init__(self, state_dim: int, action_dim: int, hidden_dim: int = 64, output_dim: int = 32):
        super(TrajectoryEncoder, self).__init__()

        # Input dimension: state + action + reward
        input_dim = state_dim + action_dim + 1

        self.rnn = nn.LSTM(input_dim, hidden_dim, batch_first=True)
        self.output_layer = nn.Linear(hidden_dim, output_dim)

        # Move to device
        self.to(DEVICE)

    def forward(self, trajectory: torch.Tensor) -> torch.Tensor:
        """
        Encode a trajectory into a fixed-size embedding.

        Args:
            trajectory: Tensor of shape (batch_size, sequence_length, input_dim)

        Returns:
            Trajectory embedding of shape (batch_size, output_dim)
        """
        # Pass through LSTM
        lstm_out, (hidden, cell) = self.rnn(trajectory)

        # Use the final hidden state as trajectory representation
        trajectory_embedding = self.output_layer(hidden[-1])

        return trajectory_embedding


class PPOTrainer:
    """
    Proximal Policy Optimization (PPO) trainer for Kubernetes autoscaling.

    Implements the PPO algorithm with clipped objective and advantage estimation.
    """

    def __init__(
        self,
        state_dim: int,
        action_dim: int = 7,  # 7 discrete actions
        learning_rate: float = 3e-4,
        hidden_dim: int = 128,
        gamma: float = 0.99,
        epsilon: float = 0.2,
        epochs: int = 10
    ):
        self.state_dim = state_dim
        self.action_dim = action_dim
        self.gamma = gamma
        self.epsilon = epsilon
        self.epochs = epochs

        # Networks
        self.actor = ActorNetwork(state_dim, action_dim, hidden_dim)
        self.critic = CriticNetwork(state_dim, hidden_dim)

        # Optimizers
        self.actor_optimizer = optim.Adam(self.actor.parameters(), lr=learning_rate)
        self.critic_optimizer = optim.Adam(self.critic.parameters(), lr=learning_rate)

        logger.info(f"Initialized PPO trainer with state_dim={state_dim}, action_dim={action_dim}")

    def select_action(self, state: List[float]) -> int:
        """Select an action given the current state."""
        state_tensor = torch.FloatTensor(state).unsqueeze(0).to(DEVICE)

        with torch.no_grad():
            action_probs = self.actor(state_tensor)

        # Sample action from probability distribution
        action_dist = torch.distributions.Categorical(action_probs)
        action = action_dist.sample()

        return action.item()

    def update(
        self,
        states: List[List[float]],
        actions: List[int],
        rewards: List[float],
        next_states: List[List[float]],
        dones: List[bool]
    ):
        """Update the actor and critic networks using PPO."""
        # Convert to tensors
        states_tensor = torch.FloatTensor(states).to(DEVICE)
        actions_tensor = torch.LongTensor(actions).to(DEVICE)
        rewards_tensor = torch.FloatTensor(rewards).to(DEVICE)
        next_states_tensor = torch.FloatTensor(next_states).to(DEVICE)
        dones_tensor = torch.BoolTensor(dones).to(DEVICE)

        # Calculate advantages and returns
        with torch.no_grad():
            values = self.critic(states_tensor).squeeze()
            next_values = self.critic(next_states_tensor).squeeze()

            # Calculate returns using GAE
            returns = self._calculate_returns(rewards_tensor, next_values, dones_tensor)
            advantages = returns - values

            # Normalize advantages
            advantages = (advantages - advantages.mean()) / (advantages.std() + 1e-8)

            # Get old action probabilities
            old_action_probs = self.actor(states_tensor)
            old_log_probs = torch.log(old_action_probs.gather(1, actions_tensor.unsqueeze(1))).squeeze()

        # PPO update loop
        for _ in range(self.epochs):
            # Current action probabilities
            current_action_probs = self.actor(states_tensor)
            current_log_probs = torch.log(current_action_probs.gather(1, actions_tensor.unsqueeze(1))).squeeze()

            # Calculate ratio
            ratio = torch.exp(current_log_probs - old_log_probs)

            # Calculate clipped objective
            surr1 = ratio * advantages
            surr2 = torch.clamp(ratio, 1 - self.epsilon, 1 + self.epsilon) * advantages
            actor_loss = -torch.min(surr1, surr2).mean()

            # Update actor
            self.actor_optimizer.zero_grad()
            actor_loss.backward()
            self.actor_optimizer.step()

            # Update critic
            current_values = self.critic(states_tensor).squeeze()
            critic_loss = F.mse_loss(current_values, returns)

            self.critic_optimizer.zero_grad()
            critic_loss.backward()
            self.critic_optimizer.step()

    def _calculate_returns(self, rewards: torch.Tensor, next_values: torch.Tensor, dones: torch.Tensor) -> torch.Tensor:
        """Calculate discounted returns using GAE."""
        returns = torch.zeros_like(rewards)

        # Bootstrap from next value if not done
        next_return = torch.where(dones[-1], torch.zeros(1).to(DEVICE), next_values[-1])

        for t in reversed(range(len(rewards))):
            returns[t] = rewards[t] + self.gamma * next_return * (1 - dones[t].float())
            next_return = returns[t]

        return returns

    def save_model(self, path: str):
        """Save the actor and critic networks."""
        torch.save({
            'actor_state_dict': self.actor.state_dict(),
            'critic_state_dict': self.critic.state_dict(),
            'actor_optimizer': self.actor_optimizer.state_dict(),
            'critic_optimizer': self.critic_optimizer.state_dict(),
        }, path)
        logger.info(f"Model saved to {path}")

    def load_model(self, path: str):
        """Load the actor and critic networks."""
        checkpoint = torch.load(path, map_location=DEVICE)

        self.actor.load_state_dict(checkpoint['actor_state_dict'])
        self.critic.load_state_dict(checkpoint['critic_state_dict'])
        self.actor_optimizer.load_state_dict(checkpoint['actor_optimizer'])
        self.critic_optimizer.load_state_dict(checkpoint['critic_optimizer'])

        logger.info(f"Model loaded from {path}")


class MetaPPOTrainer(PPOTrainer):
    """
    Meta-PPO trainer that learns to adapt quickly to new tasks.

    Extends PPO with trajectory encoding for meta-learning across
    different applications and environments.
    """

    def __init__(
        self,
        state_dim: int,
        action_dim: int = 7,
        learning_rate: float = 3e-4,
        hidden_dim: int = 128,
        trajectory_dim: int = 32,
        meta_lr: float = 1e-3,
        **kwargs
    ):
        super().__init__(state_dim, action_dim, learning_rate, hidden_dim, **kwargs)

        # Trajectory encoder for meta-learning
        self.trajectory_encoder = TrajectoryEncoder(
            state_dim, action_dim, hidden_dim=64, output_dim=trajectory_dim
        )

        # Meta-optimizer for trajectory encoder
        self.meta_optimizer = optim.Adam(self.trajectory_encoder.parameters(), lr=meta_lr)

        # Modify actor and critic to take trajectory embeddings as additional input
        self.actor = MetaActorNetwork(state_dim + trajectory_dim, action_dim, hidden_dim)
        self.critic = MetaCriticNetwork(state_dim + trajectory_dim, hidden_dim)
        self.actor_optimizer = optim.Adam(self.actor.parameters(), lr=learning_rate)
        self.critic_optimizer = optim.Adam(self.critic.parameters(), lr=learning_rate)

        logger.info(f"Initialized Meta-PPO trainer with trajectory_dim={trajectory_dim}")

    def encode_trajectory(self, trajectory: Dict[str, List]) -> torch.Tensor:
        """
        Encode a trajectory into a fixed-size embedding.

        Args:
            trajectory: Dict with 'states', 'actions', 'rewards' keys

        Returns:
            Trajectory embedding tensor
        """
        states = torch.FloatTensor(trajectory['states'])
        actions = torch.LongTensor(trajectory['actions'])
        rewards = torch.FloatTensor(trajectory['rewards'])

        # One-hot encode actions
        actions_onehot = F.one_hot(actions, num_classes=self.action_dim).float()

        # Combine state, action, reward
        trajectory_input = torch.cat([
            states,
            actions_onehot,
            rewards.unsqueeze(-1)
        ], dim=-1)

        # Add batch dimension
        trajectory_input = trajectory_input.unsqueeze(0).to(DEVICE)

        # Encode trajectory
        embedding = self.trajectory_encoder(trajectory_input)

        return embedding.squeeze(0)

    def meta_update(self, tasks_data: List[Dict[str, List]]):
        """
        Perform First-Order Meta-Learning update across multiple tasks (FOMAML-style).

        This implements a simplified meta-learning approach where:
        1. Each task gets a trajectory embedding for context
        2. We perform gradient-based updates on each task
        3. Meta-parameters are updated to improve few-shot learning

        Args:
            tasks_data: List of trajectory dictionaries from different tasks
        """
        if not tasks_data:
            return

        total_meta_loss = 0.0
        total_tasks = len(tasks_data)

        # Accumulate gradients across tasks
        self.actor_optimizer.zero_grad()
        self.critic_optimizer.zero_grad()
        self.meta_optimizer.zero_grad()

        for task_idx, task_data in enumerate(tasks_data):
            try:
                # Encode task trajectory for context
                trajectory_embedding = self.encode_trajectory(task_data)

                # Prepare task data
                states = task_data['states']
                actions = task_data['actions']
                rewards = task_data['rewards']

                # Validate input dimensions
                if len(states) != len(actions) or len(actions) != len(rewards):
                    logger.warning(f"Task {task_idx}: Mismatched data lengths")
                    continue

                # Augment states with trajectory embedding
                augmented_states = []
                for state in states:
                    if len(state) != self.state_dim:
                        logger.warning(f"Task {task_idx}: State dimension mismatch")
                        continue

                    augmented_state = torch.cat([
                        torch.FloatTensor(state).to(DEVICE),
                        trajectory_embedding.to(DEVICE)
                    ])
                    augmented_states.append(augmented_state)

                if not augmented_states:
                    logger.warning(f"Task {task_idx}: No valid states")
                    continue

                # Convert to tensors
                states_tensor = torch.stack(augmented_states)
                actions_tensor = torch.LongTensor(actions[:len(augmented_states)]).to(DEVICE)
                rewards_tensor = torch.FloatTensor(rewards[:len(augmented_states)]).to(DEVICE)

                # Ensure actions are within valid range
                actions_tensor = torch.clamp(actions_tensor, 0, self.action_dim - 1)

                # Forward pass
                action_probs = self.actor(states_tensor)
                values = self.critic(states_tensor).squeeze()

                # Prevent log(0) by clamping probabilities
                action_probs = torch.clamp(action_probs, min=1e-8, max=1.0)

                # Calculate log probabilities
                log_probs = torch.log(action_probs.gather(1, actions_tensor.unsqueeze(1))).squeeze()

                # Calculate advantages (simplified)
                advantages = rewards_tensor - values.detach()

                # Policy loss (REINFORCE-style)
                policy_loss = -(log_probs * advantages).mean()

                # Value loss
                value_loss = F.mse_loss(values, rewards_tensor)

                # Combined loss for this task
                task_loss = policy_loss + 0.5 * value_loss

                # Backward pass (accumulates gradients)
                task_loss.backward()

                total_meta_loss += task_loss.item()

            except Exception as e:
                logger.warning(f"Error processing task {task_idx}: {str(e)}")
                continue

        if total_tasks > 0:
            # Average the accumulated gradients
            for param in self.actor.parameters():
                if param.grad is not None:
                    param.grad.data.div_(total_tasks)

            for param in self.critic.parameters():
                if param.grad is not None:
                    param.grad.data.div_(total_tasks)

            for param in self.trajectory_encoder.parameters():
                if param.grad is not None:
                    param.grad.data.div_(total_tasks)

            # Update all networks
            self.actor_optimizer.step()
            self.critic_optimizer.step()
            self.meta_optimizer.step()

            avg_meta_loss = total_meta_loss / total_tasks
            logger.info(f"Meta-update completed for {total_tasks} tasks, avg_loss={avg_meta_loss:.4f}")
        else:
            logger.warning("No valid tasks processed in meta-update")



class MetaActorNetwork(nn.Module):
    """
    Actor network that takes trajectory embeddings as additional context.
    Used in Meta-PPO for conditioning on task history.
    """

    def __init__(self, input_dim: int, action_dim: int, hidden_dim: int = 128):
        super(MetaActorNetwork, self).__init__()

        self.fc1 = nn.Linear(input_dim, hidden_dim)
        self.fc2 = nn.Linear(hidden_dim, hidden_dim)
        self.fc3 = nn.Linear(hidden_dim, action_dim)

        # Initialize weights
        self._initialize_weights()

        # Move to device
        self.to(DEVICE)

    def _initialize_weights(self):
        """Initialize network weights using Xavier initialization."""
        for layer in [self.fc1, self.fc2, self.fc3]:
            nn.init.xavier_uniform_(layer.weight)
            nn.init.zeros_(layer.bias)

    def forward(self, augmented_state: torch.Tensor) -> torch.Tensor:
        """Forward pass with state + trajectory embedding."""
        x = F.relu(self.fc1(augmented_state))
        x = F.relu(self.fc2(x))
        action_logits = self.fc3(x)

        # Apply softmax to get action probabilities
        action_probs = F.softmax(action_logits, dim=-1)

        return action_probs


class MetaCriticNetwork(nn.Module):
    """
    Critic network that takes trajectory embeddings as additional context.
    Used in Meta-PPO for conditioning on task history.
    """

    def __init__(self, input_dim: int, hidden_dim: int = 128):
        super(MetaCriticNetwork, self).__init__()

        self.fc1 = nn.Linear(input_dim, hidden_dim)
        self.fc2 = nn.Linear(hidden_dim, hidden_dim)
        self.fc3 = nn.Linear(hidden_dim, 1)

        # Initialize weights
        self._initialize_weights()

        # Move to device
        self.to(DEVICE)

    def _initialize_weights(self):
        """Initialize network weights using Xavier initialization."""
        for layer in [self.fc1, self.fc2, self.fc3]:
            nn.init.xavier_uniform_(layer.weight)
            nn.init.zeros_(layer.bias)

    def forward(self, augmented_state: torch.Tensor) -> torch.Tensor:
        """Forward pass with state + trajectory embedding."""
        x = F.relu(self.fc1(augmented_state))
        x = F.relu(self.fc2(x))
        value = self.fc3(x)

        return value


class ModelRegistry:
    """
    Model registry for managing trained RL models and their versions.
    """

    def __init__(self, storage_path: str = "/tmp/futura-models"):
        self.storage_path = storage_path
        self.models_db = {}

        # Create storage directory
        os.makedirs(storage_path, exist_ok=True)

        logger.info(f"Model registry initialized with storage_path={storage_path}")

    def register_model(self, model_info: Dict[str, Any]):
        """Register a new model in the registry."""
        model_id = model_info['model_id']
        self.models_db[model_id] = model_info
        logger.info(f"Registered model {model_id}")

    def list_models(self) -> List[str]:
        """List all registered model IDs."""
        return list(self.models_db.keys())

    def save_checkpoint(self, model_id: str, checkpoint_data: Dict[str, Any]) -> str:
        """Save a model checkpoint."""
        checkpoint_path = os.path.join(self.storage_path, f"{model_id}_checkpoint.pth")
        torch.save(checkpoint_data, checkpoint_path)
        logger.info(f"Saved checkpoint for {model_id} to {checkpoint_path}")
        return checkpoint_path

    def load_checkpoint(self, model_id: str) -> Optional[Dict[str, Any]]:
        """Load a model checkpoint."""
        checkpoint_path = os.path.join(self.storage_path, f"{model_id}_checkpoint.pth")

        if os.path.exists(checkpoint_path):
            checkpoint = torch.load(checkpoint_path, map_location=DEVICE)
            logger.info(f"Loaded checkpoint for {model_id} from {checkpoint_path}")
            return checkpoint
        else:
            logger.warning(f"No checkpoint found for {model_id}")
            return None


# Convenience functions for reward calculation integration
def calculate_performance_reward(current_metrics: Dict[str, float], target_metrics: Dict[str, float]) -> float:
    """Calculate performance-based reward using v1 formula."""
    from .reward_functions import RewardCalculator

    calculator = RewardCalculator()

    # Convert to expected format
    current_state = {
        'cpu_utilization': current_metrics.get('cpu_utilization', 0.5),
        'memory_utilization': current_metrics.get('memory_utilization', 0.5),
        'request_rate': current_metrics.get('request_rate', 100.0),
        'processing_rate': current_metrics.get('processing_rate', 100.0),
        'p95_latency_ms': current_metrics.get('p95_latency_ms', 200.0),
        'error_rate': current_metrics.get('error_rate', 0.01)
    }

    slo_targets = {
        'p95_latency_ms': target_metrics.get('target_p95_latency_ms', 400.0),
        'error_rate': target_metrics.get('target_error_rate', 0.01),
        'throughput_rps': target_metrics.get('target_request_rate', 150.0)
    }

    # Use empty actions for this calculation
    empty_action = {'horizontal': 0, 'vertical_cpu': 0, 'vertical_memory': 0}

    return calculator.calculate_reward_v1(current_state, empty_action, empty_action, current_state, slo_targets)


def calculate_slo_reward(current_state: Dict[str, float], slo_targets: Dict[str, float]) -> float:
    """Calculate SLO compliance reward."""
    from .reward_functions import RewardCalculator

    calculator = RewardCalculator()
    return calculator._calculate_slo_reward(current_state, slo_targets)


def calculate_efficiency_reward(resource_utilization: Dict[str, float], cost_metrics: Dict[str, float]) -> float:
    """Calculate resource efficiency reward."""
    from .reward_functions import RewardCalculator

    calculator = RewardCalculator()

    # Simple efficiency calculation based on utilization
    cpu_util = resource_utilization.get('cpu_utilization', 0.5)
    memory_util = resource_utilization.get('memory_utilization', 0.5)
    num_replicas = resource_utilization.get('num_replicas', 1)

    # Efficiency is high when utilization is in the sweet spot (60-80%)
    target_util = 0.7
    cpu_efficiency = 1.0 - abs(cpu_util - target_util) / target_util
    memory_efficiency = 1.0 - abs(memory_util - target_util) / target_util

    # Scale penalty based on number of replicas
    scale_penalty = min(0.1, 0.01 * num_replicas)

    return (cpu_efficiency + memory_efficiency) / 2.0 - scale_penalty