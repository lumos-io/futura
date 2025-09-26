"""
RNN Embedding implementation for Futura Engine.

This module provides RNN-based embeddings for meta-learning in the
Kubernetes autoscaling context. It encodes trajectory sequences
to enable quick adaptation to new workloads and environments.
"""

import math
import torch
import torch.nn as nn
import torch.nn.functional as F
import numpy as np
import logging
from typing import Dict, List, Tuple, Optional, Any

logger = logging.getLogger(__name__)

# Import blocks for attention and temporal convolution
try:
    from .blocks import AttentionBlock, TCBlock
except ImportError:
    # Fallback for direct execution
    from blocks import AttentionBlock, TCBlock


class RNNEmbedding(nn.Module):
    """
    RNN-based embedding for trajectory encoding in meta-learning.

    This module encodes sequences of states, actions, and rewards
    to generate embeddings that capture the dynamics of different
    workloads and environments.
    """

    def __init__(
        self,
        N: int,
        K: int,
        task: str,
        num_channels: int = 10,
        embedding_dim: int = 64,
        hidden_size: int = 128,
        num_layers: int = 2,
        use_cuda: bool = False,
        verbose: bool = True
    ):
        """
        Initialize RNN embedding.

        Args:
            N: N-way (N classes, N=1 for non-classification tasks)
            K: K-shot (K samples used to generate per embedding)
            task: Task type ('wa' for workload adaptation)
            num_channels: Dimension of input features per sample
            embedding_dim: Output embedding dimension
            hidden_size: RNN hidden size
            num_layers: Number of RNN layers
            use_cuda: Whether to use CUDA
            verbose: Whether to print debug info
        """
        super(RNNEmbedding, self).__init__()

        if task == 'wa':
            # Workload adaptation task
            if verbose:
                logger.info(f'Task: {task}')
                logger.info(f'Number of features per sample: {num_channels}')
        else:
            raise ValueError(f'Not recognized task: {task}')

        # Configure 2-layer bi-directional RNN
        self.hidden_size = hidden_size
        self.num_layers = num_layers
        self.bidirectional = True
        self.directions = 2 if self.bidirectional else 1

        # GRU for sequence encoding
        self.gru = nn.GRU(
            num_channels,
            self.hidden_size,
            num_layers=self.num_layers,
            batch_first=True,
            bidirectional=self.bidirectional
        )

        # FC layer for embedding generation
        self.embedding_layer = nn.Linear(
            self.hidden_size * self.num_layers * self.directions,
            embedding_dim
        )
        self.relu = nn.ReLU()
        self.embedding_dim = embedding_dim

        self.N = N
        self.K = K
        self.use_cuda = use_cuda

        # Additional FC layers for final output (compatible with original)
        self.fc1 = nn.Linear(
            num_channels * K + embedding_dim,
            256
        )
        self.fc2 = nn.Linear(256, 64)
        self.fc3 = nn.Linear(64, N)

        logger.info(f"Initialized RNN embedding with hidden_size={hidden_size}, embedding_dim={embedding_dim}")

    def forward(self, input_data: torch.Tensor) -> torch.Tensor:
        """
        Forward pass through RNN embedding.

        Args:
            input_data: Input tensor containing trajectory features

        Returns:
            Output tensor with embedded features
        """
        # Extract features for K samples
        num_channels = input_data.shape[-1] // self.K if len(input_data.shape) > 1 else 10

        if isinstance(input_data, (list, tuple)):
            input_tensor = input_data[0]
        else:
            input_tensor = input_data

        if hasattr(input_tensor, 'numpy'):
            input_array = input_tensor.numpy()
        else:
            input_array = np.array(input_tensor)

        # Reshape input for RNN processing
        x = np.array([
            input_array[i * num_channels : (i+1) * num_channels]
            for i in range(min(self.K, len(input_array) // num_channels))
        ])

        # Convert to tensor
        x = torch.FloatTensor(x)
        x = x.view((1, self.K, -1))

        # Initialize hidden state
        hidden = self.init_hidden()

        # GRU forward pass
        output, hidden_state = self.gru(x, hidden)

        # Generate embedding from hidden state
        hidden_state = hidden_state.view((1, 1, -1))
        embedding = self.embedding_layer(hidden_state.squeeze())
        embedding = self.relu(embedding)

        # Concatenate embedding with original input
        if isinstance(input_data, (list, tuple)):
            original_input = input_data[0]
        else:
            original_input = input_data

        if not isinstance(original_input, torch.Tensor):
            original_input = torch.FloatTensor(original_input)

        original_input = original_input.view((1, 1, -1))
        embedding = embedding.view((1, 1, -1))
        combined = torch.cat((embedding, original_input), 2)

        # Final FC layers
        output = self.relu(self.fc1(combined.float()))
        output = self.relu(self.fc2(output))
        output = self.fc3(output)

        return output

    def init_hidden(self, num_sequences: int = 1) -> torch.Tensor:
        """
        Initialize hidden state for RNN.

        Args:
            num_sequences: Number of sequences in batch

        Returns:
            Initialized hidden state tensor
        """
        return torch.zeros((
            self.directions * self.num_layers,
            num_sequences,
            self.hidden_size
        ))

    def encode_trajectory(self, states: List, actions: List, rewards: List) -> torch.Tensor:
        """
        Encode a trajectory into an embedding.

        Args:
            states: List of state observations
            actions: List of actions taken
            rewards: List of rewards received

        Returns:
            Trajectory embedding tensor
        """
        # Combine states, actions, rewards into single sequence
        trajectory_features = []
        for i in range(len(states)):
            feature_vector = list(states[i])
            if isinstance(actions[i], (int, float)):
                feature_vector.append(actions[i])
            else:
                feature_vector.extend(actions[i])
            feature_vector.append(rewards[i])
            trajectory_features.append(feature_vector)

        # Convert to tensor and process through RNN
        x = torch.FloatTensor(trajectory_features)
        x = x.unsqueeze(0)  # Add batch dimension

        hidden = self.init_hidden()
        output, hidden_state = self.gru(x, hidden)

        # Generate embedding from final hidden state
        embedding = self.embedding_layer(hidden_state.view(1, -1))
        embedding = self.relu(embedding)

        return embedding


class SnailEmbedding(nn.Module):
    """
    SNAIL (Simple Neural Attentive meta-Learner) embedding.

    This implements the SNAIL architecture with attention and
    temporal convolution blocks for meta-learning.
    """

    def __init__(
        self,
        N: int,
        K: int,
        task: str,
        num_features_per_shot: int = 10,
        num_config_params: int = 3,
        use_cuda: bool = False
    ):
        """
        Initialize SNAIL embedding.

        Args:
            N: N-way classification (N=1 for regression)
            K: K-shot learning
            task: Task type ('wa' for workload adaptation)
            num_features_per_shot: Features per sample
            num_config_params: Configuration parameters
            use_cuda: Whether to use CUDA
        """
        super(SnailEmbedding, self).__init__()

        if task == 'wa':
            num_channels = num_features_per_shot
        else:
            raise ValueError(f'Not recognized task: {task}')

        # Calculate number of filters for temporal convolution
        num_filters = int(math.ceil(math.log(N * K + 1, 2)))

        # Attention and temporal convolution blocks
        self.attention1 = AttentionBlock(num_channels, 64, 32)
        num_channels += 32

        self.tc1 = TCBlock(num_channels, N * K + 1, 128)
        num_channels += num_filters * 128

        self.attention2 = AttentionBlock(num_channels, 256, 128)
        num_channels += 128

        self.tc2 = TCBlock(num_channels, N * K + 1, 128)
        num_channels += num_filters * 128

        self.attention3 = AttentionBlock(num_channels, 512, 256)
        num_channels += 256

        # Final fully connected layers
        self.fc = nn.Linear(num_channels, N)
        self.fc1 = nn.Linear(
            num_channels * K + num_features_per_shot * K + num_config_params,
            256
        )
        self.fc2 = nn.Linear(256, 64)
        self.fc3 = nn.Linear(64, N)
        self.relu = nn.ReLU()

        self.N = N
        self.K = K
        self.use_cuda = use_cuda
        self.num_features_per_shot = num_features_per_shot
        self.num_config_params = num_config_params

        logger.info(f"Initialized SNAIL embedding with N={N}, K={K}")

    def forward(self, input_data: torch.Tensor) -> torch.Tensor:
        """
        Forward pass through SNAIL embedding.

        Args:
            input_data: Input tensor

        Returns:
            Output tensor
        """
        # Extract features for K samples
        if isinstance(input_data, (list, tuple)):
            input_tensor = input_data[0]
        else:
            input_tensor = input_data

        if hasattr(input_tensor, 'numpy'):
            input_array = input_tensor.numpy()
        else:
            input_array = np.array(input_tensor)

        # Reshape for SNAIL processing
        x = np.array([
            input_array[i * self.num_features_per_shot : (i+1) * self.num_features_per_shot]
            for i in range(self.K)
        ])

        x = torch.FloatTensor(x)
        x = x.view((1, self.K, -1))

        # Apply attention and temporal convolution blocks
        x = self.attention1(x.float())
        x = self.tc1(x)
        x = self.attention2(x)
        x = self.tc2(x)
        x = self.attention3(x)

        # Concatenate with configuration parameters
        x = x.view((1, 1, -1))
        config_params = torch.tensor(
            np.array([[input_array[-self.num_config_params:]]])
        )
        x = torch.cat((x, config_params), 2)

        # Final FC layers
        x = self.relu(self.fc1(x.float()))
        x = self.relu(self.fc2(x))
        x = self.fc3(x)

        return x


class TrajectoryEncoder(nn.Module):
    """
    Standalone trajectory encoder for embedding sequences.

    This provides a clean interface for encoding trajectories
    into fixed-size embeddings for meta-learning.
    """

    def __init__(
        self,
        state_dim: int,
        action_dim: int,
        hidden_size: int = 128,
        embedding_dim: int = 64,
        num_layers: int = 2
    ):
        """
        Initialize trajectory encoder.

        Args:
            state_dim: State space dimension
            action_dim: Action space dimension
            hidden_size: RNN hidden size
            embedding_dim: Output embedding dimension
            num_layers: Number of RNN layers
        """
        super(TrajectoryEncoder, self).__init__()

        self.state_dim = state_dim
        self.action_dim = action_dim
        self.hidden_size = hidden_size
        self.embedding_dim = embedding_dim
        self.num_layers = num_layers

        # Input dimension: state + action + reward
        input_dim = state_dim + action_dim + 1

        # Bidirectional GRU
        self.gru = nn.GRU(
            input_dim,
            hidden_size,
            num_layers=num_layers,
            batch_first=True,
            bidirectional=True
        )

        # Embedding layer
        self.embedding_layer = nn.Linear(hidden_size * 2, embedding_dim)
        self.relu = nn.ReLU()

        logger.info(f"Initialized trajectory encoder with embedding_dim={embedding_dim}")

    def forward(self, states: torch.Tensor, actions: torch.Tensor, rewards: torch.Tensor) -> torch.Tensor:
        """
        Encode trajectory into embedding.

        Args:
            states: State tensor [batch_size, seq_len, state_dim]
            actions: Action tensor [batch_size, seq_len, action_dim]
            rewards: Reward tensor [batch_size, seq_len, 1]

        Returns:
            Trajectory embedding [batch_size, embedding_dim]
        """
        # Concatenate states, actions, rewards
        trajectory = torch.cat([states, actions, rewards], dim=-1)

        # Pass through GRU
        output, hidden = self.gru(trajectory)

        # Use final hidden state for embedding
        # hidden shape: [num_layers * 2, batch_size, hidden_size]
        final_hidden = hidden[-2:].transpose(0, 1).contiguous()  # Last layer, both directions
        final_hidden = final_hidden.view(final_hidden.size(0), -1)  # Flatten

        # Generate embedding
        embedding = self.relu(self.embedding_layer(final_hidden))

        return embedding

    def encode_episode(self, episode: Dict[str, List]) -> torch.Tensor:
        """
        Encode a single episode into embedding.

        Args:
            episode: Episode data with 'states', 'actions', 'rewards'

        Returns:
            Episode embedding
        """
        states = torch.FloatTensor(episode['states']).unsqueeze(0)
        actions = torch.FloatTensor(episode['actions']).unsqueeze(0).unsqueeze(-1)
        rewards = torch.FloatTensor(episode['rewards']).unsqueeze(0).unsqueeze(-1)

        return self.forward(states, actions, rewards).squeeze(0)