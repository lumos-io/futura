"""
Proximal Policy Optimization (PPO) implementation for Futura Engine.

This is the core RL algorithm from the research paper, adapted for the
Futura engine architecture with proper integration for production use.
"""

import os
import torch
from torch import nn
import numpy as np
import logging
from typing import Dict, List, Tuple, Optional, Any
from datetime import datetime

logger = logging.getLogger(__name__)

# Import from state_action_space for consistency
try:
    from .state_action_space import StateSpace, ActionSpace, ActionType
except ImportError:
    # Fallback for direct execution
    from state_action_space import StateSpace, ActionSpace, ActionType


class ActorNetwork(nn.Module):
    """
    PPO Actor Network - Policy network that outputs action probabilities.

    From the research paper: Uses a simple 3-layer fully connected network
    to map state observations to action probabilities.
    """

    def __init__(self, input_size: int, hidden_size: int, output_size: int):
        super(ActorNetwork, self).__init__()

        self.fc1 = nn.Linear(input_size, hidden_size)
        self.fc2 = nn.Linear(hidden_size, hidden_size)
        self.fc3 = nn.Linear(hidden_size, output_size)
        self.relu = nn.ReLU()
        self.softmax = nn.Softmax(dim=-1)

    def forward(self, input_):
        """Forward pass through the actor network."""
        if not isinstance(input_, torch.Tensor):
            input_ = torch.FloatTensor(input_)

        output = self.relu(self.fc1(input_))
        output = self.relu(self.fc2(output))
        output = self.fc3(output)

        # For discrete actions (Kubernetes scaling), use softmax
        output = self.softmax(output)

        return output


class CriticNetwork(nn.Module):
    """
    PPO Critic Network - Value function that estimates state values.

    From the research paper: Estimates the value of being in a particular
    state to help with advantage calculation.
    """

    def __init__(self, input_size: int, hidden_size: int, output_size: int = 1):
        super(CriticNetwork, self).__init__()

        self.fc1 = nn.Linear(input_size, hidden_size)
        self.fc2 = nn.Linear(hidden_size, hidden_size)
        self.fc3 = nn.Linear(hidden_size, output_size)
        self.relu = nn.ReLU()

    def forward(self, input_):
        """Forward pass through the critic network."""
        if not isinstance(input_, torch.Tensor):
            input_ = torch.FloatTensor(input_)

        output = self.relu(self.fc1(input_))
        output = self.relu(self.fc2(output))
        output = self.fc3(output)

        return output


