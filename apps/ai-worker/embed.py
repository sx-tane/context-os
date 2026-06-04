"""
Deterministic, dependency-free text embedding for the ContextOS AI worker.

The embedding is intentionally simple and local-first: it hashes character
trigrams into a fixed-dimension vector and L2-normalises the result. The same
text always produces the same vector, so pipeline and harness runs stay
reproducible without contacting an external model. The ``model`` field is kept
as an extension point so a real transformer can be swapped in later without
changing the request or response contract.
"""

import hashlib  # stable cross-process hashing for the embedding buckets
import math  # square root for L2 normalisation

# EMBED_DIM is the fixed embedding dimension. It is part of the response contract;
# the Go client validates that every returned vector has this length.
EMBED_DIM = 256

# DEFAULT_MODEL labels the deterministic embedding so callers can detect which
# implementation produced a vector. A real model would report its own name here.
DEFAULT_MODEL = "contextos-hashing-v1"


def _trigrams(text: str) -> list[str]:
    """Return the padded character trigrams of a lowercased string."""
    padded = f"  {text.lower()}  "
    return [padded[i : i + 3] for i in range(len(padded) - 2)]


def embed_text(text: str) -> list[float]:
    """Embed a single string into a deterministic L2-normalised vector.

    Each character trigram is hashed into one of ``EMBED_DIM`` buckets and the
    bucket count is incremented, then the vector is L2-normalised. Empty input
    yields a zero vector.
    """
    vector = [0.0] * EMBED_DIM
    if not text.strip():
        return vector  # empty or whitespace-only text has no signal
    for gram in _trigrams(text):
        digest = hashlib.sha1(gram.encode("utf-8")).digest()  # stable bucket selection
        bucket = int.from_bytes(digest[:4], "big") % EMBED_DIM
        sign = 1.0 if digest[4] & 1 else -1.0  # signed hashing reduces collisions
        vector[bucket] += sign
    norm = math.sqrt(sum(component * component for component in vector))
    if norm == 0:
        return vector  # no trigrams contributed any signal
    return [component / norm for component in vector]


def embed_texts(texts: list[str]) -> list[list[float]]:
    """Embed many strings, preserving input order."""
    return [embed_text(text) for text in texts]
