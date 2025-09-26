"""
Meta-learning PPO implementation for Futura Engine.

This implements meta-learning capabilities on top of PPO, allowing the agent
to adapt quickly to new environments using trajectory embeddings.
Based on the research paper implementation with production adaptations.
"""

import os
import torch
from torch import nn
import numpy as np
import heapq
import time
import logging
from typing import Dict, List, Tuple, Optional, Any
from datetime import datetime

logger = logging.getLogger(__name__)

# Import from state_action_space and rnn for consistency
try:
    from .state_action_space import StateSpace, ActionSpace, ActionType
    from .rnn import RNNEmbedding
except ImportError:
    # Fallback for direct execution
    from state_action_space import StateSpace, ActionSpace, ActionType
    from rnn import RNNEmbedding


def get_padded_trajectories(trajectories: Dict, state_dim: int = 10, action_dim: int = 1, max_timesteps: int = 100) -> Tuple[torch.Tensor, torch.Tensor, torch.Tensor]:
    """
    Pad variable-sized trajectories for batch processing.

    Args:
        trajectories: Dict of trajectory data
        state_dim: Dimension of state space
        action_dim: Dimension of action space
        max_timesteps: Maximum timesteps per episode

    Returns:
        Tuple of padded (states, actions, rewards) tensors
    """
    batch_size = len(trajectories)
    if batch_size == 0:
        # Empty buffer case
        batch_size = 1
        padded_states = torch.zeros(batch_size, state_dim, max_timesteps)
        padded_actions = torch.zeros(batch_size, action_dim, max_timesteps)
        padded_rewards = torch.zeros(batch_size, 1, max_timesteps)
        return padded_states, padded_actions, padded_rewards

    max_traj_len = max([len(trajectories[traj_key]['states']) for traj_key in trajectories])
    max_traj_len = min(max_traj_len, max_timesteps)

    # Pad sequences in the batch to have equal length
    padded_states = torch.zeros(batch_size, state_dim, max_traj_len)
    padded_actions = torch.zeros(batch_size, action_dim, max_traj_len)
    padded_rewards = torch.zeros(batch_size, 1, max_traj_len)

    for i, traj_key in enumerate(trajectories):
        traj = trajectories[traj_key]
        for j in range(min(len(traj['states']), max_traj_len)):
            for s in range(min(len(traj['states'][j]), state_dim)):
                padded_states[i][s][j] = torch.tensor(traj['states'][j][s])
            # For discrete actions
            padded_actions[i][0][j] = torch.tensor(traj['actions'][j])
            padded_rewards[i][0][j] = torch.tensor(traj['rewards'][j])

    return padded_states, padded_actions, padded_rewards


