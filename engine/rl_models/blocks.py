"""
Neural network building blocks for Futura Engine RL models.

This module provides reusable components for building
meta-learning architectures including attention mechanisms
and temporal convolution blocks.
"""

import math
import numpy as np
import torch
import torch.nn as nn
import torch.nn.functional as F
import logging
from typing import Optional

logger = logging.getLogger(__name__)


class FullyConnectedNet(nn.Module):
    """
    Simple fully connected network for baseline comparisons.
    """

    def __init__(self, input_dim: int = 13, hidden_dim: int = 32, output_dim: int = 8):
        """
        Initialize fully connected network.

        Args:
            input_dim: Input dimension
            hidden_dim: Hidden layer dimension
            output_dim: Output dimension
        """
        super(FullyConnectedNet, self).__init__()
        self.fc1 = nn.Linear(input_dim, hidden_dim)
        self.fc2 = nn.Linear(hidden_dim, output_dim)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """Forward pass."""
        x = self.fc1(x.float())
        x = F.relu(x)
        output = self.fc2(x)
        return output


class CausalConv1d(nn.Module):
    """
    Causal 1D convolution for temporal modeling.

    This ensures that the convolution only depends on past
    and present inputs, not future ones.
    """

    def __init__(
        self,
        in_channels: int,
        out_channels: int,
        kernel_size: int,
        stride: int = 1,
        dilation: int = 1,
        groups: int = 1,
        bias: bool = True
    ):
        """
        Initialize causal convolution.

        Args:
            in_channels: Number of input channels
            out_channels: Number of output channels
            kernel_size: Convolution kernel size
            stride: Convolution stride
            dilation: Dilation factor
            groups: Number of groups for grouped convolution
            bias: Whether to use bias
        """
        super(CausalConv1d, self).__init__()
        self.dilation = dilation
        padding = dilation * (kernel_size - 1)
        self.conv1d = nn.Conv1d(
            in_channels, out_channels, kernel_size, stride,
            padding, dilation, groups, bias
        )

    def forward(self, input_tensor: torch.Tensor) -> torch.Tensor:
        """
        Forward pass through causal convolution.

        Args:
            input_tensor: Input tensor of shape (N, in_channels, T)

        Returns:
            Output tensor of shape (N, out_channels, T)
        """
        out = self.conv1d(input_tensor)
        # Remove future information by trimming the end
        return out[:, :, :-self.dilation]


class DenseBlock(nn.Module):
    """
    Dense block with gated activation for temporal convolution.

    This implements the gated activation mechanism used in
    WaveNet and similar architectures.
    """

    def __init__(
        self,
        in_channels: int,
        dilation: int,
        filters: int,
        kernel_size: int = 2
    ):
        """
        Initialize dense block.

        Args:
            in_channels: Number of input channels
            dilation: Dilation factor for convolution
            filters: Number of filters
            kernel_size: Convolution kernel size
        """
        super(DenseBlock, self).__init__()
        self.causal_conv1 = CausalConv1d(
            in_channels, filters, kernel_size, dilation=dilation
        )
        self.causal_conv2 = CausalConv1d(
            in_channels, filters, kernel_size, dilation=dilation
        )

    def forward(self, input_tensor: torch.Tensor) -> torch.Tensor:
        """
        Forward pass through dense block.

        Args:
            input_tensor: Input tensor of shape (N, in_channels, T)

        Returns:
            Concatenated output with residual connection
        """
        # Gated activation: tanh(W * x) * sigmoid(V * x)
        xf = self.causal_conv1(input_tensor)
        xg = self.causal_conv2(input_tensor)
        # Shape: (N, filters, T)
        activations = torch.tanh(xf) * torch.sigmoid(xg)

        # Residual connection by concatenation
        return torch.cat((input_tensor, activations), dim=1)