class PPOAgent:
    """
    PPO Agent for Kubernetes resource optimization.

    This implements the PPO algorithm from the research paper, adapted
    for production use in the Futura engine.
    """

    def __init__(
        self,
        state_size: int = 10,
        action_size: int = 7,
        hidden_size: int = 64,
        learning_rate: float = 3e-4,
        gamma: float = 0.99,
        clip_epsilon: float = 0.2,
        entropy_coeff: float = 0.01,
        critic_coeff: float = 0.05,
        device: str = "cpu"
    ):
        self.state_size = state_size
        self.action_size = action_size
        self.device = torch.device(device)

        # Hyperparameters from the research paper
        self.gamma = gamma
        self.clip_epsilon = clip_epsilon
        self.entropy_coeff = entropy_coeff
        self.critic_coeff = critic_coeff

        # Initialize networks
        self.actor = ActorNetwork(
            state_size, hidden_size, action_size).to(self.device)
        self.critic = CriticNetwork(state_size, hidden_size, 1).to(self.device)

        # Optimizer for both networks
        self.optimizer = torch.optim.Adam(
            list(self.actor.parameters()) + list(self.critic.parameters()),
            lr=learning_rate
        )

        # For action sampling
        self.state_space = StateSpace()
        self.action_space = ActionSpace()

        # Training state
        self.training_mode = True

        logger.info(
            f"Initialized PPO agent with state_size={state_size}, action_size={action_size}")

    def set_training_mode(self, training: bool):
        """Set training or inference mode."""
        self.training_mode = training
        if training:
            self.actor.train()
            self.critic.train()
        else:
            self.actor.eval()
            self.critic.eval()

    def get_action(self, state: np.ndarray, deterministic: bool = False) -> Tuple[int, float]:
        """
        Get action from the policy network.

        Args:
            state: Normalized state vector (10-dimensional)
            deterministic: If True, select argmax action (for inference)

        Returns:
            Tuple of (action_index, log_probability)
        """
        with torch.no_grad():
            state_tensor = torch.FloatTensor(
                state).unsqueeze(0).to(self.device)
            action_probs = self.actor(state_tensor)

            if deterministic:
                # For inference, select the action with highest probability
                action = torch.argmax(action_probs, dim=1)
                log_prob = torch.log(action_probs.gather(
                    1, action.unsqueeze(1))).squeeze()
            else:
                # For training, sample from the distribution
                dist = torch.distributions.Categorical(action_probs)
                action = dist.sample()
                log_prob = dist.log_prob(action)

            return action.item(), log_prob.item()

    def get_action_and_value(self, state: np.ndarray) -> Tuple[int, float, float, torch.Tensor]:
        """
        Get action, log probability, value, and action probabilities for training.

        Args:
            state: Normalized state vector

        Returns:
            Tuple of (action, log_prob, value, action_probs)
        """
        state_tensor = torch.FloatTensor(state).unsqueeze(0).to(self.device)

        action_probs = self.actor(state_tensor)
        value = self.critic(state_tensor)

        dist = torch.distributions.Categorical(action_probs)
        action = dist.sample()
        log_prob = dist.log_prob(action)

        return action.item(), log_prob.item(), value.item(), action_probs

    def convert_action_to_k8s_action(self, action_index: int, current_state: Dict[str, float]) -> Dict[str, int]:
        """
        Convert RL action index to Kubernetes scaling action.

        Args:
            action_index: Action index from the policy (0-6)
            current_state: Current resource state

        Returns:
            Dictionary with scaling actions
        """
        action_type = ActionType(action_index)

        action_dict = {
            'vertical_cpu': 0,
            'vertical_memory': 0,
            'horizontal': 0
        }

        # Convert using action space from state_action_space.py
        if action_type == ActionType.NO_ACTION:
            pass  # No change
        elif action_type == ActionType.HORIZONTAL_UP:
            action_dict['horizontal'] = 1
        elif action_type == ActionType.HORIZONTAL_DOWN:
            action_dict['horizontal'] = -1
        elif action_type == ActionType.VERTICAL_CPU_UP:
            action_dict['vertical_cpu'] = 256  # milliCPU
        elif action_type == ActionType.VERTICAL_CPU_DOWN:
            action_dict['vertical_cpu'] = -256
        elif action_type == ActionType.VERTICAL_MEMORY_UP:
            action_dict['vertical_memory'] = 256  # MiB
        elif action_type == ActionType.VERTICAL_MEMORY_DOWN:
            action_dict['vertical_memory'] = -256

        return action_dict

    def compute_advantages(self, rewards: List[float], values: List[float], next_value: float = 0.0) -> Tuple[np.ndarray, np.ndarray]:
        """
        Compute Generalized Advantage Estimation (GAE).

        Args:
            rewards: List of rewards
            values: List of state values
            next_value: Value of the final state

        Returns:
            Tuple of (advantages, returns)
        """
        advantages = []
        returns = []

        gae = 0
        values = values + [next_value]

        for i in reversed(range(len(rewards))):
            delta = rewards[i] + self.gamma * values[i + 1] - values[i]
            gae = delta + self.gamma * 0.95 * gae  # lambda = 0.95 for GAE
            advantages.insert(0, gae)
            returns.insert(0, gae + values[i])

        advantages = np.array(advantages, dtype=np.float32)
        returns = np.array(returns, dtype=np.float32)

        # Normalize advantages
        advantages = (advantages - advantages.mean()) / \
            (advantages.std() + 1e-8)

        return advantages, returns

    def update(
        self,
        states: List[np.ndarray],
        actions: List[int],
        old_log_probs: List[float],
        rewards: List[float],
        advantages: np.ndarray,
        returns: np.ndarray,
        epochs: int = 5,
        batch_size: int = 32
    ) -> Dict[str, float]:
        """
        Update the PPO agent using collected trajectory data.

        Args:
            states: List of state observations
            actions: List of action indices
            old_log_probs: List of old log probabilities
            rewards: List of rewards
            advantages: Computed advantages
            returns: Computed returns
            epochs: Number of optimization epochs
            batch_size: Mini-batch size

        Returns:
            Dictionary with training metrics
        """
        if not self.training_mode:
            logger.warning("Called update() while not in training mode")
            return {}

        # Convert to tensors
        states_tensor = torch.FloatTensor(np.array(states)).to(self.device)
        actions_tensor = torch.LongTensor(actions).to(self.device)
        old_log_probs_tensor = torch.FloatTensor(old_log_probs).to(self.device)
        advantages_tensor = torch.FloatTensor(advantages).to(self.device)
        returns_tensor = torch.FloatTensor(returns).to(self.device)

        total_loss = 0
        actor_losses = []
        critic_losses = []
        entropy_losses = []

        for epoch in range(epochs):
            # Create mini-batches
            dataset_size = len(states)
            indices = np.random.permutation(dataset_size)

            for start_idx in range(0, dataset_size, batch_size):
                end_idx = min(start_idx + batch_size, dataset_size)
                batch_indices = indices[start_idx:end_idx]

                # Get current predictions
                action_probs = self.actor(states_tensor[batch_indices])
                values = self.critic(states_tensor[batch_indices]).squeeze()

                # Compute new log probabilities and entropy
                dist = torch.distributions.Categorical(action_probs)
                new_log_probs = dist.log_prob(actions_tensor[batch_indices])
                entropy = dist.entropy().mean()

                # Compute probability ratios
                ratios = torch.exp(
                    new_log_probs - old_log_probs_tensor[batch_indices])

                # Compute surrogate losses
                surrogate1 = ratios * advantages_tensor[batch_indices]
                surrogate2 = torch.clamp(
                    ratios, 1 - self.clip_epsilon, 1 + self.clip_epsilon) * advantages_tensor[batch_indices]

                # Actor loss (PPO clipped objective)
                actor_loss = -torch.min(surrogate1, surrogate2).mean()

                # Critic loss (value function loss)
                critic_loss = (
                    returns_tensor[batch_indices] - values).pow(2).mean()

                # Total loss
                loss = actor_loss + self.critic_coeff * \
                    critic_loss - self.entropy_coeff * entropy

                # Optimization step
                self.optimizer.zero_grad()
                loss.backward()
                torch.nn.utils.clip_grad_norm_(
                    list(self.actor.parameters()) + list(self.critic.parameters()), 0.5)
                self.optimizer.step()

                # Track metrics
                total_loss += loss.item()
                actor_losses.append(actor_loss.item())
                critic_losses.append(critic_loss.item())
                entropy_losses.append(entropy.item())

        metrics = {
            'total_loss': total_loss / (epochs * (dataset_size // batch_size + 1)),
            'actor_loss': np.mean(actor_losses),
            'critic_loss': np.mean(critic_losses),
            'entropy': np.mean(entropy_losses)
        }

        logger.debug(f"PPO update completed: {metrics}")
        return metrics

    def save_model(self, filepath: str):
        """Save the model state."""
        torch.save({
            'actor_state_dict': self.actor.state_dict(),
            'critic_state_dict': self.critic.state_dict(),
            'optimizer_state_dict': self.optimizer.state_dict(),
            'config': {
                'state_size': self.state_size,
                'action_size': self.action_size,
                'gamma': self.gamma,
                'clip_epsilon': self.clip_epsilon,
                'entropy_coeff': self.entropy_coeff,
                'critic_coeff': self.critic_coeff
            }
        }, filepath)
        logger.info(f"Model saved to {filepath}")

    def load_model(self, filepath: str):
        """Load the model state."""
        if not os.path.exists(filepath):
            raise FileNotFoundError(f"Model file not found: {filepath}")

        checkpoint = torch.load(filepath, map_location=self.device)

        self.actor.load_state_dict(checkpoint['actor_state_dict'])
        self.critic.load_state_dict(checkpoint['critic_state_dict'])
        self.optimizer.load_state_dict(checkpoint['optimizer_state_dict'])

        logger.info(f"Model loaded from {filepath}")

    def evaluate_action(self, state: np.ndarray, action: int) -> Tuple[float, float]:
        """
        Evaluate an action in a given state (for training).

        Args:
            state: State observation
            action: Action index

        Returns:
            Tuple of (log_probability, state_value)
        """
        with torch.no_grad():
            state_tensor = torch.FloatTensor(
                state).unsqueeze(0).to(self.device)
            action_probs = self.actor(state_tensor)
            value = self.critic(state_tensor)

            dist = torch.distributions.Categorical(action_probs)
            log_prob = dist.log_prob(torch.tensor([action]).to(self.device))

            return log_prob.item(), value.item()


def calc_gae(rewards: List[List[float]], gamma: float = 0.99, lambda_: float = 0.95) -> torch.Tensor:
    """
    Calculate Generalized Advantage Estimation for episode rewards.

    Args:
        rewards: List of episode rewards
        gamma: Discount factor
        lambda_: GAE lambda parameter

    Returns:
        Tensor of returns
    """
    returns = []
    for episode_rewards in reversed(rewards):
        discounted_return = 0.0
        for reward in reversed(episode_rewards):
            discounted_return = reward + discounted_return * gamma
            returns.insert(0, discounted_return)

    return torch.FloatTensor(returns)