class MetaActorNetwork(nn.Module):
    """
    Meta-learning Actor Network with trajectory embedding.

    This network uses RNN embeddings of past trajectories to adapt
    the policy to new environments quickly.
    """

    def __init__(
        self,
        input_size: int,
        hidden_size: int,
        output_size: int,
        env_dim: Dict[str, int],
        agent,
        embedding_dim: int = 64,
        max_timesteps: int = 100,
        verbose: bool = True
    ):
        super(MetaActorNetwork, self).__init__()

        self.env_dim = env_dim
        self.agent = agent
        self.max_timesteps = max_timesteps

        # RNN embedding for trajectory encoding
        num_features_per_sample = env_dim['state'] + env_dim['action'] + env_dim['reward']
        self.rnn = RNNEmbedding(
            N=1,
            K=5,  # Buffer size for trajectories
            task='wa',
            num_channels=max_timesteps,
            embedding_dim=embedding_dim,
            verbose=verbose
        )

        # Policy network with embedding input
        self.fc1 = nn.Linear(input_size + self.rnn.embedding_dim, hidden_size)
        self.fc2 = nn.Linear(hidden_size, hidden_size)
        self.fc3 = nn.Linear(hidden_size, output_size)
        self.relu = nn.ReLU()
        self.softmax = nn.Softmax(dim=-1)

        logger.info(f"Initialized Meta Actor with embedding_dim={embedding_dim}")

    def forward(self, input_: torch.Tensor) -> torch.Tensor:
        """Forward pass with trajectory embedding."""
        # Get trajectory embedding from episode buffer
        padded_states, padded_actions, padded_rewards = get_padded_trajectories(
            self.agent.episode_buffer,
            state_dim=self.env_dim['state'],
            action_dim=self.env_dim['action'],
            max_timesteps=self.max_timesteps
        )

        # Encode RL trajectories with RNN
        rnn_input = torch.cat((padded_states, padded_actions, padded_rewards), dim=-2)
        num_sequences = rnn_input.shape[0]
        hidden = self.rnn.init_hidden(num_sequences=num_sequences)
        rnn_output, hidden_state = self.rnn.gru(rnn_input, hidden)

        # Generate embedding from hidden state
        hidden_state = torch.mean(hidden_state, dim=1, keepdim=True)
        hidden_state = hidden_state.view((1, 1, -1))
        embedding = self.rnn.embedding_layer(hidden_state[0][0])
        embedding = self.rnn.relu(embedding)

        # Ensure input is tensor
        if not isinstance(input_, torch.Tensor):
            input_ = torch.FloatTensor(input_)

        # Concatenate input with embedding
        if len(input_.shape) > 1:
            # Batched inputs
            embedding = embedding.reshape(1, -1)
            embedding = embedding.repeat(input_.size(0), 1)
            input_ = torch.cat((input_, embedding), dim=1)
        else:
            # Single input
            input_ = torch.cat((input_, embedding), dim=-1)

        # Forward through policy network
        output = self.relu(self.fc1(input_))
        output = self.relu(self.fc2(output))
        output = self.fc3(output)

        # For discrete actions (Kubernetes scaling), use softmax
        output = self.softmax(output)

        return output


class MetaCriticNetwork(nn.Module):
    """
    Meta-learning Critic Network with trajectory embedding.

    This network uses RNN embeddings to estimate state values
    adapted to the current environment context.
    """

    def __init__(
        self,
        input_size: int,
        hidden_size: int,
        output_size: int,
        env_dim: Dict[str, int],
        agent,
        embedding_dim: int = 64,
        max_timesteps: int = 100,
        verbose: bool = True
    ):
        super(MetaCriticNetwork, self).__init__()

        self.env_dim = env_dim
        self.agent = agent
        self.max_timesteps = max_timesteps

        # RNN embedding for trajectory encoding
        num_features_per_sample = env_dim['state'] + env_dim['action'] + env_dim['reward']
        self.rnn = RNNEmbedding(
            N=1,
            K=5,  # Buffer size for trajectories
            task='wa',
            num_channels=max_timesteps,
            embedding_dim=embedding_dim,
            verbose=verbose
        )

        # Value network with embedding input
        self.fc1 = nn.Linear(input_size + self.rnn.embedding_dim, hidden_size)
        self.fc2 = nn.Linear(hidden_size, hidden_size)
        self.fc3 = nn.Linear(hidden_size, output_size)
        self.relu = nn.ReLU()

        logger.info(f"Initialized Meta Critic with embedding_dim={embedding_dim}")

    def forward(self, input_: torch.Tensor) -> torch.Tensor:
        """Forward pass with trajectory embedding."""
        # Get trajectory embedding from episode buffer
        padded_states, padded_actions, padded_rewards = get_padded_trajectories(
            self.agent.episode_buffer,
            state_dim=self.env_dim['state'],
            action_dim=self.env_dim['action'],
            max_timesteps=self.max_timesteps
        )

        # Encode RL trajectories with RNN
        rnn_input = torch.cat((padded_states, padded_actions, padded_rewards), dim=-2)
        num_sequences = rnn_input.shape[0]
        hidden = self.rnn.init_hidden(num_sequences=num_sequences)
        rnn_output, hidden_state = self.rnn.gru(rnn_input, hidden)

        # Generate embedding from hidden state
        hidden_state = torch.mean(hidden_state, dim=1, keepdim=True)
        hidden_state = hidden_state.view((1, 1, -1))
        embedding = self.rnn.embedding_layer(hidden_state[0][0])
        embedding = self.rnn.relu(embedding)

        # Ensure input is tensor
        if not isinstance(input_, torch.Tensor):
            input_ = torch.FloatTensor(input_)

        # Concatenate input with embedding
        embedding = embedding.reshape(1, -1)
        embedding = embedding.repeat(input_.size(0), 1)
        input_ = torch.cat((input_, embedding), dim=1)

        # Forward through value network
        output = self.relu(self.fc1(input_))
        output = self.relu(self.fc2(output))
        output = self.fc3(output)

        return output