class TCBlock(nn.Module):
    """
    Temporal Convolution Block for sequence modeling.

    This stacks multiple dense blocks with increasing dilation
    to capture temporal dependencies at different scales.
    """

    def __init__(self, in_channels: int, seq_length: int, filters: int):
        """
        Initialize temporal convolution block.

        Args:
            in_channels: Number of input channels
            seq_length: Length of input sequences
            filters: Number of filters per dense block
        """
        super(TCBlock, self).__init__()

        # Calculate number of dense blocks needed
        num_blocks = int(math.ceil(math.log(seq_length, 2)))

        self.dense_blocks = nn.ModuleList([
            DenseBlock(
                in_channels + i * filters,
                2 ** (i + 1),  # Exponentially increasing dilation
                filters
            )
            for i in range(num_blocks)
        ])

        logger.debug(f"TCBlock initialized with {num_blocks} dense blocks")

    def forward(self, input_tensor: torch.Tensor) -> torch.Tensor:
        """
        Forward pass through temporal convolution block.

        Args:
            input_tensor: Input tensor of shape (N, T, in_channels)

        Returns:
            Output tensor of shape (N, T, expanded_channels)
        """
        # Transpose to (N, in_channels, T) for convolution
        x = torch.transpose(input_tensor, 1, 2)

        # Apply dense blocks sequentially
        for block in self.dense_blocks:
            x = block(x)

        # Transpose back to (N, T, channels)
        return torch.transpose(x, 1, 2)


class AttentionBlock(nn.Module):
    """
    Multi-head attention block for meta-learning.

    This implements scaled dot-product attention with
    causal masking for autoregressive modeling.
    """

    def __init__(self, in_channels: int, key_size: int, value_size: int):
        """
        Initialize attention block.

        Args:
            in_channels: Input feature dimension
            key_size: Dimension of keys and queries
            value_size: Dimension of values
        """
        super(AttentionBlock, self).__init__()
        self.linear_query = nn.Linear(in_channels, key_size)
        self.linear_keys = nn.Linear(in_channels, key_size)
        self.linear_values = nn.Linear(in_channels, value_size)
        self.sqrt_key_size = math.sqrt(key_size)

        logger.debug(
            f"AttentionBlock initialized: in_channels={in_channels}, key_size={key_size}, value_size={value_size}")

    def forward(self, input_tensor: torch.Tensor) -> torch.Tensor:
        """
        Forward pass through attention block.

        Args:
            input_tensor: Input tensor of shape (N, T, in_channels)
                where N is batch size and T is sequence length

        Returns:
            Output tensor with attention applied, concatenated with input
        """
        batch_size, seq_length, _ = input_tensor.shape

        # Create causal mask (upper triangular)
        mask = np.triu(np.ones((seq_length, seq_length)), k=1).astype(bool)
        mask = torch.BoolTensor(mask)

        if input_tensor.is_cuda:
            mask = mask.cuda()

        # Compute queries, keys, values
        queries = self.linear_query(input_tensor)  # Shape: (N, T, key_size)
        keys = self.linear_keys(input_tensor)      # Shape: (N, T, key_size)
        values = self.linear_values(input_tensor)  # Shape: (N, T, value_size)

        # Scaled dot-product attention
        # Compute attention scores
        scores = torch.bmm(queries, torch.transpose(
            keys, 1, 2))  # Shape: (N, T, T)
        scores = scores / self.sqrt_key_size

        # Apply causal mask
        scores.data.masked_fill_(mask, -float('inf'))

        # Apply softmax to get attention weights
        attention_weights = F.softmax(scores, dim=-1)  # Shape: (N, T, T)

        # Apply attention to values
        # Shape: (N, T, value_size)
        attended_values = torch.bmm(attention_weights, values)

        # Concatenate with input (residual connection)
        output = torch.cat((input_tensor, attended_values), dim=2)
        return output