class MetaPPOAgent:
    """
    Meta-learning PPO Agent for Kubernetes resource optimization.

    This agent can quickly adapt to new workloads and environments
    using trajectory embeddings from past experiences.
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
        buffer_size: int = 5,
        buffer_mode: str = "latest",
        max_timesteps: int = 100,
        device: str = "cpu",
        verbose: bool = True
    ):
        self.state_size = state_size
        self.action_size = action_size
        self.device = torch.device(device)
        self.verbose = verbose

        # Environment dimensions for embedding
        self.env_dim = {
            'state': state_size,
            'action': 1,  # Discrete actions
            'reward': 1
        }

        # Hyperparameters
        self.gamma = gamma
        self.clip_epsilon = clip_epsilon
        self.entropy_coeff = entropy_coeff
        self.critic_coeff = critic_coeff
        self.max_timesteps = max_timesteps

        # Initialize meta-learning networks
        self.actor = MetaActorNetwork(
            state_size, hidden_size, action_size, self.env_dim, self,
            max_timesteps=max_timesteps, verbose=verbose
        ).to(self.device)

        self.critic = MetaCriticNetwork(
            state_size, hidden_size, 1, self.env_dim, self,
            max_timesteps=max_timesteps, verbose=verbose
        ).to(self.device)

        # Optimizer for both networks
        self.optimizer = torch.optim.Adam(
            list(self.actor.parameters()) + list(self.critic.parameters()),
            lr=learning_rate
        )

        # Episode buffer for meta-learning
        self.buffer_config = {
            'mode': buffer_mode,  # 'best' or 'latest'
            'buffer_size': buffer_size
        }
        self.clear_episode_buffer()

        # For action sampling
        self.state_space = StateSpace()
        self.action_space = ActionSpace()

        # Training state
        self.training_mode = True
        self.skip_update = False

        # Convergence tracking
        self.num_same_parameter_actor = 0
        self.num_same_parameter_critic = 0
        self.parameter_actor = None
        self.parameter_critic = None

        # Recent rewards for adaptive training
        self.recent_rewards = []

        logger.info(f"Initialized Meta-PPO agent with buffer_size={buffer_size}, mode={buffer_mode}")

    def set_training_mode(self, training: bool):
        """Set training or inference mode."""
        self.training_mode = training
        self.skip_update = not training
        if training:
            self.actor.train()
            self.critic.train()
        else:
            self.actor.eval()
            self.critic.eval()

    def clear_episode_buffer(self):
        """Clear the episode buffer for meta-learning."""
        self.episode_buffer = {}
        self.episode_buffer_rewards = []
        self.episode_buffer_index = 0

    def update_episode_buffer(self, states_ep: List, actions_ep: List, rewards_ep: List, steps_ep: int):
        """
        Update episode buffer with new trajectory.

        Args:
            states_ep: Episode states
            actions_ep: Episode actions
            rewards_ep: Episode rewards
            steps_ep: Number of steps in episode
        """
        assert len(states_ep) == steps_ep
        assert len(actions_ep) == steps_ep
        assert len(rewards_ep) == steps_ep

        reward = np.sum(rewards_ep)

        if self.buffer_config['mode'] == 'best':
            # Keep episodes with highest rewards
            if len(self.episode_buffer) >= self.buffer_config['buffer_size']:
                if reward >= self.episode_buffer_rewards[0]:
                    # Remove episode with least reward
                    removed = heapq.heappop(self.episode_buffer_rewards)
                    del self.episode_buffer[removed]
                else:
                    # Ignore new episode
                    return

            # Add new episode
            heapq.heappush(self.episode_buffer_rewards, reward)
            self.episode_buffer[reward] = {
                'states': states_ep,
                'actions': actions_ep,
                'rewards': rewards_ep
            }

        elif self.buffer_config['mode'] == 'latest':
            # Keep latest episodes
            if len(self.episode_buffer) >= self.buffer_config['buffer_size']:
                # Remove oldest episode
                oldest = self.episode_buffer_rewards[self.episode_buffer_index]
                del self.episode_buffer[oldest]

            self.episode_buffer[self.episode_buffer_index] = {
                'states': states_ep,
                'actions': actions_ep,
                'rewards': rewards_ep
            }

            if len(self.episode_buffer_rewards) < self.buffer_config['buffer_size']:
                heapq.heappush(self.episode_buffer_rewards, self.episode_buffer_index)

            self.episode_buffer_index = (self.episode_buffer_index + 1) % self.buffer_config['buffer_size']

        else:
            raise NotImplementedError(f"Buffer mode {self.buffer_config['mode']} not implemented")

    def get_action(self, state: np.ndarray, deterministic: bool = False) -> Tuple[int, float]:
        """
        Get action from the meta-policy network.

        Args:
            state: Normalized state vector
            deterministic: If True, select argmax action

        Returns:
            Tuple of (action_index, log_probability)
        """
        with torch.no_grad():
            state_tensor = torch.FloatTensor(state).unsqueeze(0).to(self.device)
            action_probs = self.actor(state_tensor)

            if deterministic:
                # For inference, select the action with highest probability
                action = torch.argmax(action_probs, dim=1)
                log_prob = torch.log(action_probs.gather(1, action.unsqueeze(1))).squeeze()
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

        # Convert using action space
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
        advantages = (advantages - advantages.mean()) / (advantages.std() + 1e-8)

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
        Update the Meta-PPO agent using collected trajectory data.

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
        if not self.training_mode or self.skip_update:
            logger.warning("Called update() while not in training mode or updates skipped")
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
                ratios = torch.exp(new_log_probs - old_log_probs_tensor[batch_indices])

                # Compute surrogate losses
                surrogate1 = ratios * advantages_tensor[batch_indices]
                surrogate2 = torch.clamp(ratios, 1 - self.clip_epsilon, 1 + self.clip_epsilon) * advantages_tensor[batch_indices]

                # Actor loss (PPO clipped objective)
                actor_loss = -torch.min(surrogate1, surrogate2).mean()

                # Critic loss (value function loss)
                critic_loss = (returns_tensor[batch_indices] - values).pow(2).mean()

                # Total loss
                loss = actor_loss + self.critic_coeff * critic_loss - self.entropy_coeff * entropy

                # Optimization step
                self.optimizer.zero_grad()
                loss.backward()
                torch.nn.utils.clip_grad_norm_(list(self.actor.parameters()) + list(self.critic.parameters()), 0.5)
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

        logger.debug(f"Meta-PPO update completed: {metrics}")
        return metrics

    def save_model(self, filepath: str):
        """Save the model state."""
        torch.save({
            'actor_state_dict': self.actor.state_dict(),
            'critic_state_dict': self.critic.state_dict(),
            'optimizer_state_dict': self.optimizer.state_dict(),
            'episode_buffer': self.episode_buffer,
            'buffer_config': self.buffer_config,
            'config': {
                'state_size': self.state_size,
                'action_size': self.action_size,
                'gamma': self.gamma,
                'clip_epsilon': self.clip_epsilon,
                'entropy_coeff': self.entropy_coeff,
                'critic_coeff': self.critic_coeff,
                'max_timesteps': self.max_timesteps
            }
        }, filepath)
        logger.info(f"Meta-PPO model saved to {filepath}")

    def load_model(self, filepath: str):
        """Load the model state."""
        if not os.path.exists(filepath):
            raise FileNotFoundError(f"Model file not found: {filepath}")

        checkpoint = torch.load(filepath, map_location=self.device)

        self.actor.load_state_dict(checkpoint['actor_state_dict'])
        self.critic.load_state_dict(checkpoint['critic_state_dict'])
        self.optimizer.load_state_dict(checkpoint['optimizer_state_dict'])

        # Restore episode buffer if available
        if 'episode_buffer' in checkpoint:
            self.episode_buffer = checkpoint['episode_buffer']
        if 'buffer_config' in checkpoint:
            self.buffer_config = checkpoint['buffer_config']

        logger.info(f"Meta-PPO model loaded from {filepath}")

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
            state_tensor = torch.FloatTensor(state).unsqueeze(0).to(self.device)
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