class MultiHeadAttention(nn.Module):
    """
    Multi-head attention mechanism.

    This extends the basic attention with multiple heads
    for better representation learning.
    """

    def __init__(
        self,
        in_channels: int,
        key_size: int,
        value_size: int,
        num_heads: int = 8,
        dropout: float = 0.1
    ):
        """
        Initialize multi-head attention.

        Args:
            in_channels: Input feature dimension
            key_size: Dimension of keys and queries
            value_size: Dimension of values
            num_heads: Number of attention heads
            dropout: Dropout probability
        """
        super(MultiHeadAttention, self).__init__()
        assert key_size % num_heads == 0
        assert value_size % num_heads == 0

        self.key_size = key_size
        self.value_size = value_size
        self.num_heads = num_heads
        self.head_key_size = key_size // num_heads
        self.head_value_size = value_size // num_heads

        self.linear_query = nn.Linear(in_channels, key_size)
        self.linear_keys = nn.Linear(in_channels, key_size)
        self.linear_values = nn.Linear(in_channels, value_size)
        self.linear_out = nn.Linear(value_size, in_channels)

        self.dropout = nn.Dropout(dropout)
        self.sqrt_key_size = math.sqrt(self.head_key_size)

    def forward(self, input_tensor: torch.Tensor) -> torch.Tensor:
        """
        Forward pass through multi-head attention.

        Args:
            input_tensor: Input tensor of shape (N, T, in_channels)

        Returns:
            Output tensor after multi-head attention
        """
        batch_size, seq_length, in_channels = input_tensor.shape

        # Create causal mask
        mask = np.triu(np.ones((seq_length, seq_length)), k=1).astype(bool)
        mask = torch.BoolTensor(mask)
        if input_tensor.is_cuda:
            mask = mask.cuda()

        # Compute queries, keys, values
        queries = self.linear_query(input_tensor)
        keys = self.linear_keys(input_tensor)
        values = self.linear_values(input_tensor)

        # Reshape for multi-head attention
        queries = queries.view(batch_size, seq_length,
                               self.num_heads, self.head_key_size)
        keys = keys.view(batch_size, seq_length,
                         self.num_heads, self.head_key_size)
        values = values.view(batch_size, seq_length,
                             self.num_heads, self.head_value_size)

        # Transpose for attention computation
        queries = queries.transpose(1, 2)  # (N, num_heads, T, head_key_size)
        keys = keys.transpose(1, 2)        # (N, num_heads, T, head_key_size)
        values = values.transpose(1, 2)    # (N, num_heads, T, head_value_size)

        # Scaled dot-product attention for each head
        scores = torch.matmul(
            queries, keys.transpose(-2, -1)) / self.sqrt_key_size
        scores = scores.masked_fill(
            mask.unsqueeze(0).unsqueeze(0), -float('inf'))

        attention_weights = F.softmax(scores, dim=-1)
        attention_weights = self.dropout(attention_weights)

        attended_values = torch.matmul(attention_weights, values)

        # Concatenate heads
        attended_values = attended_values.transpose(1, 2).contiguous().view(
            batch_size, seq_length, self.value_size
        )

        # Final linear transformation
        output = self.linear_out(attended_values)

        # Residual connection
        return input_tensor + output


class LayerNorm(nn.Module):
    """
    Layer normalization for stabilizing training.
    """

    def __init__(self, features: int, eps: float = 1e-6):
        """
        Initialize layer normalization.

        Args:
            features: Number of features
            eps: Small constant for numerical stability
        """
        super(LayerNorm, self).__init__()
        self.gamma = nn.Parameter(torch.ones(features))
        self.beta = nn.Parameter(torch.zeros(features))
        self.eps = eps

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """Apply layer normalization."""
        mean = x.mean(-1, keepdim=True)
        std = x.std(-1, keepdim=True)
        return self.gamma * (x - mean) / (std + self.eps) + self.beta


class PositionalEncoding(nn.Module):
    """
    Positional encoding for transformer-style architectures.
    """

    def __init__(self, d_model: int, max_len: int = 1000):
        """
        Initialize positional encoding.

        Args:
            d_model: Model dimension
            max_len: Maximum sequence length
        """
        super(PositionalEncoding, self).__init__()

        pe = torch.zeros(max_len, d_model)
        position = torch.arange(0, max_len, dtype=torch.float).unsqueeze(1)
        div_term = torch.exp(torch.arange(
            0, d_model, 2).float() * (-math.log(10000.0) / d_model))

        pe[:, 0::2] = torch.sin(position * div_term)
        pe[:, 1::2] = torch.cos(position * div_term)
        pe = pe.unsqueeze(0).transpose(0, 1)

        self.register_buffer('pe', pe)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Add positional encoding to input.

        Args:
            x: Input tensor of shape (seq_len, batch_size, d_model)

        Returns:
            Input with positional encoding added
        """
        x = x + self.pe[:x.size(0), :]
        return x


class FeedForward(nn.Module):
    """
    Position-wise feed-forward network.
    """

    def __init__(self, d_model: int, d_ff: int, dropout: float = 0.1):
        """
        Initialize feed-forward network.

        Args:
            d_model: Model dimension
            d_ff: Feed-forward dimension
            dropout: Dropout probability
        """
        super(FeedForward, self).__init__()
        self.linear1 = nn.Linear(d_model, d_ff)
        self.linear2 = nn.Linear(d_ff, d_model)
        self.dropout = nn.Dropout(dropout)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """Forward pass through feed-forward network."""
        return self.linear2(self.dropout(F.relu(self.linear1(x))))
